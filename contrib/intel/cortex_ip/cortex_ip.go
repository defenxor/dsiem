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

package cortex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/defenxor/dsiem/pkg/intel"
)

func init() {
	intel.RegisterExtension(new(Cortex), "CortexIP")
}

// Cortex is an intel plugin
type Cortex struct {
	Cfg              Config `json:"cfg"`
	EnabledAnalyzers []analyzer
}

// Config defines the struct for plugin configuration
type Config struct {
	URL            string   `json:"url"`
	APIKey         string   `json:"apikey"`
	Analyzers      []string `json:"analyzers"`
	MaxWaitMinutes int      `json:"max_wait_minutes"`
	SkipError      bool     `json:"skip_error"`
}
type cortexIPResult struct {
	Level     string `json:"level"`
	Namespace string `json:"namespace"`
	Predicate string `json:"predicate"`
	ValueStr  string
	// Value returned from Cortex can be string or int
	Value  json.RawMessage `json:"value"`
	JobURL string
}

type submissionResult struct {
	ID           string `json:"id"`
	AnalyzerName string `json:"analyzerName"`
}

type jobstatusResult struct {
	Report struct {
		Success      bool   `json:"success"`
		ErrorMessage string `json:"errorMessage"`
		Summary      struct {
			Taxonomies []cortexIPResult `json:"taxonomies"`
		} `json:"summary"`
	} `json:"report"`
}

type analyzer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Initialize implement iface
func (w *Cortex) Initialize(b []byte) error {
	err := json.Unmarshal(b, &w.Cfg)
	if err != nil {
		return err
	}
	if w.Cfg.URL == "" || w.Cfg.MaxWaitMinutes == 0 || w.Cfg.APIKey == "" {
		return errors.New("cannot find one of the required fields (url, max_wait_minutes, apikey) in config")
	}
	url := w.Cfg.URL + "/api/analyzer/_search"
	c := http.Client{}
	jsonStr := []byte(`{"query": {}}`)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonStr))
	req.Header.Add("Authorization", "Bearer "+w.Cfg.APIKey)
	req.Header.Add("Content-type", "application/json")

	res, err := c.Do(req)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	w.EnabledAnalyzers = []analyzer{}
	err = json.Unmarshal([]byte(body), &w.EnabledAnalyzers)
	if err != nil {
		return err
	}
	if len(w.EnabledAnalyzers) == 0 {
		return errors.New("Can't find enabled analyzers in config file")
	}
	return err
}

// CheckIP implement iface
func (w Cortex) CheckIP(ctx context.Context, ip string) (found bool, results []intel.Result, err error) {

	result := []cortexIPResult{}
	errs := []error{}

	for _, analyzer := range w.Cfg.Analyzers {
		analyzerId := ""
		for _, v := range w.EnabledAnalyzers {
			if v.Name == analyzer {
				analyzerId = v.ID
			}
		}
		url := w.Cfg.URL + "/api/analyzer/" + analyzerId + "/run"
		c := http.Client{}

		sData := `{"data":"${ip}", "dataType": "ip", "tlp": 0}`
		sData = strings.Replace(sData, "${ip}", ip, 1)
		var jsonStr = []byte(sData)

		req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonStr))
		req.Header.Add("Authorization", "Bearer "+w.Cfg.APIKey)
		req.Header.Add("Content-type", "application/json")
		req = req.WithContext(ctx)

		res, e := c.Do(req)
		if e != nil {
			errs = append(errs, e)
			continue
		}

		body, e := ioutil.ReadAll(res.Body)
		if e != nil {
			errs = append(errs, e)
			continue
		}
		res.Body.Close()
		subResult := submissionResult{}
		err = json.Unmarshal([]byte(body), &subResult)
		if err != nil {
			errs = append(errs, e)
		}

		url = w.Cfg.URL + "/api/job/" + subResult.ID + "/waitreport?atMost=" + strconv.Itoa(w.Cfg.MaxWaitMinutes) + "minute"
		req, _ = http.NewRequest(http.MethodGet, url, nil)
		req.Header.Add("Authorization", "Bearer "+w.Cfg.APIKey)
		req = req.WithContext(ctx)

		res, e = c.Do(req)
		if e != nil {
			err = e
			return
		}

		body, e = ioutil.ReadAll(res.Body)
		if e != nil {
			err = e
			return
		}
		res.Body.Close()
		jobResult := jobstatusResult{}
		err = json.Unmarshal([]byte(body), &jobResult)
		if err != nil {
			return
		}
		v := []cortexIPResult{}
		if jobResult.Report.Success {
			for _, u := range jobResult.Report.Summary.Taxonomies {
				v = append(v, cortexIPResult{
					Level:     u.Level,
					Namespace: u.Namespace,
					Predicate: u.Predicate,
					ValueStr:  string(u.Value),
					JobURL:    " - Ref: " + w.Cfg.URL + "/index.html#!/jobs/" + subResult.ID,
				})
			}
		} else if !w.Cfg.SkipError {
			v = append(v, cortexIPResult{
				Level:     "error",
				Namespace: subResult.AnalyzerName,
				Predicate: "Message",
				ValueStr:  jobResult.Report.ErrorMessage,
			})
		}
		for _, g := range v {
			result = append(result, g)
		}
	}

	for _, r := range result {
		// r.ValueStr = strings.ReplaceAll(r.ValueStr, "\"", "")
		s := r.Level + " - " + r.Predicate + ": " + r.ValueStr + r.JobURL
		fmt.Println("appending s: ", s)
		results = append(results, intel.Result{Provider: "CortexIP-" + r.Namespace, Term: ip, Result: s})
		found = true
	}

	return
}
