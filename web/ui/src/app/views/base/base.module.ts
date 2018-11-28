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

@NgModule({
  imports: [
    CommonModule,
    FormsModule,
    BaseRoutingModule,
    MomentModule,
    TooltipModule.forRoot(),
    BsDropdownModule.forRoot()
  ],
  declarations: [
    TablesComponent,
    DetailalarmComponent
  ]
})
export class BaseModule { }
