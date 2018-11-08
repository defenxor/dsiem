import { Component, OnInit, Input } from '@angular/core';

import {Alarm} from '../alarm.interface';

import {ElasticsearchService} from '../../../app/elasticsearch.service';

@Component({
  selector: 'alarm-details',
  templateUrl: './alarm-details.component.html',
  styleUrls: ['./alarm-details.component.css']
})
export class AlarmDetailsComponent implements OnInit {

  @Input() alarm: Alarm;
  private static readonly ALARMEVENT_INDEX = 'siem_alarm_events-*';
  private static readonly EVENT_INDEX = 'siem_events-*';
  private static readonly TYPE = 'doc';
  evnts = [];

  constructor(private es: ElasticsearchService) { }

  ngOnInit() {
  }

  setStatus(rule) {
    if (rule["status"] != "") return rule["status"]
    if (rule["status"] == "" && rule["start_time"] == 0) {
      return "inactive"
    }
    if (rule["status"] == "" && rule["start_time"] > 0) {
      return "active"
    }
    let deadline = rule["start_time"] + rule["timeout"]
    let now = Math.round((new Date()).getTime() / 1000)
    if (now > deadline) {
      return "timeout"
    }
  }

  seeDetail(alarm, rule){
    var that = this;
    that.evnts = [];
    this.es.getAlarmEvents(AlarmDetailsComponent.ALARMEVENT_INDEX, AlarmDetailsComponent.TYPE, alarm.id, rule.stage).then(function(alev){
      var prom = function(){
        return new Promise(function(resolve, reject){
          alev['hits']['hits'].forEach(element => {
            that.es.getEvents(AlarmDetailsComponent.EVENT_INDEX, AlarmDetailsComponent.TYPE, element['_source']['event_id']).then(function(ev){
              let jml = 0;
              ev['hits']['hits'].forEach(element2 => {
                that.evnts.push(element2['_source']);
                jml++;
                if(jml == ev['hits']['hits'].length) return resolve(that.evnts);
              });
            });
          });
        });
      }

      prom().then(v=>{
        console.log(v);
      })
    })
    document.getElementById(alarm.id).hidden = false;
  }

}
