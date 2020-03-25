import {BrowserModule} from '@angular/platform-browser';
import {NgModule} from '@angular/core';

import {AppRoutingModule} from './app-routing.module';
import {AppComponent} from './app.component';
import {UserComponent} from './card/user/user.component';
import {ExplorerComponent} from './card/explorer/explorer.component';
import {FlexLayoutModule} from "@angular/flex-layout";
import {
    MatChipsModule,
    MatDialogModule,
    MatFormFieldModule, MatInputModule,
    MatListModule,
    MatSelectModule,
    MatSliderModule
} from "@angular/material";
import {MatGridListModule} from '@angular/material/grid-list';
import {MatCardModule} from '@angular/material/card';
import {MatMenuModule} from '@angular/material/menu';
import {MatIconModule} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
import {LayoutModule} from '@angular/cdk/layout';
import {DialogTransactionComponent} from "src/app/dialogs/transaction/transaction";
import { CalypsoComponent } from './dialogs/calypso/calypso.component';
import {BrowserAnimationsModule} from "@angular/platform-browser/animations";
import {EditDarcComponent} from "src/app/dialogs/edit-darc/edit-darc";
import { NewFileComponent } from './dialogs/new-file/new-file.component';
import {FormsModule} from "@angular/forms";

@NgModule({
    declarations: [
        AppComponent,
        UserComponent,
        ExplorerComponent,
        DialogTransactionComponent,
        CalypsoComponent,
        EditDarcComponent,
        NewFileComponent
    ],
    imports: [
        BrowserModule,
        AppRoutingModule,
        FlexLayoutModule,
        MatSliderModule,
        MatGridListModule,
        MatCardModule,
        MatMenuModule,
        MatIconModule,
        MatButtonModule,
        LayoutModule,
        MatListModule,
        MatChipsModule,
        MatDialogModule,
        BrowserAnimationsModule,
        MatSelectModule,
        MatFormFieldModule,
        FormsModule,
        MatInputModule
    ],
    providers: [],
    bootstrap: [AppComponent],
    entryComponents: [
        DialogTransactionComponent,
        CalypsoComponent,
        EditDarcComponent,
        NewFileComponent
    ]
})
export class AppModule {
}
