import { TestBed, async } from '@angular/core/testing';
import { TablesComponent } from './tables.component';
import { RouterTestingModule } from '@angular/router/testing';
import { NgxSpinnerModule, NgxSpinnerService } from 'ngx-spinner';
import { ModalModule, AlertModule, ModalDirective } from 'ngx-bootstrap';
import { MomentModule } from 'ngx-moment';
import { HttpModule } from '@angular/http';
import { ElasticsearchService } from '../../elasticsearch.service';

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
    
  }));

  it('should create the app', () => {
    expect(app).toBeTruthy();
  });

  it('elasticsearch alarm index should be siem_alarms', () => {
    expect(app.esIndex).toContain('siem_alarms');
  });

});