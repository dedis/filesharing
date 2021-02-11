// Package share implements Shamir secret sharing and polynomial commitments.
// Shamir's scheme allows to split a secret value into multiple parts, so called
// shares, by evaluating a secret sharing polynomial at certain indices. The
// shared secret can only be reconstructed (via Lagrange interpolation) if a
// threshold of the participants provide their shares. A polynomial commitment
// scheme allows a committer to commit to a secret sharing polynomial so that
// a verifier can check the claimed evaluations of the committed polynomial.
// Both schemes of this package are core building blocks for more advanced
// secret sharing techniques.
package share

import (
	"crypto/cipher"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"strings"

	"gopkg.in/dedis/crypto.v0/abstract"
)

// Some error definitions
var errorGroups = errors.New("non-matching groups")
var errorCoeffs = errors.New("different number of coefficients")

// PriShare represents a private share.
type PriShare struct {
	I int             // Index of the private share
	V abstract.Scalar // Value of the private share
}

// Hash computes the hash of the private share.
func (p *PriShare) Hash(s abstract.Suite) []byte {
	h := s.Hash()
	p.V.MarshalTo(h)
	binary.Write(h, binary.LittleEndian, p.I)
	return h.Sum(nil)
}

// PriPoly represents a secret sharing polynomial.
type PriPoly struct {
	g      abstract.Group    // Cryptographic group
	coeffs []abstract.Scalar // Coefficients of the polynomial
}

// NewPriPoly creates a new secret sharing polynomial for the cryptographic
// group g, the secret sharing threshold t, and the secret to be shared s.
func NewPriPoly(g abstract.Group, t int, s abstract.Scalar, rand cipher.Stream) *PriPoly {
	coeffs := make([]abstract.Scalar, t)
	coeffs[0] = s
	if coeffs[0] == nil {
		coeffs[0] = g.Scalar().Pick(rand)
	}
	for i := 1; i < t; i++ {
		coeffs[i] = g.Scalar().Pick(rand)
	}
	return &PriPoly{g, coeffs}
}

// Threshold returns the secret sharing threshold.
func (p *PriPoly) Threshold() int {
	return len(p.coeffs)
}

// Secret returns the shared secret p(0), i.e., the constant term of the polynomial.
func (p *PriPoly) Secret() abstract.Scalar {
	return p.coeffs[0]
}

// Eval computes the private share v = p(i).
func (p *PriPoly) Eval(i int) *PriShare {
	xi := p.g.Scalar().SetInt64(1 + int64(i))
	v := p.g.Scalar().Zero()
	for j := p.Threshold() - 1; j >= 0; j-- {
		v.Mul(v, xi)
		v.Add(v, p.coeffs[j])
	}
	return &PriShare{i, v}
}

// Shares creates a list of n private shares p(1),...,p(n).
func (p *PriPoly) Shares(n int) []*PriShare {
	shares := make([]*PriShare, n)
	for i := range shares {
		shares[i] = p.Eval(i)
	}
	return shares
}

// Add computes the component-wise sum of the polynomials p and q and returns it
// as a new polynomial.
func (p *PriPoly) Add(q *PriPoly) (*PriPoly, error) {
	if p.g.String() != q.g.String() {
		return nil, errorGroups
	}
	if p.Threshold() != q.Threshold() {
		return nil, errorCoeffs
	}
	coeffs := make([]abstract.Scalar, p.Threshold())
	for i := range coeffs {
		coeffs[i] = p.g.Scalar().Add(p.coeffs[i], q.coeffs[i])
	}
	return &PriPoly{p.g, coeffs}, nil
}

// Equal checks equality of two secret sharing polynomials p and q.
func (p *PriPoly) Equal(q *PriPoly) bool {
	if p.g.String() != q.g.String() {
		return false
	}
	if len(p.coeffs) != len(q.coeffs) {
		return false
	}
	b := 1
	for i := 0; i < p.Threshold(); i++ {
		pb := p.coeffs[i].Bytes()
		qb := q.coeffs[i].Bytes()
		b &= subtle.ConstantTimeCompare(pb, qb)
	}
	return b == 1
}

// Commit creates a public commitment polynomial for the given base point b or
// the standard base if b == nil.
func (p *PriPoly) Commit(b abstract.Point) *PubPoly {
	commits := make([]abstract.Point, p.Threshold())
	for i := range commits {
		commits[i] = p.g.Point().Mul(b, p.coeffs[i])
	}
	return &PubPoly{p.g, b, commits}
}

