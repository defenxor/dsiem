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

import { Component, OnInit, ViewChildren, ViewChild, QueryList, OnDestroy } from '@angular/core';
import { sleep, isEmptyOrUndefined } from '../../utilities';
import { ActivatedRoute } from '@angular/router';
import { ElasticsearchService } from '../../elasticsearch.service';
import { HttpClient } from '@angular/common/http';
import { map } from 'rxjs/operators';
import { NgxSpinnerService } from 'ngx-spinner';
import { AlertboxComponent } from './alertbox.component';

@Component({
  templateUrl: './detailalarm.component.html',
})
export class DetailalarmComponent implements OnInit, OnDestroy {

  @ViewChildren('pages') pages: QueryList<any>;
  @ViewChild(AlertboxComponent) private alertBox: AlertboxComponent;

  sub: any;
  alarmID: string;
  stage: number;
  alarm; // type should have been Alarm
  alarmRules = []; // this should be a member of Alarm
  alarmVuln = []; // this should be a member of Alarm
  alarmIntelHits = []; // this should be a member of Alarm
  alarmCustomData = []; // this should be a member of Alarm
  events = [];
  wide: boolean;
  wideEv = [];
  wideAlarmEv = [];
  isProcessingUpdateStatus = false;
  isProcessingUpdateTag = false;
  progressLoading: boolean;
  kibanaUrl: string;
  elasticsearch: string;
  dsiemStatuses: string[] = [];
  dsiemTags: string[] = [];

  constructor(private route: ActivatedRoute, private es: ElasticsearchService, private http: HttpClient,
    private spinner: NgxSpinnerService) { }

  ngOnDestroy() {
    this.sub.unsubscribe();
  }

  ngOnInit() {
    this.sub = this.route.params.subscribe(async params => {
      this.alarmID = params['alarmID'];
      const esAlive = await this.checkES();
      if (esAlive) {
        this.kibanaUrl = this.es.kibana;
        await this.getAlarmDetail(this.alarmID);
        await this.loadDsiemConfig();
      }
    });
  }

  async checkES(): Promise<boolean> {

    let esStatus = await this.es.init();
    while (esStatus.initialized === false) {
      this.alertBox.showAlert('Fail to read or parse esconfig.json: ' +
        esStatus.errMsg + '. Will retry every 10s ..', 'danger', true);
      await sleep(10000);
      esStatus = await this.es.init();
    }

    this.elasticsearch = this.es.getServer();
    this.kibanaUrl = this.es.kibana;
    const esUser = this.es.getUser();
    const label = esUser ? this.elasticsearch + ' as ' + esUser : this.elasticsearch;
    try {
      await this.es.isAvailable();
      this.alertBox.showAlert('Connected to ES ' + label, 'success', true);
      return true;
    } catch (err) {
      this.alertBox.showAlert('Disconnected from ES ' + this.elasticsearch + ': ' + err, 'danger', true);
      this.es.reset();
     }
    return false;
  }

  async loadDsiemConfig() {
    // try to load from both /config and /assets/config, the later being used for testing ng serve
    let out;
    try {
      out = await this.http.get('/config/dsiem_config.json').toPromise();
    } catch (err) {}
    if (typeof out === 'undefined') {
      try {
        out = await this.http.get('./assets/config/dsiem_config.json').toPromise();
      } catch (err) {
        const msg = 'Cannot load dsiem_config.json from server, status and tag changes will be disabled';
        this.alertBox.showAlert(msg, 'danger', false);
      }
    }
    if (typeof out !== 'undefined') {
      this.dsiemTags = out['tags'];
      this.dsiemStatuses = out['status'];
    }
  }

  async getAlarmDetail(alarmID) {
    const that = this;
    // this.spinner.show();
    let tempAlarms;
    try {
      const resp = await this.es.getAlarms(this.es.esIndex, alarmID);
      tempAlarms = resp.hits.hits;
      await Promise.all(tempAlarms.map(async (e) => {
        await Promise.all(e['_source']['rules'].map(async (r) => {
          // if (r['status'] === 'finished') {
          //  r['events_count'] = r['occurrence'];
          // } else {
            const response = await this.es.countEvents(this.es.esIndexAlarmEvent, alarmID, r['stage']);
            r['events_count'] = response.count;
          // }
        }));
      }));
    } catch (err) {
      const msg = 'Error occurred while loading alarm ' + this.alarmID + ': ' + err;
      console.log(msg);
      this.alertBox.showAlert(msg, 'danger', false);
    } finally {
      // this.spinner.hide();
    }
    if (typeof tempAlarms === 'undefined') {
 return;
}
    this.alarm = tempAlarms;
    this.alarm[0].id = this.alarmID;
    for (const element of tempAlarms) {
      this.alarmRules = element._source.rules;
      await that.getEventsDetail(that.alarmID, that.alarmRules[0].stage, null, null, that.alarmRules[0].events_count);
      if (element._source.vulnerabilities) {
        this.alarmVuln = element._source.vulnerabilities;
      }
      if (element._source.intel_hits) {
        this.alarmIntelHits = element._source.intel_hits;
      }
      if (element._source.custom_data) {
        this.alarmCustomData = element._source.custom_data;
      }
    }
  }

