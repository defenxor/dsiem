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

package nesd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/defenxor/dsiem/pkg/vuln"
)

const (
	notFoundMessage string = "no vulnerability found\n"
)

func init() {
	vuln.RegisterExtension(new(Nesd), "Nesd")
}

// Config defins
type Config struct {
	URL              string `json:"url"`
	AllowLowSeverity bool   `json:"allow_low_severity"`
}

type nesdResult struct {
	Cve  string `json:"cve"`
	Risk string `json:"risk"`
	Name string `json:"name"`
}

// Initialize implement iface
func (n *Nesd) Initialize(b []byte) error {
	return json.Unmarshal(b, &n.Cfg)
}

// Nesd is a vuln plugin
type Nesd struct {
	Cfg    Config `json:"cfg"`
	Client *http.Client
	mu     sync.Mutex
}

func (n *Nesd) initializeHTTPClient() {
	n.mu.Lock()
	if n.Client == nil {
		n.Client = &http.Client{}
	}
	n.mu.Unlock()
}

// CheckIPPort implement iface
func (n *Nesd) CheckIPPort(ctx context.Context, ip string, port int) (found bool, results []vuln.Result, err error) {
	n.initializeHTTPClient()

	url := strings.Replace(n.Cfg.URL, "${ip}", ip, 1)
	url = strings.Replace(url, "${port}", strconv.Itoa(port), 1)

	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req = req.WithContext(ctx)

	res, err := n.Client.Do(req)
	if err != nil {
		return
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}
	defer res.Body.Close()

	str := string(body)
	if str == notFoundMessage {
		return
	}

	var result = []nesdResult{}
	err = json.Unmarshal([]byte(str), &result)
	if err != nil {
		return
	}

	for _, v := range result {
		if !n.checkRisk(v.Risk) {
			continue
		}

		s := fmt.Sprintf(`%s - %s`, v.Risk, v.Name)
		if v.Cve != "" {
			s = fmt.Sprintf(`%s ("%s")`, s, v.Cve)
		}
		term := fmt.Sprintf(`%s:%s`, ip, strconv.Itoa(port))
		results = append(results, vuln.Result{Provider: "Nesd", Term: term, Result: s})
		found = true
	}

	return
}

func (n *Nesd) checkRisk(risk string) bool {
	if n.Cfg.AllowLowSeverity {
		return slices.Contains([]string{"Low", "Medium", "High", "Critical"}, risk)
	}

	return slices.Contains([]string{"Medium", "High", "Critical"}, risk)
}
