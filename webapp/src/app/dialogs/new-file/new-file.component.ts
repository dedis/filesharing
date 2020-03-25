import {Component, Inject, OnInit} from '@angular/core';
import {MAT_DIALOG_DATA} from "@angular/material";
import {DarcsBS} from "src/dynacred2/byzcoin";

export interface INewFile {
    name: string;
    content: string;
    darcs: DarcsBS;
    chosen: string;
}

@Component({
    selector: 'app-new-file',
    templateUrl: './new-file.component.html',
    styleUrls: ['./new-file.component.css']
})
export class NewFileComponent implements OnInit {

    constructor(
        @Inject(MAT_DIALOG_DATA) public data: INewFile
    ) {
        data.chosen = data.darcs.getValue()[0].getValue().getBaseID().toString("hex");
    }

    ngOnInit() {
    }

}
