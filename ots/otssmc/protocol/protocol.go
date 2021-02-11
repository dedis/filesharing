package protocol

import (
	"crypto/sha256"
	"errors"

	ocs "github.com/calypso-demo/filesharing/ots/onchain-secrets"
	"github.com/calypso-demo/filesharing/ots/otsclient/util"

	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/crypto.v0/cosi"
	"gopkg.in/dedis/crypto.v0/random"
	"gopkg.in/dedis/crypto.v0/share/pvss"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/crypto"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
)

var Name = "otssmc"

func init() {
	network.RegisterMessage(AnnounceDecrypt{})
	network.RegisterMessage(DecryptReply{})
	network.RegisterMessage(&util.OTSDecryptReqData{})
	network.RegisterMessage(&util.DecryptedShare{})
	network.RegisterMessage(&pvss.PubVerShare{})
	onet.GlobalProtocolRegister(Name, NewProtocol)
}

type OTSDecrypt struct {
	*onet.TreeNodeInstance
	ChannelAnnounce chan StructAnnounceDecrypt
	ChannelReply    chan []StructDecryptReply
	DecShares       chan []*util.DecryptedShare
	DecReqData      *util.OTSDecryptReqData
	Signature       *crypto.SchnorrSig
	RootIndex       int
}

func NewProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {

	otsDecrypt := &OTSDecrypt{
		TreeNodeInstance: n,
		DecShares:        make(chan []*util.DecryptedShare),
	}
	err := otsDecrypt.RegisterChannelLength(&otsDecrypt.ChannelAnnounce, 65536)

	if err != nil {
		return nil, errors.New("couldn't register announcement-channel: " + err.Error())
	}
	err = otsDecrypt.RegisterChannelLength(&otsDecrypt.ChannelReply, 65536)

	if err != nil {
		return nil, errors.New("couldn't register reply-channel: " + err.Error())
	}
	return otsDecrypt, nil
}

func (p *OTSDecrypt) Start() error {
	log.Lvl3("Starting OTSDecrypt")
	for _, c := range p.Children() {
		err := p.SendTo(c, &AnnounceDecrypt{
			DecReqData: p.DecReqData,
			Signature:  p.Signature,
			RootIndex:  p.RootIndex,
		})

		if err != nil {
			log.Error(p.Info(), "failed to send to", c.Name(), err)
			return err
		}
	}
	return nil
}

