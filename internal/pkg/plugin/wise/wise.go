package wise

import (
	"context"
	"dsiem/pkg/intel"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
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
	Len   int    `json:len"`
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
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}
	req = req.WithContext(ctx)

	res, err := c.Do(req)
	if err != nil {
		return
	}

	body, err := ioutil.ReadAll(res.Body)
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
		results = append(results, intel.Result{"Wise", ip, r.Field + ": " + r.Value})
		found = true
	}
	return
}
