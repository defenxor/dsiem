import { Component, OnInit, ViewChildren, QueryList } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { ElasticsearchService } from '../../elasticsearch.service';

@Component({
  templateUrl: './detailalarm.component.html',
})
export class DetailalarmComponent implements OnInit {

  @ViewChildren('pages') pages: QueryList<any>;
  private sub: any;
  private alarmID;
  private stage;
  public alarm = [];
  public alarmRules = [];
  public alarmVuln = [];
  public evnts = [];
  private isShowEventDetails: boolean;
  private static readonly ALARM_INDEX = 'siem_alarms';
  private static readonly ALARMEVENT_INDEX = 'siem_alarm_events-*';
  private static readonly EVENT_INDEX = 'siem_events-*';
  private static readonly TYPE = 'doc';
  totalItems;
  itemsPerPage = 5;
  numberOfVisiblePaginators = 10;
  numberOfPaginators: number;
  paginators: Array<any> = [];
  activePage = 1;
  firstVisibleIndex = 1;
  lastVisibleIndex: number = this.itemsPerPage;
  firstVisiblePaginator = 0;
  lastVisiblePaginator = this.numberOfVisiblePaginators;

  constructor(private route: ActivatedRoute, private es: ElasticsearchService) { }

  ngOnDestroy(){
    this.sub.unsubscribe();
  }

  ngOnInit() {
    this.sub = this.route.params.subscribe(params => {
      this.alarmID = params['alarmID'];
      this.getAlarmDetail(this.alarmID);
    })
  }

  async getAlarmDetail(alarmID){
    var that = this;
    let resp = await this.es.getAlarms(DetailalarmComponent.ALARM_INDEX, DetailalarmComponent.TYPE, alarmID)
    var tempAlarms = resp.hits.hits;
    await Promise.all(tempAlarms.map(async (e) => {
      await Promise.all(e["_source"]["rules"].map(async (r) => {
        if (r["status"] == "finished") {
          r["events_count"] = r["occurrence"]
          Promise.resolve()
        } else {
          let response = await this.es.countEvents("siem_alarm_events-*", alarmID, r["stage"])
          r["events_count"] = response.count  
        }
      }))
    }))
    this.alarm = tempAlarms;
    tempAlarms.forEach(element => {
      this.alarmRules = element._source.rules;
      this.es.getAlarmEventsPagination(DetailalarmComponent.ALARMEVENT_INDEX, DetailalarmComponent.TYPE, this.alarmID, this.alarmRules[0].stage, 0, this.itemsPerPage).then(function(alev){
        console.log(alev);
        if(alev['hits'] != undefined){
          that.getEventsDetail('init', that.alarmID, that.alarmRules[0].stage, null, null, that.alarmRules[0].events_count);
        } else {
          that.getEventsDetail('init', that.alarmID, that.alarmRules[1].stage, null, null, that.alarmRules[1].events_count);
        }
      });
      if(element._source.vulnerabilities) this.alarmVuln = element._source.vulnerabilities;
    });
  }

  setStatus(rule) {
    if (rule["status"] != "") return rule["status"]
    if (rule["status"] == "" && rule["start_time"] == 0) {
      return "inactive"
    }
    if (rule["status"] == "" && rule["start_time"] > 0) {
      return "active"
    }
    let deadline = rule["start_time"] + rule["timeout"]
    let now = Math.round((new Date()).getTime() / 1000)
    if (now > deadline) {
      return "timeout"
    }
  }

  getEventsDetail(type, id, stage, from=0, size=0, allsize=0){
    var that = this;
    that.evnts = [];
    that.paginators = [];
    that.stage = stage;
    that.totalItems = allsize;
    // that.isShowEventDetails = false;
    if(type == 'init'){
      from = 0;
      size = that.itemsPerPage;
    } else if(type == 'pagination'){
      from = from-1;
      size = size;
    }
    
    this.es.getAlarmEventsPagination(DetailalarmComponent.ALARMEVENT_INDEX, DetailalarmComponent.TYPE, id, stage, from, size).then(function(alev){
      console.log(alev['hits']['hits']);
      var prom = function(){
        return new Promise(function(resolve, reject){
          alev['hits']['hits'].forEach(element => {
            that.es.getEvents(DetailalarmComponent.EVENT_INDEX, DetailalarmComponent.TYPE, element['_source']['event_id']).then(function(ev){
              let jml = 0;
              ev['hits']['hits'].forEach(element2 => {
                that.evnts.push(element2['_source']);
                jml++;
                if(jml == ev['hits']['hits'].length) return resolve(that.evnts);
              });
            });
          });
        });
      }

      prom().then(v=>{
        that.isShowEventDetails = true;
        if (type == 'init') that.activePage = 1;
        if (that.totalItems % that.itemsPerPage === 0) {
          that.numberOfPaginators = Math.floor(that.totalItems / that.itemsPerPage);
        } else {
          that.numberOfPaginators = Math.floor(that.totalItems / that.itemsPerPage + 1);
        }
      
        for (let i = 1; i <= that.numberOfPaginators; i++) {
          that.paginators.push(i);
        }
      })
    }).catch(err=>{
      console.log('ERROR: ', err);
    })
  }

  async changePage(event: any) {
    if (event.target.text >= 1 && event.target.text <= this.numberOfPaginators) {
      this.activePage = +event.target.text;
      this.firstVisibleIndex = this.activePage * this.itemsPerPage - this.itemsPerPage + 1;
      this.lastVisibleIndex = this.activePage * this.itemsPerPage;
      console.log(this.firstVisibleIndex + ' - ' + this.lastVisibleIndex);
      this.getEventsDetail('pagination', this.alarmID, this.stage, this.firstVisibleIndex, this.itemsPerPage, this.totalItems);
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
    this.getEventsDetail('pagination', this.alarmID, this.stage, this.firstVisibleIndex, this.itemsPerPage, this.totalItems);
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
    this.getEventsDetail('pagination', this.alarmID, this.stage, this.firstVisibleIndex, this.itemsPerPage, this.totalItems);
  }

  firstPage() {
    this.activePage = 1;
    this.firstVisibleIndex = this.activePage * this.itemsPerPage - this.itemsPerPage + 1;
    this.lastVisibleIndex = this.activePage * this.itemsPerPage;
    this.firstVisiblePaginator = 0;
    this.lastVisiblePaginator = this.numberOfVisiblePaginators;
    this.getEventsDetail('pagination', this.alarmID, this.stage, this.firstVisibleIndex, this.itemsPerPage, this.totalItems);
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
    this.getEventsDetail('pagination', this.alarmID, this.stage, this.firstVisibleIndex, this.itemsPerPage, this.totalItems);
  }

}
