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
import { FormControl } from '@angular/forms';
import { DsiemService } from './dsiem.service';

import { AUTH_ERROR } from './errors';

@Component({
  selector: 'app-dsiem-ui',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss']
})

export class AppComponent implements OnInit {
  public readonly status_init_waiting = 0;
  public readonly status_init_success = 1;
  public readonly status_init_fail = 2;
  public readonly status_init_auth_error = 3;

  public init_status: number = this.status_init_waiting;
  public init_error:string = ""

  public username = new FormControl('')
  public password = new FormControl('')

  constructor(private dsiem: DsiemService) {}

  get initialized(): boolean {
    return this.init_status === this.status_init_success;
  }

  ngOnInit() {
    this.dsiem.init()
      .then(() => this.init_status = this.status_init_success)
      .catch((err) => this.handleInitError(err))
  }

  private handleInitError(err: any) {
    if(err === AUTH_ERROR) {
      this.init_status = this.status_init_auth_error;
    } else {
      this.init_status = this.status_init_fail;
      this.init_error = err
    }
  }

  public submit() {
    const username = this.username.value;
    const password = this.password.value;

    this.dsiem.initWithCredentials(username, password)
    .then(() => this.init_status = this.status_init_success)
    .catch((err) => this.handleInitError(err))
  }
}

