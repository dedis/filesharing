import {Component, Input, OnInit} from '@angular/core';
import {InstanceID, Instruction} from "src/lib/byzcoin";
import {SkipBlock} from "src/lib/skipchain";
import {ByzCoinService} from "src/app/byz-coin.service";
import SkipchainRPC from "src/lib/skipchain/skipchain-rpc";
import {DataBody, DataHeader} from "src/lib/byzcoin/proto";

@Component({
    selector: 'app-explorer',
    templateUrl: './explorer.component.html',
    styleUrls: ['./explorer.component.css']
})
export class ExplorerComponent implements OnInit {
    public blocks: Block[] = [];
    public last: SkipBlock;
    @Input() bcID: InstanceID;
    private sc: SkipchainRPC;

    constructor(
        private bcs: ByzCoinService
    ) {
        this.sc = new SkipchainRPC(bcs.bc.latest.roster);
    }

    async ngOnInit() {
        (await this.bcs.bc.getNewBlocks()).subscribe((sb) => this.updateBlocks(sb));
    }

    async getBlock(i: number): Promise<Block> {
        const sbRep = await this.sc.getSkipBlockByIndex(this.bcs.bc.genesisID, i);
        return new Block(sbRep.skipblock);
    }

    async prependBlock(i: number) {
        this.blocks.unshift(await this.getBlock(i));
        this.blocks.splice(4);
    }

    async appendBlock(i: number) {
        this.blocks.push(await this.getBlock(i));
        this.blocks.splice(0, this.blocks.length - 4);
    }

    async updateBlocks(sb: SkipBlock) {
        this.last = sb;
        if (this.blocks.length > 0 &&
            this.blocks[this.blocks.length - 1].index != sb.index - 1) {
            return;
        }
        this.blocks.push(new Block(sb));
        this.blocks.splice(0, this.blocks.length - 4);
        while (this.blocks.length < 4) {
            await this.prependBlock(this.blocks[0].index - 1);
        }
    }

}

class Block {
    index: number;
    timestamp: string;
    insts: InstructionDisplay[] = [];

    constructor(sb: SkipBlock) {
        this.index = sb.index;
        const bch = DataHeader.decode(sb.data);
        const d = new Date(bch.timestamp.div(1e6).toNumber());
        this.timestamp = [d.getHours(), d.getMinutes(), d.getSeconds()]
            .map(d => d.toString().padStart(2, "0"))
            .join(":");
        const bcb = DataBody.decode(sb.payload);
        bcb.txResults.forEach(tx => {
            tx.clientTransaction.instructions.forEach(inst => {
                this.insts.push(new InstructionDisplay(inst, tx.accepted));
            })
        })
    }
}

class InstructionDisplay {
    public str: string;
    public ok: string;

    constructor(inst: Instruction,
                ok: boolean) {
        this.ok = ok ? "txOK" : "txRejected";
        switch (inst.type) {
            case Instruction.typeSpawn:
                this.str = "Spawn: " + inst.spawn.contractID;
                break;
            case Instruction.typeInvoke:
                this.str = "Invoke: " + inst.invoke.contractID + "/" +
                    inst.invoke.command;
                break;
            case Instruction.typeDelete:
                this.str = "Delete: " + inst.delete.contractID;
                break;
        }
    }
}
