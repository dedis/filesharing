import {Component, OnInit} from '@angular/core';
import {Genesis, KeyPair, User} from "src/dynacred2";
import {CONFIG_INSTANCE_ID, InstanceID} from "src/lib/byzcoin";
import {ByzCoinService} from "src/app/byz-coin.service";
import {CredentialsInstance} from "src/lib/byzcoin/contracts";
import Log from "src/lib/log";
import {Darc, Rule} from "src/lib/darc";
import {CalypsoReadInstance, LongTermSecret, OnChainSecretRPC} from "src/lib/calypso";
import {CredentialTransaction} from "src/dynacred2/credentialTransaction";

@Component({
    selector: 'app-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.css']
})
export class AppComponent implements OnInit {
    loading = true;
    percentage: number;
    text = "stand by";
    log = "Loaded in browser";
    users: User[] = [undefined, undefined, undefined];
    bcID: InstanceID;

    constructor(private bcs: ByzCoinService) {
    }

    logAppend(msg: string, perc: number) {
        this.log = `${msg}\n` + this.log;
        this.text = msg;
        this.percentage = perc;
        Log.lvl1(perc, msg)
    }

    async ngOnInit() {
        Log.lvl = 2;
        await this.bcs.loadConfig((msg: string, perc: number) => {
            this.logAppend(msg, perc * 0.6);
        });
        // TODO: remove this - it makes sure things are properly initialized - this is a bug in cothority :(
        await this.bcs.bc.getProofFromLatest(CONFIG_INSTANCE_ID);
        this.logAppend("got latest", 62);
        this.logAppend(`bcID is: ${this.bcs.bc.genesisID.toString("hex")}`, 65);

        const privBuf = Buffer.alloc(32);
        for (let i = 0; i <= 2; i++) {
            this.logAppend(`Getting user ${i + 1}`, 70 + i * 10);
            privBuf.writeUInt8(i + 1, 0);
            const kp = KeyPair.fromPrivate(privBuf);
            const credID = CredentialsInstance.credentialIID(kp.pub.marshalBinary());
            try {
                this.users[i] = await this.bcs.retrieveUser(credID, privBuf, `db${i}`);
            } catch (e) {
                Log.catch(e);
                if (i > 0) {
                    Log.error("couldn't get user");
                    return;
                } else {
                    return this.initChain();
                }
            }
        }
        this.logAppend("Done", 100);
        this.loading = false;
    }

    async initChain() {
        const darcID = Buffer.from("92a997e06cfce83a6b56ab30736a22a7db2a487d42fbf74eaa0eea5f78412c90", "hex");
        const privBuf = Buffer.from("ba9709e9ce407e65d43abb7282600329d77a84080892f1a460c0b607344a210c", "hex");
        const keyPair = KeyPair.fromPrivate(privBuf);

        this.logAppend("Updating genesis darc", 30);
        const genesis = new Genesis(this.bcs.db, this.bcs.bc, {keyPair, darcID});
        await genesis.evolveGenesisDarc();
        this.logAppend("Creating Coin", 35);
        await genesis.createCoin();
        this.logAppend("Creating Spawner", 40);
        await genesis.createSpawner();
        const userPrivBuf = Buffer.alloc(32);
        this.logAppend("Creating Long Term Secret", 45);
        const ocs = new OnChainSecretRPC(this.bcs.bc);
        await ocs.authorizeRoster();
        const lts = await LongTermSecret.spawn(this.bcs.bc, darcID, genesis.signers, this.bcs.config.roster);

        for (let i = 0; i <= 2; i++) {
            userPrivBuf.writeUInt8(i + 1, 0);
            const kp = KeyPair.fromPrivate(userPrivBuf);
            this.logAppend(`Creating User ${i + 1}`, 50 + i * 10);
            this.users[i] = await genesis.createUser(`User ${i + 1}`, kp.priv, `user${i}`, lts);
        }

        const pwds = ["WohGii4a", "ReeChee4", "Beijahz3"];
        const budgets = ["Give it to the poor", "Take from the rich", "UBI for everyone"];
        for (let i = 0; i < 3; i++) {
            this.logAppend(`Adding contacts and groups for user ${i+1}`, 80 + i * 5);
            await this.users[i].executeTransactions(tx => {
                const j = (i + 1) % 3;
                const k = (i + 2) % 3;
                const ab = this.users[i].addressBook;
                ab.contacts.link(tx, this.users[j].credStructBS.id);
                ab.contacts.link(tx, this.users[k].credStructBS.id);
                const darcJ = this.addCalypsoDarc(tx, i, j);
                const darcK = this.addCalypsoDarc(tx, i, k);
                this.users[i].calypso.addFile(tx, darcJ.getBaseID(), "password.txt", Buffer.from(pwds[j]));
                this.users[i].calypso.addFile(tx, darcK.getBaseID(), "budget.txt", Buffer.from(budgets[k]));
            }, 10);
        }

        this.logAppend("Done", 100);

        this.loading = false;
    }

    addCalypsoDarc(tx: CredentialTransaction, owner: number, reader: number): Darc {
        const s1 = this.users[owner].identityDarcSigner;
        const s2 = this.users[reader].identityDarcSigner;
        const d = Darc.createBasic([s1], [s1, s2], Buffer.from(`group_${owner+1}_${reader+1}`),
            ["spawn:" + CalypsoReadInstance.contractID]);
        d.rules.appendToRule("delete:calypsoWrite", s1, Rule.OR);
        tx.spawnDarc(d);
        this.users[owner].addressBook.groups.link(tx, d.getBaseID());
        return d;
    }
}
