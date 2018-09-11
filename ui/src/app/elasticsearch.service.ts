import { Injectable } from '@angular/core';

import { Client } from 'elasticsearch';

import { environment } from '../environments/environment';

@Injectable()
export class ElasticsearchService {

  private client: Client;
  private server: string;

  querylast5mins = {
    "size" : 50,
    "query": {
      "range" : {
        "@timestamp" : {
          "gte" : "now-5m",
            "lt" :  "now"
           }
        }
     },
     "sort": { "@timestamp" : "desc" }
  }


  queryalldocs = {
    'query': {
      'match_all': {}
    }
  };

  constructor() {
    this.server = environment.elasticsearch
    if (!this.client) {
      this.connect();
    }
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

  private connect() {
    this.client = new Client({
      host:  environment.elasticsearch,
      log: 'trace'
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
      filterPath: ['hits.hits._source']
    });
  }

  getLast5Minutes(_index, _type): any {
    return this.client.search({
      index: _index,
      type: _type,
      body: this.querylast5mins
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
}
