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

