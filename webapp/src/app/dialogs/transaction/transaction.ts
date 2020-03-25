import {
    AfterViewInit,
    Component,
    ElementRef,
    Inject,
    Renderer2,
    ViewChild
} from "@angular/core";
import {MAT_DIALOG_DATA, MatDialog, MatDialogRef} from "@angular/material/dialog";
import {map, startWith} from "rxjs/operators";
import {Subscription} from "rxjs";
import Log from "src/lib/log";
import {ByzCoinService} from "src/app/byz-coin.service";

/**
 * Shows a nice pop-up with some animation of the blockchain and the transaction being processed.
 * The worker callback can store anything and call progress as many times as it wants. Every time
 * progress is called, the progress-bar is updated and the text is shown as a transaction (for positive
 * percentage values) or as a text in the progress-bar (for negative percentage values).
 *
 * When the percentage value reaches +-100, the window will be closed.
 *
 * If an error occurs, the promise will be rejected.
 *
 * The type of the returned promise is the type returned by the worker callback.
 *
 * @param dialog reference to MatDialog
 * @param title shown in h1 in the dialog
 * @param worker the callback that will execute the one or more transactions
 */
export async function showTransactions<T>(dialog: MatDialog, title: string, worker: TWorker<T>): Promise<T> {
    const tc = dialog.open(DialogTransactionComponent, {
        data: {
            title,
            worker,
        },
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

// Progress type to be used in showTransactions.
export type TProgress = (percentage: number, text: string) => void;

// Worker callback that implements multiple steps and calls progress before each step.
export type TWorker<T> = (progress: TProgress) => Promise<T>;

export interface IDialogTransactionConfig<T> {
    title: string;
    worker: TWorker<T>;
}

@Component({
    selector: "app-dialog-transaction",
    styleUrls: ["./transaction.scss"],
    templateUrl: "transaction.html",
})
export class DialogTransactionComponent<T> implements AfterViewInit {

    percentage: number = 0;
    text: string;
    error: Error | undefined;

    private blocks: Element[] = [];
    private transaction: Element;
    private ub: Subscription;
    @ViewChild("main", {static: false}) private main?: ElementRef;

    constructor(
        private bcs: ByzCoinService,
        readonly dialogRef: MatDialogRef<DialogTransactionComponent<T>>,
        private readonly renderer: Renderer2,
        @Inject(MAT_DIALOG_DATA) public data: IDialogTransactionConfig<T>) {
    }

    async ngAfterViewInit() {
        // TODO: replace the setTimeout with ChangeDetectorRef
        setTimeout(async () => {
            const last = this.bcs.bc.latest.index;
            this.ub = (await this.bcs.bc.getNewBlocks()).pipe(
                map((block) => block.index),
                startWith(last - 2, last - 1),
            ).subscribe((nb) => this.updateBlocks(nb));
        }, 100);
    }

    updateBlocks(index: number) {
        this.addBlock(index);
        if (this.blocks.length === 3) {
            this.startTransactions();
        }
    }

    async startTransactions() {
        const prog = (p: number, t: string) => this.progress(p, t);
        try {
            // await new Promise(resolve => setTimeout(resolve, 500));
            const result = await this.data.worker(prog);
            prog(-100, "Done");
            this.ub.unsubscribe();
            setTimeout(() => {
                this.dialogRef.close(result);
            }, 300);
        } catch (e) {
            this.ub.unsubscribe();
            Log.catch(e);
            this.error = e;
        }
    }

    addBlock(index: number): Element {
        const block = this.renderer.createElement("DIV");
        const txt = this.renderer.createText(index.toString());
        block.appendChild(txt);
        this.main.nativeElement.appendChild(block);
        block.classList.add("block");
        this.blocks.unshift(block);
        for (let i = 0; i < this.blocks.length; i++) {
            this.blocks[i].classList.remove("block" + i);
            this.blocks[i].classList.add("block" + (i + 1));
        }
        if (this.blocks.length > 4) {
            this.main.nativeElement.removeChild(this.blocks[4]);
            this.blocks.splice(4);
        }
        if (this.transaction) {
            this.transaction.classList.remove("tx-send");
            // void this.transaction.offsetWidth;
            this.transaction.classList.add("tx-block");
            this.transaction = null;
        }
        return block;
    }

    addTransaction(text: string) {
        this.transaction = this.renderer.createElement("DIV");
        const txt = this.renderer.createText(text);
        this.transaction.appendChild(txt);
        this.main.nativeElement.appendChild(this.transaction);
        this.transaction.classList.add("transaction");
        this.transaction.classList.add("tx-send");
    }

    progress(percentage: number, text: string) {
        this.percentage = percentage >= 0 ? percentage : percentage * -1;
        Log.lvl2("Progress:", percentage, text);

        if (percentage >= 0) {
            this.text = "";
            this.addTransaction(text);
        } else {
            this.text = text;
        }
    }
}
