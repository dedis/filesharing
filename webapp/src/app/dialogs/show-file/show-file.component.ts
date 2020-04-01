import {Component, Inject, OnInit} from '@angular/core';
import {ByzCoinService} from "src/app/byz-coin.service";
import {MAT_DIALOG_DATA} from "@angular/material";
import {InstanceID, Instruction} from "src/lib/byzcoin";
import {DarcBS} from "src/dynacred2/byzcoin";
import {CalypsoReadInstance, CalypsoWriteInstance} from "src/lib/calypso";
import {SkipchainRPC} from "src/lib/skipchain";
import {DataBody, DataHeader} from "src/lib/byzcoin/proto";
import {User} from "src/dynacred2";
import {BehaviorSubject} from "rxjs";
import {Darc} from "src/lib/darc";
import {sprintf} from "sprintf-js";

export interface IShowFile {
    user: User;
    wrID: InstanceID;
}

@Component({
    selector: 'app-show-file',
    templateUrl: './show-file.component.html',
    styleUrls: ['./show-file.component.scss']
})
export class ShowFileComponent implements OnInit {
    public name: string;
    public darc: DarcBS | undefined;
    public history = "";
    public status = "";
    private wrInst: CalypsoWriteInstance;

    constructor(private bcs: ByzCoinService,
                @Inject(MAT_DIALOG_DATA) public sf: IShowFile) {
        this.name = sf.user.calypso.cim.getEntry(sf.wrID);
    }

    async ngOnInit() {
        this.wrInst = await CalypsoWriteInstance.fromByzcoin(this.bcs.bc, this.sf.wrID);
        this.darc = await this.bcs.retrieveDarcBS(this.wrInst.darcID);
        await this.getHistory();
    }

    async getHistory() {
        this.history = "";
        const skip = new SkipchainRPC(this.bcs.conn);
        for (let sb = this.bcs.bc.latest; sb.index > 0;) {
            this.status = `Searching block ${sb.index}\n`;
            const db = DataBody.decode(sb.payload);
            const dh = DataHeader.decode(sb.data);
            for (const tx of db.txResults) {
                for (const inst of tx.clientTransaction.instructions) {
                    if (inst.instanceID.equals(this.wrInst.id)) {
                        if (inst.type === Instruction.typeSpawn &&
                            inst.spawn.contractID === CalypsoReadInstance.contractID) {
                            const blockDate = new Date(dh.timestamp.div(1e6).toNumber());
                            const date = sprintf("%02d/%02d %02d:%02d",
                                blockDate.getMonth() + 1, blockDate.getDate(),
                                blockDate.getHours(), blockDate.getMinutes());
                            this.history += `\nBlock ${sb.index} at ${date}`;
                            if (!tx.accepted) {
                                this.history += "  Refused "
                            } else {
                                this.history += "  Accepted "
                            }
                            this.history += "read request from: ";
                            let alias = "unknown";
                            const idents = inst.signerIdentities.map(si => si.ed25519);
                            for (const cr of this.sf.user.addressBook.contacts.getValue()) {
                                const darcs = await this.bcs.retrieveDarcsBS(new BehaviorSubject<InstanceID[]>(
                                    cr.credDevices.getValue().toInstanceIDs()));
                                for (const darc of darcs.getValue()) {
                                    const iids = await darc.getValue().ruleMatch(Darc.ruleSign, idents,
                                        () => new Promise(undefined));
                                    if (iids.length > 0) {
                                        alias = cr.credPublic.alias.getValue();
                                    }
                                }
                            }
                            this.history += alias;
                        }
                    }
                }
            }
            sb = await skip.getSkipBlock(sb.backlinks[0]);
        }
        this.status = "";
    }
}
