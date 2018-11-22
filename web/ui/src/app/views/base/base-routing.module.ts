import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';
import { TablesComponent } from './tables.component';

const routes: Routes = [
  {
    path: '',
    component: TablesComponent,
    data: {
      title: 'Data'
    },
    children: [
      {
        path: 'alarm-list',
        component: TablesComponent,
        data: {
          title: 'Alarm List'
        }
      },
    ]
  }
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class BaseRoutingModule {}
