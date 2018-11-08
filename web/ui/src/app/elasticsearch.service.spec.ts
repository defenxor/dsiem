import { TestBed, inject } from '@angular/core/testing';

import { ElasticsearchService } from './elasticsearch.service';

describe('ElasticsearchService', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [ElasticsearchService]
    });
  });

  it('should be created', inject([ElasticsearchService], (service: ElasticsearchService) => {
    expect(service).toBeTruthy();
  }));
});
