package main

import (
	"bytes"
	"errors"

	"github.com/BurntSushi/toml"
	ocs "github.com/calypso-demo/filesharing/ots/onchain-secrets"

	"github.com/calypso-demo/filesharing/ots/otsclient"
	"github.com/calypso-demo/filesharing/ots/otsclient/util"
	"github.com/calypso-demo/filesharing/ots/otssmc/protocol"
	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/crypto.v0/ed25519"
	"gopkg.in/dedis/crypto.v0/random"
	"gopkg.in/dedis/crypto.v0/share/pvss"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
	"gopkg.in/dedis/onet.v1/simul/monitor"
)

func init() {
	onet.SimulationRegister("OTS", NewOTSSimulation)
}

type OTSSimulation struct {
	onet.SimulationBFTree
}

func NewOTSSimulation(config string) (onet.Simulation, error) {
	otss := &OTSSimulation{}
	_, err := toml.Decode(config, otss)
	if err != nil {
		return nil, err
	}
	return otss, nil
}

func (otss *OTSSimulation) Setup(dir string, hosts []string) (*onet.SimulationConfig, error) {

	sc := &onet.SimulationConfig{}
	//TODO: 3rd parameter to CreateRoster is port #
	otss.CreateRoster(sc, hosts, 2000)
	err := otss.CreateTree(sc)
	if err != nil {
		return nil, err
	}
	return sc, nil
}

func (otss *OTSSimulation) Node(config *onet.SimulationConfig) error {
	return otss.SimulationBFTree.Node(config)
}

