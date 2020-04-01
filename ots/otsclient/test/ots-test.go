package main

import (
	"bytes"
	"flag"
	"os"

	otsclient "github.com/calypso-demo/filesharing/ots/otsclient"
	util "github.com/calypso-demo/filesharing/ots/otsclient/util"
	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/crypto.v0/ed25519"
	"gopkg.in/dedis/crypto.v0/random"
	"gopkg.in/dedis/crypto.v0/share/pvss"
	"gopkg.in/dedis/onet.v1/log"
)

func main() {

	numTrusteePtr := flag.Int("t", 0, "size of the SC cothority")
	filePtr := flag.String("g", "", "group.toml file for trustees")
	pkFilePtr := flag.String("p", "", "pk.txt file")
	dbgPtr := flag.Int("d", 0, "debug level")
	flag.Parse()
	log.SetDebugVisible(*dbgPtr)

	el, err := util.ReadRoster(*filePtr)
	if err != nil {
		log.Errorf("Couldn't read group.toml file: %v", err)
		os.Exit(1)
	}
	scurl, err := otsclient.CreateSkipchain(el)
	if err != nil {
		log.Errorf("Could not create skipchain: %v", err)
		os.Exit(1)
	}

	scPubKeys, err := otsclient.GetPubKeys(pkFilePtr)
	if err != nil {
		log.Errorf("Couldn't read pk file: %v", err)
		os.Exit(1)
	}

	dataPVSS := util.DataPVSS{
		Suite:        ed25519.NewAES128SHA256Ed25519(false),
		SCPublicKeys: scPubKeys,
		NumTrustee:   *numTrusteePtr,
	}
	// Writer's pk/sk pair
	wrPrivKey := dataPVSS.Suite.Scalar().Pick(random.Stream)
	wrPubKey := dataPVSS.Suite.Point().Mul(nil, wrPrivKey)
	// Reader's pk/sk pair
	privKey := dataPVSS.Suite.Scalar().Pick(random.Stream)
	pubKey := dataPVSS.Suite.Point().Mul(nil, privKey)

	err = otsclient.SetupPVSS(&dataPVSS, pubKey)
	if err != nil {
		log.Errorf("Could not setup PVSS: %v", err)
		os.Exit(1)
	}

	mesgSize := 1024 * 1024
	mesg := make([]byte, mesgSize)
	for i := 0; i < mesgSize; i++ {
		mesg[i] = 'w'
	}
	encMesg, hashEnc, err := otsclient.EncryptMessage(&dataPVSS, mesg)
	if err != nil {
		log.Errorf("Could not encrypt message: %v", err)
		os.Exit(1)
	}

	// Creating write transaction
	writeSB, err := otsclient.CreateWriteTxn(scurl, &dataPVSS, hashEnc, pubKey, wrPrivKey)
	if err != nil {
		log.Errorf("Could not create write transaction: %v", err)
		os.Exit(1)
	}

	// Bob gets it from Alice
	writeID := writeSB.Hash
	// Get write transaction from skipchain
	writeSB, writeTxnData, sig, err := otsclient.GetWriteTxnSB(scurl, writeID)
	if err != nil {
		log.Errorf("Could not retrieve write transaction block: %v", err)
		os.Exit(1)
	}

	sigVerErr := otsclient.VerifyTxnSignature(writeTxnData, sig, wrPubKey)
	if sigVerErr != nil {
		log.Errorf("Signature verification failed on the write transaction: %v", sigVerErr)
		os.Exit(1)
	}

	log.Info("Signature verified on the retrieved write transaction")
	validHash := otsclient.VerifyEncMesg(writeTxnData, encMesg)
	if validHash == 0 {
		log.Info("Valid hash for encrypted message")
	} else {
		log.Errorf("Invalid hash for encrypted message")
		os.Exit(1)
	}

	// Creating read transaction
	readSB, err := otsclient.CreateReadTxn(scurl, writeID, privKey)
	if err != nil {
		log.Errorf("Could not create read transaction: %v", err)
		os.Exit(1)
	}

	updWriteSB, err := otsclient.GetUpdatedWriteTxnSB(scurl, writeID)
	if err != nil {
		log.Errorf("Could not retrieve updated write txn SB: %v", err)
		os.Exit(1)
	}

	acPubKeys := readSB.Roster.Publics()
	// Bob obtains the SC public keys from T_W
	scPubKeys = writeTxnData.SCPublicKeys
	decShares, err := otsclient.GetDecryptedShares(scurl, el, updWriteSB, readSB.SkipBlockFix, acPubKeys, scPubKeys, privKey, readSB.Index)
	if err != nil {
		log.Errorf("Could not get the decrypted shares: %v", err)
		os.Exit(1)
	}

	var validKeys []abstract.Point
	var validEncShares []*pvss.PubVerShare
	var validDecShares []*pvss.PubVerShare
	sz := len(decShares)
	for i := 0; i < sz; i++ {
		validKeys = append(validKeys, writeTxnData.SCPublicKeys[i])
		validEncShares = append(validEncShares, writeTxnData.EncShares[i])
		validDecShares = append(validDecShares, decShares[i])
	}

	// Normally Bob doesn't have dataPVSS but we are
	// using it only for PVSS parameters for simplicity
	recSecret, err := pvss.RecoverSecret(dataPVSS.Suite, writeTxnData.G, validKeys, validEncShares, validDecShares, dataPVSS.Threshold, dataPVSS.NumTrustee)
	if err != nil {
		log.Errorf("Could not recover secret: %v", err)
		os.Exit(1)
	}

	log.Info("Recovered secret")
	recMesg, err := otsclient.DecryptMessage(recSecret, encMesg, writeTxnData)
	if err != nil {
		log.Errorf("Could not decrypt message: %v", err)
		os.Exit(1)
	}
	log.Info("Recovered message?:", (bytes.Compare(recMesg, mesg) == 0))
}
