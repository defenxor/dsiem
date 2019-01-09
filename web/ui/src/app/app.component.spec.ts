import { RouterTestingModule } from '@angular/router/testing';
import { TestBed, async } from '@angular/core/testing';
import { AppComponent } from './app.component';
import { HttpModule } from "@angular/http";
import { ElasticsearchService } from './elasticsearch.service';
import { of } from 'rxjs';

describe('App Component', () => {
  let serviceStub;
  let fixture;
  let app;

  beforeEach(async(() => {
    serviceStub = {
      isAvailable: () => new Promise((resolve)=>{ resolve('')}),
      getServer: () => of(),
    }

    TestBed.configureTestingModule({
      declarations: [
        AppComponent
      ],
      imports: [ 
        RouterTestingModule,
        HttpModule
      ],
      providers: [
        { provide: ElasticsearchService, useValue: serviceStub }
      ]
    }).compileComponents();
  }));

  beforeEach(()=>{
    fixture = TestBed.createComponent(AppComponent);
    app = fixture.debugElement.componentInstance;
  })

  afterEach(()=>{
    app.checkES = ()=>{
      return null;
    }
  })

  it('should create the app', () => {
    app.checkES();
    expect(app).toBeTruthy();
  });

});