func (otss *OTSSimulation) Run(config *onet.SimulationConfig) error {
	log.Info("Total # of rounds:", otss.Rounds)
	// HARD-CODING AC COTHORITY SIZE!

	// acSize := 10
	// acRoster := onet.NewRoster(config.Roster.List[:acSize])
	scPubKeys := config.Roster.Publics()
	log.Info("SC PubKeys Size", len(scPubKeys))
	// log.Info("AC Size:", len(acRoster.List))

	numTrustee := config.Tree.Size()
	log.Info("# of trustees:", numTrustee)
	mesgSize := 1024 * 1024
	mesg := make([]byte, mesgSize)
	for i := 0; i < mesgSize; i++ {
		mesg[i] = 'w'
	}

	scurl, err := otsclient.CreateSkipchain(config.Roster)
	if err != nil {
		return err
	}
	// Transactions with trustee size = 10
	// Total block # = 2 x dummyTxnCount
	// dummyTxnCount := 32
	// dummyerr := prepareDummyDP(scurl, acRoster, dummyTxnCount)
	// if dummyerr != nil {
	// 	log.Errorf("Dummy errors is: %v", dummyerr)
	// 	return err
	// }

	for round := 0; round < otss.Rounds; round++ {
		log.Info("Round:", round)

		dataPVSS := util.DataPVSS{
			Suite:        ed25519.NewAES128SHA256Ed25519(false),
			SCPublicKeys: scPubKeys,
			NumTrustee:   numTrustee,
		}

		wrPrivKey := dataPVSS.Suite.Scalar().Pick(random.Stream)
		wrPubKey := dataPVSS.Suite.Point().Mul(nil, wrPrivKey)
		// Reader's pk/sk pair
		privKey := dataPVSS.Suite.Scalar().Pick(random.Stream)
		pubKey := dataPVSS.Suite.Point().Mul(nil, privKey)

		write_txn_prep := monitor.NewTimeMeasure("WriteTxnPrep")
		err = otsclient.SetupPVSS(&dataPVSS, pubKey)
		if err != nil {
			return err
		}

		encMesg, hashEnc, err := otsclient.EncryptMessage(&dataPVSS, mesg)
		write_txn_prep.Record()
		if err != nil {
			return err
		}

		create_wrt_txn := monitor.NewTimeMeasure("CreateWriteTxn")
		writeSB, err := otsclient.CreateWriteTxn(scurl, &dataPVSS, hashEnc, pubKey, wrPrivKey)
		create_wrt_txn.Record()
		if err != nil {
			return err
		}

		// Bob gets it from Alice
		writeID := writeSB.Hash
		// Get write transaction from skipchain
		get_write_txn_sb := monitor.NewTimeMeasure("GetWriteTxnSB")
		writeSB, writeTxnData, txnSig, err := otsclient.GetWriteTxnSB(scurl, writeID)
		get_write_txn_sb.Record()
		if err != nil {
			return err
		}

		log.Info("Write index is:", writeSB.Index)

		// ver_txn_sig := monitor.NewTimeMeasure("VerifyTxnSig")
		sigVerErr := otsclient.VerifyTxnSignature(writeTxnData, txnSig, wrPubKey)
		// ver_txn_sig.Record()
		if sigVerErr != nil {
			return sigVerErr
		}

		// log.Info("Signature verified on the retrieved write transaction")
		// ver_enc_mesg := monitor.NewTimeMeasure("VerifyEncMesg")
		validHash := otsclient.VerifyEncMesg(writeTxnData, encMesg)
		// ver_enc_mesg.Record()
		if validHash != 0 {
			return errors.New("Cannot verify encrypted message")
		}

		create_read_txn := monitor.NewTimeMeasure("CreateReadTxn")
		readSB, err := otsclient.CreateReadTxn(scurl, writeID, privKey)
		create_read_txn.Record()
		if err != nil {
			return err
		}

		get_upd_wsb := monitor.NewTimeMeasure("GetUpdatedWriteSB")
		updWriteSB, err := otsclient.GetUpdatedWriteTxnSB(scurl, writeID)
		get_upd_wsb.Record()
		if err != nil {
			return err
		}

		acPubKeys := readSB.Roster.Publics()
		log.Info("# of AC public keys:", len(acPubKeys))
		readTxnSBF := readSB.SkipBlockFix
		p, err := config.Overlay.CreateProtocol("otssmc", config.Tree, onet.NilServiceID)
		if err != nil {
			return err
		}

		// GetDecryptedShares call preparation
		// log.Info("Write index is:", updWriteSB.Index)
		idx := readSB.Index - updWriteSB.Index - 1
		if idx < 0 {
			return errors.New("Forward-link index is negative")
		}

		inclusionProof := updWriteSB.GetForward(idx)
		if inclusionProof == nil {
			return errors.New("Forward-link does not exist")
		}

		data := &util.OTSDecryptReqData{
			WriteTxnSBF:    updWriteSB.SkipBlockFix,
			ReadTxnSBF:     readTxnSBF,
			InclusionProof: inclusionProof,
			ACPublicKeys:   acPubKeys,
		}
		proto := p.(*protocol.OTSDecrypt)
		proto.DecReqData = data
		proto.RootIndex = 0
		// prep_decreq := monitor.NewTimeMeasure("PrepDecReq")
		msg, err := network.Marshal(data)
		if err != nil {
			return err
		}

		sig, err := util.SignMessage(msg, privKey)
		if err != nil {
			return err
		}
		// prep_decreq.Record()

		proto.Signature = &sig
		dec_req := monitor.NewTimeMeasure("DecReq")
		go p.Start()
		reencShares := <-proto.DecShares
		dec_req.Record()

		// dec_reenc_shares := monitor.NewTimeMeasure("DecryptReencShares")
		tmpDecShares, err := otsclient.ElGamalDecrypt(reencShares, privKey)
		// dec_reenc_shares.Record()
		if err != nil {
			return err
		}

		size := len(tmpDecShares)
		decShares := make([]*pvss.PubVerShare, size)
		for i := 0; i < size; i++ {
			decShares[tmpDecShares[i].S.I] = tmpDecShares[i]
		}

		recover_sec := monitor.NewTimeMeasure("RecoverSecret")
		var validKeys []abstract.Point
		var validEncShares []*pvss.PubVerShare
		var validDecShares []*pvss.PubVerShare
		for i := 0; i < size; i++ {
			validKeys = append(validKeys, writeTxnData.SCPublicKeys[i])
			validEncShares = append(validEncShares, writeTxnData.EncShares[i])
			validDecShares = append(validDecShares, decShares[i])
		}

		// ver_recons_pvss := monitor.NewTimeMeasure("VerifyandReconstructPVSS")
		recSecret, err := pvss.RecoverSecret(dataPVSS.Suite, writeTxnData.G, validKeys, validEncShares, validDecShares, dataPVSS.Threshold, dataPVSS.NumTrustee)
		// ver_recons_pvss.Record()
		if err != nil {
			return err
		}

		// dec_mesg := monitor.NewTimeMeasure("DecryptMessage")
		recvMesg, err := otsclient.DecryptMessage(recSecret, encMesg, writeTxnData)
		recover_sec.Record()
		if err != nil {
			return err
		}
		log.Info("Recovered message?:", bytes.Compare(mesg, recvMesg) == 0)
	}

	return nil
}

func prepareDummyDP(scurl *ocs.SkipChainURL, scRoster *onet.Roster, pairCount int) error {

	scPubKeys := scRoster.Publics()
	numTrustee := len(scPubKeys)
	dp := util.DataPVSS{
		Suite:        ed25519.NewAES128SHA256Ed25519(false),
		SCPublicKeys: scPubKeys,
		NumTrustee:   numTrustee,
	}

	// err := otsclient.SetupPVSS(&dp, scPubKeys[0])
	// if err != nil {
	// 	return err
	// }
	return otsclient.AddDummyTxnPairs(scurl, &dp, pairCount)
}
