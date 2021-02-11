package util

import (
	"crypto/sha256"
	"errors"
	"os"
	"syscall"

	"gopkg.in/dedis/crypto.v0/abstract"
	onet "gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/app"
	"gopkg.in/dedis/onet.v1/crypto"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
)

func iiToF(sec int64, usec int64) float64 {
	return float64(sec) + float64(usec)/1000000.0
}

func GetRTime() (tSys, tUsr float64) {
	rusage := &syscall.Rusage{}
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, rusage); err != nil {
		log.Error("Couldn't get rusage time:", err)
		return -1, -1
	}
	s, u := rusage.Stime, rusage.Utime
	return iiToF(int64(s.Sec), int64(s.Usec)), iiToF(int64(u.Sec), int64(u.Usec))
}

func GetDiffRTime(tSys, tUsr float64) (tDiffSys, tDiffUsr float64) {
	nowSys, nowUsr := GetRTime()
	return nowSys - tSys, nowUsr - tUsr
}

func SignMessage(msg []byte, privKey abstract.Scalar) (crypto.SchnorrSig, error) {
	tmpHash := sha256.Sum256(msg)
	msgHash := tmpHash[:]
	return crypto.SignSchnorr(network.Suite, privKey, msgHash)
}

func CreatePointH(suite abstract.Suite, pubKey abstract.Point) (abstract.Point, error) {

	binPubKey, err := pubKey.MarshalBinary()
	if err != nil {
		return nil, err
	}
	tmpHash := sha256.Sum256(binPubKey)
	labelHash := tmpHash[:]
	h, _ := suite.Point().Pick(nil, suite.Cipher(labelHash))
	return h, nil
}

func ReadRoster(tomlFileName string) (*onet.Roster, error) {
	log.Lvl3("Reading in the roster from group.toml")
	f, err := os.Open(tomlFileName)
	if err != nil {
		return nil, err
	}
	el, err := app.ReadGroupToml(f)
	if err != nil {
		return nil, err
	}
	if len(el.List) <= 0 {
		return nil, errors.New("Empty or invalid group file:" +
			tomlFileName)
	}
	log.Lvl3(el)
	return el, err
}
