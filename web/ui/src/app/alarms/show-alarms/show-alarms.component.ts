import {
  Component,
  OnInit
} from '@angular/core';
import {
  timer
} from 'rxjs';

import {
  AlarmSource
} from '../alarm.interface';
import {
  ElasticsearchService
} from '../../elasticsearch.service';

@Component({
  selector: 'show-alarms',
  templateUrl: './show-alarms.component.html',
  styleUrls: ['./show-alarms.component.css']
})
export class ShowAlarmsComponent implements OnInit {

  private static readonly INDEX = 'siem_alarms';
  private static readonly TYPE = 'doc';

  alarmSources: AlarmSource[];
  tempAlarms: AlarmSource[];
  timerSubscription: any;

  constructor(private es: ElasticsearchService) {}

  ngOnInit() {
    this.getData()
  }
  ngOnDestroy() {
    if (this.timerSubscription) {
      this.timerSubscription.unsubscribe();
    }
  }
  async getData() {
    try {
      let resp = await this.es.getAllDocuments(ShowAlarmsComponent.INDEX, ShowAlarmsComponent.TYPE)
      this.tempAlarms = resp.hits.hits
      await Promise.all(this.tempAlarms.map(async (e) => {
        // e["_source"].timestamp = e["_source"]["@timestamp"]
        e["_source"].id = e["_id"]
        await Promise.all(e["_source"]["rules"].map(async (r) => {
          if (r["status"] == "finished") {
            r["events_count"] = r["occurrence"] // speed hack to avoid overloading ES with counting
            Promise.resolve()
          } else {
            let response = await this.es.countEvents("siem_alarm_events-*", e["_id"], r["stage"])
            r["events_count"] = response.count  
          }
        }))
      }))
      this.alarmSources = this.tempAlarms
      console.log('Show Alarms Completed!');
    } catch (err) {
      console.error('Error: ' + err);
    } finally {
      this.timerSubscription = timer(5000).subscribe(() => this.getData());
    }
  }
}