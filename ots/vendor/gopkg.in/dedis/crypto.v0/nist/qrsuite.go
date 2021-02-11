package nist

import (
	"crypto/cipher"
	"crypto/sha256"
	"hash"
	"io"
	"math/big"
	"reflect"

	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/crypto.v0/cipher/sha3"
	"gopkg.in/dedis/crypto.v0/random"
)

type qrsuite struct {
	ResidueGroup
}

// SHA256 hash function
func (s qrsuite) Hash() hash.Hash {
	return sha256.New()
}

// SHA3/SHAKE128 Sponge Cipher
func (s qrsuite) Cipher(key []byte, options ...interface{}) abstract.Cipher {
	return sha3.NewShakeCipher128(key, options...)
}

func (s *qrsuite) Read(r io.Reader, objs ...interface{}) error {
	return abstract.SuiteRead(s, r, objs)
}

func (s *qrsuite) Write(w io.Writer, objs ...interface{}) error {
	return abstract.SuiteWrite(s, w, objs)
}

func (s *qrsuite) New(t reflect.Type) interface{} {
	return abstract.SuiteNew(s, t)
}

func (s *qrsuite) NewKey(rand cipher.Stream) abstract.Scalar {
	if rand == nil {
		rand = random.Stream
	}
	return s.Scalar().Pick(rand)
}

// Ciphersuite based on AES-128, SHA-256,
// and a residue group of quadratic residues modulo a 512-bit prime.
// This group size should be used only for testing and experimentation;
// 512-bit DSA-style groups are no longer considered secure.
func NewAES128SHA256QR512() abstract.Suite {
	p, _ := new(big.Int).SetString("10198267722357351868598076141027380280417188309231803909918464305012113541414604537422741096561285049775792035177041672305646773132014126091142862443826263", 10)
	q, _ := new(big.Int).SetString("5099133861178675934299038070513690140208594154615901954959232152506056770707302268711370548280642524887896017588520836152823386566007063045571431221913131", 10)
	r := new(big.Int).SetInt64(2)
	g := new(big.Int).SetInt64(4)

	suite := new(qrsuite)
	suite.SetParams(p, q, r, g)
	return suite
}

// Ciphersuite based on AES-128, SHA-256,
// and a residue group of quadratic residues modulo a 1024-bit prime.
// 1024-bit DSA-style groups may no longer be secure.
func newAES128SHA256QR1024() abstract.Suite {
	suite := new(qrsuite)
	suite.QuadraticResidueGroup(1024, random.Stream) // XXX
	return suite
}

// Ciphersuite based on AES-128, SHA-256,
// and a residue group of quadratic residues modulo a 1024-bit prime.
func newAES128SHA256QR2048() abstract.Suite {
	suite := new(qrsuite)
	suite.QuadraticResidueGroup(2048, random.Stream) // XXX
	return suite
}
