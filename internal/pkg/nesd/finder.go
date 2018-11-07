package nesd

import (
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

type vulnerability struct {
	CVE  string `json:"cve"`
	Risk string `json:"risk"`
	Name string `json:"name"`
}

type vulnerabilities struct {
	V []vulnerability `json:"vulnerability"`
}

func findMatch(ip string, port int) (found bool, vs vulnerabilities) {
	vs = vulnerabilities{}
	for _, v := range vulns.entries {
		if v.Host != ip || v.Port != port {
			continue
		}
		log.Debug(log.M{Msg: "Found match: " + v.Risk + " - " + v.Name + " - " + v.CVE})
		vs.V = append(vs.V, vulnerability{v.CVE, v.Risk, v.Name})
		found = true
	}
	return found, vs
}
