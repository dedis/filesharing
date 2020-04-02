# One-Time Secrets

One-time secrets (OTS) is an implementation of the _on-chain secrets_ protocol. It is
more suitable for being deployed in permissionless environments, as oppposed to
long-term secrets (LTS), which is a better fit for the permissioned setting.

OTS uses both the access-control and secret-management cothority:

- The access-control cothority (ACC) is implemented using skipchains and distributed
  access right controls (Darcs).
- The secret-management cothority (SMC) uses an onet service, called `OTSSMC`, to
  handle the decryption requests. Unlike LTS, SMC nodes do not run a distributed
  key-generation protocol. Instead, they each have a unique Ed25519 key pair.

## OTS Workflow

1. Writer runs PVSS on the client-side using the `SetupPVSS` function in the
   [client-side library](./otsclient/ots.go). Writer establishes the SMC at this stage
   by using the public keys of the nodes in PVSS. She uses the secret generated
   by PVSS as the symmetric key to encrypt the data that she wants to share.
   Additionally, she creates a simple access control policy that specifies the
   authorized readers.
2. Writer sends a write transaction to ACC by calling
   [`WriteTxnRequest`](./onchain-secrets/api.go), which serves as an endpoint for [the ACC
   service](./onchain-secrets/service.go).
3. Reader first fetches the proof for the write transaction from the skipchain.
   He then creates a read transaction and sends it to ACC by calling
   [`ReadTxnRequest`](./onchain-secrets/api.go), which serves as an endpoint for [the ACC
   service](./onchain-secrets/service.go).
4. Reader prepares the decryption request using the proofs for read and write
   transactions. He sends the request to SMC by calling
   [`OTSDecrypt`](./otssmc/service/api.go), which serves as an endpoint for the
   [`OTSSMC` service](./otssmc/service/service.go).
5. Each trustee in SMC receives the decryption request and does the following:
   (1) verify the read and write transaction proofs, (2) verify that the
   decryption request is coming from an authorized reader as specified in the
   write transaction, (3) verify that the writer created its encrypted PVSS
   share correctly (done by verifying a non-interactive zero-knowledge proof),
   (4) decrypt its share and encrypt it under reader's public key. All of these
   steps are performed by executing the [`otssmc`
   protocol](./otssmc/protocol/protocol.go) at each trustee.
6. Reader gets back the decrypted shares and runs the Lagrange interpolation. If
   there are at least _t_ correctly decrypted shares (out of _n_), he recovers
   the secret (_i.e.,_ the symmetric key) and can decrypt the data.

## Directory Information

* `otsclient/`: This directory contains the client-side operations:
  * `ots.go`: This file mainly contains two types of functions: (1) client-side helper functions and (2) API functions of OTS. The API functions serve as wrappers around the `onchain-secrets` API.
  * `test/ots-test.go`: This is what should have been a proper go-test file. It essentially runs the workflow above.
* `otssmc/`: This directory contains the service that is run by SMC.
  * `service/api.go`: Endpoint for the `OTSSMC` service.
  * `service/service.go`: Implementation of the `OTSSMC` service. It handles the decryption request.
  * `protocol/`: The protocol used by the `OTSSMC` service to perform step 5 of the workflow.
* `onchain-secrets/`: This contains the original `onchain-secrets` service.
  * `api.go`: This file contains the original API functions of the `onchain-secrets` service and the new ones that are added for OTS.