func (p *OTSDecrypt) Dispatch() error {
	if p.IsLeaf() {
		announcement := <-p.ChannelAnnounce
		writeTxnData, sigErr := verifyDecryptionRequest(announcement.DecReqData, announcement.Signature)
		if sigErr != nil {
			return sigErr
		}

		idx := p.Index()
		// This is not needed anymore because indexes
		// are swapped instead of shifting
		// log.Info(p.Name(), idx)
		// if idx <= announcement.RootIndex {
		// 	idx--
		// }
		if idx == announcement.RootIndex {
			idx = 0
		}

		h, err := util.CreatePointH(network.Suite, writeTxnData.ReaderPk)
		if err != nil {
			log.Error(p.Info(), "Failed to generate point h", p.Name(), err)
			return err
		}

		// ds := &util.DecryptedShare{
		// 	Index: p.Index(),
		// }
		ds := &util.DecryptedShare{
			K:  nil,
			Cs: nil,
		}
		tempSh, err := pvss.DecShare(network.Suite, h, p.Public(), writeTxnData.EncProofs[idx], p.Private(), writeTxnData.EncShares[idx])
		if err != nil {
			log.Error(p.Info(), "Failed to decrypt share", p.Name(), err)
			// ds.K = nil
			// ds.Cs = nil
		} else {
			K, Cs := elGamalEncrypt(tempSh, writeTxnData.ReaderPk)
			ds.K = K
			ds.Cs = Cs
		}

		err = p.SendTo(p.Parent(), &DecryptReply{ds})
		if err != nil {
			log.Error(p.Info(), "Failed to send reply to", p.Parent().Name(), err)
			return err
		}
		return nil
	}

	var decShares []*util.DecryptedShare
	idx := p.RootIndex
	reply := <-p.ChannelReply
	for _, c := range reply {
		decShares = append(decShares, c.DecryptReply.DecShare)
	}

	writeTxnData, sigErr := verifyDecryptionRequest(p.DecReqData, p.Signature)
	if sigErr != nil {
		return sigErr
	}

	h, err := util.CreatePointH(network.Suite, writeTxnData.ReaderPk)
	if err != nil {
		log.Error(p.Info(), "Failed to generate point h", p.Name(), err)
		return err
	}

	// ds := &util.DecryptedShare{
	// 	Index: p.Index(),
	// }
	ds := &util.DecryptedShare{
		K:  nil,
		Cs: nil,
	}
	tempSh, err := pvss.DecShare(network.Suite, h, p.Public(), writeTxnData.EncProofs[idx], p.Private(), writeTxnData.EncShares[idx])

	if err != nil {
		log.Error(p.Info(), "Failed to decrypt share", p.Name(), err)
		ds.K = nil
		ds.Cs = nil
	} else {
		K, Cs := elGamalEncrypt(tempSh, writeTxnData.ReaderPk)
		ds.K = K
		ds.Cs = Cs
	}

	decShares = append(decShares, ds)
	log.Lvl3(p.ServerIdentity().Address, "is done with total of", len(decShares))
	p.DecShares <- decShares
	return nil
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func elGamalEncrypt(ds *pvss.PubVerShare, rPubKey abstract.Point) (abstract.Point, []abstract.Point) {

	msg, err := network.Marshal(ds)
	if err != nil {
		log.Errorf("Marshaling PubVerShare failed: %v", err)
		return nil, nil
	}

	var Cs []abstract.Point
	k := network.Suite.Scalar().Pick(random.Stream)
	K := network.Suite.Point().Mul(nil, k)
	S := network.Suite.Point().Mul(rPubKey, k)
	for len(msg) > 0 {
		kp, _ := network.Suite.Point().Pick(msg, random.Stream)
		Cs = append(Cs, network.Suite.Point().Add(S, kp))
		msg = msg[min(len(msg), kp.PickLen()):]
	}
	return K, Cs
}

func verifyDecryptionRequest(decReqData *util.OTSDecryptReqData, sig *crypto.SchnorrSig) (*util.WriteTxnData, error) {
	_, tmp, err := network.Unmarshal(decReqData.WriteTxnSBF.Data)
	if err != nil {
		log.Errorf("Unmarshaling WriteTxnSBF failed: %v", err)
		return nil, err
	}

	writeTxn := tmp.(*ocs.DataOCS).WriteTxn.Data
	_, tmp, err = network.Unmarshal(decReqData.ReadTxnSBF.Data)
	if err != nil {
		log.Errorf("Unmarshaling ReadTxnSBF failed: %v", err)
		return nil, err
	}

	readTxn := tmp.(*ocs.DataOCS).Read
	// 1) Check signature on the DecReq message
	drd, err := network.Marshal(decReqData)
	if err != nil {
		log.Errorf("Marshaling DecryptReqData failed: %v", err)
		return nil, err
	}

	tmpHash := sha256.Sum256(drd)
	drdHash := tmpHash[:]
	sigErr := crypto.VerifySchnorr(network.Suite, writeTxn.ReaderPk, drdHash, *sig)
	if sigErr != nil {
		log.Errorf("Cannot verify DecReq message signature: %v", sigErr)
		return nil, sigErr
	}

	// 2) Check inclusion proof
	readSBHash := decReqData.ReadTxnSBF.CalculateHash()
	proof := decReqData.InclusionProof
	if len(proof.Signature) == 0 {
		log.Error("No signature present on forward-link")
		return nil, errors.New("No signature present on forward-link")
	}

	hc := proof.Hash.Equal(readSBHash)
	if !hc {
		log.Error("Forward link hash does not match read transaction hash")
		return nil, errors.New("Forward link hash does not match read transaction hash")
	}

	sigErr = cosi.VerifySignature(network.Suite, decReqData.ACPublicKeys, proof.Hash, proof.Signature)
	if sigErr != nil {
		log.Error("Cannot verify forward-link signature")
		return nil, sigErr
	}

	// 3) Check that read contains write's hash
	writeSBHash := decReqData.WriteTxnSBF.CalculateHash()
	hc = readTxn.DataID.Equal(writeSBHash)
	if !hc {
		log.Error("Invalid write block hash in the read block")
		return nil, errors.New("Invalid write block hash in the read block")
	}
	return writeTxn, nil
}
