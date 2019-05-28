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
