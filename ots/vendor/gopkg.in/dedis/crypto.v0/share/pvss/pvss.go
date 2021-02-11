// Package pvss implements public verifiable secret sharing as introduced in
// "A Simple Publicly Verifiable Secret Sharing Scheme and its Application to
// Electronic Voting" by Berry Schoenmakers. In comparison to regular verifiable
// secret sharing schemes, PVSS enables any third party to verify shares
// distributed by a dealer using zero-knowledge proofs. PVSS runs in three steps:
//  1. The dealer creates a list of encrypted public verifiable shares using
//     EncShares() and distributes them to the trustees.
//  2. Upon the announcement that the secret should be released, each trustee
//     uses DecShare() to first verify and, if valid, decrypt his share.
//  3. Once a threshold of decrypted shares has been released, anyone can
//     verify them and, if enough shares are valid, recover the shared secret
//     using RecoverSecret().
// For concrete examples see pvss_test.go.
package pvss

import (
	"errors"

	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/crypto.v0/proof"
	"gopkg.in/dedis/crypto.v0/random"
	"gopkg.in/dedis/crypto.v0/share"
)

// Some error definitions.
var errorTooFewShares = errors.New("not enough shares to recover secret")
var errorDifferentLengths = errors.New("inputs of different lengths")
var errorEncVerification = errors.New("verification of encrypted share failed")
var errorDecVerification = errors.New("verification of decrypted share failed")

// PubVerShare is a public verifiable share.
type PubVerShare struct {
	S share.PubShare  // Share
	P proof.DLEQProof // Proof
}

// EncShares creates a list of encrypted publicly verifiable PVSS shares for
// the given secret and the list of public keys X using the sharing threshold
// t and the base point H. The function returns the list of shares and the
// public commitment polynomial.
func EncShares(suite abstract.Suite, H abstract.Point, X []abstract.Point, secret abstract.Scalar, t int) ([]*PubVerShare, *share.PubPoly, error) {
	n := len(X)
	encShares := make([]*PubVerShare, n)

	// Create secret sharing polynomial
	priPoly := share.NewPriPoly(suite, t, secret, random.Stream)

	// Create secret set of shares
	priShares := priPoly.Shares(n)

	// Create public polynomial commitments with respect to basis H
	pubPoly := priPoly.Commit(H)

	// Prepare data for encryption consistency proofs ...
	indices := make([]int, n)
	values := make([]abstract.Scalar, n)
	HS := make([]abstract.Point, n)
	for i := 0; i < n; i++ {
		indices[i] = priShares[i].I
		values[i] = priShares[i].V
		HS[i] = H
	}

	// Create NIZK discrete-logarithm equality proofs
	proofs, _, sX, err := proof.NewDLEQProofBatch(suite, HS, X, values)
	if err != nil {
		return nil, nil, err
	}

	for i := 0; i < n; i++ {
		ps := &share.PubShare{indices[i], sX[i]}
		encShares[i] = &PubVerShare{*ps, *proofs[i]}
	}

	return encShares, pubPoly, nil
}

// VerifyEncShare checks that the encrypted share sX satisfies
// log_{H}(sH) == log_{X}(sX) where sH is the public commitment computed by
// evaluating the public commitment polynomial at the encrypted share's index i.
func VerifyEncShare(suite abstract.Suite, H abstract.Point, X abstract.Point, sH abstract.Point, encShare *PubVerShare) error {
	if err := encShare.P.Verify(suite, H, X, sH, encShare.S.V); err != nil {
		return errorEncVerification
	}
	return nil
}

// VerifyEncShareBatch provides the same functionality as VerifyEncShare but for
// slices of encrypted shares. The function returns the valid encrypted shares
// together with the corresponding public keys.
func VerifyEncShareBatch(suite abstract.Suite, H abstract.Point, X []abstract.Point, sH []abstract.Point, encShares []*PubVerShare) ([]abstract.Point, []*PubVerShare, error) {
	if len(X) != len(sH) || len(sH) != len(encShares) {
		return nil, nil, errorDifferentLengths
	}
	var K []abstract.Point // good public keys
	var E []*PubVerShare   // good encrypted shares
	for i := 0; i < len(X); i++ {
		if err := VerifyEncShare(suite, H, X[i], sH[i], encShares[i]); err == nil {
			K = append(K, X[i])
			E = append(E, encShares[i])
		}
	}
	return K, E, nil
}

