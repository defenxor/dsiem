import { Component, OnInit } from '@angular/core';
import { Router, NavigationEnd } from '@angular/router';
import {ElasticsearchService} from './elasticsearch.service'
import { timer } from 'rxjs';

@Component({
  // tslint:disable-next-line
  selector: 'body',
  template: '<router-outlet></router-outlet>'
})
export class AppComponent implements OnInit {
  private elasticsearch: string
  timerSubscription = null

  constructor(private router: Router, private es: ElasticsearchService) { 
    this.elasticsearch = this.es.getServer()
    this.checkES()
  }

  ngOnInit() {
    this.router.events.subscribe((evt) => {
      if (!(evt instanceof NavigationEnd)) {
        return;
      }
      window.scrollTo(0, 0);
    });
  }

  checkES() {
    console.log('checkES executed.')
    this.es.isAvailable().then(() => {
      console.log('Connected to ES ' + this.elasticsearch)
    }, error => {
      console.log('Disconnected from ES ' + this.elasticsearch)
      console.error('Elasticsearch is down', error)
    }).then(() => {
      this.timerSubscription = timer(5000).subscribe(() => this.checkES());
      // this.cd.detectChanges()
      // console.log('detectChanges executed.')
    })
  }
}