// Mul multiples p  and q together. The result is a polynomial of the sum of
// the two degrees of p and q. NOTE: it does not check for null coefficients
// after the multiplication, so the degree of the polynomial is "always" as
// described above. This is only to use in secret sharing schemes, and is not to
// be considered a general polynomial manipulation routine.
func (p *PriPoly) Mul(q *PriPoly) *PriPoly {
	d1 := len(p.coeffs) - 1
	d2 := len(q.coeffs) - 1
	newDegree := d1 + d2
	coeffs := make([]abstract.Scalar, newDegree+1)
	for i := range coeffs {
		coeffs[i] = p.g.Scalar().Zero()
	}
	for i := range p.coeffs {
		for j := range q.coeffs {
			tmp := p.g.Scalar().Mul(p.coeffs[i], q.coeffs[j])
			coeffs[i+j] = tmp.Add(coeffs[i+j], tmp)
		}
	}
	return &PriPoly{p.g, coeffs}
}

// RecoverSecret reconstructs the shared secret p(0) from a list of private
// shares using Lagrange interpolation.
func RecoverSecret(g abstract.Group, shares []*PriShare, t, n int) (abstract.Scalar, error) {
	x := xScalar(g, shares, t, n)

	if len(x) < t {
		return nil, errors.New("share: not enough shares to recover secret")
	}

	acc := g.Scalar().Zero()
	num := g.Scalar()
	den := g.Scalar()
	tmp := g.Scalar()

	for i, xi := range x {
		num.Set(shares[i].V)
		den.One()
		for j, xj := range x {
			if i == j {
				continue
			}
			num.Mul(num, xj)
			den.Mul(den, tmp.Sub(xj, xi))
		}
		acc.Add(acc, num.Div(num, den))
	}

	return acc, nil
}

func xScalar(g abstract.Group, shares []*PriShare, t, n int) map[int]abstract.Scalar {
	x := make(map[int]abstract.Scalar)
	for i, s := range shares {
		if s == nil || s.V == nil || s.I < 0 || n <= s.I {
			continue
		}
		x[i] = g.Scalar().SetInt64(1 + int64(s.I))
		if len(x) == t {
			break
		}
	}
	return x
}

func xMinusConst(g abstract.Group, c abstract.Scalar) *PriPoly {
	neg := g.Scalar().Neg(c)
	return &PriPoly{
		g:      g,
		coeffs: []abstract.Scalar{neg, g.Scalar().One()},
	}
}

// RecoverPriPoly takes a list of shares and the parameters t and n to
// reconstruct the secret polynomial completely, i.e., all private coefficients.
// It is up to the caller to make sure there are enough shares to correctly
// re-construct the polynomial. There must be at least t shares.
func RecoverPriPoly(g abstract.Group, shares []*PriShare, t, n int) (*PriPoly, error) {
	x := xScalar(g, shares, t, n)
	if len(x) != t {
		return nil, errors.New("share: not enough shares to recove private polynomial")
	}

	var accPoly *PriPoly
	var err error
	den := g.Scalar()
	// notations following the wikipedia article on Lagrange interpolation
	// https://en.wikipedia.org/wiki/Lagrange_polynomial
	for j, xj := range x {
		var basis = &PriPoly{
			g:      g,
			coeffs: []abstract.Scalar{g.Scalar().One()},
		}
		var acc = g.Scalar().Set(shares[j].V)
		// compute lagrange basis l_j
		for m, xm := range x {
			if j == m {
				continue
			}
			basis = basis.Mul(xMinusConst(g, xm)) // basis = basis * (x - xm)

			den.Sub(xj, xm)   // den = xj - xm
			den.Inv(den)      // den = 1 / den
			acc.Mul(acc, den) // acc = acc * den
		}

		for i := range basis.coeffs {
			basis.coeffs[i] = basis.coeffs[i].Mul(basis.coeffs[i], acc)
		}

		if accPoly == nil {
			accPoly = basis
			continue
		}

		// add all L_j * y_j together
		accPoly, err = accPoly.Add(basis)
		if err != nil {
			return nil, err
		}
	}
	return accPoly, nil
}

func (p *PriPoly) String() string {
	var strs = make([]string, len(p.coeffs))
	for i, c := range p.coeffs {
		strs[i] = c.String()
	}
	return "[ " + strings.Join(strs, ", ") + " ]"
}

// PubShare represents a public share.
type PubShare struct {
	I int            // Index of the public share
	V abstract.Point // Value of the public share
}

