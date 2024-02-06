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

package xcorrelator

import (
	"context"

	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	"github.com/defenxor/dsiem/internal/pkg/shared/cache"

	"time"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"

	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/defenxor/dsiem/pkg/vuln"
)

var (
	// VulnEnabled mark whether vuln lookup is enabled
	VulnEnabled            bool
	vulnFileGlob           = "vuln_*.json"
	maxSecondToWaitForVuln = time.Duration(5)
	vulnCache              *cache.Cache
	vulns                  VulnSources
	vulnPlugins            = vuln.Checkers
	vulnCheckers           = []vulnChecker{}
)

// VulnSource represents entry in vuln_*.json config file
type VulnSource struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
	Plugin  string `json:"plugin"`
	Config  string `json:"config"`
}

// VulnSources represents collection of VulnSource
type VulnSources struct {
	VulnSources []VulnSource `json:"vuln_sources"`
}

type vulnChecker struct {
	vuln.Checker
	name string
}

// CheckVulnIPPort lookup ip-port pair on vulnerability scan result references
func CheckVulnIPPort(ip string, port int, th *apm.TraceHeader) (found bool, results []vuln.Result) {
	/*
		defer func() {
			if r := recover(); r != nil {
				log.Warn(log.M{Msg: "Panic occurred while checking vulnerability scan result for " + ip})
			}
		}()
	*/
	p := strconv.Itoa(port)
	term := ip + ":" + p

	if res, err := vulnCache.Get(term); err == nil {
		if string(res) == "n/f" {
			log.Debug(log.M{Msg: "Returning vuln cache entry (not found) for " + term})
			return
		}
		err := json.Unmarshal(res, &results)
		if err == nil {
			log.Debug(log.M{Msg: "Returning vuln cache entry (found) for " + term})
			found = true
			return
		}
		log.Debug(log.M{Msg: "Failed to unmarshal vuln cache for " + term})
	}

	// flag to store cache only on successful query
	successQuery := false

	for _, v := range vulnCheckers {
		var tx *apm.Transaction
		if apm.Enabled() && th != nil {
			tx = apm.StartTransaction("Vulnerability Lookup", "Plugin", nil, th)
			tx.SetCustom("Term", term)
			tx.SetCustom("Provider", v.name)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*maxSecondToWaitForVuln)
		f, r, err := v.CheckIPPort(ctx, ip, port)
		if err != nil {
			log.Warn(log.M{Msg: "Error received from vuln checker " + v.name + ": " + err.Error()})
			cancel()
			if apm.Enabled() {
				tx.Result(err.Error())
				tx.End()
			}
			continue
		}
		cancel()
		successQuery = true

		if f {
			found = true
			results = append(results, r...)
		}

		if apm.Enabled() && th != nil {
			if found {
				tx.Result("Vuln found")
			} else {
				tx.Result("Vuln not found")
			}
			tx.End()
		}
	}

	if !successQuery {
		return
	}

	if found {
		b, err := json.Marshal(results)
		if err == nil {
			vulnCache.Set(term, b)
			log.Debug(log.M{Msg: "Storing vuln result for " + term + " in cache"})
		}
	} else {
		vulnCache.Set(term, []byte("n/f"))
		log.Debug(log.M{Msg: "Storing vuln not found result for " + term + " in cache"})
	}
	return

}

// InitVuln initialize vulnerability scan result cross-correlation
func InitVuln(confDir string, cacheDuration int) error {
	p := path.Join(confDir, vulnFileGlob)
	files, err := filepath.Glob(p)
	if err != nil {
		return err
	}
	vulnCache, err = cache.New("vuln", cacheDuration, 0)
	if err != nil {
		return err
	}

	for i := range files {
		var it VulnSources
		file, err := os.Open(files[i])
		if err != nil {
			return err
		}
		defer file.Close()

		byteValue, _ := io.ReadAll(file)
		err = json.Unmarshal(byteValue, &it)
		if err != nil {
			return err
		}
		for j := range it.VulnSources {
			if it.VulnSources[j].Enabled {
				vulns.VulnSources = append(vulns.VulnSources, it.VulnSources[j])
				pPlugin := it.VulnSources[j].Plugin
				p := vulnPlugins.Lookup(pPlugin)
				if p == nil {
					log.Warn(log.M{Msg: "Cannot find vuln plugin " + pPlugin})
					continue
				}
				if err := p.Initialize([]byte(it.VulnSources[j].Config)); err != nil {
					log.Warn(log.M{Msg: "Cannot initialize vuln plugin " + pPlugin + ": " + err.Error()})
					continue
				}
				log.Info(log.M{Msg: "Adding vuln plugin " + pPlugin})
				c := vulnChecker{p, pPlugin}
				vulnCheckers = append(vulnCheckers, c)
			}
		}
	}

	total := len(vulnCheckers)
	if total > 0 {
		VulnEnabled = true
	}
	log.Info(log.M{Msg: "Loaded " + strconv.Itoa(total) + " vulnerability scan result sources."})

	return nil
}
