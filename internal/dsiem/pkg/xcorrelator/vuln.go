package xcorrelator

import (
	"dsiem/internal/shared/pkg/fs"
	log "dsiem/internal/shared/pkg/logger"
	"net/http"
	"regexp"
	"strings"
	"time"

	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

const (
	vulnFileGlob = "vuln_*.json"
)

// VulnEnabled mark whether intel lookup is enabled
var VulnEnabled bool

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
func CheckVulnIPPort(ip string, port int, connID uint64) (found bool, results []VulnResult) {
	defer func() {
		if r := recover(); r != nil {
			log.Warn("Panic occurred while checking vulnerability scan result for "+ip, connID)
		}
	}()

	for _, v := range vulns.VulnSources {
		p := strconv.Itoa(port)
		log.Debug("Checking url "+v.URL+" with ip: "+ip, 0)
		url := strings.Replace(v.URL, "${ip}", ip, 1)
		url = strings.Replace(url, "${port}", p, 1)
		log.Debug("result url "+url, 0)

		c := http.Client{Timeout: time.Second * maxSecondToWaitForIntel}
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			log.Warn("Cannot create new HTTP request for "+v.Name+" VS.", connID)
			continue
		}
		res, err := c.Do(req)
		if err != nil {
			log.Warn("Failed to query "+v.Name+" VS for IP "+ip+":"+p, connID)
			continue
		}
		body, readErr := ioutil.ReadAll(res.Body)
		if readErr != nil {
			log.Warn("Cannot read result from "+v.Name+" VS for IP "+ip+":"+p, connID)
			continue
		}

		strRegex := v.ResultRegex
		vResult := string(body)
		// loop over the strRegex, applying it one by one to vResult
		for _, v := range strRegex {
			if strings.HasPrefix(v, "match:") {
				r := strings.Split(v, ":")
				re := regexp.MustCompile(r[len(r)-1])
				s := re.FindAllString(vResult, -1)
				if s == nil {
					vResult = ""
					break
				}
				vResult = s[len(s)-1]
			}
			if strings.HasPrefix(v, "remove:") {
				r := strings.Split(v, ":")
				re := regexp.MustCompile(r[len(r)-1])
				s := re.ReplaceAllLiteralString(vResult, "")
				if s == "" {
					vResult = ""
					break
				}
				vResult = s
			}
			// special, natively supported, and cheating rule
			if strings.HasPrefix(v, "nesd") {
				if vResult == "no vulnerability found\n" {
					log.Debug("nesd: no vulnerability found for "+ip+":"+strconv.Itoa(port), connID)
					vResult = ""
					continue
				}
				var n = []nesdResult{}
				err := json.Unmarshal([]byte(vResult), &n)
				if err != nil {
					log.Debug("Error unmarshalling nesd result "+err.Error(), connID)
					continue
				}
				vResult = ""
				last := len(n) - 1
				for i, v := range n {
					if v.Risk != "Medium" && v.Risk != "High" && v.Risk != "Critical" {
						continue
					}
					vResult = vResult + "[" + v.Risk + "] " + v.Name
					if v.Cve != "" {
						vResult = vResult + " (" + v.Cve + ")"
					}
					if i != last {
						vResult = vResult + ", "
					}
				}
			}

		}
		vResult = strings.Trim(vResult, " ")
		if vResult == "" {
			continue
		}
		results = append(results, VulnResult{v.Name, ip + ":" + p, vResult})
		// log.Debug("Appending: "+vResult, connID)
		found = true
	}
	return
}

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
		VulnEnabled = true
	}
	log.Info("Loaded "+strconv.Itoa(total)+" vulnerability scan result sources.", 0)

	return nil
}
