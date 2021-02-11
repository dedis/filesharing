package vss

import (
	"crypto/aes"
	"crypto/cipher"
	"hash"

	"golang.org/x/crypto/hkdf"

	"gopkg.in/dedis/crypto.v0/abstract"
)

// dhExchange computes the shared key from a private key and a public key
func dhExchange(suite abstract.Suite, ownPrivate abstract.Scalar, remotePublic abstract.Point) abstract.Point {
	sk := suite.Point()
	sk.Mul(remotePublic, ownPrivate)
	return sk
}

var sharedKeyLength = 32

// newAEAD returns the AEAD cipher to be use to encrypt a share
func newAEAD(fn func() hash.Hash, preSharedKey abstract.Point, context []byte) (cipher.AEAD, error) {
	preBuff, _ := preSharedKey.MarshalBinary()
	reader := hkdf.New(fn, preBuff, nil, context)

	sharedKey := make([]byte, sharedKeyLength)
	if _, err := reader.Read(sharedKey); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(sharedKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm, nil
}

// context returns the context slice to be used when encrypting a share
func context(suite abstract.Suite, dealer abstract.Point, verifiers []abstract.Point) []byte {
	h := suite.Hash()
	h.Write([]byte("vss-dealer"))
	dealer.MarshalTo(h)
	h.Write([]byte("vss-verifiers"))
	for _, v := range verifiers {
		v.MarshalTo(h)
	}
	return h.Sum(nil)
}
