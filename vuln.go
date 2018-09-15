package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

const (
	vulnFileGlob = "vulnscan_*.json"
)

var vulnEnabled bool

type vulnSource struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Enabled     bool     `json:"enabled"`
	URL         string   `json:"url"`
	ResultRegex []string `json:"result_regex"`
}

type vulnSources struct {
	VulnSources []vulnSource `json:"vuln_sources"`
}

var vulns vulnSources

func initVuln() error {
	p := path.Join(progDir, confDir, vulnFileGlob)
	files, err := filepath.Glob(p)
	if err != nil {
		return err
	}

	for i := range files {
		var it vulnSources
		if !fileExist(files[i]) {
			return errors.New("Cannot find " + files[i])
		}
		file, err := os.Open(files[i])
		if err != nil {
			return err
		}
		defer file.Close()

		byteValue, _ := ioutil.ReadAll(file)
		err = json.Unmarshal(byteValue, &it)
		if err != nil {
			return err
		}
		for j := range it.VulnSources {
			if it.VulnSources[j].Enabled {
				vulns.VulnSources = append(vulns.VulnSources, it.VulnSources[j])
			}
		}
	}

	total := len(vulns.VulnSources)
	if total > 0 {
		vulnEnabled = true
	}
	logInfo("Loaded "+strconv.Itoa(total)+" vulnerability scan result sources.", 0)

	return nil
}
