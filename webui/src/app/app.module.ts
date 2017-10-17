import { BrowserModule } from '@angular/platform-browser';
import { NgModule } from '@angular/core';
import {BrowserAnimationsModule} from '@angular/platform-browser/animations';

import { AppComponent, DialogJobDetailDialog } from './app.component';

import {HttpClientModule} from "@angular/common/http"

import {
  MatButtonToggleModule, MatPaginatorModule, MatRadioModule, MatSnackBarModule,
  MatToolbarModule
} from "@angular/material";
import { MatButtonModule } from "@angular/material";
import {MatSelectModule} from '@angular/material';
import {MatCardModule} from '@angular/material';
import {MatTableModule} from '@angular/material';
import {MatSidenavModule} from '@angular/material';
import {MatListModule} from '@angular/material';
import {MatMenuModule} from '@angular/material';
import {MatGridListModule} from '@angular/material';
import {MatTooltipModule} from '@angular/material';
import {MatDialogModule} from '@angular/material';
import {MatInputModule} from '@angular/material';
import {MatFormFieldModule} from '@angular/material';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import {MatSlideToggleModule} from '@angular/material';
import {MatChipsModule} from '@angular/material';
import {MatIconModule} from '@angular/material';

import 'hammerjs';

import {DialogJobStatsDialog} from "./job.stats.component";
import {AppCreateComponent} from "./app.create.component";
import {DialogConfirmDialog} from "./confirm.component";
import {KalaService} from "./kala.service";

@NgModule({
  declarations: [
    AppComponent,
    DialogJobDetailDialog,
    DialogJobStatsDialog,
    AppCreateComponent,
    DialogConfirmDialog,
  ],
  imports: [
    BrowserModule,
    HttpClientModule,
    BrowserAnimationsModule,
    MatToolbarModule,
    MatButtonModule,
    MatSelectModule,
    MatCardModule,
    MatTableModule,
    MatSidenavModule,
    MatListModule,
    MatMenuModule,
    MatGridListModule,
    MatTooltipModule,
    FormsModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatSlideToggleModule,
    MatChipsModule,
    MatIconModule,
    MatButtonToggleModule,
    MatPaginatorModule,
    MatRadioModule,
    MatSnackBarModule,
  ],
  providers: [
    KalaService,
  ],
  bootstrap: [AppComponent],
  entryComponents: [
    DialogJobDetailDialog,
    DialogConfirmDialog,
    DialogJobStatsDialog,
    AppCreateComponent,
  ],
})
export class AppModule { }
