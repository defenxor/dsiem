/*
Copyright (c) 2019 PT Defender Nusa Semesta and contributors, All rights reserved.

This file is part of Dsiem.

Dsiem is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation version 3 of the License.

Dsiem is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Dsiem. If not, see <https:www.gnu.org/licenses/>.
*/
import { Component, OnInit, ViewChildren, QueryList, OnDestroy } from '@angular/core';
import { sleep } from '../../utilities';
import { ActivatedRoute } from '@angular/router';
import { ElasticsearchService } from '../../elasticsearch.service';
import { Http } from '@angular/http';
import { NgxSpinnerService } from 'ngx-spinner';

@Component({
  templateUrl: './detailalarm.component.html',
})
export class DetailalarmComponent implements OnInit, OnDestroy {

  @ViewChildren('pages') pages: QueryList<any>;
  sub: any;
  alarmID: string;
  stage: number;
  alarm;
  alarmRules = [];
  alarmVuln = [];
  alarmIntelHits = [];
  events = [];
  wide;
  wideEv = [];
  wideAlarmEv = [];
  isProcessingUpdateStatus = false;
  isProcessingUpdateTag = false;
  progressLoading: boolean;
  kibanaUrl: string;

  constructor(private route: ActivatedRoute, private es: ElasticsearchService, private http: Http,
    private spinner: NgxSpinnerService) { }

  ngOnDestroy() {
    this.sub.unsubscribe();
  }

  ngOnInit() {
    this.kibanaUrl = this.es.kibana;
    this.sub = this.route.params.subscribe(async params => {
      this.alarmID = params['alarmID'];
      try {
        await this.es.isAvailable()
      } catch (err) {
        // TODO: replace this with a better popup
        alert('Cannot access ES server ' + this.es.server + ': ' + err)
        return
      }
      return this.getAlarmDetail(this.alarmID)
    })
  }

  async getAlarmDetail(alarmID) {
    const that = this;
    that.spinner.show();
    let tempAlarms;
    try {
      const resp = await this.es.getAlarms(this.es.esIndex, alarmID);
      tempAlarms = resp.hits.hits;
      await Promise.all(tempAlarms.map(async (e) => {
        await Promise.all(e['_source']['rules'].map(async (r) => {
          if (r['status'] === 'finished') {
            r['events_count'] = r['occurrence'];
          } else {
            const response = await this.es.countEvents(this.es.esIndexAlarmEvent, alarmID, r['stage']);
            r['events_count'] = response.count;
          }
        }));
      }));
    } catch (err) {
      // TODO: Replace this with a better popup
      alert('Cannot load alarm ' + this.alarmID + ': ' + err);
    } finally {
      that.spinner.hide();
    }
    this.alarm = tempAlarms;
    for (const element of tempAlarms) {
      this.alarmRules = element._source.rules;
      await that.getEventsDetail(that.alarmID, that.alarmRules[0].stage, null, null, that.alarmRules[0].events_count);
      if (element._source.vulnerabilities) {
        this.alarmVuln = element._source.vulnerabilities;
      }
      if (element._source.intel_hits) {
        this.alarmIntelHits = element._source.intel_hits;
      }
    };
  }

  isEmptyOrUndefined(v): boolean {
    if (v === '' || v === 0 || v === undefined) return true
  }

  setStatus(rule) {
    if (!this.isEmptyOrUndefined(rule['status'])) {
      return rule['status'];
    }
    if (this.isEmptyOrUndefined(rule['status']) && this.isEmptyOrUndefined(rule['start_time'])) {
      return 'inactive';
    }
    if (this.isEmptyOrUndefined(rule['status']) && rule['start_time'] > 0) {
      return 'active';
    }
    const deadline = rule['start_time'] + rule['timeout'];
    const now = Math.round((new Date()).getTime() / 1000);
    if (now > deadline) {
      return 'timeout';
    }
  }

  async getEventsDetail(id, stage, from= 0, size= 0, allSize= 0) {
    this.progressLoading = true
    from = 0;
    size = allSize;
    let ev: any;
    let alev: any;

    try {
      alev = await this.es.getAlarmEventsPagination(this.es.esIndexAlarmEvent, id, stage, from, size);
      this.events = [];
      let elArray = [];
      for (const el of alev['hits']['hits']) {
        elArray.push(el['_source']['event_id'])
      }
      ev = await this.es.getEventsMulti(this.es.esIndexEvent, elArray)
    } catch (err) {
      throw 'error from getEventsMulti: ' + err      
    } finally {
      this.progressLoading = false
    }
    for (const el of ev['hits']['hits']) {
      this.events.push(el['_source'])
    }
  }

  openDropdown(key, param) {
    document.getElementById(key + param).style.display = 'block';
    document.getElementById('close-' + key + param).style.display = 'block';
  }

  closeDropdown(key, param) {
    document.getElementById(key + param).style.display = 'none';
    document.getElementById('close-' + key + param).style.display = 'none';
  }

  resetHeight(key, alarmID) {
    const a = document.getElementById(key + alarmID).getAttribute('class');
    if ( a.indexOf('open') > -1) {
      this.wide = false;
    } else {
      this.wide = true;
    }
  }

