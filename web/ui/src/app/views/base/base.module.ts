// Angular
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { NgModule } from '@angular/core';

// Tabel Component
import { TablesComponent } from './tables.component';

// Components Routing
import { BaseRoutingModule } from './base-routing.module';

import { MomentModule } from 'ngx-moment';
import { DetailalarmComponent } from './detailalarm.component';

@NgModule({
  imports: [
    CommonModule,
    FormsModule,
    BaseRoutingModule,
    MomentModule
  ],
  declarations: [
    TablesComponent,
    DetailalarmComponent
  ]
})
export class BaseModule { }
