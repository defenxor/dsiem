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
