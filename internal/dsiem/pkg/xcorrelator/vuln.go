package xcorrelator

import (
	log "dsiem/internal/dsiem/pkg/logger"
	"dsiem/internal/shared/pkg/fs"

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

// InitVuln initialize vulnerability scan result cross-correlation
func InitVuln(confDir string) error {
	p := path.Join(confDir, vulnFileGlob)
	files, err := filepath.Glob(p)
	if err != nil {
		return err
	}

	for i := range files {
		var it vulnSources
		if !fs.FileExist(files[i]) {
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
	log.Info("Loaded "+strconv.Itoa(total)+" vulnerability scan result sources.", 0)

	return nil
}
