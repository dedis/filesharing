package otsclient

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"errors"
	"os"

	ocs "github.com/calypso-demo/filesharing/ots/onchain-secrets"
	"github.com/calypso-demo/filesharing/ots/otsclient/util"
	"gopkg.in/dedis/cothority.v1/skipchain"
	"gopkg.in/dedis/crypto.v0/share/pvss"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/network"

	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/crypto.v0/random"
	"gopkg.in/dedis/onet.v1/crypto"
)

func AddDummyTxnPairs(scurl *ocs.SkipChainURL, dp *util.DataPVSS, pairCount int) error {
	mesg := "On Wisconsin!"
	writerSK := make([]abstract.Scalar, pairCount)
	writerPK := make([]abstract.Point, pairCount)
	readerSK := make([]abstract.Scalar, pairCount)
	readerPK := make([]abstract.Point, pairCount)
	sbWrite := make([]*skipchain.SkipBlock, pairCount)
	sbRead := make([]*skipchain.SkipBlock, pairCount)

	for i := 0; i < pairCount; i++ {
		readerSK[i] = dp.Suite.Scalar().Pick(random.Stream)
		readerPK[i] = dp.Suite.Point().Mul(nil, readerSK[i])
		writerSK[i] = dp.Suite.Scalar().Pick(random.Stream)
		writerPK[i] = dp.Suite.Point().Mul(nil, writerSK[i])
		err := SetupPVSS(dp, readerPK[i])
		if err != nil {
			return err
		}
		_, hashEnc, _ := EncryptMessage(dp, []byte(mesg))

		tmp, err := CreateWriteTxn(scurl, dp, hashEnc, readerPK[i], writerSK[i])
		if err != nil {
			return err
		}
		sbWrite[i] = tmp
	}

	for i := 0; i < pairCount-1; i++ {
		tmp, err := CreateReadTxn(scurl, sbWrite[i].Hash, readerSK[i])
		if err != nil {
			return err
		}
		sbRead[i] = tmp
	}
	return nil
}

func ElGamalDecrypt(shares []*util.DecryptedShare, privKey abstract.Scalar) ([]*pvss.PubVerShare, error) {
	size := len(shares)
	decShares := make([]*pvss.PubVerShare, size)
	for i := 0; i < size; i++ {
		tmp := shares[i]
		var decSh []byte
		for _, C := range tmp.Cs {
			S := network.Suite.Point().Mul(tmp.K, privKey)
			decShPart := network.Suite.Point().Sub(C, S)
			decShPartData, _ := decShPart.Data()
			decSh = append(decSh, decShPartData...)
		}
		_, tmpSh, err := network.Unmarshal(decSh)
		if err != nil {
			return nil, err
		}

		sh := tmpSh.(*pvss.PubVerShare)
		decShares[i] = sh
	}
	return decShares, nil
}

func GetDecryptedShares(scurl *ocs.SkipChainURL, el *onet.Roster, writeTxnSB *skipchain.SkipBlock, readTxnSBF *skipchain.SkipBlockFix, acPubKeys []abstract.Point, scPubKeys []abstract.Point, privKey abstract.Scalar, index int) ([]*pvss.PubVerShare, error) {
	cl := otssmc.NewClient()
	defer cl.Close()
	idx := index - writeTxnSB.Index - 1
	if idx < 0 {
		return nil, errors.New("Forward-link index is negative")
	}

	inclusionProof := writeTxnSB.GetForward(idx)
	if inclusionProof == nil {
		return nil, errors.New("Forward-link does not exist")
	}

	reencShares, cerr := cl.OTSDecrypt(el, writeTxnSB.SkipBlockFix, readTxnSBF, inclusionProof, acPubKeys, privKey)
	if cerr != nil {
		return nil, cerr
	}

	tmpDecShares, err := ElGamalDecrypt(reencShares, privKey)
	if err != nil {
		return nil, err
	}

	size := len(tmpDecShares)
	decShares := make([]*pvss.PubVerShare, size)
	for i := 0; i < size; i++ {
		decShares[tmpDecShares[i].S.I] = tmpDecShares[i]
	}
	return decShares, nil
}

func GetUpdatedWriteTxnSB(scurl *ocs.SkipChainURL, sbid skipchain.SkipBlockID) (*skipchain.SkipBlock, error) {
	cl := skipchain.NewClient()
	defer cl.Close()
	sb, err := cl.GetSingleBlock(scurl.Roster, sbid)
	return sb, err
}

func CreateReadTxn(scurl *ocs.SkipChainURL, dataID skipchain.SkipBlockID, privKey abstract.Scalar) (*skipchain.SkipBlock, error) {
	cl := ocs.NewClient()
	defer cl.Close()
	sb, err := cl.ReadTxnRequest(scurl, dataID, privKey)
	return sb, err
}

