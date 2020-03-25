import {BehaviorSubject} from "rxjs";
import * as Long from "long";

import {CoinInstance} from "src/lib/byzcoin/contracts";
import {Argument, InstanceID} from "src/lib/byzcoin";

import {Transaction} from "./transaction";

export class CoinBS extends BehaviorSubject<CoinInstance> {

    constructor(coin: BehaviorSubject<CoinInstance>) {
        super(coin.getValue());
        coin.subscribe(this);
    }

    public transferCoins(tx: Transaction, dest: InstanceID, amount: Long) {
        tx.invoke(this.getValue().id, CoinInstance.contractID, CoinInstance.commandTransfer,
            [new Argument({name: CoinInstance.argumentDestination, value: dest}),
                new Argument({name: CoinInstance.argumentCoins, value: Buffer.from(amount.toBytesLE())})])
    }
}
