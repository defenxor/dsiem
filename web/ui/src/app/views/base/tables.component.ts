/* eslint-disable @typescript-eslint/naming-convention */
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
import { Component, ViewChildren, QueryList, ViewChild, ChangeDetectorRef } from '@angular/core';
import { ElasticsearchService } from '../../elasticsearch.service';
import { sleep, removeItemFromObjectArray } from '../../utilities';
import { ModalDirective } from 'ngx-bootstrap';
import { NgxSpinnerService } from 'ngx-spinner';
import { CountdownComponent } from 'ngx-countdown';
import { SearchboxComponent } from './searchbox.component';
import { AlertboxComponent } from './alertbox.component';

@Component({
  templateUrl: 'tables.component.html'
})
export class TablesComponent {

  @ViewChildren('pages') pages: QueryList<any>;

  @ViewChild('confirmModalRemove') confirmModalRemove: ModalDirective;

  @ViewChild('counter', {static: true}) counter: CountdownComponent;

  @ViewChild(SearchboxComponent) private searchBox: SearchboxComponent;

  @ViewChild(AlertboxComponent) private alertBox: AlertboxComponent;

  elasticsearch: string;
  tableData: any[] = [];
  counterPreText = 'Turn-off auto-refresh (Refreshing in ';
  counterPostText = ' seconds)';
  counterPaused = false;
  animateProgress = false;
  totalItems = 20;
  alarmIdToRemove: string;
  alarmIndexToRemove: string;
  disabledBtn: boolean;

  constructor(private es: ElasticsearchService, private spinner: NgxSpinnerService, private cd: ChangeDetectorRef) {
  }

  async onSearchboxReady() {
    this.toggleCounter(true);
    // disabling button cause too many color changes at once
    // this.disabledBtn = true
    this.animateProgress = true;
    try {
      await this.getData(this.searchBox.resultIDs);
    } catch (err) {
      console.log('error in doSearch():', err);
    } finally {
      await sleep(500);
      this.animateProgress = false;
      // this.disabledBtn = false
    }
  }

  async onSearchboxEmpty() {
    this.toggleCounter(false);
  }

  async counterStart() {
    await this.syncES();
  }

  counterClick() {
    this.toggleCounter(!this.counterPaused);
  }

  async counterFinished() {
    await this.syncES();
    await sleep(100).then(() => this.counter.restart());
  }

  toggleCounter(pause: boolean) {
    this.counterPaused = pause;
    if (this.counterPaused) {
      this.counter.pause();
      this.counterPreText = 'Turn-on auto-refresh (Continue refreshing in ';
    } else {
      this.counterPreText = 'Turn-off auto-refresh (Refreshing in ';
      this.counter.resume();
    }
  }

  async syncES() {
    this.disabledBtn = true;
    this.counter.pause();
    this.animateProgress = true;
    this.cd.detectChanges();
    const esAlive = await this.checkES();
    try {
      if (esAlive) {
        await this.getData();
      }
    } catch (err) {
      console.log('Fail to sync data with elasticsearch: ' + err);
    } finally {
      this.disabledBtn = false;
      this.animateProgress = false;
      this.counter.resume();
    }
  }

  async checkES(): Promise<boolean> {

    let esStatus = await this.es.init();
    while (esStatus.initialized === false) {
      this.alertBox.showAlert('Fail to read or parse esconfig.json: ' +
        esStatus.errMsg + '. Will retry every 5s ..', 'danger', true);
      await sleep(10000);
      esStatus = await this.es.init();
    }

    this.elasticsearch = this.es.getServer();
    const esUser = this.es.getUser();
    const label = esUser ? this.elasticsearch + ' as ' + esUser : this.elasticsearch;
    try {
      await this.es.isAvailable();
      this.alertBox.showAlert('Connected to ES ' + label, 'success', true);
      return true;
    } catch (err) {
      this.alertBox.showAlert('Disconnected from ES ' + this.elasticsearch + ': ' + err, 'danger', true);
      this.es.reset();
     }
    return false;
  }

  async getData(alarmIds: string[] = []) {
    try {
      let resp;
      if (alarmIds.length > 0) {
        resp = await this.es.getAlarmsMulti(this.es.esIndex, alarmIds);
      } else {
        resp = await this.es.getAllDocumentsPaging(this.es.esIndex, 0, this.totalItems);
      }

      const tempAlarms = resp.hits.hits;
      this.tableData = [];
      tempAlarms.forEach((a) => {
        const tempArr = {
          id: a['_id'],
          title: a['_source']['title'],
          timestamp: a['_source']['timestamp'],
          updated_time: a['_source']['updated_time'],
          status: a['_source']['status'],
          risk_class: a['_source']['risk_class'],
          tag: a['_source']['tag'],
          src_ips: a['_source']['src_ips'],
          dst_ips: a['_source']['dst_ips'],
          actions: '<i class=\'fa fa-eye\' title=\'click here to see details\' style=\'cursor:pointer; color:#ff9800\'></i>'
        };
        this.tableData.push(tempArr);
      });
    } catch (err) {
      this.tableData = [];
      throw err;
    }
  }

  confirmBeforeRemove(alarmID, alarmIndex) {
    this.alarmIdToRemove = alarmID;
    this.alarmIndexToRemove = alarmIndex;
    this.confirmModalRemove.show();
  }

  async deleteAlarm() {
    const targetID = this.alarmIdToRemove;
    this.spinner.show();
    this.confirmModalRemove.hide();
    const savedCounterState = this.counterPaused;
    try {
      await this.es.deleteAlarm(targetID);
      this.alertBox.showAlert('alarm ' + targetID + ' removed successfully', 'success', false );
      removeItemFromObjectArray(this.tableData, 'id', targetID);
    } catch (err) {
      console.log('Error in deleteAlarm: ', err);
      this.alertBox.showAlert('Error occurred while removing alarm ' + targetID + ': ' + err, 'danger', false);
    } finally {
      this.disabledBtn = false;
      this.spinner.hide();
      this.toggleCounter(savedCounterState);
    }
  }

}
