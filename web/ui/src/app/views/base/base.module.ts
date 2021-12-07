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
// Angular
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { NgModule } from '@angular/core';

// Tabel Component
import { TablesComponent } from './tables.component';
import { DetailalarmComponent } from './detailalarm.component';
import { SearchboxComponent } from './searchbox.component';
import { AlertboxComponent } from './alertbox.component';

// Components Routing
import { BaseRoutingModule } from './base-routing.module';

import { MomentModule } from 'ngx-moment';
import { TooltipModule} from 'ngx-bootstrap';
import { BsDropdownModule} from 'ngx-bootstrap/dropdown';
import { ModalModule } from 'ngx-bootstrap/modal';
import { AlertModule } from 'ngx-bootstrap/alert';
import { NgxSpinnerModule } from 'ngx-spinner';
import { CountdownModule } from 'ngx-countdown';
import { NgxInputSearchModule } from 'ngx-input-search';
@NgModule({
  imports: [
    CommonModule,
    FormsModule,
    BaseRoutingModule,
    MomentModule,
    TooltipModule.forRoot(),
    BsDropdownModule.forRoot(),
    ModalModule.forRoot(),
    AlertModule.forRoot(),
    NgxSpinnerModule,
    CountdownModule,
    NgxInputSearchModule
  ],
  declarations: [
    TablesComponent,
    DetailalarmComponent,
    SearchboxComponent,
    AlertboxComponent
  ],
})
export class BaseModule { }
