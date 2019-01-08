import { Component, AfterViewInit, ViewChildren, QueryList, ViewChild } from '@angular/core';
import { ElasticsearchService } from '../../elasticsearch.service';
import { AlarmSource } from './alarm.interface';
import { timer } from 'rxjs';
import { ModalDirective } from 'ngx-bootstrap';
import { NgxSpinnerService } from 'ngx-spinner';


@Component({
  templateUrl: 'tables.component.html'
})
export class TablesComponent implements AfterViewInit {

  @ViewChildren('pages') pages: QueryList<any>;
  @ViewChild('confirmModalRemove') confirmModalRemove: ModalDirective;
  esIndex = 'siem_alarms';
  esIndexAlarmEvent = 'siem_alarm_events-*';
  esIndexEvent = 'siem_events-*';
  esType = 'doc';
  elasticsearch: string;
  tempAlarms: AlarmSource[];
  tableData: object[] = [];
  timerSubscription: any;
  totalItems = 20;
  itemsPerPage = 10;
  numberOfVisiblePaginators = 10;
  numberOfPaginators: number;
  paginators: Array<any> = [];
  activePage = 1;
  firstVisibleIndex = 1;
  lastVisibleIndex: number = this.itemsPerPage;
  firstVisiblePaginator = 0;
  lastVisiblePaginator = this.numberOfVisiblePaginators;
  timer_status = 'on';
  refreshSec;
  intrvl;
  alarmIdToRemove;
  alarmIndexToRemove;
  isRemoved;
  isNotRemoved;
  errMsg;
  disabledBtn;

  constructor(private es: ElasticsearchService, private spinner: NgxSpinnerService) {
    this.elasticsearch = this.es.getServer();
  }

  ngAfterViewInit(){
    setTimeout(()=>{
      this.getData('init');
    }, 100);
  }

  ngOnDestroy() {
    if (this.timerSubscription) {
      this.timerSubscription.unsubscribe();
    }
  }

  async getData(type, from=0, size=0) {
    var that = this;
    that.spinner.show();
    clearInterval(this.intrvl);
    try {
      this.refreshSec = 10;
      this.intrvl = setInterval(function(){
        if(that.refreshSec > 0) that.refreshSec--;
      }, 1000);
      let resp;
      if(type == 'init'){
        resp = await this.es.getAllDocumentsPaging(this.esIndex, this.esType, 0, this.itemsPerPage);
      } else if(type == 'pagination'){
        resp = await this.es.getAllDocumentsPaging(this.esIndex, this.esType, from-1, size);
      }
      this.tempAlarms = resp.hits.hits
      await Promise.all(this.tempAlarms.map(async (e) => {
        // e["_source"].timestamp = e["_source"]["@timestamp"]
        e["_source"].id = e["_id"]
        await Promise.all(e["_source"]["rules"].map(async (r) => {
          if (r["status"] == "finished") {
            r["events_count"] = r["occurrence"]
            Promise.resolve()
          } else {
            let response = await this.es.countEvents("siem_alarm_events-*", e["_id"], r["stage"])
            r["events_count"] = response.count  
          }
        }))
      }))
      this.tableData = [];
      this.paginators = [];
      if (type == 'init') this.activePage = 1;
      this.tempAlarms.forEach((a)=>{
        var tempArr = {
          id: a['_source']['id'],
          title: a['_source']['title'],
          timestamp: a['_source']['timestamp'],
          updated_time: a['_source']['updated_time'],
          status: a['_source']['status'],
          risk_class: a['_source']['risk_class'],
          tag: a['_source']['tag'],
          src_ips: a['_source']['src_ips'],
          dst_ips: a['_source']['dst_ips'],
          actions: '<i class="fa fa-eye" title="click here to see details" style="cursor:pointer; color:#ff9800"></i>'
        };
        this.tableData.push(tempArr);
      })
      // console.log(this.tableData);
      // console.log('Show Alarms Completed!');
      if (this.totalItems % this.itemsPerPage === 0) {
        this.numberOfPaginators = Math.floor(this.totalItems / this.itemsPerPage);
      } else {
        this.numberOfPaginators = Math.floor(this.totalItems / this.itemsPerPage + 1);
      }
    
      for (let i = 1; i <= this.numberOfPaginators; i++) {
        this.paginators.push(i);
      }
      that.spinner.hide();
    } catch (err) {
      console.error('Error: ' + err);
      this.tableData = [];
      this.paginators = [];
      this.spinner.hide();
    } finally {
      if(type == 'init'){
        this.timerSubscription = timer(9000).subscribe(() => this.getData('init'));
      }
    }
  }

