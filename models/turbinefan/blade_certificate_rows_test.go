package turbinefan

import (
	"fmt"
	"math/big"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/sketch/geom"
)

// bladeCertificateRow represents the two exact planar curves at the ends of an axial band.
// Each coordinate numerator and the shared denominator are in ascending powers of local u.
type bladeCertificateRow struct {
	Band, Chart, Face int
	Z0, Z1            float64
	AX, AY, BX, BY, W []*big.Rat
	Linear            bool
}

type bladePolynomial []*big.Rat

func bprInt(n int64) *big.Rat       { return big.NewRat(n, 1) }
func bprAdd(a, b *big.Rat) *big.Rat { return new(big.Rat).Add(a, b) }
func bprSub(a, b *big.Rat) *big.Rat { return new(big.Rat).Sub(a, b) }
func bprMul(a, b *big.Rat) *big.Rat { return new(big.Rat).Mul(a, b) }
func bprDiv(a, b *big.Rat) *big.Rat { return new(big.Rat).Quo(a, b) }
func bprFloat(x float64) *big.Rat   { return new(big.Rat).SetFloat64(x) }
func bpTrim(a bladePolynomial) bladePolynomial {
	for len(a) > 0 && a[len(a)-1].Sign() == 0 {
		a = a[:len(a)-1]
	}
	return a
}
func bpAdd(a, b bladePolynomial) bladePolynomial {
	c := make(bladePolynomial, max(len(a), len(b)))
	for i := range c {
		c[i] = new(big.Rat)
		if i < len(a) {
			c[i].Add(c[i], a[i])
		}
		if i < len(b) {
			c[i].Add(c[i], b[i])
		}
	}
	return bpTrim(c)
}
func bpMul(a, b bladePolynomial) bladePolynomial {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	c := make(bladePolynomial, len(a)+len(b)-1)
	for i := range c {
		c[i] = new(big.Rat)
	}
	for i, x := range a {
		for j, y := range b {
			c[i+j].Add(c[i+j], bprMul(x, y))
		}
	}
	return bpTrim(c)
}
func bpScaled(p bladePolynomial, x *big.Rat) bladePolynomial { return bpMul(p, bladePolynomial{x}) }
func bpLerp(a, b, t bladePolynomial) bladePolynomial {
	return bpAdd(a, bpMul(bpAdd(b, bpScaled(a, bprInt(-1))), t))
}
func bpAffine(a, b *big.Rat) bladePolynomial { return bpTrim(bladePolynomial{a, bprSub(b, a)}) }

type bcPoint struct{ x, y *big.Rat }
type bcCurve struct{ x, y, w bladePolynomial }

func bcXY(p *sketch.Point) bcPoint { return bcPoint{bprFloat(p.X()), bprFloat(p.Y())} }
func bcSame(a, b bcPoint) bool     { return a.x.Cmp(b.x) == 0 && a.y.Cmp(b.y) == 0 }
func bcLine(a, b bcPoint) bcCurve {
	return bcCurve{bpAffine(a.x, b.x), bpAffine(a.y, b.y), bladePolynomial{bprInt(1)}}
}

// bcSpan applies exact de Boor recursion to a polynomial parameter. The span is already
// selected, including at its endpoints, so identifying cubic coefficients needs no sampling.
func bcSpan(control []*sketch.Point, knots []*big.Rat, i int, reversed bool) bcCurve {
	lo, hi := knots[i], knots[i+1]
	if reversed {
		lo, hi = hi, lo
	}
	t := bpAffine(lo, hi)
	var p [4]bcCurve
	for j := range p {
		v := bcXY(control[i-3+j])
		p[j] = bcCurve{bladePolynomial{v.x}, bladePolynomial{v.y}, nil}
	}
	for r := 1; r <= 3; r++ {
		for j := 3; j >= r; j-- {
			k := i - 3 + j
			den := bprSub(knots[k+4-r], knots[k])
			w := bpScaled(bpAdd(t, bladePolynomial{new(big.Rat).Neg(knots[k])}), bprDiv(bprInt(1), den))
			p[j].x = bpLerp(p[j-1].x, p[j].x, w)
			p[j].y = bpLerp(p[j-1].y, p[j].y, w)
		}
	}
	return bcCurve{p[3].x, p[3].y, bladePolynomial{bprInt(1)}}
}

type bcCircle struct {
	center bcPoint
	radius *big.Rat
	first  int
}

