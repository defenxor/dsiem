import {Component, OnInit} from '@angular/core';
import {Router, NavigationEnd} from '@angular/router';
import {ElasticsearchService} from './elasticsearch.service';
import {timer} from 'rxjs';

@Component({
  selector: 'app-dsiem-ui',
  template: '<router-outlet></router-outlet>'
})
export class AppComponent implements OnInit {
  private elasticsearch: string;

  constructor(private router: Router, private es: ElasticsearchService) {
  }

  ngOnInit() {
    setTimeout(() => {
      this.checkES();
      this.elasticsearch = this.es.getServer();
    }, 500);
  }

  checkES() {
    this.es.isAvailable().then(() => {
      console.log(`[ES Check] Connected to ${this.elasticsearch}`);
    }, error => {
      console.log(`[ES Check] Disconnected from ${this.elasticsearch} - ${error}`);
    }).then(() => {
      // changed observable subscription to promise
      timer(5000).toPromise().then(
        () => this.checkES(),
        err => console.log('unable to finish timer', err.message)
      );
    });
  }
}
