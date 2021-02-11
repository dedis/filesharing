// This package provides functionality to create and verify non-interactive
// zero-knowledge (NIZK) proofs for the equality (EQ) of discrete logarithms (DL).
// This means, for two values xG and xH one can check that
//   log_{G}(xG) == log_{H}(xH)
// without revealing the secret value x.
package proof

import (
	"errors"

	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/crypto.v0/hash"
	"gopkg.in/dedis/crypto.v0/random"
)

var errorDifferentLengths = errors.New("inputs of different lengths")
var errorInvalidProof = errors.New("invalid proof")

// DLEQProof represents a NIZK dlog-equality proof.
type DLEQProof struct {
	C  abstract.Scalar // challenge
	R  abstract.Scalar // response
	VG abstract.Point  // public commitment with respect to base point G
	VH abstract.Point  // public commitment with respect to base point H
}

// NewDLEQProof computes a new NIZK dlog-equality proof for the scalar x with
// respect to base points G and H. It therefore randomly selects a commitment v
// and then computes the challenge c = H(xG,xH,vG,vH) and response r = v - cx.
// Besides the proof, this function also returns the encrypted base points xG
// and xH.
func NewDLEQProof(suite abstract.Suite, G abstract.Point, H abstract.Point, x abstract.Scalar) (proof *DLEQProof, xG abstract.Point, xH abstract.Point, err error) {
	// Encrypt base points with secret
	xG = suite.Point().Mul(G, x)
	xH = suite.Point().Mul(H, x)

	// Commitment
	v := suite.Scalar().Pick(random.Stream)
	vG := suite.Point().Mul(G, v)
	vH := suite.Point().Mul(H, v)

	// Challenge
	cb, err := hash.Structures(suite.Hash(), xG, xH, vG, vH)
	if err != nil {
		return nil, nil, nil, err
	}
	c := suite.Scalar().Pick(suite.Cipher(cb))

	// Response
	r := suite.Scalar()
	r.Mul(x, c).Sub(v, r)

	return &DLEQProof{c, r, vG, vH}, xG, xH, nil
}

// NewDLEQProofBatch computes lists of NIZK dlog-equality proofs and of
// encrypted base points xG and xH. Note that the challenge is computed over all
// input values.
func NewDLEQProofBatch(suite abstract.Suite, G []abstract.Point, H []abstract.Point, secrets []abstract.Scalar) (proof []*DLEQProof, xG []abstract.Point, xH []abstract.Point, err error) {
	if len(G) != len(H) || len(H) != len(secrets) {
		return nil, nil, nil, errorDifferentLengths
	}

	n := len(secrets)
	proofs := make([]*DLEQProof, n)
	v := make([]abstract.Scalar, n)
	xG = make([]abstract.Point, n)
	xH = make([]abstract.Point, n)
	vG := make([]abstract.Point, n)
	vH := make([]abstract.Point, n)

	for i, x := range secrets {
		// Encrypt base points with secrets
		xG[i] = suite.Point().Mul(G[i], x)
		xH[i] = suite.Point().Mul(H[i], x)

		// Commitments
		v[i] = suite.Scalar().Pick(random.Stream)
		vG[i] = suite.Point().Mul(G[i], v[i])
		vH[i] = suite.Point().Mul(H[i], v[i])
	}

	// Collective challenge
	cb, err := hash.Structures(suite.Hash(), xG, xH, vG, vH)
	if err != nil {
		return nil, nil, nil, err
	}
	c := suite.Scalar().Pick(suite.Cipher(cb))

	// Responses
	for i, x := range secrets {
		r := suite.Scalar()
		r.Mul(x, c).Sub(v[i], r)
		proofs[i] = &DLEQProof{c, r, vG[i], vH[i]}
	}

	return proofs, xG, xH, nil
}

// Verify examines the validity of the NIZK dlog-equality proof.
// The proof is valid if the following two conditions hold:
//   vG == rG + c(xG)
//   vH == rH + c(xH)
func (p *DLEQProof) Verify(suite abstract.Suite, G abstract.Point, H abstract.Point, xG abstract.Point, xH abstract.Point) error {
	rG := suite.Point().Mul(G, p.R)
	rH := suite.Point().Mul(H, p.R)
	cxG := suite.Point().Mul(xG, p.C)
	cxH := suite.Point().Mul(xH, p.C)
	a := suite.Point().Add(rG, cxG)
	b := suite.Point().Add(rH, cxH)
	if !(p.VG.Equal(a) && p.VH.Equal(b)) {
		return errorInvalidProof
	}
	return nil
}
