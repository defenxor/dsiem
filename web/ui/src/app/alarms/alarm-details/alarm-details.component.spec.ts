import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { AlarmDetailsComponent } from './alarm-details.component';

describe('AlarmDetailsComponent', () => {
  let component: AlarmDetailsComponent;
  let fixture: ComponentFixture<AlarmDetailsComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ AlarmDetailsComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(AlarmDetailsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
