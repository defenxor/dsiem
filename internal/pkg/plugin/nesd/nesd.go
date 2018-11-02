package nesd

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/defenxor/dsiem/pkg/vuln"
)

func init() {
	vuln.RegisterExtension(new(Nesd), "Nesd")
}

// Config defins
type Config struct {
	URL string `json:"url"`
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
	Cfg Config `json:"cfg"`
}

// CheckIPPort implement iface
func (n Nesd) CheckIPPort(ctx context.Context, ip string, port int) (found bool, results []vuln.Result, err error) {

	url := strings.Replace(n.Cfg.URL, "${ip}", ip, 1)
	url = strings.Replace(url, "${port}", strconv.Itoa(port), 1)

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

	str := string(body)
	if str == "no vulnerability found\n" {
		return
	}

	var result = []nesdResult{}
	err = json.Unmarshal([]byte(str), &result)
	if err != nil {
		return
	}

	for _, v := range result {
		if v.Risk != "Medium" && v.Risk != "High" && v.Risk != "Critical" {
			continue
		}
		s := v.Risk + " - " + v.Name
		if v.Cve != "" {
			s = s + " (" + v.Cve + ")"
		}
		term := ip + ":" + strconv.Itoa(port)
		results = append(results, vuln.Result{"Nesd", term, s})
		found = true
	}

	return
}
