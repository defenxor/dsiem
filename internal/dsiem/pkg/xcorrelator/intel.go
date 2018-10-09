package xcorrelator

import (
	"dsiem/internal/shared/pkg/cache"
	"dsiem/internal/shared/pkg/fs"
	log "dsiem/internal/shared/pkg/logger"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/apm-agent-go"
)

const (
	intelFileGlob           = "intel_*.json"
	maxSecondToWaitForIntel = 5
)

// IntelEnabled mark whether intel lookup is enabled
var IntelEnabled bool
var intelCache *cache.Cache

type intelSource struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Enabled     bool     `json:"enabled"`
	URL         string   `json:"url"`
	Matcher     string   `json:"matcher"`
	ResultRegex []string `json:"result_regex"`
}

// IntelResult contain results from threat intel queries
type IntelResult struct {
	Provider string `json:"provider"`
	Term     string `json:"term"`
	Result   string `json:"result"`
}

type intelSources struct {
	IntelSources []intelSource `json:"intel_sources"`
}

var intels intelSources

// CheckIntelIP lookup ip on threat intel references
func CheckIntelIP(ip string, connID uint64) (found bool, results []IntelResult) {
	defer func() {
		if r := recover(); r != nil {
			log.Warn(log.M{Msg: "Panic occurred while checking intel for " + ip})
		}
	}()

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

	// flag to store cache only on succesful query
	successQuery := false

	for _, v := range intels.IntelSources {
		url := strings.Replace(v.URL, "${ip}", ip, 1)
		c := http.Client{Timeout: time.Second * maxSecondToWaitForIntel}
		req, err := http.NewRequest(http.MethodGet, url, nil)

		tx := elasticapm.DefaultTracer.StartTransaction("Threat Intel Lookup", "SIEM")
		tx.Context.SetCustom("term", term)
		tx.Context.SetCustom("provider", v.Name)
		tx.Context.SetCustom("Url", url)

		if err != nil {
			log.Warn(log.M{Msg: "Cannot create new HTTP request for " + v.Name + " TI."})
			tx.Result = "Cannot create HTTP request"
			tx.End()
			continue
		}
		res, err := c.Do(req)
		if err != nil {
			// log.Warn(log.M{Msg: "Failed to query " + v.Name + " TI for IP " + ip + ": " + err.Error()})
			log.Warn(log.M{Msg: "Failed to query " + v.Name + " TI for IP " + ip})
			tx.Result = "Failed to query " + v.Name
			tx.End()
			continue
		}
		body, readErr := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			log.Warn(log.M{Msg: "Cannot read result from " + v.Name + " TI for IP " + ip})
			tx.Result = "Cannot read result from " + v.Name
			tx.End()
			continue
		}

		successQuery = true

		if v.Matcher == "regex" {
			f, r := matcherRegexIntel(body, v.Name, term, v.ResultRegex)
			if f {
				found = true
				results = append(results, r...)
			}
		}

		if found {
			tx.Result = "Intel found"
		} else {
			tx.Result = "Intel not found"
		}
		tx.End()
	}

	if !successQuery {
		return
	}

	if found {
		b, err := json.Marshal(results)
		if err == nil {
			intelCache.Set(term, b)
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
	intelCache, err = cache.New("intel", cacheDuration)
	if err != nil {
		return err
	}

	for i := range files {
		var it intelSources
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
		for j := range it.IntelSources {
			if it.IntelSources[j].Enabled {
				intels.IntelSources = append(intels.IntelSources, it.IntelSources[j])
			}
		}
	}

	total := len(intels.IntelSources)
	if total > 0 {
		IntelEnabled = true
	}
	log.Info(log.M{Msg: "Loaded " + strconv.Itoa(total) + " threat intelligence sources."})

	return nil
}