  async reload(from, size){
    try {
      let resp = await this.es.getAllDocumentsPaging(this.esIndex, this.esType, from-1, size)
      this.tempAlarms = resp.hits.hits
      await Promise.all(this.tempAlarms.map(async (e) => {
        // e["_source"].timestamp = e["_source"]["@timestamp"]
        e["_source"].id = e["_id"]
        await Promise.all(e["_source"]["rules"].map(async (r) => {
          if (r["status"] == "finished") {
            r["events_count"] = r["occurrence"]
            Promise.resolve()
          } else {
            let response = await this.es.countEvents("siem_alarm_events-*", e["_id"], r["stage"])
            r["events_count"] = response.count  
          }
        }))
      }))
      this.tableData = [];
      this.paginators = [];
      this.tempAlarms.forEach((a)=>{
        var tempArr = {
          id: a['_source']['id'],
          title: a['_source']['title'],
          timestamp: a['_source']['timestamp'],
          updated_time: a['_source']['updated_time'],
          status: a['_source']['status'],
          risk_class: a['_source']['risk_class'],
          tag: a['_source']['tag'],
          src_ips: a['_source']['src_ips'],
          dst_ips: a['_source']['dst_ips'],
          actions: '<i class="fa fa-eye" title="click here to see details" style="cursor:pointer; color:#ff9800"></i>'
        };
        this.tableData.push(tempArr);
      })
      // console.log(this.tableData);
      // console.log('Show Alarms Completed!');
      if (this.totalItems % this.itemsPerPage === 0) {
        this.numberOfPaginators = Math.floor(this.totalItems / this.itemsPerPage);
      } else {
        this.numberOfPaginators = Math.floor(this.totalItems / this.itemsPerPage + 1);
      }
    
      for (let i = 1; i <= this.numberOfPaginators; i++) {
        this.paginators.push(i);
      }
    } catch (err) {
      console.error('Error: ' + err);
      this.tableData = [];
      this.paginators = [];
    } finally {
    }
  }

  async changePage(event: any) {
    if (event.target.text >= 1 && event.target.text <= this.numberOfPaginators) {
      this.activePage = +event.target.text;
      this.firstVisibleIndex = this.activePage * this.itemsPerPage - this.itemsPerPage + 1;
      this.lastVisibleIndex = this.activePage * this.itemsPerPage;
      console.log(this.firstVisibleIndex + ' - ' + this.lastVisibleIndex);
      this.reload(this.firstVisibleIndex, this.itemsPerPage);
    }
  }

  nextPage(event: any) {
    if (this.pages.last.nativeElement.classList.contains('active')) {
      if ((this.numberOfPaginators - this.numberOfVisiblePaginators) >= this.lastVisiblePaginator) {
        this.firstVisiblePaginator += this.numberOfVisiblePaginators;
      this.lastVisiblePaginator += this.numberOfVisiblePaginators;
      } else {
        this.firstVisiblePaginator += this.numberOfVisiblePaginators;
      this.lastVisiblePaginator = this.numberOfPaginators;
      }
    }

    this.activePage += 1;
    this.firstVisibleIndex = this.activePage * this.itemsPerPage - this.itemsPerPage + 1;
    this.lastVisibleIndex = this.activePage * this.itemsPerPage;
    this.reload(this.firstVisibleIndex, this.itemsPerPage);
  }

  previousPage(event: any) {
    if (this.pages.first.nativeElement.classList.contains('active')) {
      if ((this.lastVisiblePaginator - this.firstVisiblePaginator) === this.numberOfVisiblePaginators)  {
        this.firstVisiblePaginator -= this.numberOfVisiblePaginators;
        this.lastVisiblePaginator -= this.numberOfVisiblePaginators;
      } else {
        this.firstVisiblePaginator -= this.numberOfVisiblePaginators;
        this.lastVisiblePaginator -= (this.numberOfPaginators % this.numberOfVisiblePaginators);
      }
    }

    this.activePage -= 1;
    this.firstVisibleIndex = this.activePage * this.itemsPerPage - this.itemsPerPage + 1;
    this.lastVisibleIndex = this.activePage * this.itemsPerPage;
    this.reload(this.firstVisibleIndex, this.itemsPerPage);
  }

  firstPage() {
    this.activePage = 1;
    this.firstVisibleIndex = this.activePage * this.itemsPerPage - this.itemsPerPage + 1;
    this.lastVisibleIndex = this.activePage * this.itemsPerPage;
    this.firstVisiblePaginator = 0;
    this.lastVisiblePaginator = this.numberOfVisiblePaginators;
    this.reload(this.firstVisibleIndex, this.itemsPerPage);
  }

  lastPage() {
    this.activePage = this.numberOfPaginators;
    this.firstVisibleIndex = this.activePage * this.itemsPerPage - this.itemsPerPage + 1;
    this.lastVisibleIndex = this.activePage * this.itemsPerPage;

    if (this.numberOfPaginators % this.numberOfVisiblePaginators === 0) {
      this.firstVisiblePaginator = this.numberOfPaginators - this.numberOfVisiblePaginators;
      this.lastVisiblePaginator = this.numberOfPaginators;
    } else {
      this.lastVisiblePaginator = this.numberOfPaginators;
      this.firstVisiblePaginator = this.lastVisiblePaginator - (this.numberOfPaginators % this.numberOfVisiblePaginators);
    }
    this.reload(this.firstVisibleIndex, this.itemsPerPage);
  }

