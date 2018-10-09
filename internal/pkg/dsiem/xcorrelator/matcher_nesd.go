package xcorrelator

import (
	log "dsiem/internal/pkg/shared/logger"
	"encoding/json"
)

func matcherNesd(body []byte, provider string, term string) (found bool, results []VulnResult) {
	vResult := string(body)
	if vResult == "no vulnerability found\n" {
		log.Debug(log.M{Msg: "nesd: no vulnerability found for " + term})
		return
	}
	var n = []nesdResult{}
	err := json.Unmarshal([]byte(vResult), &n)
	if err != nil {
		log.Debug(log.M{Msg: "Error unmarshalling nesd result " + err.Error()})
		return
	}
	for _, v := range n {
		if v.Risk != "Medium" && v.Risk != "High" && v.Risk != "Critical" {
			continue
		}
		s := v.Risk + " - " + v.Name
		if v.Cve != "" {
			s = s + " (" + v.Cve + ")"
		}
		results = append(results, VulnResult{provider, term, s})
		found = true
	}
	return
}