  async changeAlarmStatus(_id, status) {
    try {
      if (this.alarm[0]._source.status === status) { return; }
      this.progressLoading = true
      const res = await this.es.updateAlarmStatusById(this.alarm[0]._source.perm_index, _id, status);
      if (res.result !== 'updated') {
        throw new Error(('index not updated, result: ' + res.result));
      }
      this.isProcessingUpdateStatus = true;
      this.closeDropdown('alrm-status-', this.alarmID);
      this.wide = false;
      await sleep (1000);
      const resp = await this.es.getAlarm(this.alarm[0]._source.perm_index, _id);
      this.alarm[0] = resp;
    } catch (err) {
      // TODO: should alert here too
      console.log('Error occur while changing alarm status: ' + err);
    } finally {
      this.isProcessingUpdateStatus = false;
      this.progressLoading = false
    }
  }

  async changeAlarmTag(_id, tag) {
    try {
      if (this.alarm[0]._source.tag === tag) { return; }
      this.progressLoading = true
      const res = await this.es.updateAlarmTagById(this.alarm[0]._source.perm_index, _id, tag);
      if (res.result !== 'updated') {
        throw new Error(('index not updated, result: ' + res.result));
      }
      this.isProcessingUpdateTag = true;
      this.closeDropdown('alrm-tag-', this.alarmID);
      this.wide = false;
      await sleep (1000);
      const resp = await this.es.getAlarm(this.alarm[0]._source.perm_index, _id);
      this.alarm[0] = resp;
      this.isProcessingUpdateTag = false;
    } catch (err) {
      // TODO: should alert here too
      console.log('Error occur while changing alarm tag: ' + err);
    } finally {
      this.progressLoading = false
    }
  }

  openKibana(index, key, value) {
    let url;
    if (index === 'suricata') {
      url = this.kibanaUrl + '/app/kibana#/discover?_g=(refreshInterval:(display:Off,pause:!f,value:0)';
      url += ',time:(from:now-24h,mode:quick,to:now))&_a=(columns:!(_source),filters:!((\'$state\':(store:appState)';
      url += ',meta:(alias:!n,disabled:!f,index:' + index + ',key:' + key + ',negate:!f,params:(query:\'' + value + '\',type:phrase)';
      url += ',type:phrase,value:\'' + value + '\'),query:(match:(' + key + ':(query:\'' + value + '\',type:phrase))))),index:\'';
      url += index + '-*\',interval:auto,query:(language:lucene,query:\'\'),sort:!(\'@timestamp\',desc))';
    } else {
      url = this.kibanaUrl + '/app/kibana#/discover?_g=(refreshInterval:(display:Off,pause:!f,value:0)';
      url += ',time:(from:now-24h,mode:quick,to:now))&_a=(columns:!(_source),filters:!((\'$state\':(store:appState)';
      url += ',meta:(alias:!n,disabled:!f,index:' + index + ',key:' + key + ',negate:!f,params:(query:\'' + value + '\',type:phrase)';
      url += ',type:phrase,value:\'' + value + '\'),query:(match:(' + key + ':(query:\'' + value + '\',type:phrase))))),index:';
      url += index + ',interval:auto,query:(language:lucene,query:\'\'),sort:!(\'@timestamp\',desc))';
    }

    window.open(url, '_blank');
  }

  openKibanaCorrStage(index, alarmID, stage, indexArray) {
    let url;
    url = this.kibanaUrl + '/app/kibana#/discover?_g=(refreshInterval:(display:Off,pause:!f,value:0)';
    url += ',time:(from:now-24h,mode:quick,to:now))&_a=(columns:!(_source),filters:!((\'$state\':(store:appState)';
    url += ',meta:(alias:!n,disabled:!f,index:' + index + ',key:alarm_id,negate:!f,params:(query:' + alarmID + ',type:phrase)';
    url += ',type:phrase,value:' + alarmID + '),query:(match:(alarm_id:(query:' + alarmID + ',type:phrase))))';
    url += ',(\'$state\':(store:appState),meta:(alias:!n,disabled:!f,index:' + index + ',key:stage,negate:!f';
    url += ',params:(query:' + stage + ',type:phrase),type:phrase,value:\'' + stage + '\')';
    url += ',query:(match:(stage:(query:' + stage + ',type:phrase))))),index:' + index + ',interval:auto';
    url += ',query:(language:lucene,parsed:(match_all:()),query:\'*\',suggestions:!()),sort:!(\'@timestamp\',desc))';
    this.wideAlarmEv[indexArray] = false;
    window.open(url, '_blank');
  }

  resetHeightEv(key, alarmID, index) {
    const a = document.getElementById(key + alarmID).getAttribute('class');
    if (a.indexOf('open') > -1) {
      this.wideEv[index] = false;
    } else {
      this.wideEv[index] = true;
    }
  }

  resetHeightAlarmEv(key, stage, index) {
    const a = document.getElementById(key + stage).getAttribute('class');
    if (a.indexOf('open') > -1) {
      this.wideAlarmEv[index] = false;
    } else {
      this.wideAlarmEv[index] = true;
    }
  }

}