func bcCircleFrom(edges []sketch.BoundaryEdge) (bcCircle, error) {
	first := edges[0].Entity.(*sketch.Arc)
	c := bcXY(first.Center)
	e := bcXY(first.Start)
	dx, dy := bprSub(e.x, c.x), bprSub(e.y, c.y)
	r := bprAdd(new(big.Rat).Abs(dx), new(big.Rat).Abs(dy))
	cards := []bcPoint{{bprAdd(c.x, r), c.y}, {c.x, bprAdd(c.y, r)}, {bprSub(c.x, r), c.y}, {c.x, bprSub(c.y, r)}}
	seam := -1
	for i, p := range cards {
		if bcSame(p, e) {
			seam = i
		}
	}
	if r.Sign() <= 0 || seam < 0 {
		return bcCircle{}, fmt.Errorf("circle has no cardinal seam")
	}
	for i, edge := range edges {
		a, ok := edge.Entity.(*sketch.Arc)
		if !ok || edge.Reversed || !bcSame(bcXY(a.Center), c) || !bcSame(bcXY(a.Start), cards[(seam+i)%4]) || !bcSame(bcXY(a.End), cards[(seam+i+1)%4]) {
			return bcCircle{}, fmt.Errorf("invalid cardinal arc %d", i)
		}
	}
	return bcCircle{c, r, seam}, nil
}
func bcCircleCurve(c bcCircle, q int, t bladePolynomial) bcCurve {
	// The rational quarter circle has homogeneous coordinates (1-t²,2t,1+t²).
	one := bladePolynomial{bprInt(1)}
	tt := bpMul(t, t)
	w := bpAdd(one, tt)
	x := bpScaled(bpAdd(one, bpScaled(tt, bprInt(-1))), c.radius)
	y := bpScaled(t, bprMul(bprInt(2), c.radius))
	switch q {
	case 1:
		x, y = bpScaled(y, bprInt(-1)), x
	case 2:
		x, y = bpScaled(x, bprInt(-1)), bpScaled(y, bprInt(-1))
	case 3:
		x, y = y, bpScaled(x, bprInt(-1))
	}
	return bcCurve{bpAdd(x, bpScaled(w, c.center.x)), bpAdd(y, bpScaled(w, c.center.y)), w}
}

type bcFitted struct {
	curves []bcCurve
	knots  []*big.Rat
}

func bcFittedFrom(edges []sketch.BoundaryEdge) (bcFitted, error) {
	at := -1
	for i, e := range edges {
		if _, ok := e.Entity.(*sketch.Spline); ok {
			if at >= 0 {
				return bcFitted{}, fmt.Errorf("multiple control splines")
			}
			at = i
		}
	}
	if at < 0 {
		return bcFitted{}, fmt.Errorf("missing control spline")
	}
	e := edges[at]
	sp := e.Entity.(*sketch.Spline)
	n := len(sp.Control) - 3
	if n < 4 || n%4 != 0 {
		return bcFitted{}, fmt.Errorf("spline span count %d cannot form four equal groups", n)
	}
	curves := make([]bcCurve, 0, n+3)
	var lastY *big.Rat
	for j := 1; j <= 3; j++ {
		ed := edges[(at+j)%4]
		ln, ok := ed.Entity.(*sketch.Line)
		if !ok {
			return bcFitted{}, fmt.Errorf("unexpected flat edge %d", j)
		}
		a, b := bcXY(ln.Start), bcXY(ln.End)
		if ed.Reversed {
			a, b = b, a
		}
		if a.x.Cmp(b.x) != 0 || a.y.Cmp(b.y) >= 0 || (lastY != nil && a.y.Cmp(lastY) != 0) {
			return bcFitted{}, fmt.Errorf("flat edge %d is not ascending", j)
		}
		curves = append(curves, bcLine(a, b))
		lastY = b.y
	}
	raw := geom.ClampedKnots(len(sp.Control))
	kv := make([]*big.Rat, len(raw))
	for k, v := range raw {
		kv[k] = bprFloat(v)
	}
	ordered := make([]*big.Rat, n+1)
	for j := range ordered {
		k := j + 3
		if e.Reversed {
			k = n + 3 - j
			ordered[j] = bprSub(bprInt(1), kv[k])
		} else {
			ordered[j] = kv[k]
		}
	}
	for j := range n {
		k := j + 3
		if e.Reversed {
			k = n + 2 - j
		}
		curves = append(curves, bcSpan(sp.Control, kv, k, e.Reversed))
	}
	return bcFitted{curves, ordered}, nil
}

