import {Component, ChangeDetectorRef} from '@angular/core';
import {ElasticsearchService} from './elasticsearch.service'
import { timer } from 'rxjs';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent {
  private elasticsearch: string
  
  title = 'DSIEM';
  status = "";
  timerSubscription = null

  constructor(private es: ElasticsearchService, private cd: ChangeDetectorRef) {
    this.elasticsearch = this.es.getServer()
    this.checkES()
  }

  checkES() {
    console.log('checkES executed.')
    this.es.isAvailable().then(() => {
      this.status = "Connected to ES " + this.elasticsearch
    }, error => {
      this.status = "Disconnected from ES " + this.elasticsearch
      console.error('Elasticsearch is down', error)
    }).then(() => {
      this.timerSubscription = timer(5000).subscribe(() => this.checkES());
      // this.cd.detectChanges()
      // console.log('detectChanges executed.')
    })
  }
}
