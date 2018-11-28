import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';
import { TablesComponent } from './tables.component';
import { DetailalarmComponent } from './detailalarm.component';

const routes: Routes = [
  {
    path: '',
    component: TablesComponent,
    data: {
      title: 'Data'
    }
  },
  {
    path: 'alarm-list',
    component: TablesComponent,
    data: {
      title: 'Alarm List'
    }
  },
  {
    path: 'alarm-detail/:alarmID',
    component: DetailalarmComponent,
    data: {
      title: 'Alarm Detail'
    }
  }
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class BaseRoutingModule {}
