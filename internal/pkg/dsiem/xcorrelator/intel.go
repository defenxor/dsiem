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
	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	"github.com/defenxor/dsiem/internal/pkg/shared/cache"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/defenxor/dsiem/pkg/intel"
)

var (
	// IntelEnabled mark whether intel lookup is enabled
	IntelEnabled            bool
	intelCache              *cache.Cache
	intelFileGlob           = "intel_*.json"
	maxSecondToWaitForIntel = time.Duration(5)
	intels                  IntelSources
	intelPlugins            = intel.Checkers
	checkers                = []intelCheckers{}
)

// IntelSource represents entry in intel_*.json config file
type IntelSource struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
	Plugin  string `json:"plugin"`
	Config  string `json:"config"`
}

// IntelSources represents collection of IntelSource
type IntelSources struct {
	IntelSources []IntelSource `json:"intel_sources"`
}

type intelCheckers struct {
	intel.Checker
	name string
}

// CheckIntelIP lookup ip on threat intel references
func CheckIntelIP(ip string, connID uint64, th *apm.TraceHeader) (found bool, results []intel.Result) {
	/*
		defer func() {
			if r := recover(); r != nil {
				log.Warn(log.M{Msg: "Panic occurred while checking intel for " + ip})
			}
		}()
	*/

	term := ip

	if res, err := intelCache.Get(term); err == nil {
		if string(res) == "n/f" {
			log.Debug(log.M{Msg: "Returning intel cache entry (not found) for " + term})
			return
		}
		err := json.Unmarshal(res, &results)
		if err == nil {
			log.Debug(log.M{Msg: "Returning intel cache entry (found) for " + term})
			found = true
			return
		}
		log.Debug(log.M{Msg: "Failed to unmarshal intel cache for " + term})
	}

	// flag to store cache only on successful query
	successQuery := false

	for _, v := range checkers {
		var tx *apm.Transaction
		if apm.Enabled() && th != nil {
			tx = apm.StartTransaction("Threat Intel Lookup", "Plugin", nil, th)
			tx.SetCustom("Term", term)
			tx.SetCustom("Provider", v.name)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*maxSecondToWaitForIntel)
		// ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		f, r, err := v.CheckIP(ctx, term)
		if err != nil {
			log.Warn(log.M{Msg: "Error received from intel checker " + v.name + ": " + err.Error()})
			cancel()
			if apm.Enabled() && th != nil {
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
				tx.Result("Intel found")
			} else {
				tx.Result("Intel not found")
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
			intelCache.Set(term, b)
			// fmt.Println("result: ", string(b))
			log.Debug(log.M{Msg: "Storing intel result for " + term + " in cache"})
		}
	} else {
		intelCache.Set(term, []byte("n/f"))
		log.Debug(log.M{Msg: "Storing intel not found result for " + term + " in cache"})
	}
	return
}

// InitIntel initialize threat intel cross-correlation
func InitIntel(confDir string, cacheDuration int) error {
	p := path.Join(confDir, intelFileGlob)
	files, err := filepath.Glob(p)
	if err != nil {
		return err
	}
	intelCache, err = cache.New("intel", cacheDuration, 0)
	if err != nil {
		return err
	}

	for i := range files {
		var it IntelSources
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
		for j := range it.IntelSources {
			if it.IntelSources[j].Enabled {
				intels.IntelSources = append(intels.IntelSources, it.IntelSources[j])
				pPlugin := it.IntelSources[j].Plugin
				p := intelPlugins.Lookup(pPlugin)
				if p == nil {
					log.Warn(log.M{Msg: "Cannot find intel plugin " + pPlugin})
					continue
				}
				if err := p.Initialize([]byte(it.IntelSources[j].Config)); err != nil {
					log.Warn(log.M{Msg: "Cannot initialize intel plugin " + pPlugin + ": " + err.Error()})
					continue
				}
				log.Info(log.M{Msg: "Adding intel plugin " + pPlugin})
				c := intelCheckers{p, pPlugin}
				checkers = append(checkers, c)
			}
		}
	}

	total := len(checkers)
	if total > 0 {
		IntelEnabled = true
	}
	log.Info(log.M{Msg: "Loaded " + strconv.Itoa(total) + " threat intelligence sources."})

	return nil
}
