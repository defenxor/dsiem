import { Component, AfterViewInit, ViewChildren, QueryList } from '@angular/core';
import { ElasticsearchService } from '../../elasticsearch.service';
import { AlarmSource } from './alarm.interface';
import { timer } from 'rxjs';

@Component({
  templateUrl: 'tables.component.html'
})
export class TablesComponent implements AfterViewInit {

  @ViewChildren('pages') pages: QueryList<any>;
  private static readonly INDEX = 'siem_alarms';
  private static readonly TYPE = 'doc';
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

  constructor(private es: ElasticsearchService) {
    this.elasticsearch = this.es.getServer();
  }

  ngAfterViewInit(){
    setTimeout(()=>{
      this.getData('init');
    }, 100);
  }

  async getData(type, from=0, size=0) {
    var that = this;
    try {
      let resp;
      if(type == 'init'){
        resp = await this.es.getAllDocumentsPaging(TablesComponent.INDEX, TablesComponent.TYPE, 0, this.itemsPerPage);
      } else if(type == 'pagination'){
        resp = await this.es.getAllDocumentsPaging(TablesComponent.INDEX, TablesComponent.TYPE, from-1, size);
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
    } catch (err) {
      console.error('Error: ' + err);
      this.tableData = [];
      this.paginators = [];
    } finally {
      if(type == 'init'){
        this.timerSubscription = timer(9000).subscribe(() => this.getData('init'));
      }
    }
  }

}
