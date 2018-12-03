// Angular
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { NgModule } from '@angular/core';

// Tabel Component
import { TablesComponent } from './tables.component';
import { DetailalarmComponent } from './detailalarm.component';

// Components Routing
import { BaseRoutingModule } from './base-routing.module';

import { MomentModule } from 'ngx-moment';
import { TooltipModule} from 'ngx-bootstrap';
import { BsDropdownModule} from 'ngx-bootstrap/dropdown';
import { ModalModule } from 'ngx-bootstrap/modal';
import { AlertModule } from 'ngx-bootstrap/alert';

@NgModule({
  imports: [
    CommonModule,
    FormsModule,
    BaseRoutingModule,
    MomentModule,
    TooltipModule.forRoot(),
    BsDropdownModule.forRoot(),
    ModalModule.forRoot(),
    AlertModule.forRoot()
  ],
  declarations: [
    TablesComponent,
    DetailalarmComponent
  ]
})
export class BaseModule { }
