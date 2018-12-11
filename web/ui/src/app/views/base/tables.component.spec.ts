import { TestBed, async, fakeAsync } from '@angular/core/testing';
import { TablesComponent } from './tables.component';
import { RouterTestingModule } from '@angular/router/testing';
import { NgxSpinnerModule, NgxSpinnerService } from 'ngx-spinner';
import { ModalModule, AlertModule, ModalDirective } from 'ngx-bootstrap';
import { MomentModule } from 'ngx-moment';
import { HttpModule } from '@angular/http';
import { ElasticsearchService } from '../../elasticsearch.service';
import { timer } from 'rxjs';

describe('Alarm List Component', ()=>{

  let fixture;
  let app;

  beforeEach(async(() => {
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
        HttpModule
      ],
      providers: [
        NgxSpinnerService,
        ElasticsearchService
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(TablesComponent);
    app = fixture.debugElement.componentInstance;
    app.timerSubscription =  timer(9000).subscribe();
    
  }));

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

  it('timer should off when turn-off button clicked', fakeAsync(()=>{
    app.startStopTimer('off');
    expect(app.timer_status).toBe('off');
  }));

});