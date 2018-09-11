import { NgModule } from '@angular/core';
import { MomentModule } from 'angular2-moment';
import { BrowserModule } from '@angular/platform-browser';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { AppRoutingModule } from './app-routing.module';

import { AppComponent } from './app.component';
import { ElasticsearchService } from './elasticsearch.service';
import { ShowAlarmsComponent } from './alarms/show-alarms/show-alarms.component';
import { AlarmDetailsComponent } from './alarms/alarm-details/alarm-details.component';
import { ShowEventsComponent } from './events/show-events/show-events.component';

import {BrowserAnimationsModule} from "@angular/platform-browser/animations"

@NgModule({
  declarations: [
    AppComponent,
    ShowAlarmsComponent,
    AlarmDetailsComponent,
    ShowEventsComponent
  ],
  imports: [
    BrowserModule,
    FormsModule,
    ReactiveFormsModule,
    AppRoutingModule,
    MomentModule,
    BrowserAnimationsModule
  ],
  providers: [ElasticsearchService],
  bootstrap: [AppComponent]
})

export class AppModule { }
