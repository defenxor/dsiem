import { Component, OnInit } from '@angular/core';
import { Router, ActivatedRoute, ParamMap } from '@angular/router';

import {ElasticsearchService} from '../../../app/elasticsearch.service';

@Component({
  selector: 'app-event-details',
  templateUrl: './event-details.component.html',
  styleUrls: ['./event-details.component.css']
})
export class EventDetailsComponent implements OnInit {

  private static readonly ALARMEVENT_INDEX = 'siem_alarm_events-*';
  private static readonly EVENT_INDEX = 'siem_events-*';
  private static readonly TYPE = 'doc';
  evnts = [];
  private sub: any;

  constructor(
    private es: ElasticsearchService,
    private route: ActivatedRoute,
    private router: Router
  ) { }

  ngOnDestroy(){
    this.sub.unsubscribe();
  }

  ngOnInit() {
    this.sub = this.route.params.subscribe(params => {
      // console.log(params['alarmID']);
      // console.log(params['stage']);
      this.getData(params['alarmID'], params['stage']);
    })
  }

  getData(id, stage){
    var that = this;
    that.evnts = [];
    this.es.getAlarmEvents(EventDetailsComponent.ALARMEVENT_INDEX, EventDetailsComponent.TYPE, id, stage).then(function(alev){
      var prom = function(){
        return new Promise(function(resolve, reject){
          alev['hits']['hits'].forEach(element => {
            that.es.getEvents(EventDetailsComponent.EVENT_INDEX, EventDetailsComponent.TYPE, element['_source']['event_id']).then(function(ev){
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
  }

}