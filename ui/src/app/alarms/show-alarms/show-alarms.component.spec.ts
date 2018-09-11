import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { ShowAlarmsComponent } from './show-alarms.component';

describe('ShowAlarmsComponent', () => {
  let component: ShowAlarmsComponent;
  let fixture: ComponentFixture<ShowAlarmsComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ ShowAlarmsComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ShowAlarmsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
