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

package alarm

import (
	"reflect"
	"strconv"
	"strings"

	xc "github.com/defenxor/dsiem/internal/pkg/dsiem/xcorrelator"
	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	"github.com/defenxor/dsiem/internal/pkg/shared/ip"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/defenxor/dsiem/internal/pkg/shared/str"
)

type vulnSearchTerm struct {
	ip   string
	port string
}

func construcTerms(termsIn []vulnSearchTerm, IPs []string, ports []string, port string) (terms []vulnSearchTerm) {
	terms = termsIn
	for _, z := range IPs {
		if z == "ANY" || z == "HOME_NET" || z == "!HOME_NET" || strings.Contains(z, "/") {
			continue
		}
		for _, y := range ports {
			if y == "ANY" {
				continue
			}
			terms = append(terms, vulnSearchTerm{z, y})
		}
		// also try to use port from last event
		if port != "0" {
			terms = append(terms, vulnSearchTerm{z, port})
		}
	}
	return
}

func asyncVulnCheck(aSource *alarm, srcPort, dstPort int, connID uint64, tx *apm.Transaction) {
	var th *apm.TraceHeader
	if apm.Enabled() && tx != nil {
		defer tx.Recover()
		th = tx.GetTraceContext()
	}

	go func() {
		// this lock is specifically for concurrent access to vuln result
		aSource.VulnMu.Lock()
		defer aSource.VulnMu.Unlock()
		// avoid general lock by copying the alarm source
		var a = &alarm{}
		copyAlarm(a, aSource)
		// record prev value
		pVulnerabilities := a.Vulnerabilities

		// build IP:Port list
		terms := []vulnSearchTerm{}
		for _, v := range a.Rules {
			sIps := str.UniqStringSlice(v.From)
			ports := str.UniqStringSlice(v.PortFrom)
			sPort := strconv.Itoa(srcPort)
			terms = construcTerms(terms, sIps, ports, sPort)
			dIps := str.UniqStringSlice(v.To)
			ports = str.UniqStringSlice(v.PortTo)
			dPort := strconv.Itoa(dstPort)
			terms = construcTerms(terms, dIps, ports, dPort)
		}
		terms = sliceUniqMap(terms)

		for i := range terms {
			log.Debug(log.M{Msg: "Evaluating " + terms[i].ip + ":" + terms[i].port, BId: a.ID, CId: connID})
			// skip existing entries
			alreadyExist := false
			for _, v := range a.Vulnerabilities {
				s := terms[i].ip + ":" + terms[i].port
				if v.Term == s {
					alreadyExist = true
					break
				}
			}
			if alreadyExist {
				log.Debug(log.M{Msg: "vuln checker: " + terms[i].ip + ":" + terms[i].port + " already exist", BId: a.ID, CId: connID})
				continue
			}
			p, err := strconv.Atoi(terms[i].port)
			if err != nil {
				continue
			}
			log.Debug(log.M{Msg: "actually checking vuln for " + terms[i].ip + ":" + terms[i].port, BId: a.ID, CId: connID})

			if found, res := xc.CheckVulnIPPort(terms[i].ip, p, th); found {
				// append to the source too here
				a.Vulnerabilities = append(a.Vulnerabilities, res...)
				aSource.Lock()
				aSource.Vulnerabilities = append(aSource.Vulnerabilities, res...)
				aSource.Unlock()
				log.Info(log.M{Msg: "found vulnerability for " + terms[i].ip + ":" + terms[i].port, CId: connID, BId: a.ID})
			}
		}

		// compare content of slice
		if reflect.DeepEqual(pVulnerabilities, a.Vulnerabilities) {
			return
		}
		updateElasticsearch(a, "AsyncVulnCheck", connID, tx)
	}()
}

func asyncIntelCheck(aSource *alarm, connID uint64, checkPrivateIP bool, tx *apm.Transaction) {
	var th *apm.TraceHeader
	if apm.Enabled() && tx != nil {
		defer tx.Recover()
		th = tx.GetTraceContext()
	}

	go func() {

		// this lock is specifically for concurrent access to intel result
		aSource.IntelMu.Lock()
		defer aSource.IntelMu.Unlock()
		// avoid general locks by copying the alarm source
		var a = &alarm{}
		copyAlarm(a, aSource)

		IPIntel := a.ThreatIntels
		p := append(a.SrcIPs, a.DstIPs...)
		p = str.RemoveDuplicatesUnordered(p)

		// loop over srcips and dstips
		for i := range p {
			// skip private IP unless flag is set
			if !checkPrivateIP {
				priv, err := ip.IsPrivateIP(p[i])
				if priv || err != nil {
					continue
				}
			}

			// skip existing entries
			alreadyExist := false
			for _, v := range a.ThreatIntels {
				if v.Term == p[i] {
					alreadyExist = true
					break
				}
			}
			if alreadyExist {
				continue
			}
			if found, res := xc.CheckIntelIP(p[i], connID, th); found {
				a.ThreatIntels = append(a.ThreatIntels, res...)
				aSource.Lock()
				aSource.ThreatIntels = append(aSource.ThreatIntels, res...)
				aSource.Unlock()
				log.Info(log.M{Msg: "Found intel result for " + p[i], CId: connID, BId: a.ID})
			}
		}

		// compare content of slice
		if reflect.DeepEqual(IPIntel, a.ThreatIntels) {
			return
		}
		updateElasticsearch(a, "AsyncIntelCheck", connID, tx)
	}()
}

func sliceUniqMap(s []vulnSearchTerm) []vulnSearchTerm {
	seen := make(map[vulnSearchTerm]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}
