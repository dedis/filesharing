import {Component, Input, OnInit} from '@angular/core';
import {CredentialStructBS, User} from "src/dynacred2";
import {DarcBS} from "src/dynacred2/byzcoin";
import {InstanceMapKV} from "src/dynacred2/credentialStructBS";
import {Calypso} from "src/dynacred2/calypso";
import {showCalypso} from "src/app/dialogs/calypso/calypso.component";
import {MatDialog} from "@angular/material";
import {showTransactions, TProgress} from "src/app/dialogs/transaction/transaction";
import {EditDarcComponent} from "src/app/dialogs/edit-darc/edit-darc";
import Log from "src/lib/log";
import {map} from "rxjs/operators";
import {INewFile, NewFileComponent} from "src/app/dialogs/new-file/new-file.component";

@Component({
    selector: 'app-user',
    templateUrl: './user.component.html',
    styleUrls: ['./user.component.css']
})
export class UserComponent implements OnInit {
    @Input() user: User;

    public calypsoKVs: InstanceMapKV[];

    constructor(
        private dialog: MatDialog
    ) {
    }

    ngOnInit() {
        this.user.credStructBS.credCalypso.pipe(
            map(im => im.toKVs())
        ).subscribe({next: kvs => this.calypsoKVs = kvs})
    }

    async lookup(u: CredentialStructBS) {
        const c = new Calypso(this.user.calypso.lts,
            this.user.credSignerBS.getValue().getBaseID(), u.credCalypso);
        return showCalypso(this.dialog, c, this.user.startTransaction());
    }

    async fileNew() {
        this.dialog.open(NewFileComponent,
            {
                data: {
                    darcs: this.user.addressBook.groups
                },
                height: "400px",
                width: "400px",
            })
            .afterClosed().subscribe(async (result: INewFile) => {
            if (result && result.content && result.name && result.chosen &&
                result.content != "" && result.name != "") {
                await showTransactions(this.dialog, "Adding new file",
                    async (progress: TProgress) => {
                        progress(50, "Creating CalypsoWrite");
                        await this.user.executeTransactions(tx => {
                            const darcID = new Buffer(result.chosen, "hex");
                            this.user.calypso.addFile(tx, darcID, result.name,
                                Buffer.from(result.content));
                        }, 10);
                    });
            }
        });
    }

    async fileDelete(c: InstanceMapKV) {
        await showTransactions(this.dialog, "Deleting file",
            async (progress: TProgress) => {
                progress(50, "Deleting file");
                await this.user.executeTransactions(tx =>
                    this.user.calypso.rmFile(tx, c.value), 10);
            })

    }

    async editShow(u: DarcBS) {
        this.dialog.open(EditDarcComponent,
            {
                data: {
                    darc: u.getValue(),
                    filter: 'contact',
                    rule: 'spawn:calypsoRead',
                    title: u.getValue().description.toString(),
                    user: this.user
                },
                height: "400px",
                width: "400px",
            })
            .afterClosed().subscribe(async (result) => {
            if (result) {
                await showTransactions(this.dialog, "Updating Darc",
                    async (progress: TProgress) => {
                        progress(50, "Storing new DARC");
                        await this.user.executeTransactions(tx => {
                            u.evolve(tx, result);
                        });
                    });
            }
        });
    }
}
