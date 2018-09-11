import { Component, OnInit } from '@angular/core';
import { Observable} from 'rxjs/Rx';

import { AlarmSource } from '../alarm.interface';
import { ElasticsearchService } from '../../elasticsearch.service';

@Component({
  selector: 'show-alarms',
  templateUrl: './show-alarms.component.html',
  styleUrls: ['./show-alarms.component.css']
})
export class ShowAlarmsComponent implements OnInit {

  private static readonly INDEX = 'siem_alarms';
  private static readonly TYPE = 'doc';

  alarmSources: AlarmSource[];
  timerSubscription: any;

  constructor(private es: ElasticsearchService) { }

  ngOnInit() {
    this.getData()
  }
  ngOnDestroy() {
    if (this.timerSubscription) {
      this.timerSubscription.unsubscribe(); }
  }
  getData() {
    this.es.getLast5Minutes(ShowAlarmsComponent.INDEX, ShowAlarmsComponent.TYPE)
      .then(response => {
        this.alarmSources = response.hits.hits
        this.alarmSources.forEach( e => {
          e["_source"].timestamp = e["_source"]["@timestamp"]
          e["_source"].id = e["_id"]
        })
        console.log(this.alarmSources)
        console.log(response);
      }, error => {
        console.error(error);
      }).then(() => {
        console.log('Show Alarm Completed!');
      });
      this.timerSubscription = Observable.timer(5000).first().subscribe(() => this.getData());
  }
}
