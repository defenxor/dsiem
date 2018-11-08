import { ShowAlarmsComponent } from './alarms/show-alarms/show-alarms.component';
import { ShowEventsComponent } from './events/show-events/show-events.component';
import { EventDetailsComponent } from './events/event-details/event-details.component';

import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

const routes: Routes = [
    { path: '', redirectTo: 'alarms', pathMatch: 'full' },
    { path: 'alarms', component: ShowAlarmsComponent },
    { path: 'events', component: ShowEventsComponent },
    { path: 'eventdetails/:alarmID/:stage', component: EventDetailsComponent }
];

@NgModule({
    imports: [RouterModule.forRoot(routes)],
    exports: [RouterModule]
})

export class AppRoutingModule { }