  setStatus(rule) {
    if (!isEmptyOrUndefined(rule['status'])) {
      return rule['status'];
    }
    if (isEmptyOrUndefined(rule['status']) && isEmptyOrUndefined(rule['start_time'])) {
      return 'inactive';
    }
    if (isEmptyOrUndefined(rule['status']) && rule['start_time'] > 0) {
      return 'active';
    }
    const deadline = rule['start_time'] + rule['timeout'];
    const now = Math.round((new Date()).getTime() / 1000);
    if (now > deadline) {
      return 'timeout';
    }
  }

  async getEventsDetail(id, stage, from= 0, size= 0, allSize= 0) {
    if (this.progressLoading === true) {
      return this.alertBox.showAlert('Still processing previous search, try again later', 'success', false);
    }
    this.progressLoading = true;
    from = 0;
    size = allSize;
    let ev: any;
    let alev: any;

    try {
      alev = await this.es.getAlarmEventsPagination(this.es.esIndexAlarmEvent, id, stage, from, size);
      this.events = [];
      const elArray = [];
      for (const el of alev['hits']['hits']) {
        elArray.push(el['_source']['event_id']);
      }
      ev = await this.es.getEventsMulti(this.es.esIndexEvent, elArray);
    } catch (err) {
      const msg = 'failed to get events for alarm ' + id + ' stage ' + stage + ': ' + err;
      this.alertBox.showAlert(msg, 'danger', false);
    } finally {
      this.progressLoading = false;
    }
    if (typeof ev !== 'undefined') {
      for (const el of ev['hits']['hits']) {
        this.events.push(el['_source']);
      }
    }
  }

  openDropdown(key, param) {
    if (key === 'alrm-tag-' && this.dsiemTags.length === 0) {
 return;
}
    if (key === 'alrm-status-' && this.dsiemStatuses.length === 0) {
 return;
}
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
      if (this.alarm[0]._source.status === status) {
 return;
}
      this.progressLoading = true;
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
      this.alarm[0].id = _id;
    } catch (err) {
      const msg = 'Error occurred while changing alarm status: ' + err;
      console.log(msg);
      this.alertBox.showAlert(msg, 'danger', false);
    } finally {
      this.isProcessingUpdateStatus = false;
      this.progressLoading = false;
    }
  }

  async changeAlarmTag(_id, tag) {
    try {
      if (this.alarm[0]._source.tag === tag) {
 return;
}
      this.progressLoading = true;
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
      this.alarm[0].id = _id;
      this.isProcessingUpdateTag = false;
    } catch (err) {
      const msg = 'Error occurred while changing alarm tag: ' + err;
      console.log(msg);
      this.alertBox.showAlert(msg, 'danger', false);
    } finally {
      this.progressLoading = false;
    }
  }

  openKibana(index, key, value) {
    index = '\'' + index + '\'';
    let url = this.kibanaUrl + '/app/kibana#/discover?_g=(refreshInterval:(display:Off,pause:!f,value:0)';
    url += '';
    if (key !== '') {
      url += ',time:(from:now-24h,mode:quick,to:now))&_a=(columns:!(_source),filters:!((\'$state\':(store:appState)';
      url += ',meta:(alias:!n,disabled:!f,index:' + index + ',key:' + key + ',negate:!f,params:(query:\'' + value + '\',type:phrase)';
      url += ',type:phrase,value:\'' + value + '\'),query:(match:(' + key + ':(query:\'' + value + '\',type:phrase))))),index:';
      url += index + ',interval:auto,query:(language:lucene,query:\'\'),sort:!(\'@timestamp\',desc))';
    } else {
      // for non dsiem indices, use 1h and don't restrict field
      url += ',time:(from:now-1h,mode:quick,to:now))&_a=(columns:!(_source)';
      url += ',index:' + index + ',interval:auto,query:(language:lucene,query:\'"' + value + '"\'),sort:!(\'@timestamp\',desc))';
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