// DecShare first verifies the encrypted share against the encryption
// consistency proof and, if valid, decrypts it and creates a decryption
// consistency proof.
func DecShare(suite abstract.Suite, H abstract.Point, X abstract.Point, sH abstract.Point, x abstract.Scalar, encShare *PubVerShare) (*PubVerShare, error) {
	if err := VerifyEncShare(suite, H, X, sH, encShare); err != nil {
		return nil, err
	}
	G := suite.Point().Base()
	V := suite.Point().Mul(encShare.S.V, suite.Scalar().Inv(x)) // decryption: x^{-1} * (xS)
	ps := &share.PubShare{encShare.S.I, V}
	P, _, _, err := proof.NewDLEQProof(suite, G, V, x)
	if err != nil {
		return nil, err
	}
	return &PubVerShare{*ps, *P}, nil
}

// DecShareBatch provides the same functionality as DecShare but for slices of
// encrypted shares. The function returns the valid encrypted and decrypted
// shares as well as the corresponding public keys.
func DecShareBatch(suite abstract.Suite, H abstract.Point, X []abstract.Point, sH []abstract.Point, x abstract.Scalar, encShares []*PubVerShare) ([]abstract.Point, []*PubVerShare, []*PubVerShare, error) {
	if len(X) != len(sH) || len(sH) != len(encShares) {
		return nil, nil, nil, errorDifferentLengths
	}
	var K []abstract.Point // good public keys
	var E []*PubVerShare   // good encrypted shares
	var D []*PubVerShare   // good decrypted shares
	for i := 0; i < len(encShares); i++ {
		if ds, err := DecShare(suite, H, X[i], sH[i], x, encShares[i]); err == nil {
			K = append(K, X[i])
			E = append(E, encShares[i])
			D = append(D, ds)
		}
	}
	return K, E, D, nil
}

// VerifyDecShare checks that the decrypted share sG satisfies
// log_{G}(X) == log_{sG}(sX). Note that X = xG and sX = s(xG) = x(sG).
func VerifyDecShare(suite abstract.Suite, G abstract.Point, X abstract.Point, encShare *PubVerShare, decShare *PubVerShare) error {
	if err := decShare.P.Verify(suite, G, decShare.S.V, X, encShare.S.V); err != nil {
		return errorDecVerification
	}
	return nil
}

// VerifyDecShareBatch provides the same functionality as VerifyDecShare but for
// slices of decrypted shares. The function returns the the valid decrypted shares.
func VerifyDecShareBatch(suite abstract.Suite, G abstract.Point, X []abstract.Point, encShares []*PubVerShare, decShares []*PubVerShare) ([]*PubVerShare, error) {
	if len(X) != len(encShares) || len(encShares) != len(decShares) {
		return nil, errorDifferentLengths
	}
	var D []*PubVerShare // good decrypted shares
	for i := 0; i < len(X); i++ {
		if err := VerifyDecShare(suite, G, X[i], encShares[i], decShares[i]); err == nil {
			D = append(D, decShares[i])
		}
	}
	return D, nil
}

// RecoverSecret first verifies the given decrypted shares against their
// decryption consistency proofs and then tries to recover the shared secret.
func RecoverSecret(suite abstract.Suite, G abstract.Point, X []abstract.Point, encShares []*PubVerShare, decShares []*PubVerShare, t int, n int) (abstract.Point, error) {
	D, err := VerifyDecShareBatch(suite, G, X, encShares, decShares)
	if err != nil {
		return nil, err
	}
	if len(D) < t {
		return nil, errorTooFewShares
	}
	var shares []*share.PubShare
	for _, s := range D {
		shares = append(shares, &s.S)
	}
	return share.RecoverCommit(suite, shares, t, n)
}
