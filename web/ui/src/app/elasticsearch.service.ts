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
import { HttpClient } from '@angular/common/http';
import { url2obj } from './utilities';

@Injectable({
  providedIn: 'root'
})
export class ElasticsearchService {
  server: string;
  kibana: string;
  user: string;
  esType = 'doc'; // default to ES6
  initialized: boolean;
  esIndexAlarmEvent = 'siem_alarm_events-*';
  esIndex = 'siem_alarms';
  esIndexEvent = 'siem_events-*';

  // eslint-disable-next-line @typescript-eslint/naming-convention
  readonly MAX_DOCS_RETURNED = 200;
  private client: Client;
  constructor(private http: HttpClient) {}

  loadConfig() {
    return this.http.get('./assets/config/esconfig.json').toPromise();
  }

  reset() {
    this.initialized = false;
  }

  async init() {
    const ret = {
      initialized: this.initialized,
      errMsg: ''
    };
    if (ret.initialized) {
      return ret;
    }
    this.initialized = false;
    try {
      const res = await this.loadConfig();
      this.kibana = res['kibana'];
      const rgxp = url2obj(res['elasticsearch']);
      this.server = rgxp.protocol + '://' + rgxp.host;
      this.user = rgxp.user;
      this.initialized = true;
      this.client = new Client({
        host: res['elasticsearch'],
        log: 'info'
      });
    } catch (err) {
      ret.errMsg = err;
    }
    ret.initialized = this.initialized;
    if (this.initialized) {
      this.esType = await this.getESType();
    }
    return ret;
  }

  async getESType() {
    try {
      const res = await this.client.info();
      const fullVer = res.version.number;
      // disable type if es major version >= 7
      const re = new RegExp(/^\d+/);
      const reVer = re.exec(fullVer);
      if (parseInt(reVer[0], 10) >= 7) {
        return '_doc';
      }
      return 'doc';
    } catch (err) {
      console.log('err', err);
    }
    // default to ES6
    return 'doc';
  }

  queryAllDocsPaging(from, size) {
    return {
      from,
      size,
      query: {
        // eslint-disable-next-line @typescript-eslint/naming-convention
        match_all: {}
      },
      sort: { timestamp: 'desc' }
    };
  }

  buildQueryAlarmEvents(alarmId, stage) {
    return {
      query: {
        bool: {
          must: [
            { term: { stage } },
            // eslint-disable-next-line @typescript-eslint/naming-convention
            { term: { 'alarm_id.keyword': alarmId } }
          ]
        }
      }
    };
  }

  buildQueryAlarmEventsPagination(alarmId, stage, from, size) {
    return {
      from,
      size,
      query: {
        bool: {
          must: [
            {
              term: { stage }
            },
            {
              // eslint-disable-next-line @typescript-eslint/naming-convention
              term: { 'alarm_id.keyword': alarmId }
            }
          ]
        }
      }
    };
  }

  buildQueryAllAlarmEvents(alarmId, size) {
    return {
      size,
      query: {
        // eslint-disable-next-line @typescript-eslint/naming-convention
        term: { 'alarm_id.keyword': alarmId }
      }
    };
  }

  buildQueryEvents(eventId) {
    return {
      query: {
        // eslint-disable-next-line @typescript-eslint/naming-convention
        term: { 'event_id.keyword': eventId }
      }
    };
  }

  buildQueryMultipleEvents(keywords: string[]) {
    const k = keywords.join(',');
    return {
      query: {
        // eslint-disable-next-line @typescript-eslint/naming-convention
        terms: { 'event_id.keyword': keywords }
      },
      sort: { timestamp: 'desc' }
    };
  }

  buildQueryMultipleAlarms(keywords: string[]) {
    const k = keywords.join(',');
    return {
      query: {
        terms: { _id: keywords }
      },
      sort: { timestamp: 'desc' }
    };
  }

  buildQueryAlarms(alarmId) {
    return {
      query: {
        term: { _id: alarmId }
      }
    };
  }

