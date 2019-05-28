/*
Copyright (c) 2019 PT Defender Nusa Semesta and contributors, All rights reserved.

This file is part of Dsiem.

Dsiem is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation version 3 of the License.

Dsiem is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Dsiem. If not, see <https:www.gnu.org/licenses/>.
*/
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
    // setTimeout(() => {
      // this.checkES();
      // this.elasticsearch = this.es.getServer();
    // }, 500);
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
