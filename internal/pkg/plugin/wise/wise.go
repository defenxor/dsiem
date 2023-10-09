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

package wise

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/defenxor/dsiem/pkg/intel"
)

func init() {
	intel.RegisterExtension(new(Wise), "Wise")
}

// Wise is an intel plugin
type Wise struct {
	Cfg Config `json:"cfg"`
}

// Config defins
type Config struct {
	URL string `json:"url"`
}
type wiseResult struct {
	Field string `json:"field"`
	Len   int    `json:"len"`
	Value string `json:"value"`
}

// Initialize implement iface
func (w *Wise) Initialize(b []byte) error {
	return json.Unmarshal(b, &w.Cfg)
}

// CheckIP implement iface
func (w Wise) CheckIP(ctx context.Context, ip string) (found bool, results []intel.Result, err error) {

	url := strings.Replace(w.Cfg.URL, "${ip}", ip, 1)

	c := http.Client{}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req = req.WithContext(ctx)

	res, err := c.Do(req)
	if err != nil {
		return
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}
	defer res.Body.Close()

	// convert Wise JS object literal to valid JSON
	s := strings.Replace(string(body), "field:", `"field":`, -1)
	s = strings.Replace(s, "len:", `"len":`, -1)
	s = strings.Replace(s, "value:", `"value":`, -1)

	result := []wiseResult{}
	err = json.Unmarshal([]byte(s), &result)
	if err != nil {
		return
	}

	for _, r := range result {
		// len < 5 is ID or metadata, not the actual result text
		if r.Len < 5 {
			continue
		}
		// Example {field:value} returned is:
		// alienvault.activity:Malicious Host
		results = append(results, intel.Result{Provider: "Wise", Term: ip, Result: r.Field + ": " + r.Value})
		found = true
	}
	return
}
