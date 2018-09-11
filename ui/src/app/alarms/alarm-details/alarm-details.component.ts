import { Component, OnInit, Input } from '@angular/core';

import {Alarm} from '../alarm.interface';

@Component({
  selector: 'alarm-details',
  templateUrl: './alarm-details.component.html',
  styleUrls: ['./alarm-details.component.css']
})
export class AlarmDetailsComponent implements OnInit {

  @Input() alarm: Alarm;

  constructor() { }

  ngOnInit() {
  }

}