// Hash computes the hash of the public share.
func (p *PubShare) Hash(s abstract.Suite) []byte {
	h := s.Hash()
	p.V.MarshalTo(h)
	binary.Write(h, binary.LittleEndian, p.I)
	return h.Sum(nil)
}

// PubPoly represents a public commitment polynomial to a secret sharing polynomial.
type PubPoly struct {
	g       abstract.Group   // Cryptographic group
	b       abstract.Point   // Base point, nil for standard base
	commits []abstract.Point // Commitments to coefficients of the secret sharing polynomial
}

// NewPubPoly creates a new public commitment polynomial.
func NewPubPoly(g abstract.Group, b abstract.Point, commits []abstract.Point) *PubPoly {
	return &PubPoly{g, b, commits}
}

// Info returns the base point and the commitments to the polynomial coefficients.
func (p *PubPoly) Info() (abstract.Point, []abstract.Point) {
	return p.b, p.commits
}

// Threshold returns the secret sharing threshold.
func (p *PubPoly) Threshold() int {
	return len(p.commits)
}

// Commit returns the secret commitment p(0), i.e., the constant term of the polynomial.
func (p *PubPoly) Commit() abstract.Point {
	return p.commits[0]
}

// Eval computes the public share v = p(i).
func (p *PubPoly) Eval(i int) *PubShare {
	xi := p.g.Scalar().SetInt64(1 + int64(i)) // x-coordinate of this share
	v := p.g.Point().Null()
	for j := p.Threshold() - 1; j >= 0; j-- {
		v.Mul(v, xi)
		v.Add(v, p.commits[j])
	}
	return &PubShare{i, v}
}

// Shares creates a list of n public commitment shares p(1),...,p(n).
func (p *PubPoly) Shares(n int) []*PubShare {
	shares := make([]*PubShare, n)
	for i := range shares {
		shares[i] = p.Eval(i)
	}
	return shares
}

// Add computes the component-wise sum of the polynomials p and q and returns it
// as a new polynomial. NOTE: If the base points p.b and q.b are different then the
// base point of the resulting PubPoly cannot be computed without knowing the
// discrete logarithm between p.b and q.b. In this particular case, we are using
// p.b as a default value which of course does not correspond to the correct
// base point and thus should not be used in further computations.
func (p *PubPoly) Add(q *PubPoly) (*PubPoly, error) {
	if p.g.String() != q.g.String() {
		return nil, errorGroups
	}

	if p.Threshold() != q.Threshold() {
		return nil, errorCoeffs
	}

	commits := make([]abstract.Point, p.Threshold())
	for i := range commits {
		commits[i] = p.g.Point().Add(p.commits[i], q.commits[i])
	}

	return &PubPoly{p.g, p.b, commits}, nil
}

// Equal checks equality of two public commitment polynomials p and q.
func (p *PubPoly) Equal(q *PubPoly) bool {
	if p.g.String() != q.g.String() {
		return false
	}
	b := 1
	for i := 0; i < p.Threshold(); i++ {
		pb, _ := p.commits[i].MarshalBinary()
		qb, _ := q.commits[i].MarshalBinary()
		b &= subtle.ConstantTimeCompare(pb, qb)
	}
	return b == 1
}

// Check a private share against a public commitment polynomial.
func (p *PubPoly) Check(s *PriShare) bool {
	pv := p.Eval(s.I)
	ps := p.g.Point().Mul(p.b, s.V)
	return pv.V.Equal(ps)
}

// RecoverCommit reconstructs the secret commitment p(0) from a list of public
// shares using Lagrange interpolation.
func RecoverCommit(g abstract.Group, shares []*PubShare, t, n int) (abstract.Point, error) {
	x := make(map[int]abstract.Scalar)
	for i, s := range shares {
		if s == nil || s.V == nil || s.I < 0 || n <= s.I {
			continue
		}
		x[i] = g.Scalar().SetInt64(1 + int64(s.I))
	}

	if len(x) < t {
		return nil, errors.New("not enough good public shares to reconstruct secret commitment")
	}

	num := g.Scalar()
	den := g.Scalar()
	tmp := g.Scalar()
	Acc := g.Point().Null()
	Tmp := g.Point()

	for i, xi := range x {
		num.One()
		den.One()
		for j, xj := range x {
			if i == j {
				continue
			}
			num.Mul(num, xj)
			den.Mul(den, tmp.Sub(xj, xi))
		}
		Tmp.Mul(shares[i].V, num.Div(num, den))
		Acc.Add(Acc, Tmp)
	}

	return Acc, nil
}
