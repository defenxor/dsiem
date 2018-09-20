package server

import (
	log "dsiem/internal/shared/pkg/logger"
)

type vulnerability struct {
	CVE  string `json:"cve"`
	Risk string `json:"risk"`
	Name string `json:"name"`
}

type vulnerabilities struct {
	v []vulnerability `json:"vulnerability"`
}

func findMatch(ip string, port int) (found bool, vs vulnerabilities) {
	vs = vulnerabilities{}
	for _, v := range vulns.entries {
		if v.Host != ip || v.Port != port {
			continue
		}
		log.Debug("Found match: "+v.CVE+": "+v.Risk+": "+v.Name, 0)
		vs.v = append(vs.v, vulnerability{v.CVE, v.Risk, v.Name})
		found = true
	}
	return found, vs
}
