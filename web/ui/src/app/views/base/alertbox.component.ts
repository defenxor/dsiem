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
import { Component } from '@angular/core';

@Component({
  selector: 'app-alert-box',
  templateUrl: 'alertbox.component.html',
})

export class AlertboxComponent {
  /** @internal */
  alertType: string;
  alertMsg: string;
  alertVisible: boolean;
  alertIcon: string;
  prevType: string;
  prevMsg: string;
  prevIcon: string;

  async showAlert(msg: string, type: string, persistent: boolean = true) {
    this.alertMsg = msg;
    this.alertType = type;
    this.alertIcon = type === 'success' ? 'fa-check-circle' : 'fa-exclamation-triangle';
    if (persistent) {
      this.prevMsg = this.alertMsg;
      this.prevIcon = this.alertIcon;
      this.prevType = this.alertType;
    } else {
      setTimeout(() => {
        this.alertMsg = this.prevMsg;
        this.alertIcon = this.prevIcon;
        this.alertType = this.prevType;
      }, 5000);
    }
    this.alertVisible = true;

  }

}
