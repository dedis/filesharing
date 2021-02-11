package protocol

import (
	"errors"

	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/crypto.v0/random"
	"gopkg.in/dedis/crypto.v0/share/dkg"
	"gopkg.in/dedis/onet.v1/log"
)

// EncodeKey can be used by the writer to an onchain-secret skipchain
// to encode his symmetric key under the collective public key created
// by the DKG.
// As this method uses `Pick` to encode the key, depending on the key-length
// more than one point is needed to encode the data.
//
// Input:
//   - suite - the cryptographic suite to use
//   - X - the aggregate public key of the DKG
//   - key - the symmetric key for the document
//
// Output:
//   - U - the schnorr commit
//   - Cs - encrypted key-slices
func EncodeKey(suite abstract.Suite, X abstract.Point, key []byte) (U abstract.Point, Cs []abstract.Point) {
	r := suite.Scalar().Pick(random.Stream)
	U = suite.Point().Mul(nil, r)

	rem := make([]byte, len(key))
	copy(rem, key)
	for len(rem) > 0 {
		var kp abstract.Point
		kp, rem = suite.Point().Pick(rem, random.Stream)
		C := suite.Point().Mul(X, r)
		Cs = append(Cs, C.Add(C, kp))
	}
	return
}

// DecodeKey can be used by the reader of an onchain-secret to convert the
// re-encrypted secret back to a symmetric key that can be used later to
// decode the document.
//
// Input:
//   - suite - the cryptographic suite to use
//   - X - the aggregate public key of the DKG
//   - Cs - the encrypted key-slices
//   - XhatEnc - the re-encrypted schnorr-commit
//   - xc - the private key of the reader
//
// Output:
//   - key - the re-assembled key
//   - err - an eventual error when trying to recover the data from the points
func DecodeKey(suite abstract.Suite, X abstract.Point, Cs []abstract.Point, XhatEnc abstract.Point,
	xc abstract.Scalar) (key []byte, err error) {
	xcInv := suite.Scalar().Neg(xc)
	XhatDec := suite.Point().Mul(X, xcInv)
	Xhat := suite.Point().Add(XhatEnc, XhatDec)
	XhatInv := suite.Point().Neg(Xhat)

	// Decrypt Cs to keyPointHat
	for _, C := range Cs {
		keyPointHat := suite.Point().Add(C, XhatInv)
		keyPart, err := keyPointHat.Data()
		if err != nil {
			return nil, err
		}
		key = append(key, keyPart...)
	}
	return
}

// CreateDKGs can be used for testing to set up a set of DKGs.
//
// Input:
//   - suite - the suite to use
//   - nbrNodes - how many nodes to set up
//   - threshold - how many nodes can recover the secret
//
// Output:
//   - dkgs - a slice of dkg-structures
//   - err - an eventual error
func CreateDKGs(suite abstract.Suite, nbrNodes, threshold int) (dkgs []*dkg.DistKeyGenerator, err error) {
	// 1 - share generation
	dkgs = make([]*dkg.DistKeyGenerator, nbrNodes)
	scalars := make([]abstract.Scalar, nbrNodes)
	points := make([]abstract.Point, nbrNodes)
	// 1a - initialisation
	for i := range scalars {
		scalars[i] = suite.Scalar().Pick(random.Stream)
		points[i] = suite.Point().Mul(nil, scalars[i])
	}

	// 1b - key-sharing
	for i := range dkgs {
		dkgs[i], err = dkg.NewDistKeyGenerator(suite,
			scalars[i], points, random.Stream, threshold)
		if err != nil {
			return
		}
	}
	// Exchange of Deals
	responses := make([][]*dkg.Response, nbrNodes)
	for i, p := range dkgs {
		responses[i] = make([]*dkg.Response, nbrNodes)
		deals, err := p.Deals()
		if err != nil {
			return nil, err
		}
		for j, d := range deals {
			responses[i][j], err = dkgs[j].ProcessDeal(d)
			if err != nil {
				return nil, err
			}
		}
	}
	// ProcessResponses
	for i, resp := range responses {
		for j, r := range resp {
			for k, p := range dkgs {
				if r != nil && j != k {
					log.Lvl3("Response from-to-peer:", i, j, k)
					justification, err := p.ProcessResponse(r)
					if err != nil {
						return nil, err
					}
					if justification != nil {
						return nil, errors.New("there should be no justification")
					}
				}
			}
		}
	}

	// Secret commits
	for _, p := range dkgs {
		commit, err := p.SecretCommits()
		if err != nil {
			return nil, err
		}
		for _, p2 := range dkgs {
			compl, err := p2.ProcessSecretCommits(commit)
			if err != nil {
				return nil, err
			}
			if compl != nil {
				return nil, errors.New("there should be no complaint")
			}
		}
	}

	// Verify if all is OK
	for _, p := range dkgs {
		if !p.Finished() {
			return nil, errors.New("one of the dkgs is not finished yet")
		}
	}
	return
}
