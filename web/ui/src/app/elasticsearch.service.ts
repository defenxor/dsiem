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
import { Injectable } from '@angular/core';
import { Client } from 'elasticsearch-browser';
import { Http } from '@angular/http';
import { map } from 'rxjs/operators';

@Injectable({
  providedIn: 'root'
})
export class ElasticsearchService {
  private client: Client;
  server: string;
  kibana: string;
  esVersion: string;
  logstashType: boolean;

  querylast5mins = {
    'size' : 50,
    'query': {
      'range' : {
        'timestamp' : {
          'gte' : 'now-5m',
            'lt' : 'now'
           }
        }
     },
     'sort': { '@timestamp' : 'desc' }
  };

  queryalldocs = {
    'size': 20,
    'query': {
      'match_all': {}
    },
    'sort': { '@timestamp' : 'desc' }
  };

  queryalldocspaging(from, size) {
    return {
      'from': from,
      'size': size,
      'query': {
        'match_all': {}
      },
      'sort': { '@timestamp' : 'desc' }
    };
  }

  constructor(private http: Http) {
    this.loadConfig()
    .then(res => {
      this.server = res['elasticsearch'];
      this.kibana = res['kibana'];
      if (!this.client) {
        this.client = new Client({
          host:  this.server,
          log: 'info',
        });
      }
      return this.getESVersion();
    }).catch(err => {
      console.log(`[ES] error in constructor, ${err}`);
    });
  }

  loadConfig() {
    return this.http.get('./assets/config/esconfig.json')
      .pipe(map(res => res.json()))
      .toPromise();
  }

  async getESVersion() {
    try {
      const res = await this.http.get(this.server)
        .pipe(map(out => out.json()))
        .toPromise();
      const fullVer = res['version']['number'];
      this.esVersion = fullVer;
      // disable type if es major version >= 7
      const re = new RegExp(/^\d+/);
      const reVer = re.exec(fullVer);
      if (parseInt(reVer[0], 10) >= 7) {
        this.logstashType = false;
      } else {
          this.logstashType = true;
      }
    } catch (err) {}
  }

  buildQueryAlarmEvents(alarmId, stage) {
    return {
      'query': {
        'bool': {
          'must': [
            { 'term': { 'stage': stage }},
            { 'term': { 'alarm_id.keyword': alarmId }}
          ]
        }
      }
    };
  }

  buildQueryAlarmEventsPagination(alarmId, stage, from, size) {
    return {
      'from': from,
      'size': size,
      'query': {
        'bool': {
          'must': [
            {
              'term': { 'stage': stage }
            },
            {
              'term': { 'alarm_id.keyword': alarmId }
            }
          ]
        }
      }
    };
  }

  buildQueryAllAlarmEvents(alarmId, size) {
    return {
      'size': size,
      'query': {
        'term': { 'alarm_id.keyword': alarmId }
      }
    };
  }

  buildQueryEvents(eventId) {
    return {
      'query': {
        'term': { 'event_id.keyword': eventId }
      }
    };
  }

  buildQueryAlarms(alarmId) {
    return {
      'query': {
        'term': { '_id': alarmId }
      }
    };
  }

  buildQueryAlarmEventsWithoutStage(alarmId) {
    return {
      'size': 10000,
      'query': {
        'term': { 'alarm_id': alarmId }
      }
    };
  }

  getServer() {
    return this.server;
  }

  createIndex(name): any {
    return this.client.indices.create(name);
  }

  isAvailable(): any {
    return this.client.ping({
      requestTimeout: Infinity,
      body: 'hello'
    });
  }

  getType(): string {
    if (this.esVersion === '') {
      // esGetVersion in constructor failed, just default to use es 6.x
      this.logstashType = true;
    }
    if (this.logstashType) {
      return 'doc';
    } else {
      return "_doc";
    }
  }

  addToIndex(value): any {
    return this.client.create(value);
  }

