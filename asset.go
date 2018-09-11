package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"

	"github.com/yl2chen/cidranger"
)

const (
	assetsFileGlob = "assets_*.json"
)

var ranger cidranger.Ranger

type networkAsset struct {
	Name  string `json:"name"`
	Cidr  string `json:"cidr"`
	Value int    `json:"value"`
}

type networkAssets struct {
	NetworkAssets []networkAsset `json:"assets"`
}

var assets networkAssets

type assetEntry struct {
	ipNet net.IPNet
	value int
	name  string
}

func (b *assetEntry) Network() net.IPNet {
	return b.ipNet
}

func newAssetEntry(ipNet net.IPNet, value int, name string) cidranger.RangerEntry {
	return &assetEntry{
		ipNet: ipNet,
		value: value,
		name:  name,
	}
}

func initAssets() error {
	p := path.Join(progDir, confDir, assetsFileGlob)
	files, err := filepath.Glob(p)
	if err != nil {
		return err
	}

	for i := range files {
		var a networkAssets
		if !fileExist(files[i]) {
			return errors.New("Cannot find " + files[i])
		}
		file, err := os.Open(files[i])
		if err != nil {
			return err
		}
		defer file.Close()

		byteValue, _ := ioutil.ReadAll(file)
		err = json.Unmarshal(byteValue, &a)
		if err != nil {
			return err
		}
		for j := range a.NetworkAssets {
			assets.NetworkAssets = append(assets.NetworkAssets, a.NetworkAssets[j])
		}
	}

	ranger = cidranger.NewPCTrieRanger()

	for i := range assets.NetworkAssets {
		cidr := assets.NetworkAssets[i].Cidr
		value := assets.NetworkAssets[i].Value
		name := assets.NetworkAssets[i].Name

		_, net, err := net.ParseCIDR(cidr)
		if err != nil {
			logger.Info("Cannot parse ", cidr, "!")
			return err
		}

		logger.Info("Inserting ", cidr, " network.")
		err = ranger.Insert(newAssetEntry(*net, value, name))
		if err != nil {
			logger.Info("Cannot insert ", cidr, " to HOME_NET!")
			return err
		}
	}
	return nil
}

func isInHomeNet(ip string) (bool, error) {
	contains, err := ranger.Contains(net.ParseIP(ip)) // returns true, nil
	return contains, err
}

func getAssetName(ip string) string {
	val := ""
	containingNetworks, err := ranger.ContainingNetworks(net.ParseIP(ip))
	if err != nil || len(containingNetworks) == 0 {
		return val
	}
	// return the one with /32
	for i := range containingNetworks {
		r := containingNetworks[i].(*assetEntry)
		m := r.ipNet.Mask.String()
		if m == "ffffffff" {
			val = r.name
			break
		}
	}
	return val
}

func getAssetValue(ip string) int {
	val := 0
	containingNetworks, err := ranger.ContainingNetworks(net.ParseIP(ip))
	if err != nil || len(containingNetworks) == 0 {
		return val
	}
	// return the highest asset value
	for i := range containingNetworks {
		r, ok := containingNetworks[i].(*assetEntry)
		if !ok {
			continue
		}
		if r.value > val {
			val = r.value
		}
	}
	return val
}

func getAssetNetworks(ip string) []string {
	val := []string{}
	containingNetworks, err := ranger.ContainingNetworks(net.ParseIP(ip))
	if err != nil || len(containingNetworks) == 0 {
		return val
	}
	// return all network string except those with /32
	for i := range containingNetworks {
		r := containingNetworks[i].(*assetEntry)
		m := r.ipNet.Mask.String()
		if m != "ffffffff" {
			s := r.ipNet.String()
			val = appendStringUniq(val, s)
		}
	}
	return val
}