// bladeCertificateRows reads solved boundary entities. It accepts the gallery's explicit emerging-flat
// construction with four consecutive cardinal circles followed by a single cubic spline and three lines.
func bladeCertificateRows(sections []decad.LoftSection, corr decad.CircularLoftCorrespondence) ([]bladeCertificateRow, error) {
	if !corr.EmergingFlat || !corr.SmoothCircularBands || len(sections) < 5 {
		return nil, fmt.Errorf("expected a smooth circular prefix and emerging flat")
	}
	z := make([]float64, len(sections))
	var circles []bcCircle
	var fits []bcFitted
	var origin r3.Vec
	for i, sec := range sections {
		if sec.Sketch == nil || sec.Profile == nil || !sec.Profile.Valid || len(sec.Profile.Holes) != 0 || len(sec.Profile.Outer) != 4 {
			return nil, fmt.Errorf("invalid section %d", i)
		}
		f, err := sec.Sketch.Plane().Frame()
		if err != nil {
			return nil, err
		}
		if i == 0 {
			origin = f.Origin()
		}
		o := f.Origin()
		if f.U() != (r3.Vec{X: 1}) || f.V() != (r3.Vec{Y: 1}) || o.X != origin.X || o.Y != origin.Y || (i > 0 && o.Z <= z[i-1]) {
			return nil, fmt.Errorf("incompatible section frame %d", i)
		}
		z[i] = o.Z
		for _, e := range sec.Profile.Outer {
			if e.Partial || e.TStart != 0 || e.TEnd != 1 {
				return nil, fmt.Errorf("trimmed edge at section %d", i)
			}
		}
		if _, ok := sec.Profile.Outer[0].Entity.(*sketch.Arc); ok {
			if len(fits) > 0 {
				return nil, fmt.Errorf("circle after fitted section")
			}
			c, err := bcCircleFrom(sec.Profile.Outer)
			if err != nil {
				return nil, err
			}
			if i > 0 && (!bcSame(c.center, circles[0].center) || c.first != circles[0].first) {
				return nil, fmt.Errorf("circle center or seam changed")
			}
			circles = append(circles, c)
		} else {
			fit, err := bcFittedFrom(sec.Profile.Outer)
			if err != nil {
				return nil, err
			}
			fits = append(fits, fit)
		}
	}
	if len(circles) != 4 || len(fits) == 0 {
		return nil, fmt.Errorf("expected four circular stations before the foil")
	}
	first := fits[0]
	n := len(first.knots) - 1
	for _, fit := range fits {
		if len(fit.knots) != len(first.knots) {
			return nil, fmt.Errorf("different spline knot counts")
		}
		for i := range first.knots {
			if fit.knots[i].Cmp(first.knots[i]) != 0 {
				return nil, fmt.Errorf("different spline knot at %d", i)
			}
		}
	}
	for q := range 4 {
		nums, den := corr.Numerators[q], corr.Denominators[q]
		if den <= 0 || len(nums) < 2 || len(nums) > 6 || nums[0] != 0 || nums[len(nums)-1] != den {
			return nil, fmt.Errorf("invalid correspondence %d", q)
		}
		for k := 1; k < len(nums); k++ {
			if nums[k] <= nums[k-1] {
				return nil, fmt.Errorf("non-increasing correspondence %d", q)
			}
		}
	}
	var rows []bladeCertificateRow
	for chart := range first.curves {
		q := 0
		face := 4 + chart
		var t bladePolynomial
		if chart >= 3 {
			j := chart - 3
			per := n / 4
			q = j / per
			face = (q - circles[0].first + 4) % 4
			base := first.knots[q*per]
			width := bprSub(first.knots[(q+1)*per], base)
			u := bpAffine(bprDiv(bprSub(first.knots[j], base), width), bprDiv(bprSub(first.knots[j+1], base), width))
			mapLevels := make([]bladePolynomial, len(corr.Numerators[q]))
			for k, v := range corr.Numerators[q] {
				mapLevels[k] = bladePolynomial{big.NewRat(v, corr.Denominators[q])}
			}
			for level := len(mapLevels) - 1; level > 0; level-- {
				for k := 0; k < level; k++ {
					mapLevels[k] = bpLerp(mapLevels[k], mapLevels[k+1], u)
				}
			}
			t = mapLevels[0]
		}
		curves := make([]bcCurve, 0, len(sections))
		for _, c := range circles {
			curves = append(curves, bcCircleCurve(c, q, t))
		}
		for _, f := range fits {
			curves = append(curves, f.curves[chart])
		}
		for i := 0; i+1 < len(curves); i++ {
			a, b := curves[i], curves[i+1]
			rows = append(rows, bladeCertificateRow{i, chart, face, z[i], z[i+1], bpMul(a.x, b.w), bpMul(a.y, b.w), bpMul(b.x, a.w), bpMul(b.y, a.w), bpMul(a.w, b.w), i == 0})
		}
	}
	return rows, nil
}
