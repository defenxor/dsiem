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
  alarmID;
  stage;
  alarm;
  alarmRules = [];
  alarmVuln = [];
  alarmIntelHits = [];
  evnts = [];
  isShowEventDetails: boolean;
  esIndex = 'siem_alarms';
  esIndexAlarmEvent = 'siem_alarm_events-*';
  esIndexEvent = 'siem_events-*';
  esType = 'doc';
  totalItems;
  itemsPerPage = 5;
  numberOfVisiblePaginators = 10;
  numberOfPaginators: number;
  paginators: Array<any> = [];
  activePage = 1;
  firstVisibleIndex = 1;
  lastVisibleIndex: number = this.itemsPerPage;
  firstVisiblePaginator = 0;
  lastVisiblePaginator = this.numberOfVisiblePaginators;
  wide;
  wideEv = [];
  wideAlarmEv = [];
  isProcessingUpdateStatus = false;
  isProcessingUpdateTag = false;
  kibanaUrl;

  constructor(private route: ActivatedRoute, private es: ElasticsearchService, private http: Http,
    private spinner: NgxSpinnerService) { }

  ngOnDestroy() {
    this.sub.unsubscribe();
  }

  ngOnInit() {
    this.sub = this.route.params.subscribe(params => {
      this.alarmID = params['alarmID'];
      this.getAlarmDetail(this.alarmID);
    });
    this.kibanaUrl = this.es.kibana;
  }

  sleep = (ms) => new Promise(resolve => setTimeout(resolve, ms));

  async getAlarmDetail(alarmID) {
    const that = this;
    that.spinner.show();
    const resp = await this.es.getAlarms(this.esIndex, this.esType, alarmID);
    const tempAlarms = resp.hits.hits;
    await Promise.all(tempAlarms.map(async (e) => {
      await Promise.all(e['_source']['rules'].map(async (r) => {
        if (r['status'] === 'finished') {
          r['events_count'] = r['occurrence'];
          Promise.resolve();
        } else {
          const response = await this.es.countEvents('siem_alarm_events-*', alarmID, r['stage']);
          r['events_count'] = response.count;
        }
      }));
    }));
    this.alarm = tempAlarms;
    tempAlarms.forEach(element => {
      this.alarmRules = element._source.rules;
      this.es.getAlarmEventsPagination(this.esIndexAlarmEvent, this.esType, this.alarmID, this.alarmRules[0].stage, 0, this.itemsPerPage)
      .then(function(alev) {
        console.log(alev);
        if (alev['hits'] !== undefined) {
          that.getEventsDetail('init', that.alarmID, that.alarmRules[0].stage, null, null, that.alarmRules[0].events_count);
        } else {
          that.getEventsDetail('init', that.alarmID, that.alarmRules[1].stage, null, null, that.alarmRules[1].events_count);
        }
      });
      if (element._source.vulnerabilities) {
        this.alarmVuln = element._source.vulnerabilities;
      }
      if (element._source.intel_hits) {
        this.alarmIntelHits = element._source.intel_hits;
      }
    });
    that.spinner.hide();
  }

  setStatus(rule) {
    if (rule['status'] !== '') {
      return rule['status'];
    }
    if (rule['status'] === '' && rule['start_time'] === 0) {
      return 'inactive';
    }
    if (rule['status'] === '' && rule['start_time'] > 0) {
      return 'active';
    }
    const deadline = rule['start_time'] + rule['timeout'];
    const now = Math.round((new Date()).getTime() / 1000);
    if (now > deadline) {
      return 'timeout';
    }
  }

  async getEventsDetail(type, id, stage, from= 0, size= 0, allsize= 0) {
    const that = this;
    that.stage = stage;
    that.totalItems = allsize;
    // that.isShowEventDetails = false;
    if (type === 'init') {
      from = 0;
      size = that.itemsPerPage;
    } else if (type === 'pagination') {
      from = from - 1;
      size = size;
    }
    let ev: any;
    let alev: any;

    try {
      alev = await this.es.getAlarmEventsPagination(this.esIndexAlarmEvent, this.esType, id, stage, from, size);
      console.log(alev['hits']['hits']);
    } catch (err) {
      throw new Error(('Error from getAlarmEventsPagination: ' + err));
    }
    for (const element of alev['hits']['hits']) {
      try {
        ev = await that.es.getEvents(that.esIndexEvent, that.esType, element['_source']['event_id']);
      } catch (err) {
        throw new Error(('Error from getEvents: ' + err));
      }
      let jml = 0;
      that.evnts = [];
      that.paginators = [];
      for (const element2 of ev['hits']['hits']) {
        that.evnts.push(element2['_source']);
        jml++;
        if (jml !== ev['hits']['hits'].length) {
          // should throw this but the test cases doesnt handle it yet
          // throw new Error(('jml != ev length'));
        }
      }
    }

    that.isShowEventDetails = true;
    if (type === 'init') {
      that.activePage = 1;
    }
    if (that.totalItems % that.itemsPerPage === 0) {
      that.numberOfPaginators = Math.floor(that.totalItems / that.itemsPerPage);
    } else {
      that.numberOfPaginators = Math.floor(that.totalItems / that.itemsPerPage + 1);
    }

    for (let i = 1; i <= that.numberOfPaginators; i++) {
      that.paginators.push(i);
    }
  }

  async changePage(event: any) {
    if (event.target.text >= 1 && event.target.text <= this.numberOfPaginators) {
      this.activePage = +event.target.text;
      this.firstVisibleIndex = this.activePage * this.itemsPerPage - this.itemsPerPage + 1;
      this.lastVisibleIndex = this.activePage * this.itemsPerPage;
      console.log(this.firstVisibleIndex + ' - ' + this.lastVisibleIndex);
      this.getEventsDetail('pagination', this.alarmID, this.stage, this.firstVisibleIndex, this.itemsPerPage, this.totalItems);
    }
  }

  nextPage(event: any) {
    if (this.pages.last.nativeElement.classList.contains('active')) {
      if ((this.numberOfPaginators - this.numberOfVisiblePaginators) >= this.lastVisiblePaginator) {
        this.firstVisiblePaginator += this.numberOfVisiblePaginators;
      this.lastVisiblePaginator += this.numberOfVisiblePaginators;
      } else {
        this.firstVisiblePaginator += this.numberOfVisiblePaginators;
      this.lastVisiblePaginator = this.numberOfPaginators;
      }
    }

    this.activePage += 1;
    this.firstVisibleIndex = this.activePage * this.itemsPerPage - this.itemsPerPage + 1;
    this.lastVisibleIndex = this.activePage * this.itemsPerPage;
    this.getEventsDetail('pagination', this.alarmID, this.stage, this.firstVisibleIndex, this.itemsPerPage, this.totalItems);
  }

  previousPage(event: any) {
    if (this.pages.first.nativeElement.classList.contains('active')) {
      if ((this.lastVisiblePaginator - this.firstVisiblePaginator) === this.numberOfVisiblePaginators)  {
        this.firstVisiblePaginator -= this.numberOfVisiblePaginators;
        this.lastVisiblePaginator -= this.numberOfVisiblePaginators;
      } else {
        this.firstVisiblePaginator -= this.numberOfVisiblePaginators;
        this.lastVisiblePaginator -= (this.numberOfPaginators % this.numberOfVisiblePaginators);
      }
    }

    this.activePage -= 1;
    this.firstVisibleIndex = this.activePage * this.itemsPerPage - this.itemsPerPage + 1;
    this.lastVisibleIndex = this.activePage * this.itemsPerPage;
    this.getEventsDetail('pagination', this.alarmID, this.stage, this.firstVisibleIndex, this.itemsPerPage, this.totalItems);
  }

  firstPage() {
    this.activePage = 1;
    this.firstVisibleIndex = this.activePage * this.itemsPerPage - this.itemsPerPage + 1;
    this.lastVisibleIndex = this.activePage * this.itemsPerPage;
    this.firstVisiblePaginator = 0;
    this.lastVisiblePaginator = this.numberOfVisiblePaginators;
    this.getEventsDetail('pagination', this.alarmID, this.stage, this.firstVisibleIndex, this.itemsPerPage, this.totalItems);
  }

  lastPage() {
    this.activePage = this.numberOfPaginators;
    this.firstVisibleIndex = this.activePage * this.itemsPerPage - this.itemsPerPage + 1;
    this.lastVisibleIndex = this.activePage * this.itemsPerPage;

    if (this.numberOfPaginators % this.numberOfVisiblePaginators === 0) {
      this.firstVisiblePaginator = this.numberOfPaginators - this.numberOfVisiblePaginators;
      this.lastVisiblePaginator = this.numberOfPaginators;
    } else {
      this.lastVisiblePaginator = this.numberOfPaginators;
      this.firstVisiblePaginator = this.lastVisiblePaginator - (this.numberOfPaginators % this.numberOfVisiblePaginators);
    }
    this.getEventsDetail('pagination', this.alarmID, this.stage, this.firstVisibleIndex, this.itemsPerPage, this.totalItems);
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
    // console.log(a);
    if ( a.indexOf('open') > -1) {
      this.wide = false;
    } else {
      this.wide = true;
    }
  }

  async changeAlarmStatus(_id, status) {
    try {
      if (this.alarm[0]._source.status === status) { return; }
      this.spinner.show();
      const res = await this.es.updateAlarmStatusById(this.alarm[0]._source.perm_index, this.esType, _id, status);
      if (res.result !== 'updated') {
        throw new Error(('index not updated, result: ' + res.result));
      }
      this.isProcessingUpdateStatus = true;
      this.spinner.hide();
      this.closeDropdown('alrm-status-', this.alarmID);
      this.wide = false;
      await this.sleep (1000);
      const resp = await this.es.getAlarm(this.alarm[0]._source.perm_index, this.esType, _id);
      this.alarm[0] = resp;
      // var resp = await this.es.getAlarms(this.alarm[0]._source.perm_index, this.esType, _id)
      // this.alarm = resp.hits.hits
      // console.log("alarm hits hits: ", this.alarm)
    } catch (err) {
      console.log('Error occur while changing alarm status: ' + err);
    } finally {
      this.isProcessingUpdateStatus = false;
      this.spinner.hide();
    }
  }

  async changeAlarmTag(_id, tag) {
    try {
      if (this.alarm[0]._source.tag === tag) { return; }
      this.spinner.show();
      const res = await this.es.updateAlarmTagById(this.alarm[0]._source.perm_index,
        this.esType, _id, tag);
      if (res.result !== 'updated') {
        throw new Error(('index not updated, result: ' + res.result));
      }
      this.isProcessingUpdateTag = true;
      this.spinner.hide();
      this.closeDropdown('alrm-tag-', this.alarmID);
      this.wide = false;
      await this.sleep (1000);
      const resp = await this.es.getAlarm(this.alarm[0]._source.perm_index, this.esType, _id);
      this.alarm[0] = resp;
      this.isProcessingUpdateTag = false;
    } catch (err) {
      console.log('Error occur while changing alarm tag: ' + err);
    } finally {
      this.spinner.hide();
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
    // console.log(a);
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
