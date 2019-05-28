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
import { map } from 'rxjs/operators';
import { NgxSpinnerService } from 'ngx-spinner';

@Component({
  templateUrl: './detailalarm.component.html',
})
export class DetailalarmComponent implements OnInit, OnDestroy {

  @ViewChildren('pages') pages: QueryList<any>;
  sub: any;
  alarmID;
  stage;
  alarm = [];
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
    this.loadConfig().then(res => {
      this.kibanaUrl = res['kibana'];
    });
  }

  loadConfig() {
    const that = this;
    return new Promise((resolve, reject) => {
      that.http.get('./assets/config/esconfig.json').pipe(
        map(res => res.json())
      ).toPromise()
      .then(
        res => resolve(res),
        err => reject(err)
      );
    });
  }

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

  getEventsDetail(type, id, stage, from= 0, size= 0, allsize= 0) {
    const that = this;
    that.evnts = [];
    that.paginators = [];
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

    this.es.getAlarmEventsPagination(this.esIndexAlarmEvent, this.esType, id, stage, from, size).then(function(alev) {
      console.log(alev['hits']['hits']);
      const prom = function() {
        return new Promise(function(resolve, reject) {
          alev['hits']['hits'].forEach(element => {
            that.es.getEvents(that.esIndexEvent, that.esType, element['_source']['event_id']).then(function(ev) {
              let jml = 0;
              ev['hits']['hits'].forEach(element2 => {
                that.evnts.push(element2['_source']);
                jml++;
                if (jml === ev['hits']['hits'].length) {
                  return resolve(that.evnts);
                }
              });
            });
          });
        });
      };

      prom().then(v => {
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
      });
    }).catch(err => {
      console.log('ERROR: ', err);
    });
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

  changeAlarmStatus(_id, status) {
    this.spinner.show();
    this.es.updateAlarmStatusById(this.esIndex, this.esType, _id, status).then((res) => {
      console.log(res);
      this.spinner.hide();
      this.isProcessingUpdateStatus = true;
      this.closeDropdown('alrm-status-', this.alarmID);
      this.wide = false;
      setTimeout(() => {
        this.es.getAlarms(this.esIndex, this.esType, _id).then((resp) => {
          this.alarm = resp.hits.hits;
          console.log(this.alarm);
          this.isProcessingUpdateStatus = false;
        });
      }, 5000);
    }).catch(err => {
      console.log('ERROR: ', err);
      this.spinner.hide();
    });
  }

  changeAlarmTag(_id, tag) {
    this.spinner.show();
    this.es.updateAlarmTagById(this.esIndex, this.esType, _id, tag).then((res) => {
      console.log(res);
      this.spinner.hide();
      this.isProcessingUpdateTag = true;
      this.closeDropdown('alrm-tag-', this.alarmID);
      this.wide = false;
      setTimeout(() => {
        this.es.getAlarms(this.esIndex, this.esType, _id).then((resp) => {
          this.alarm = resp.hits.hits;
          console.log(this.alarm);
          this.isProcessingUpdateTag = false;
        });
      }, 5000);
    }).catch(err => {
      console.log('ERROR: ', err);
      this.spinner.hide();
    });
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

  resetHeightEv(key, alarmID, index) {
    const a = document.getElementById(key + alarmID).getAttribute('class');
    // console.log(a);
    if (a.indexOf('open') > -1) {
      this.wideEv[index] = false;
    } else {
      this.wideEv[index] = true;
    }
  }

}
