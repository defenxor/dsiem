// Copyright (c) 2018 PT Defender Nusa Semesta and contributors, All rights reserved.
//
// This file is part of Dsiem.
//
// Dsiem is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation version 3 of the License.
//
// Dsiem is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Dsiem. If not, see <https://www.gnu.org/licenses/>.

package asset

import (
	"sync"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/defenxor/dsiem/internal/pkg/shared/str"

	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/yl2chen/cidranger"
)

const (
	assetsFileGlob = "assets_*.json"
)

var ranger cidranger.Ranger
var whitelist cidranger.Ranger

// NetworkAsset represents a single entry in assets_*.json config file
type NetworkAsset struct {
	Name        string `json:"name"`
	Cidr        string `json:"cidr"`
	Value       int    `json:"value"`
	Whitelisted bool   `json:"whitelisted"`
}

// NetworkAssets represents collection of NetworkAsset
type NetworkAssets struct {
	NetworkAssets []NetworkAsset `json:"assets"`
}

var assets NetworkAssets

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

var mu = sync.RWMutex{}

// Init read assets from all asset_* files in confDir
func Init(confDir string) error {
	mu.Lock()
	defer mu.Unlock()
	p := path.Join(confDir, assetsFileGlob)
	files, _ := filepath.Glob(p)
	if len(files) == 0 {
		return errors.New("Cannot find asset files in " + p)
	}

	for i := range files {
		var a NetworkAssets
		file, err := os.Open(files[i])
		if err != nil {
			return err
		}
		defer file.Close()

		byteValue, _ := io.ReadAll(file)
		err = json.Unmarshal(byteValue, &a)
		if err != nil {
			log.Info(log.M{Msg: "Cannot unmarshal asset!"})
			return err
		}
		for j := range a.NetworkAssets {
			assets.NetworkAssets = append(assets.NetworkAssets, a.NetworkAssets[j])
		}
	}

	ranger = cidranger.NewPCTrieRanger()
	whitelist = cidranger.NewPCTrieRanger()

	for i := range assets.NetworkAssets {
		cidr := assets.NetworkAssets[i].Cidr
		value := assets.NetworkAssets[i].Value
		name := assets.NetworkAssets[i].Name

		_, net, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Info(log.M{Msg: "Cannot parse " + cidr + "!"})
			return err
		}

		if value == 0 || name == "" {
			return errors.New("value cannot be 0 and name cannot be empty for " + cidr)
		}

		if assets.NetworkAssets[i].Whitelisted {
			_ = whitelist.Insert(newAssetEntry(*net, value, name))
		} else {
			_ = ranger.Insert(newAssetEntry(*net, value, name))
		}
	}

	// total := len(assets.NetworkAssets)
	_, allIPs, _ := net.ParseCIDR("::/0")
	r, _ := ranger.CoveredNetworks(*allIPs)
	ttlAssets := len(r)
	r, _ = whitelist.CoveredNetworks(*allIPs)
	ttlWhitelist := len(r)

	log.Info(log.M{Msg: "Loaded " + strconv.Itoa(ttlAssets) + " host and/or network assets."})
	log.Info(log.M{Msg: "Loaded " + strconv.Itoa(ttlWhitelist) + " whitelisted host and/or network assets."})

	return nil
}

// IsInHomeNet checks if IP is in HOME_NET
func IsInHomeNet(ip string) (bool, error) {
	contains, err := ranger.Contains(net.ParseIP(ip)) // returns true, nil
	return contains, err
}

// IsWhiteListed checks if IP is whitelisted
func IsWhiteListed(ip string) (bool, error) {
	contains, err := whitelist.Contains(net.ParseIP(ip)) // returns true, nil
	return contains, err
}

// GetName returns the asset name
func GetName(ip string) string {
	val := ""
	containingNetworks, err := ranger.ContainingNetworks(net.ParseIP(ip))
	if err != nil || len(containingNetworks) == 0 {
		return val
	}
	// return the one with /32 or /128
	for i := range containingNetworks {
		r := containingNetworks[i].(*assetEntry)
		m := r.ipNet.Mask.String()
		if m == "ffffffff" || m == "ffffffffffffffffffffffffffffffff" {
			val = r.name
			break
		}
	}
	return val
}

// GetValue returns asset value
func GetValue(ip string) int {
	val := 0
	containingNetworks, err := ranger.ContainingNetworks(net.ParseIP(ip))
	if err != nil || len(containingNetworks) == 0 {
		return val
	}
	// return the highest asset value
	for i := range containingNetworks {
		r, ok := containingNetworks[i].(*assetEntry)
		if ok && r.value > val {
			val = r.value
		}
	}
	return val
}

// GetAssetNetworks return the CIDR network that the IP is in
func GetAssetNetworks(ip string) []string {
	val := []string{}
	mu.RLock()
	containingNetworks, err := ranger.ContainingNetworks(net.ParseIP(ip))
	if err != nil || len(containingNetworks) == 0 {
		mu.RUnlock()
		return val
	}
	// return all network string except those with /32
	for i := range containingNetworks {
		r := containingNetworks[i].(*assetEntry)
		m := r.ipNet.Mask.String()
		if m != "ffffffff" && m != "ffffffffffffffffffffffffffffffff" {
			s := r.ipNet.String()
			val = str.AppendUniq(val, s)
		}
	}
	mu.RUnlock()
	return val
}
