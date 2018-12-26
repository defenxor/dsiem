import { DetailalarmComponent } from "./detailalarm.component";
import { async, TestBed } from "@angular/core/testing";
import { NgxSpinnerModule, NgxSpinnerService } from "ngx-spinner";
import { MomentModule } from "ngx-moment";
import { HttpModule } from "@angular/http";
import { RouterTestingModule } from "@angular/router/testing";
import { ElasticsearchService } from "../../elasticsearch.service";
import { ModalModule, AlertModule, TooltipModule } from "ngx-bootstrap";
import { of } from "rxjs";

describe('Detail Alarm Component', ()=>{

  let fixture;
  let app: DetailalarmComponent;
  let alarmID: string = 'iM0V7PdTp';
  let serviceStub;
  let responseAlarmDetail;
  let responseCount;
  let responseAlarmEvent;
  let responseEvents;
  let originalTimeout;

  beforeEach(async(() => {

    responseAlarmDetail = {
      "hits": {
        "hits": [
          {
            "_source":
              {
                "@timestamp": "2018-12-17T17:19:52.171Z",
                "category": "Misc Activity",
                "dst_ips": [
                  "10.8.100.1"
                ],
                "kingdom": "Reconnaissance & Probing",
                "networks": [
                  "10.0.0.0/8"
                ],
                "risk": 1,
                "risk_class": "Low",
                "rules": [
                  {
                    "category": "",
                    "end_time": 1545067044,
                    "events_count": 1,
                    "from": "ANY",
                    "name": "ICMP Ping",
                    "occurrence": 1,
                    "plugin_id": 1001,
                    "plugin_sid": [
                      2100384
                    ],
                    "port_from": "ANY",
                    "port_to": "ANY",
                    "protocol": "ICMP",
                    "rcvd_time": 1545067090,
                    "reliability": 1,
                    "stage": 1,
                    "start_time": 1545067044,
                    "status": "finished",
                    "timeout": 0,
                    "to": "HOME_NET",
                    "type": "PluginRule"
                  },
                  {
                    "category": "",
                    "end_time": 1545067146,
                    "events_count": 6,
                    "from": "ANY",
                    "name": "ICMP Ping",
                    "occurrence": 400,
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
                    "start_time": 1545067044,
                    "status": "finished",
                    "timeout": 3600,
                    "to": "10.8.100.1",
                    "type": "PluginRule"
                  },
                  {
                    "category": "",
                    "end_time": 0,
                    "events_count": 0,
                    "from": "ANY",
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
                    "to": "10.8.100.1",
                    "type": "PluginRule"
                  }
                ],
                "src_ips": [
                  "10.8.100.58"
                ],
                "status": "Open",
                "tag": "Identified Threat",
                "timestamp": "2018-12-17T17:18:10.000Z",
                "title": "Ping Flood to 10.8.100.1",
                "updated_time": "2018-12-17T17:19:51.000Z",
                "intel_hits": [
                  {
                    "provider": "Wise",
                    "term": "115.79.79.91",
                    "result": "Malicious Host"
                  }
                ],
                "vulnerabilities": [
                  {
                    "provider": "Nessus",
                    "term": "10.23.51.67:88",
                    "result": "Critical - PHP Unsupported Version Detection"
                  },
                  {
                    "provider": "Nessus",
                    "term": "10.23.51.67:88",
                    "result": "High - PHP 5.4.x < 5.4.17 Buffer Overflow"
                  }
                ]
              }
          }
        ]
      }
    }

    responseCount = {
      count: 10,
    }

    responseAlarmEvent = {
      "hits": {
        "hits":
        [
          {
            "_source": {
              "@timestamp": "2018-12-17T17:16:20.063Z",
              "alarm_id": "xblyZpeTp",
              "event_id": "e65294a7-17f3-46d8-a71d-2e1cf7066e2a",
              "stage": 2
            }
          },
          {
            "_source": {
              "@timestamp": "2018-12-17T17:18:17.171Z",
              "alarm_id": "xblyZpeTp",
              "event_id": "5f03dfbc-45c7-42f3-8442-8e4556ab7ebb",
              "stage": 2
            }
          },
          {
            "_source": {
              "@timestamp": "2018-12-17T17:20:20.063Z",
              "alarm_id": "xblyZpeTp",
              "event_id": "a65294a7-17f3-46d8-a71d-2e1cf7066abc",
              "stage": 2
            }
          },
          {
            "_source": {
              "@timestamp": "2018-12-17T17:21:17.171Z",
              "alarm_id": "xblyZpeTp",
              "event_id": "8f03dfbc-45c7-42f3-8442-8e4556ab7def",
              "stage": 2
            }
          },
          {
            "_source": {
              "@timestamp": "2018-12-17T17:23:20.063Z",
              "alarm_id": "xblyZpeTp",
              "event_id": "g65294a7-17f3-46d8-a71d-2e1cf7066ghi",
              "stage": 2
            }
          },
          {
            "_source": {
              "@timestamp": "2018-12-17T17:24:17.171Z",
              "alarm_id": "xblyZpeTp",
              "event_id": "7f03dfbc-45c7-42f3-8442-8e4556ab7jkl",
              "stage": 2
            }
          }
        ]
      }
    }

    responseEvents = {
      "hits": {
        "hits": [
          {
            "_source": {
              "@timestamp": "2018-12-17T17:17:32.036Z",
              "category": "Attempted Information Leak",
              "dst_ip": "10.7.105.191",
              "dst_port": 22,
              "event_id": "5f03dfbc-45c7-42f3-8442-8e4556ab7ebb",
              "plugin_id": 1001,
              "plugin_sid": 2001219,
              "product": "Intrusion Detection System",
              "protocol": "TCP",
              "sensor": "k8sworker1d",
              "src_index_pattern": "suricata-*",
              "src_ip": "10.8.100.58",
              "src_port": 50341,
              "timestamp": "2018-12-17T17:17:31.083Z",
              "title": "ET SCAN Potential SSH Scan"
            }
          },
          {
            "_source": {
              "@timestamp": "2018-12-17T17:17:32.036Z",
              "category": "Attempted Information Leak",
              "dst_ip": "10.7.105.187",
              "dst_port": 22,
              "event_id": "e65294a7-17f3-46d8-a71d-2e1cf7066e2a",
              "plugin_id": 1001,
              "plugin_sid": 2001219,
              "product": "Intrusion Detection System",
              "protocol": "TCP",
              "sensor": "k8sworker1d",
              "src_index_pattern": "suricata-*",
              "src_ip": "10.8.100.58",
              "src_port": 50341,
              "timestamp": "2018-12-17T17:17:31.083Z",
              "title": "ET SCAN Potential SSH Scan"
            }
          },
          {
            "_source": {
              "@timestamp": "2018-12-17T17:17:32.036Z",
              "category": "Attempted Information Leak",
              "dst_ip": "10.7.105.10",
              "dst_port": 22,
              "event_id": "a65294a7-17f3-46d8-a71d-2e1cf7066abc",
              "plugin_id": 1001,
              "plugin_sid": 2001219,
              "product": "Intrusion Detection System",
              "protocol": "TCP",
              "sensor": "k8sworker1d",
              "src_index_pattern": "suricata-*",
              "src_ip": "10.8.100.58",
              "src_port": 50341,
              "timestamp": "2018-12-17T17:17:31.083Z",
              "title": "ET SCAN Potential SSH Scan"
            }
          },
          {
            "_source": {
              "@timestamp": "2018-12-17T17:17:32.036Z",
              "category": "Attempted Information Leak",
              "dst_ip": "10.7.105.8",
              "dst_port": 22,
              "event_id": "8f03dfbc-45c7-42f3-8442-8e4556ab7def",
              "plugin_id": 1001,
              "plugin_sid": 2001219,
              "product": "Intrusion Detection System",
              "protocol": "TCP",
              "sensor": "k8sworker1d",
              "src_index_pattern": "suricata-*",
              "src_ip": "10.8.100.58",
              "src_port": 50341,
              "timestamp": "2018-12-17T17:17:31.083Z",
              "title": "ET SCAN Potential SSH Scan"
            }
          },
          {
            "_source": {
              "@timestamp": "2018-12-17T17:17:32.036Z",
              "category": "Attempted Information Leak",
              "dst_ip": "10.7.105.153",
              "dst_port": 22,
              "event_id": "g65294a7-17f3-46d8-a71d-2e1cf7066ghi",
              "plugin_id": 1001,
              "plugin_sid": 2001219,
              "product": "Intrusion Detection System",
              "protocol": "TCP",
              "sensor": "k8sworker1d",
              "src_index_pattern": "suricata-*",
              "src_ip": "10.8.100.58",
              "src_port": 50341,
              "timestamp": "2018-12-17T17:17:31.083Z",
              "title": "ET SCAN Potential SSH Scan"
            }
          },
          {
            "_source": {
              "@timestamp": "2018-12-17T17:17:32.036Z",
              "category": "Attempted Information Leak",
              "dst_ip": "10.7.105.80",
              "dst_port": 22,
              "event_id": "7f03dfbc-45c7-42f3-8442-8e4556ab7jkl",
              "plugin_id": 1001,
              "plugin_sid": 2001219,
              "product": "Intrusion Detection System",
              "protocol": "TCP",
              "sensor": "k8sworker1d",
              "src_index_pattern": "suricata-*",
              "src_ip": "10.8.100.58",
              "src_port": 50341,
              "timestamp": "2018-12-17T17:17:31.083Z",
              "title": "ET SCAN Potential SSH Scan"
            }
          }
        ]
      }
    }

    serviceStub = {
      getAlarms: () => new Promise((resolve)=>{ resolve(responseAlarmDetail)}),
      getServer: () => of(),
      countEvents: () => responseCount,
      getAlarmEventsPagination: () => new Promise((resolve)=>{ resolve(responseAlarmEvent)}),
      getEvents: () => new Promise((resolve)=>{ resolve(responseEvents)}),
      updateAlarmStatusById: () => new Promise((resolve)=>{ resolve('')}),
      updateAlarmTagById: () => new Promise((resolve)=>{ resolve('')})
    }

    TestBed.configureTestingModule({
      declarations: [
        DetailalarmComponent
      ],
      imports: [ 
        RouterTestingModule,
        NgxSpinnerModule,
        ModalModule.forRoot(),
        AlertModule.forRoot(),
        TooltipModule.forRoot(),
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
    jasmine.DEFAULT_TIMEOUT_INTERVAL = 15000;
    fixture = TestBed.createComponent(DetailalarmComponent);
    app = fixture.debugElement.componentInstance;
    fixture.detectChanges();
  });

  afterEach(()=>{
    app.alarm = [];
    app.alarmRules = [];
    app.alarmVuln = [];
    app.alarmIntelHits = [];
    app.evnts = [];
    jasmine.DEFAULT_TIMEOUT_INTERVAL = originalTimeout;
    fixture.detectChanges();
  })

  it('should create the app', () => {
    expect(app).toBeTruthy();
  });

  it('elasticsearch alarm index should be siem_alarms', (done) => {
    app.alarmID = alarmID;
    fixture.detectChanges();
    setTimeout(() => {
      fixture.detectChanges();
      expect(app.esIndex).toContain('siem_alarms');
      done();
    }, 100);
  });

  it('elasticsearch alarm event index should be siem_alarm_events-*', (done) => {
    app.alarmID = alarmID;
    fixture.detectChanges();
    setTimeout(() => {
      fixture.detectChanges();
      expect(app.esIndexAlarmEvent).toContain('siem_alarm_events-*');
      done();
    }, 100);
  });

  it('elasticsearch event index should be siem_events-*', (done) => {
    app.alarmID = alarmID;
    fixture.detectChanges();
    setTimeout(() => {
      fixture.detectChanges();
      expect(app.esIndexEvent).toContain('siem_events-*');
      done();
    }, 100);
  });

  it('elasticsearch type should be doc', (done) => {
    app.alarmID = alarmID;
    fixture.detectChanges();
    setTimeout(() => {
      fixture.detectChanges();
      expect(app.esType).toContain('doc');
      done();
    }, 100);
  });

  it('should have initial 5 data displayed per page', (done) => {
    app.alarmID = alarmID;
    fixture.detectChanges();
    setTimeout(() => {
      fixture.detectChanges();
      expect(app.itemsPerPage).toEqual(5);
      done();
    }, 100);
  });

  it('should have initial 10 number of visible paginators', (done) => {
    app.alarmID = alarmID;
    fixture.detectChanges();
    setTimeout(() => {
      fixture.detectChanges();
      expect(app.numberOfVisiblePaginators).toEqual(10);
      done();
    }, 100);
  });

  it('should return alarm detail data', (done)=>{
    app.getAlarmDetail(alarmID);
    app.alarmID = alarmID;
    fixture.detectChanges();
    setTimeout(() => {
      fixture.detectChanges();
      expect(app.alarm).toEqual(responseAlarmDetail.hits.hits);
      done();
    }, 100);
  });

  it('should have alarm title on card header', (done)=>{
    app.alarmID = alarmID;
    fixture.detectChanges();
    setTimeout(() => {
      fixture.detectChanges();
      const title = fixture.nativeElement.querySelector('.card-header').textContent;
      expect(title).toContain(responseAlarmDetail.hits.hits[0]['_source']['title']);
      done();
    }, 100);
  });

  it('should return alarm id on datatable', (done)=>{
    app.alarmID = alarmID;
    fixture.detectChanges();
    setTimeout(() => {
      fixture.detectChanges();
      const alarmid = document.querySelectorAll('label')[0].textContent;
      expect(alarmid).toEqual(alarmID);
      done();
    }, 100);
  });

  it('should return alarm status on datatable', (done)=>{
    app.alarmID = alarmID;
    fixture.detectChanges();
    setTimeout(() => {
      fixture.detectChanges();
      const status = document.querySelectorAll('label')[1].textContent;
      expect(status).toEqual(responseAlarmDetail.hits.hits[0]['_source']['status']);
      done();
    }, 100);
  });

  it('should return alarm risk on datatable', (done)=>{
    app.alarmID = alarmID;
    fixture.detectChanges();
    setTimeout(() => {
      fixture.detectChanges();
      const risk = document.querySelectorAll('label')[2].textContent;
      expect(risk).toEqual(responseAlarmDetail.hits.hits[0]['_source']['risk_class']);
      done();
    }, 100);
  });

  it('should return alarm tag on datatable', (done)=>{
    app.alarmID = alarmID;
    fixture.detectChanges();
    setTimeout(() => {
      fixture.detectChanges();
      const tag = document.querySelectorAll('label')[3].textContent;
      expect(tag).toEqual(responseAlarmDetail.hits.hits[0]['_source']['tag']);
      done();
    }, 100);
  });

  it('should return alarm source ips on datatable', (done)=>{
    const tempArray = [];
    app.alarmID = alarmID;
    fixture.detectChanges();
    setTimeout(() => {
      fixture.detectChanges();
      const source_ips = document.querySelectorAll('label')[4].textContent;
      tempArray.push(source_ips);
      expect(tempArray).toEqual(responseAlarmDetail.hits.hits[0]['_source']['src_ips']);
      done();
    }, 100);
  });

  it('should have title \'rules\' on card header', (done)=>{
    app.alarmID = alarmID;
    fixture.detectChanges();
    setTimeout(() => {
      fixture.detectChanges();
      const title = document.querySelectorAll('.card-header')[1].textContent;
      expect(title).toContain('Rules');
      done();
    }, 100);
  });

  it('should return alarm rules data', (done)=>{
    app.alarmID = alarmID;
    fixture.detectChanges();
    setTimeout(() => {
      fixture.detectChanges();
      expect(app.alarmRules).toEqual(responseAlarmDetail.hits.hits[0]['_source']['rules']);
      done();
    }, 100);
  });

  it('should return alarm intel hits data if exists', (done)=>{
    app.alarmID = alarmID;
    fixture.detectChanges();
    setTimeout(() => {
      fixture.detectChanges();
      expect(app.alarmIntelHits).toEqual(responseAlarmDetail.hits.hits[0]['_source']['intel_hits']);
      done();
    }, 100);
  });

  it('should return alarm vulnerabilities data if exists', (done)=>{
    app.alarmID = alarmID;
    fixture.detectChanges();
    setTimeout(() => {
      fixture.detectChanges();
      expect(app.alarmVuln).toEqual(responseAlarmDetail.hits.hits[0]['_source']['vulnerabilities']);
      done();
    }, 100);
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

});
