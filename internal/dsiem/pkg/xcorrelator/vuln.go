package xcorrelator

import (
	"dsiem/internal/shared/pkg/apm"
	"dsiem/internal/shared/pkg/cache"

	"dsiem/internal/shared/pkg/fs"
	log "dsiem/internal/shared/pkg/logger"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/apm-agent-go"

	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

const (
	vulnFileGlob           = "vuln_*.json"
	maxSecondToWaitForVuln = 5
)

// VulnEnabled mark whether intel lookup is enabled
var VulnEnabled bool
var vulnCache *cache.Cache

type vulnSource struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Enabled     bool     `json:"enabled"`
	URL         string   `json:"url"`
	Matcher     string   `json:"matcher"`
	ResultRegex []string `json:"result_regex"`
}

type vulnSources struct {
	VulnSources []vulnSource `json:"vuln_sources"`
}

type nesdResult struct {
	Cve  string `json:"cve"`
	Risk string `json:"risk"`
	Name string `json:"name"`
}

// VulnResult contain results from vulnerability scan result queries
type VulnResult struct {
	Provider string `json:"provider"`
	Term     string `json:"term"`
	Result   string `json:"result"`
}

var vulns vulnSources

// CheckVulnIPPort lookup ip-port pair on vulnerability scan result references
func CheckVulnIPPort(ip string, port int) (found bool, results []VulnResult) {
	defer func() {
		if r := recover(); r != nil {
			log.Warn(log.M{Msg: "Panic occurred while checking vulnerability scan result for " + ip})
		}
	}()

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

	// flag to store cache only on succesful query
	successQuery := false

	for _, v := range vulns.VulnSources {
		url := strings.Replace(v.URL, "${ip}", ip, 1)
		url = strings.Replace(url, "${port}", p, 1)
		log.Debug(log.M{Msg: "result url " + url})

		var tx *elasticapm.Transaction
		if apm.Enabled() {
			tx = elasticapm.DefaultTracer.StartTransaction("Vulnerability Lookup", "SIEM")
			tx.Context.SetCustom("term", term)
			tx.Context.SetCustom("provider", v.Name)
			tx.Context.SetCustom("Url", url)
		}

		c := http.Client{Timeout: time.Second * maxSecondToWaitForVuln}
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			log.Warn(log.M{Msg: "Cannot create new HTTP request for " + v.Name + " VS."})
			if apm.Enabled() {
				tx.Result = "Cannot create HTTP request"
				tx.End()
			}
			continue
		}
		res, err := c.Do(req)
		if err != nil {
			log.Warn(log.M{Msg: "Failed to query " + v.Name + " VS for IP " + term})
			if apm.Enabled() {
				tx.Result = "Failed to query " + v.Name
				tx.End()
			}
			continue
		}
		body, readErr := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			log.Warn(log.M{Msg: "Cannot read result from " + v.Name + " VS for IP " + term})
			if apm.Enabled() {
				tx.Result = "Cannot create read result from " + v.Name
				tx.End()
			}
			continue
		}

		successQuery = true

		if v.Matcher == "regex" {
			f, r := matcherRegexVuln(body, v.Name, term, v.ResultRegex)
			if f {
				found = true
				results = append(results, r...)
			}
		}

		if v.Matcher == "nesd" {
			f, r := matcherNesd(body, v.Name, term)
			if f {
				found = true
				results = append(results, r...)
			}
		}
		if apm.Enabled() {
			if found {
				tx.Result = "Vuln found"
			} else {
				tx.Result = "Vuln not found"
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
	vulnCache, err = cache.New("intel", cacheDuration)
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
		VulnEnabled = true
	}
	log.Info(log.M{Msg: "Loaded " + strconv.Itoa(total) + " vulnerability scan result sources."})

	return nil
}