func VerifyTxnSignature(writeTxnData *util.WriteTxnData, sig *crypto.SchnorrSig, wrPubKey abstract.Point) error {
	// network.RegisterMessage(&util.WriteTxnData{})
	wtd, err := network.Marshal(writeTxnData)
	if err != nil {
		return err
	}

	tmpHash := sha256.Sum256(wtd)
	wtdHash := tmpHash[:]
	return crypto.VerifySchnorr(network.Suite, wrPubKey, wtdHash, *sig)
}

func GetWriteTxnSB(scurl *ocs.SkipChainURL, dataID skipchain.SkipBlockID) (*skipchain.SkipBlock, *util.WriteTxnData, *crypto.SchnorrSig, error) {
	cl := ocs.NewClient()
	defer cl.Close()
	sbWrite, tmpTxn, err := cl.GetWriteTxn(scurl, dataID)
	if err != nil {
		return nil, nil, nil, err
	}

	sig := tmpTxn.Signature
	writeTxnData := &util.WriteTxnData{
		G:            tmpTxn.Data.G,
		SCPublicKeys: tmpTxn.Data.SCPublicKeys,
		EncShares:    tmpTxn.Data.EncShares,
		EncProofs:    tmpTxn.Data.EncProofs,
		HashEnc:      tmpTxn.Data.HashEnc,
		ReaderPk:     tmpTxn.Data.ReaderPk,
	}
	return sbWrite, writeTxnData, sig, nil
}

func CreateWriteTxn(scurl *ocs.SkipChainURL, dp *util.DataPVSS, hashEnc []byte, pubKey abstract.Point, wrPrivKey abstract.Scalar) (*skipchain.SkipBlock, error) {
	cl := ocs.NewClient()
	defer cl.Close()
	readList := make([]abstract.Point, 1)
	readList[0] = pubKey
	sb, err := cl.WriteTxnRequest(scurl, dp.G, dp.SCPublicKeys, dp.EncShares, dp.EncProofs, hashEnc, readList, wrPrivKey)
	return sb, err
}

func CreateSkipchain(el *onet.Roster) (*ocs.SkipChainURL, error) {
	cl := ocs.NewClient()
	defer cl.Close()
	scurl, err := cl.CreateSkipchain(el)
	return scurl, err
}

func VerifyEncMesg(wtd *util.WriteTxnData, encMesg []byte) int {
	tmpHash := sha256.Sum256(encMesg)
	cmptHash := tmpHash[:]
	return bytes.Compare(cmptHash, wtd.HashEnc)
}

func DecryptMessage(recSecret abstract.Point, encMesg []byte, wtd *util.WriteTxnData) ([]byte, error) {
	g_s, err := recSecret.MarshalBinary()
	if err != nil {
		return nil, err
	}

	tempSymKey := sha256.Sum256(g_s)
	symKey := tempSymKey[:]
	cipher := network.Suite.Cipher(symKey)
	decMesg, err := cipher.Open(nil, encMesg)
	return decMesg, err
}

func EncryptMessage(dp *util.DataPVSS, mesg []byte) ([]byte, []byte, error) {
	g_s, err := dp.Suite.Point().Mul(nil, dp.Secret).MarshalBinary()
	if err != nil {
		return nil, nil, err
	}

	tempSymKey := sha256.Sum256(g_s)
	symKey := tempSymKey[:]
	cipher := network.Suite.Cipher(symKey)
	encMesg := cipher.Seal(nil, mesg)
	tempHash := sha256.Sum256(encMesg)
	hashEnc := tempHash[:]
	return encMesg, hashEnc, nil
}

func SetupPVSS(dp *util.DataPVSS, pubKey abstract.Point) error {
	g := dp.Suite.Point().Base()
	h, err := util.CreatePointH(dp.Suite, pubKey)
	if err != nil {
		return err
	}

	secret := dp.Suite.Scalar().Pick(random.Stream)
	threshold := 2*dp.NumTrustee/3 + 1
	// PVSS step
	encShares, commitPoly, err := pvss.EncShares(dp.Suite, h, dp.SCPublicKeys, secret, threshold)
	if err == nil {
		encProofs := make([]abstract.Point, dp.NumTrustee)
		for i := 0; i < dp.NumTrustee; i++ {
			encProofs[i] = commitPoly.Eval(encShares[i].S.I).V
		}
		dp.Threshold = threshold
		dp.G = g
		dp.H = h
		dp.Secret = secret
		dp.EncShares = encShares
		dp.EncProofs = encProofs
		return nil
	} else {
		return err
	}
}

func GetPubKeys(fname *string) ([]abstract.Point, error) {
	var keys []abstract.Point
	fh, err := os.Open(*fname)
	defer fh.Close()
	if err != nil {
		return nil, err
	}

	fs := bufio.NewScanner(fh)
	for fs.Scan() {
		tmp, err := crypto.String64ToPoint(network.Suite, fs.Text())
		if err != nil {
			return nil, err
		}
		keys = append(keys, tmp)
	}
	return keys, nil
}