  buildQueryAlarmEventsWithoutStage(alarmId) {
    return {
      size: 10000,
      query: {
        // eslint-disable-next-line @typescript-eslint/naming-convention
        term: { alarm_id: alarmId }
      }
    };
  }

  getServer() {
    return this.server;
  }

  getUser() {
    return this.user;
  }

  isAvailable(): any {
    return this.client.ping({
      requestTimeout: 10000,
      body: 'hello'
    });
  }

  getType(): string {
    return this.esType;
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

  getEventsMulti(_index, eventIds: string[]): any {
    const len =
      eventIds.length > this.MAX_DOCS_RETURNED
        ? this.MAX_DOCS_RETURNED
        : eventIds.length;
    return this.client.search({
      index: _index,
      size: len,
      type: this.getType(),
      body: this.buildQueryMultipleEvents(eventIds),
      filterPath: ['hits.hits._source']
    });
  }

  getAllDocumentsPaging(_index, from, size): any {
    return this.client.search({
      index: _index,
      type: this.getType(),
      body: this.queryAllDocsPaging(from, size)
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

  getAlarmsMulti(_index, alarmIds: string[]): any {
    const len =
      alarmIds.length > this.MAX_DOCS_RETURNED
        ? this.MAX_DOCS_RETURNED
        : alarmIds.length;
    return this.client.search({
      index: _index,
      size: len,
      type: this.getType(),
      body: this.buildQueryMultipleAlarms(alarmIds),
      filterPath: ['hits.hits']
    });
  }

  updateAlarmStatusById(_index, _id, status) {
    return this.client.update({
      index: _index,
      type: this.getType(),
      id: _id,
      body: {
        doc: {
          status
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
          tag
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
                // eslint-disable-next-line @typescript-eslint/naming-convention
                match_all: {}
              },
              {
                // eslint-disable-next-line @typescript-eslint/naming-convention
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
                // eslint-disable-next-line @typescript-eslint/naming-convention
                match_all: {}
              },
              {
                // eslint-disable-next-line @typescript-eslint/naming-convention
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
                // eslint-disable-next-line @typescript-eslint/naming-convention
                match_all: {}
              },
              {
                // eslint-disable-next-line @typescript-eslint/naming-convention
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

  async deleteAlarm(targetID: string) {
    const res = await this.getAlarmEventsWithoutStage(
      this.esIndexAlarmEvent,
      targetID
    );
    if (typeof res.hits.hits === 'undefined') {
      throw new Error('getAlarmEventsWithoutStage return undefined hits');
    }

    const tempAlarmEvent = res.hits.hits;
    const numOfAlarmEvent = tempAlarmEvent.length;

    const loopTimes = Math.floor(numOfAlarmEvent / 4500) + 1;
    for (let i = 1; i <= loopTimes; i++) {
      await this.deleteAllAlarmEvents(targetID);
    }
    const resAlarm = await this.removeAlarmById(this.esIndex, targetID);
    if (resAlarm.deleted === 1) {
      console.log('Deleting alarm ' + targetID + ' done');
    }
  }

  async deleteAllAlarmEvents(alarmID: string) {
    const arrDelete = [];
      const size = 4500;
    const res = await this.getAllAlarmEvents(
      this.esIndexAlarmEvent,
      alarmID,
      size
    );
    if (typeof res.hits.hits === 'undefined') {
      throw new Error('getAllAlarmEvents return undefined hits');
    }

    const tempAlarmEvent = res.hits.hits;

    for (let i = 0; i <= tempAlarmEvent.length - 1; i++) {
      const idx = tempAlarmEvent[i]['_index'];
      arrDelete.push({
        delete: {
          _index: idx,
          _type: tempAlarmEvent[i]['_type'],
          _id: tempAlarmEvent[i]['_id']
        }
      });
    }
    return await this.bulk(arrDelete);
  }

  async bulk(params) {
    return await this.client.bulk({
      body: params
    });
  }
}
