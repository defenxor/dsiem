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
	"encoding/json"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/rule"
	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/defenxor/dsiem/pkg/intel"
	"github.com/defenxor/dsiem/pkg/vuln"
)

func removeAlarm(id string) {
	alarms.Lock()
	log.Debug(log.M{Msg: "Lock obtained. Removing alarm", BId: id})
	delete(alarms.al, id)
	alarms.Unlock()
}

func removalListener() {
	go func() {
		for {
			// handle incoming event, id should be the ID to remove
			m := <-alarmRemovalChannel
			go removeAlarm(m)
		}
	}()
}

func findOrCreateAlarm(id string) (a *alarm) {
	alarms.RLock()
	for k := range alarms.al {
		if alarms.al[k].ID == id {
			a = alarms.al[id]
			alarms.RUnlock()
			return
		}
	}
	alarms.RUnlock()
	alarms.Lock()
	alarms.al[id] = &alarm{}
	alarms.al[id].ID = id
	a = alarms.al[id]
	alarms.Unlock()
	return
}

// to avoid copying mutex
func copyAlarm(dst *alarm, src *alarm) {
	src.RLock()
	defer src.RUnlock()
	dst.Lock()
	defer dst.Unlock()

	dst.ID = src.ID
	dst.Title = src.Title
	dst.Status = src.Status
	dst.Kingdom = src.Kingdom
	dst.Category = src.Category
	dst.CreatedTime = src.CreatedTime
	dst.UpdateTime = src.UpdateTime
	dst.Risk = src.Risk
	dst.RiskClass = src.RiskClass
	dst.Tag = src.Tag

	dst.SrcIPs = make([]string, len(src.SrcIPs))
	copy(dst.SrcIPs, src.SrcIPs)

	dst.DstIPs = make([]string, len(src.DstIPs))
	copy(dst.DstIPs, src.DstIPs)

	dst.ThreatIntels = make([]intel.Result, len(src.ThreatIntels))
	copy(dst.ThreatIntels, src.ThreatIntels)

	dst.Vulnerabilities = make([]vuln.Result, len(src.Vulnerabilities))
	copy(dst.Vulnerabilities, src.Vulnerabilities)

	dst.Networks = make([]string, len(src.Networks))
	copy(dst.Networks, src.Networks)

	dst.Rules = make([]rule.DirectiveRule, len(src.Rules))
	copy(dst.Rules, src.Rules)
}

func updateElasticsearch(a *alarm, checker string, connID uint64, tx *apm.Transaction) {
	a.RLock()
	defer a.RUnlock()
	if a.Risk == 0 {
		log.Debug(log.M{Msg: "Risk is 0, alarm not updating ES", CId: connID})
		return
	}
	err := logToES(a, connID)
	if err == nil {
		if apm.Enabled() && tx != nil {
			tx.Result("Alarm updated")
			tx.End()
		}
		return
	}
	log.Warn(log.M{Msg: checker + ": failed to update Elasticsearch!" + err.Error(), BId: a.ID, CId: connID})
	if apm.Enabled() && tx != nil {
		tx.Result("Alarm failed to update ES")
		tx.SetError(err)
		tx.End()
	}
}

func logToES(a *alarm, connID uint64) error {
	a.RLock()
	log.Info(log.M{Msg: "alarm updating Elasticsearch", BId: a.ID, CId: connID})
	aJSON, _ := json.Marshal(a)
	a.RUnlock()

	return fWriter.EnqueueWrite(string(aJSON))
}
