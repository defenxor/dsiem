import { TestBed } from '@angular/core/testing';
import { ElasticsearchService } from './elasticsearch.service';
import { HttpModule } from '@angular/http';

describe('Elasticsearch Service', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [
      ],
      imports: [ 
        HttpModule
      ]
    }).compileComponents();
  });
  
  it('should be created', () => {
    const service: ElasticsearchService = TestBed.get(ElasticsearchService);
    expect(service).toBeTruthy();
  });
});
