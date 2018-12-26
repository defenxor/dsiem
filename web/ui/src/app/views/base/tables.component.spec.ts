import { TestBed, async, fakeAsync } from '@angular/core/testing';
import { TablesComponent } from './tables.component';
import { RouterTestingModule } from '@angular/router/testing';
import { NgxSpinnerModule, NgxSpinnerService } from 'ngx-spinner';
import { ModalModule, AlertModule } from 'ngx-bootstrap';
import { MomentModule } from 'ngx-moment';
import { HttpModule } from '@angular/http';
import { ElasticsearchService } from '../../elasticsearch.service';
import { timer, of } from 'rxjs';

describe('Alarm List Component', ()=>{

  let fixture;
  let app: TablesComponent;
  let serviceStub: any;
  let responseAllDocument;
  let responseCount;
  let responseRemoveById;
  let originalTimeout;
  
  beforeEach(async(() => {

    responseAllDocument = {
      "hits": {
        "hits": [
          {
            "sort": [
              1544338159026
            ],
            "_id": "iM0V7PdTp",
            "_index": "siem_alarms",
            "_score": null,
            "_source": {
                "@timestamp": "2018-12-09T06:49:19.026Z",
                "category": "Misc Activity",
                "dst_ips": [
                  "10.8.100.1"
                ],
                "id": "iM0V7PdTp",
                "kingdom": "Reconnaissance & Probing",
                "networks": [
                  "10.0.0.0/8"
                ],
                "risk": 1,
                "risk_class": "Low",
                "rules": [
                  {
                    "category": "",
                    "end_time": 1544338032,
                    "events_count": 1,
                    "from": "HOME_NET",
                    "name": "ICMP Ping",
                    "occurrence": 1,
                    "plugin_id": 1001,
                    "plugin_sid": [
                      2100384
                    ],
                    "port_from": "ANY",
                    "port_to": "ANY",
                    "protocol": "ICMP",
                    "rcvd_time": 1544338073,
                    "reliability": 1,
                    "stage": 1,
                    "start_time": 1544338032,
                    "status": "finished",
                    "timeout": 0,
                    "to": "ANY",
                    "type": "PluginRule"
                  },
                  {
                    "category": "",
                    "end_time": 1544338109,
                    "events_count": 300,
                    "from": "10.8.100.58",
                    "name": "ICMP Ping",
                    "occurrence": 300,
                    "plugin_id": 1001,
                    "plugin_sid": [
                      2100384
                    ],
                    "port_from": "ANY",
                    "port_to": "ANY",
                    "protocol": "ICMP",
                    "rcvd_time": 0,
                    "reliability": 6,
                    "stage": 2,
                    "start_time": 1544338032,
                    "status": "finished",
                    "timeout": 600,
                    "to": "ANY",
                    "type": "PluginRule"
                  },
                  {
                    "category": "",
                    "end_time": 0,
                    "events_count": 917,
                    "from": "10.8.100.58",
                    "name": "ICMP Ping",
                    "occurrence": 10000,
                    "plugin_id": 1001,
                    "plugin_sid": [
                      2100384
                    ],
                    "port_from": "ANY",
                    "port_to": "ANY",
                    "protocol": "ICMP",
                    "rcvd_time": 0,
                    "reliability": 10,
                    "stage": 3,
                    "start_time": 0,
                    "status": "",
                    "timeout": 3600,
                    "to": "ANY",
                    "type": "PluginRule"
                  }
                ],
                "src_ips": [
                  "10.8.100.58"
                ],
                "status": "Open",
                "tag": "Identified Threat",
                "timestamp": "2018-12-09T06:47:53.000Z",
                "title": "Ping Flood from 10.8.100.58",
                "updated_time": "2018-12-09T06:49:10.000Z"
            },
            "_type": "doc"
          }
        ],
        "max_score": null,
        "total": 364292
      },
      "timed_out": false,
      "took": 8,
      "_shards": {
        "failed": 0,
        "skipped": 0,
        "successful": 2,
        "total": 2
      }
    }

    responseCount = {
      count: 10,
    }

    responseRemoveById = {
      deleted: 1,
    }

    serviceStub = {
      getAllDocumentsPaging: () => responseAllDocument,
      getServer: () => of(),
      countEvents: () => responseCount,
      getAlarmEventsWithoutStage: ()  => new Promise((resolve)=>{ resolve(responseAllDocument)}),
      removeAlarmById: () => new Promise((resolve)=>{ resolve(responseRemoveById)}),
      getAllAlarmEvents: () => new Promise((resolve)=>{ resolve(responseAllDocument)}),
      removeAlarmEvent: () => new Promise((resolve)=>{ resolve('')})
    }

    TestBed.configureTestingModule({
      declarations: [
        TablesComponent
      ],
      imports: [ 
        RouterTestingModule,
        NgxSpinnerModule,
        ModalModule.forRoot(),
        AlertModule.forRoot(),
        MomentModule,
        HttpModule,
      ],
      providers: [
        NgxSpinnerService,
        { provide: ElasticsearchService, useValue: serviceStub }
      ]
    }).compileComponents();
  }));

  beforeEach(()=>{
    originalTimeout = jasmine.DEFAULT_TIMEOUT_INTERVAL;
    jasmine.DEFAULT_TIMEOUT_INTERVAL = 60000;
    fixture = TestBed.createComponent(TablesComponent);
    app = fixture.debugElement.componentInstance;
    fixture.detectChanges();
  });

  afterEach(()=>{
    jasmine.DEFAULT_TIMEOUT_INTERVAL = originalTimeout;
    fixture.detectChanges();
  });

  it('should create the app', () => {
    expect(app).toBeTruthy();
  });

  it('elasticsearch alarm index should be siem_alarms', () => {
    expect(app.esIndex).toContain('siem_alarms');
  });

  it('elasticsearch alarm event index should be siem_alarm_events-*', () => {
    expect(app.esIndexAlarmEvent).toContain('siem_alarm_events-*');
  });

  it('elasticsearch event index should be siem_events-*', () => {
    expect(app.esIndexEvent).toContain('siem_events-*');
  });

  it('elasticsearch type should be doc', () => {
    expect(app.esType).toContain('doc');
  });

  it('shoud have alarm list title', ()=>{
    fixture.detectChanges();
    const title = fixture.nativeElement.querySelector('.card-header').textContent;
    expect(title).toContain('Alarm List');
  });

  it('shoud have warning modal title', ()=>{
    fixture.detectChanges();
    const title = fixture.nativeElement.querySelector('#myModalLabel').textContent;
    expect(title).toContain('Warning');
  });

  it('shoud have turn-off button when timer is on', ()=>{
    fixture.detectChanges();
    const title = fixture.nativeElement.querySelector('.btn-primary').textContent;
    expect(title).toContain('Turn-Off Auto Refresh');
  });

  it('shoud have turn-on button when timer is off', ()=>{
    app.timer_status = 'off';
    fixture.detectChanges();
    const title = fixture.nativeElement.querySelector('.btn-dark').textContent;
    expect(title).toContain('Turn On Auto Refresh');
  });

  it('should have initial 20 total data displayed', () => {
    expect(app.totalItems).toEqual(20);
  });

  it('should have initial 10 data displayed per page', () => {
    expect(app.itemsPerPage).toEqual(10);
  });

  it('should have initial timer status on', () => {
    expect(app.timer_status).toBe('on');
  });

  it('should have initial 10 number of visible paginators', () => {
    expect(app.numberOfVisiblePaginators).toEqual(10);
  });

  it('timer should off when turn-off button clicked', fakeAsync(()=>{
    app.timerSubscription =  timer(9000).subscribe();
    app.startStopTimer('off');
    expect(app.timer_status).toBe('off');
    app.timerSubscription.unsubscribe();
  }));

  it('shoud have alert success when alarm deleted succesfully', ()=>{
    app.isRemoved = true;
    fixture.detectChanges();
    const title = fixture.nativeElement.querySelector('#alert-success').textContent;
    expect(title).toContain('successfully removed');
    app.isRemoved = false;
  });

  it('shoud have alert danger when alarm deleted occured error', ()=>{
    app.isNotRemoved = true;
    fixture.detectChanges();
    const title = fixture.nativeElement.querySelector('#alert-failed').textContent;
    expect(title).toContain('Error!');
    app.isNotRemoved = false;
  });

  it('shoud have alarm list header table ', ()=>{
    const title = fixture.nativeElement.querySelector('tr').textContent;
    expect(title).toContain('Action');
    expect(title).toContain('AlarmID');
    expect(title).toContain('Title');
    expect(title).toContain('Created');
    expect(title).toContain('Updated');
    expect(title).toContain('Status');
    expect(title).toContain('Risk');
    expect(title).toContain('Tag');
    expect(title).toContain('Sources');
    expect(title).toContain('Destinations');
  });

  it('should return alarm data', (done)=>{
    app.getData('init');
    setTimeout(() => {
      fixture.detectChanges();
      expect(app.tempAlarms).toEqual(responseAllDocument.hits.hits);
      done();
    }, 1000);
  });

  it('should return alarm id on datatable', (done)=>{
    app.getData('init');
    setTimeout(() => {
      fixture.detectChanges();
      const id = fixture.nativeElement.querySelector('.table').textContent;
      expect(id).toContain(app.tableData[0]['id']);
      done();
    }, 1000);
  });

  it('should return alarm title on datatable', (done)=>{
    app.getData('init');
    setTimeout(() => {
      fixture.detectChanges();
      const title = fixture.nativeElement.querySelector('.table').textContent;
      expect(title).toContain(app.tableData[0]['title']);
      done();
    }, 1000);
  });

  it('should return alarm status on datatable', (done)=>{
    app.getData('init');
    setTimeout(() => {
      fixture.detectChanges();
      const status = fixture.nativeElement.querySelector('.table').textContent;
      expect(status).toContain(app.tableData[0]['status']);
      done();
    }, 1000);
  });

  it('should return alarm risk on datatable', (done)=>{
    app.getData('init');
    setTimeout(() => {
      fixture.detectChanges();
      const risk = fixture.nativeElement.querySelector('.table').textContent;
      expect(risk).toContain(app.tableData[0]['risk_class']);
      done();
    }, 1000);
  });

  it('should return alarm tag on datatable', (done)=>{
    app.getData('init');
    setTimeout(() => {
      fixture.detectChanges();
      const tag = fixture.nativeElement.querySelector('.table').textContent;
      expect(tag).toContain(app.tableData[0]['tag']);
      done();
    }, 1000);
  });

  it('should return alarm source ip on datatable', (done)=>{
    app.getData('init');
    setTimeout(() => {
      fixture.detectChanges();
      const src_ips = fixture.nativeElement.querySelector('.table').textContent;
      expect(src_ips).toContain(app.tableData[0]['src_ips']);
      done();
    }, 1000);
  });

  it('should have link to first page on pagination', ()=>{
    fixture.detectChanges();
    const first = fixture.nativeElement.querySelector('.pagination').textContent;
    expect(first).toContain('First');
  });

  it('should have link to previous page on pagination', ()=>{
    fixture.detectChanges();
    const prev = fixture.nativeElement.querySelector('.pagination').textContent;
    expect(prev).toContain('Previous');
  });

  it('should have link to next page on pagination', ()=>{
    fixture.detectChanges();
    const next = fixture.nativeElement.querySelector('.pagination').textContent;
    expect(next).toContain('Next');
  });

  it('should have link to last page on pagination', ()=>{
    fixture.detectChanges();
    const last = fixture.nativeElement.querySelector('.pagination').textContent;
    expect(last).toContain('Last');
  });

  it('active page should first page when first link on pagination clicked', ()=>{
    app.firstPage();
    fixture.detectChanges();
    expect(app.activePage).toEqual(1);
  });

  it('active page should last page when last link on pagination clicked', (done)=>{
    app.getData('init');
    setTimeout(() => {
      app.lastPage();
      fixture.detectChanges();
      const lastPage = app.numberOfPaginators;
      const activePage = app.activePage;
      expect(activePage).toEqual(lastPage);
      done();
    }, 1000);
  });

  it('active page should page 2 when link 2 on pagination clicked', (done)=>{
    app.getData('init');
    setTimeout(() => {
      const destinationPage = 2;
      app.changePage({ 'target' : { 'text' : destinationPage }});
      fixture.detectChanges();
      expect(app.activePage).toEqual(destinationPage);
      done();
    }, 1000);
  });

  it('should remove alarm', (done)=>{
    app.timerSubscription = timer(9000).subscribe();
    app.getData('init');
    setTimeout(() => {
      fixture.detectChanges();
    }, 1000);
    setTimeout(() => {
      app.alarmIdToRemove = 'iM0V7PdTp';
      app.alarmIndexToRemove = 0;
      app.removeAlarm();
      fixture.detectChanges();
    }, 10000);
    setTimeout(() => {
      fixture.detectChanges();
      expect(app.tableData).toEqual([]);
      app.timerSubscription.unsubscribe();
      done();
    }, 15000);
  });

});