  getAllDocuments(_index): any {
    return this.client.search({
      index: _index,
      type: this.getType(),
      body: this.queryalldocs,
    });
  }

  getLast5Minutes(_index): any {
    return this.client.search({
      index: _index,
      type: this.getType(),
      body: this.querylast5mins
    });
  }

  countEvents(_index, alarmId, stage): any {
    const b = `{
      "query": {
        "bool": {
          "must": [
            { "term": { "stage": "${stage}" }},
            { "term": { "alarm_id.keyword": "${alarmId}" }}
          ]
        }
      }
    }`;
    return this.client.count({
      index: _index,
      body: b
    });
  }

  getAlarmEvents(_index, alarmId, stage): any {
    return this.client.search({
      index: _index,
      type: this.getType(),
      body: this.buildQueryAlarmEvents(alarmId, stage),
      filterPath: ['hits.hits._source']
    });
  }

  getAlarmEventsPagination(_index, alarmId, stage, from, size): any {
    return this.client.search({
      index: _index,
      type: this.getType(),
      body: this.buildQueryAlarmEventsPagination(alarmId, stage, from, size),
      filterPath: ['hits.hits._source']
    });
  }

  getEvents(_index, eventId): any {
    return this.client.search({
      index: _index,
      type: this.getType(),
      body: this.buildQueryEvents(eventId),
      filterPath: ['hits.hits._source']
    });
  }

  getAllDocumentsPaging(_index, from, size): any {
    return this.client.search({
      index: _index,
      type: this.getType(),
      body: this.queryalldocspaging(from, size),
    });
  }

  getAlarms(_index, alarmId): any {
    return this.client.search({
      index: _index,
      type: this.getType(),
      body: this.buildQueryAlarms(alarmId),
      filterPath: ['hits.hits._source']
    });
  }

  getAlarm(_index, alarmId): any {
    return this.client.get({
      id: alarmId,
      index: _index,
      type: this.getType()
    });
  }

  updateAlarmStatusById(_index, _id, status) {
    return this.client.update({
      index: _index,
      type: this.getType(),
      id: _id,
      body: {
        doc: {
          status: status
        }
      }
    });
  }

  updateAlarmTagById(_index, _id, tag) {
    return this.client.update({
      index: _index,
      type: this.getType(),
      id: _id,
      body: {
        doc: {
          tag: tag
        }
      }
    });
  }

  getAlarmEventsWithoutStage(_index, alarmId): any {
    return this.client.search({
      index: _index,
      type: this.getType(),
      body: this.buildQueryAlarmEventsWithoutStage(alarmId),
      filterPath: ['hits.hits']
    });
  }

  getAllAlarmEvents(_index, alarmId, size): any {
    return this.client.search({
      index: _index,
      type: this.getType(),
      body: this.buildQueryAllAlarmEvents(alarmId, size),
      filterPath: ['hits.hits']
    });
  }

  async removeEventById(_index, _id) {
    return await this.client.deleteByQuery({
      index: _index,
      body: {
        query: {
          bool: {
            must: [
              {
                match_all: {}
              },
              {
                match_phrase: {
                  _id: {
                    query: _id
                  }
                }
              }
            ]
          }
        }
      }
    });
  }

  async removeAlarmById(_index, _id) {
    return await this.client.deleteByQuery({
      index: _index,
      body: {
        query: {
          bool: {
            must: [
              {
                match_all: {}
              },
              {
                match_phrase: {
                  _id: {
                    query: _id
                  }
                }
              }
            ]
          }
        }
      }
    });
  }

  async removeAlarmEventById(_index, _id) {
    return await this.client.deleteByQuery({
      index: _index,
      body: {
        query: {
          bool: {
            must: [
              {
                match_all: {}
              },
              {
                match_phrase: {
                  _id: {
                    query: _id
                  }
                }
              }
            ]
          }
        }
      }
    });
  }

  async removeAlarmEvent(params) {
    return await this.client.bulk({
      body: params
    }, function (err, resp) {
    });
  }
}

