import { Component, Inject } from "@angular/core";
import { MAT_DIALOG_DATA, MatDialogRef } from "@angular/material/dialog";

import { curve } from "@dedis/kyber";

import { Darc, IdentityEd25519, IIdentity, Rule } from "src/lib/darc";
import IdentityDarc from "src/lib/darc/identity-darc";
import IdentityWrapper from "src/lib/darc/identity-wrapper";
import {ByzCoinService} from "src/app/byz-coin.service";
import {User} from "src/dynacred2";

export interface IEditDarc {
    darc: Darc;
    filter: string;
    rule: string;
    title: string;
    user: User;
}

interface IItem {
    label: string;
    identity: IdentityWrapper;
}

interface INewRule {
    value: string;
}

@Component({
    selector: "app-edit-darc",
    styleUrls: ["edit-darc.css"],
    templateUrl: "edit-darc.html",
})
export class EditDarcComponent {
    newDarc: Darc;
    available: IItem[] = [];
    chosen: IItem[] = [];
    rule: string = Darc.ruleSign;

    constructor(
        private dialogRef: MatDialogRef<EditDarcComponent>,
        private builder: ByzCoinService,
        @Inject(MAT_DIALOG_DATA) public data: IEditDarc) {
        if (!data.title || data.title === "") {
            data.title = "Edit access rights";
        }
        if (data.rule && data.rule !== "") {
            this.rule = data.rule;
        }
        this.newDarc = data.darc.evolve();
        this.getItems(data.filter).then((items) => {
            this.available = items;
            this.ruleChange({value: this.rule});
        });
    }

    ruleChange(newRule: INewRule) {
        this.available = this.available.concat(this.chosen);
        this.available = this.available.filter((i) => i.label !== "Unknown");
        this.chosen = [];

        const expr = this.newDarc.rules.getRule(this.rule).getExpr().toString();
        if (expr.indexOf("&") >= 0) {
            throw new Error("cannot handle darcs with AND");
        }
        const identities = expr.split("|");
        for (const identity of identities) {
            const idStr = identity.trim();
            this.add(IdentityWrapper.fromIdentity(new IdStub(idStr)), false);
        }
    }

    createItem(label: string, iid: IIdentity): IItem {
        const idw = IdentityWrapper.fromIdentity(iid);
        return {
            identity: idw,
            label,
        };
    }

    async getItems(filter: string): Promise<IItem[]> {
        const items: IItem[] = [];
        if (filter.indexOf("contact") >= 0) {
            for (const contact of this.data.user.addressBook.contacts.getValue()) {
                items.push(this.createItem("Contact: " + contact.credPublic.alias.getValue(),
                    await this.builder.retrieveSignerIdentityDarc(contact.darcID)));
            }
        }
        if (filter.indexOf("action") >= 0) {
            for (const action of await this.data.user.addressBook.actions.getValue()) {
                items.push(this.createItem("Action: " + action.darc.getValue().description.toString(),
                    new IdentityDarc({id: action.darc.getValue().id})));
            }
        }
        if (filter.indexOf("group") >= 0) {
            for (const group of await this.data.user.addressBook.groups.getValue()) {
                items.push(this.createItem("Group: " + group.getValue().description.toString(),
                    new IdentityDarc({id: group.getValue().id})));
            }
        }
        items.unshift(this.createItem("Ourselves: " + this.data.user.credStructBS.credPublic.alias.getValue(),
            this.data.user.identityDarcSigner));
        return items;
    }

    add(id: IdentityWrapper, update: boolean = true) {
        const index = this.available.findIndex((i) => i.identity.toString() === id.toString());
        if (index >= 0) {
            this.chosen.push(this.available[index]);
            this.available.splice(index, 1);
        } else {
            this.chosen.push({label: "Unknown", identity: id});
        }
        if (update) {
            this.updateDarc();
        }
    }

    remove(id: IdentityWrapper) {
        const index = this.chosen.findIndex((i) => i.identity.toString() === id.toString());
        if (index >= 0) {
            this.available.push(this.chosen[index]);
            this.chosen.splice(index, 1);
        }
        this.updateDarc();
    }

    updateDarc() {
        if (this.chosen.length > 0) {
            this.newDarc.rules.setRule(this.rule, this.idWrapToId(this.chosen[0].identity));
            this.chosen.slice(1).forEach((item) => {
                this.newDarc.rules.appendToRule(this.rule, this.idWrapToId(item.identity), Rule.OR);
            });
        }
    }

    idWrapToId(idW: IdentityWrapper): IIdentity {
        const str = idW.toString();
        const curve25519 = curve.newCurve("edwards25519");

        if (str.startsWith("ed25519:")) {
            return new IdentityEd25519({point: Buffer.from(str.slice(8), "hex")});
        }
        if (str.startsWith("darc:")) {
            return new IdentityDarc({id: Buffer.from(str.slice(5), "hex")});
        }
    }
}

class IdStub {
    constructor(private id: string) {
    }

    verify(msg: Buffer, signature: Buffer): boolean {
        return false;
    }

    /**
     * Get the byte array representation of the public key of the identity
     * @returns the public key as buffer
     */
    toBytes(): Buffer {
        return null;
    }

    /**
     * Get the string representation of the identity
     * @return a string representation
     */
    toString(): string {
        return this.id;
    }

}