  startStopTimer(status){
    if(status == 'off'){
      if (this.timerSubscription) {
        this.timerSubscription.unsubscribe();
        this.timer_status = 'off';
        clearInterval(this.intrvl);
      }
    } else {
      this.getData('init');
      this.timer_status = 'on';
    }
  }

  confirmBeforeRemove(alarmID, alarmIndex){
    this.alarmIdToRemove = alarmID;
    this.alarmIndexToRemove = alarmIndex;
    this.confirmModalRemove.show();
  }

  async removeAlarm(){
    var that = this;
    that.spinner.show();
    that.startStopTimer('off');
    that.disabledBtn = true;
    console.log('id to remove: ', that.alarmIdToRemove);
    that.confirmModalRemove.hide();

    var promRemoveAE = function(){
      new Promise((resolveAE, rejectAE)=>{
        that.es.getAlarmEventsWithoutStage(that.esIndexAlarmEvent, that.esType, that.alarmIdToRemove).then(res=>{
          
          if(res.hits.hits){

            var tempAlarmEvent = res.hits.hits;
            var numOfAlarmEvent = tempAlarmEvent.length;

            if(numOfAlarmEvent < 4500){
              var removeAlarmEvent = function(){
                return new Promise((resolveRemove, reject)=>{
                  that.removeAllAlarmEvent().then((r)=>{
                    // console.log(r);
                    return resolveRemove('remove alarm event done');
                  });
                });
              }
              
            } else {
              
              var removeAlarmEvent = function(){
                return new Promise((resolveRemove, reject)=>{
                  for(var i=1; i <= Math.floor(numOfAlarmEvent/4500)+1; i++){
                    new Promise((resolveFor, reject2)=>{
                      that.removeAllAlarmEvent().then((r)=>{
                        // console.log(i + ': ', r);
                        resolveFor('done');
                        if(i == Math.floor(numOfAlarmEvent/9000)+1) return resolveRemove('remove all alarm event done');
                      });
                    });
                  }
                });
              }
              
            }
          
            removeAlarmEvent().then((res)=>{
              // console.log(res);
              resolveAE('done');
            });

          }
          
        },
        (error)=>{
          console.log(error);
        });
      });
    }

    promRemoveAE();
    setTimeout(() => {
      promRemoveAE();

      var removeAlarm = function(){
        return new Promise(function(resolve, reject){
          that.es.removeAlarmById(that.esIndex, that.esType, that.alarmIdToRemove).then(resAlarm => {
            if(resAlarm.deleted == 1) resolve('Deleting alarm '+ that.alarmIdToRemove + ' done');
          }, (error)=>{
            reject(error);
          });
        });
      }

      removeAlarm().then((c)=>{
        console.log(c);
        that.spinner.hide();
        that.isRemoved = true;
        that.tableData.splice(that.alarmIndexToRemove, 1);
        setTimeout(() => {
          that.isRemoved = false;
          that.disabledBtn = false;
          that.startStopTimer('on');
        }, 5000);
      }, 
      (error)=>{
        console.log(error);
        that.spinner.hide();
        that.isNotRemoved = true;
        that.errMsg = error;
        that.tableData.splice(that.alarmIndexToRemove, 1);
        setTimeout(() => {
          that.isNotRemoved = false;
          that.disabledBtn = false;
          that.startStopTimer('on');
        }, 5000);
      });

    }, 3000);

  }

  async removeAllAlarmEvent(){
    var that = this, arrDelete = [], size = 4500;

    var prom = function(){
      return new Promise(function(resolve, reject){
        that.es.getAllAlarmEvents(that.esIndexAlarmEvent, that.esType, that.alarmIdToRemove, size).then(res=>{
          var tempAlarmEvent = res.hits.hits;
          
          //delete alarm event
          for(var i=0; i <= tempAlarmEvent.length-1; i++){
            let idx = tempAlarmEvent[i]['_index'];
            arrDelete.push(
              {
                delete: {
                  _index: idx,
                  _type: tempAlarmEvent[i]['_type'],
                  _id: tempAlarmEvent[i]['_id']
                }
              }
            );
          }

          //delete event
          for(var i=0; i <= tempAlarmEvent.length-1; i++){
            let arridx = tempAlarmEvent[i]['_index'].split('-')[1];
            arrDelete.push(
              {
                delete: {
                  _index: 'siem_events-' + arridx,
                  _type: that.esType,
                  _id: tempAlarmEvent[i]['_source']['event_id']
                }
              }
            );
            if(i == tempAlarmEvent.length-1) resolve('done');
          }
        });
      });
    }

    return prom().then(()=>{
      this.es.removeAlarmEvent(arrDelete).then((r)=>{
      });
    });

  }

}

