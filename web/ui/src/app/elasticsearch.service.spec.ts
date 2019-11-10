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
import { TestBed } from '@angular/core/testing';
import { ElasticsearchService } from './elasticsearch.service';
import { HttpClientModule } from '@angular/common/http';

describe('Elasticsearch Service', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [
      ],
      imports: [
        HttpClientModule
      ]
    }).compileComponents();
  });

  it('should be created', () => {
    const service: ElasticsearchService = TestBed.get(ElasticsearchService);
    expect(service).toBeTruthy();
  });
});
