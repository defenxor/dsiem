import { RouterTestingModule } from '@angular/router/testing';
import { TestBed, async, ComponentFixture } from '@angular/core/testing';
import { AppComponent } from './app.component';
import { HttpModule } from '@angular/http';
import { ElasticsearchService } from './elasticsearch.service';
import { of } from 'rxjs';

describe('App Component', () => {
  let serviceStub;
  let fixture: ComponentFixture<AppComponent>;
  let app;
  let esService: ElasticsearchService;

  beforeEach(async(() => {

    serviceStub = {
      isAvailable: () => new Promise((resolve, reject) => { resolve('done'); }),
      getServer: () => of(),
    };

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

  beforeEach(() => {
    fixture = TestBed.createComponent(AppComponent);
    app = fixture.debugElement.componentInstance;
    esService = TestBed.get(ElasticsearchService);
  });

  afterEach(() => {
    app.checkES = () => {
      return null;
    };
  });

  it('should create the app', () => {
    app.checkES();
    app.es.isAvailable().then(() => {

    });
    expect(app).toBeTruthy();
  });

  it('should resolve isAvailable method', () => {
    const spy = spyOn(esService, 'isAvailable').and.callFake(() => {
      return Promise.resolve();
    });
    const component = fixture.componentInstance;
    component.checkES();
    expect(spy).toHaveBeenCalled();
  });

  it('should reject isAvailable method', () => {
    const spy = spyOn(esService, 'isAvailable').and.callFake(() => {
      return Promise.reject();
    });
    const component = fixture.componentInstance;
    component.checkES();
    expect(spy).toHaveBeenCalled();
  });

});

