import {Component, Inject, OnInit} from '@angular/core';
import {MAT_DIALOG_DATA, MatDialog} from "@angular/material";
import {Calypso} from "src/dynacred2/calypso";
import {ByzCoinService} from "src/app/byz-coin.service";
import {InstanceMapKV} from "src/dynacred2/credentialStructBS";
import {showTransactions, TProgress} from "src/app/dialogs/transaction/transaction";
import {KeyPair} from "src/dynacred2";
import {CredentialTransaction} from "src/dynacred2/credentialTransaction";

/**
 *
 * @param dialog
 * @param cal
 */
export async function showCalypso(dialog: MatDialog, cal: Calypso, tx: CredentialTransaction) {
    const tc = dialog.open(CalypsoComponent, {
        data: {cal, tx},
        disableClose: true,
    });

    return new Promise((resolve, reject) => {
        tc.afterClosed().subscribe({
            error: reject,
            next: (v) => {
                if (v instanceof Error) {
                    reject(v);
                } else {
                    resolve(v);
                }
            },
        });
    });
}

interface ICalypsoComponent {
    cal: Calypso,
    tx: CredentialTransaction,
}

interface kvsOK extends InstanceMapKV {
    ok: boolean;
}

@Component({
    selector: 'app-calypso',
    templateUrl: './calypso.component.html',
    styleUrls: ['./calypso.component.css']
})
export class CalypsoComponent implements OnInit {

    public kvs: kvsOK[] = [];
    public display = 0;
    public name: string;
    public content: string;

    constructor(
        private bcs: ByzCoinService,
        private dialog: MatDialog,
        @Inject(MAT_DIALOG_DATA) public data: ICalypsoComponent
    ) {
    }

    async ngOnInit() {
        for (const kv of this.data.cal.cim.getValue().toKVs()) {
            this.kvs.push({
                key: kv.key, value: kv.value,
                ok: await this.data.cal.hasAccess(kv.key)
            })
        }
    }

    async download(c: InstanceMapKV) {
        await showTransactions(this.dialog, "Asking to Decrypt",
            async (progress: TProgress) => {
                const kp = KeyPair.rand();
                const tx = this.data.tx.clone();
                try {
                    this.content = (await this.data.cal.getFile(tx, c.key, kp, progress)).toString();
                    this.name = c.value;
                    this.display = 1;
                } catch(e){
                    this.content = e.toString();
                    this.display = 2;
                }
            });
    }
}
