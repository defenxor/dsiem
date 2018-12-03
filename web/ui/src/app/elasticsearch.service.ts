import { Injectable } from '@angular/core';
import { Client } from 'elasticsearch-browser';
import { Http } from "@angular/http";
import { map } from "rxjs/operators";
import { environment } from '../environments/environment';

@Injectable({
  providedIn: 'root'
})
export class ElasticsearchService {
  private client: Client;
  private server: string;

  querylast5mins = {
    "size" : 50,
    "query": {
      "range" : {
        "timestamp" : {
          "gte" : "now-5m",
            "lt" :  "now"
           }
        }
     },
     "sort": { "@timestamp" : "desc" }
  }

  queryalldocs = {
    "size": 20,
    'query': {
      'match_all': {}
    },
    "sort": { "@timestamp" : "desc" }
  };

  private queryalldocspaging(from, size){
    return {
      "from": from,
      "size": size,
      'query': {
        'match_all': {}
      },
      "sort": { "@timestamp" : "desc" }
    }
  };

  constructor(private http:Http) {
    this.loadConfig().then(
      res => {
        this.server = res['elasticsearch'];
        if (!this.client) {
          this.connect();
        }
      },
      err => console.log(`[ES] Unable to load config file, ${err}`)
    )
    // this.server = environment.elasticsearch
    // if (!this.client) {
    //   this.connect();
    // }
  }

  loadConfig(){
    return new Promise( (resolve, reject) => {
      this.http.get('./assets/config/esconfig.json').pipe(
        map(res => res.json())
      ).toPromise()
      .then( 
        res => resolve(res),
        err => reject(err) 
      )
    })
  }

  private buildQueryAlarmEvents (alarmId, stage) {
    return {
      "query": {
        "bool": {
          "must": [
            {
              "match_all": {}
            },
            {
              "match_phrase": {
                "stage": {
                  "query": stage
                }
              }
            },
            {
              "match_phrase": {
                "alarm_id": {
                  "query": alarmId
                }
              }
            }
          ]
        }
      }
    }
  }

  private buildQueryAlarmEventsPagination (alarmId, stage, from, size) {
    return {
      "from": from,
      "size": size,
      "query": {
        "bool": {
          "must": [
            {
              "match_all": {}
            },
            {
              "match_phrase": {
                "stage": {
                  "query": stage
                }
              }
            },
            {
              "match_phrase": {
                "alarm_id": {
                  "query": alarmId
                }
              }
            }
          ]
        }
      }
    }
  }

  private buildQueryAlarmEventsWithoutStage (alarmId) {
    return {
      "size": 10000,
      "query": {
        "bool": {
          "must": [
            {
              "match_all": {}
            },
            {
              "match_phrase": {
                "alarm_id": {
                  "query": alarmId
                }
              }
            }
          ]
        }
      }
    }
  }

  private buildQueryAllAlarmEvents (alarmId, size) {
    return {
      "size": size,
      "query": {
        "bool": {
          "must": [
            {
              "match_all": {}
            },
            {
              "match_phrase": {
                "alarm_id": {
                  "query": alarmId
                }
              }
            }
          ]
        }
      }
    }
  }

  private buildQueryEvents (eventId) {
    return {
      "query": {
        "bool": {
          "must": [
            {
              "match_all": {}
            },
            {
              "match_phrase": {
                "event_id": {
                  "query": eventId
                }
              }
            }
          ]
        }
      }
    }
  }

  private buildQueryAlarms (alarmId) {
    return {
      "query": {
        "bool": {
          "must": [
            {
              "match_all": {}
            },
            {
              "match_phrase": {
                "_id": {
                  "query": alarmId
                }
              }
            }
          ]
        }
      }
    }
  }

  private connect() {
    this.client = new Client({
      // host:  environment.elasticsearch,
      host:  this.server,
      log: 'info',
      apiVersion: '6.3'
    });
  }

  public getServer() {
    return this.server
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

  addToIndex(value): any {
    return this.client.create(value);
  }

  getAllDocuments(_index, _type): any {
    return this.client.search({
      index: _index,
      type: _type,
      body: this.queryalldocs,
    });
  }

  getLast5Minutes(_index, _type): any {
    return this.client.search({
      index: _index,
      type: _type,
      body: this.querylast5mins
    })
  }

  countEvents(_index, alarmId, stage): any {
    let b = `{
      "query": {
        "bool": {
          "must": [
            {
              "match_all": {}
            },
            {
              "match_phrase": {
                "stage": {
                  "query": ${stage}
                }
              }
            },
            {
              "match_phrase": {
                "alarm_id": {
                  "query": "${alarmId}"
                }
              }
            }
          ]
        }
      }
    }`
    return this.client.count({
      index: _index,
      body: b
    })
  }

  getAlarmEvents(_index, _type, alarmId, stage): any {
    return this.client.search({
      index: _index,
      type: _type,
      body: this.buildQueryAlarmEvents(alarmId, stage),
      filterPath: ['hits.hits._source']
    })
  }

  getAlarmEventsPagination(_index, _type, alarmId, stage, from, size): any {
    return this.client.search({
      index: _index,
      type: _type,
      body: this.buildQueryAlarmEventsPagination(alarmId, stage, from, size),
      filterPath: ['hits.hits._source']
    })
  }

  getEvents(_index, _type, eventId): any {
    return this.client.search({
      index: _index,
      type: _type,
      body: this.buildQueryEvents(eventId),
      filterPath: ['hits.hits._source']
    })
  }

  getAllDocumentsPaging(_index, _type, from, size): any {
    return this.client.search({
      index: _index,
      type: _type,
      body: this.queryalldocspaging(from, size),
    });
  }

  getAlarms(_index, _type, alarmId): any {
    return this.client.search({
      index: _index,
      type: _type,
      body: this.buildQueryAlarms(alarmId),
      filterPath: ['hits.hits._source']
    })
  }

  async updateAlarmStatusById(_index, _type, _id, status){
    return await this.client.update({
      index: _index,
      type: _type,
      id: _id,
      body: {
        doc: {
          status: status
        }
      }
    });
  }

  async updateAlarmTagById(_index, _type, _id, tag){
    return await this.client.update({
      index: _index,
      type: _type,
      id: _id,
      body: {
        doc: {
          tag: tag
        }
      }
    });
  }

  getAlarmEventsWithoutStage(_index, _type, alarmId): any {
    return this.client.search({
      index: _index,
      type: _type,
      body: this.buildQueryAlarmEventsWithoutStage(alarmId),
      filterPath: ['hits.hits']
    })
  }

  getAllAlarmEvents(_index, _type, alarmId, size): any {
    return this.client.search({
      index: _index,
      type: _type,
      body: this.buildQueryAllAlarmEvents(alarmId, size),
      filterPath: ['hits.hits']
    })
  }

  async removeEventById(_index, _type, _id){
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

  async removeAlarmById(_index, _type, _id){
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

  async removeAlarmEventById(_index, _type, _id){
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

  async removeAlarmEvent(params){
    return await this.client.bulk({
      body: params
    }, function (err, resp) {
    });
  }
}
