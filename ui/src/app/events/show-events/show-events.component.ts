import { Component, OnInit } from '@angular/core';
import { Observable} from 'rxjs/Rx';
import { ElasticsearchService } from '../../elasticsearch.service';

import { animate } from "@angular/core";
import { trigger } from "@angular/core";
import { transition } from "@angular/core";
import { style } from "@angular/core";

@Component({
  selector: 'show-events',
  templateUrl: './show-events.component.html',
  styleUrls: ['./show-events.component.css'],
  animations: [
    trigger('fadeIn', [
      transition(':enter', [
        style({ opacity: '0' }),
        animate('.3s ease-out', style({ opacity: '1' })),
      ]),
    ]),
  ]
})

export class ShowEventsComponent implements OnInit {

  private static readonly INDEX = 'siem_events-*';
  private static readonly TYPE = 'doc';

  timerSubscription: any;
  events: any[]
  status: string
  isConnected: boolean

  constructor(private es: ElasticsearchService) { }

  ngOnInit() {
    this.getData()
  }
  ngOnDestroy() {
    if (this.timerSubscription) {
      this.timerSubscription.unsubscribe(); }
  }
  getData() {
    this.es.getLast5Minutes(ShowEventsComponent.INDEX, ShowEventsComponent.TYPE)
      .then(response => {
        this.events = response.hits.hits
        this.events.forEach( e => {
          e["_source"].timestamp = e["_source"]["@timestamp"]
          e["_source"].id = e["_id"]
        })
        console.log(this.events)
        console.log(response);
      }, error => {
        console.error(error);
      }).then(() => {
        console.log('Show Event Completed!');
      });
      this.timerSubscription = Observable.timer(5000).first().subscribe(() => this.getData());
  }

  
}
