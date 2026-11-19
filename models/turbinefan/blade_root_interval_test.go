package turbinefan

import (
	"fmt"
	"math"
	"math/big"
)

type bladeProofPatch struct {
	band, face        int
	z0, z1            float64
	ax, ay, bx, by, w []*big.Rat
	linear            bool
}

type bladeInterval struct{ lo, hi float64 }

func bi(x float64) bladeInterval { return bladeInterval{x, x} }
func bidown(x float64) float64   { return math.Nextafter(x, math.Inf(-1)) }
func biup(x float64) float64     { return math.Nextafter(x, math.Inf(1)) }
func (a bladeInterval) add(b bladeInterval) bladeInterval {
	return bladeInterval{bidown(a.lo + b.lo), biup(a.hi + b.hi)}
}
func (a bladeInterval) sub(b bladeInterval) bladeInterval {
	return bladeInterval{bidown(a.lo - b.hi), biup(a.hi - b.lo)}
}
func (a bladeInterval) mul(b bladeInterval) bladeInterval {
	x := [4]float64{a.lo * b.lo, a.lo * b.hi, a.hi * b.lo, a.hi * b.hi}
	return bladeInterval{bidown(math.Min(math.Min(x[0], x[1]), math.Min(x[2], x[3]))), biup(math.Max(math.Max(x[0], x[1]), math.Max(x[2], x[3])))}
}
func (a bladeInterval) div(b bladeInterval) bladeInterval {
	return a.mul(bladeInterval{bidown(1 / b.hi), biup(1 / b.lo)})
}
func (a bladeInterval) square() bladeInterval {
	if a.lo >= 0 {
		return bladeInterval{bidown(a.lo * a.lo), biup(a.hi * a.hi)}
	}
	if a.hi <= 0 {
		return bladeInterval{bidown(a.hi * a.hi), biup(a.lo * a.lo)}
	}
	return bladeInterval{0, biup(math.Max(a.lo*a.lo, a.hi*a.hi))}
}
func bihull(a, b bladeInterval) bladeInterval {
	return bladeInterval{math.Min(a.lo, b.lo), math.Max(a.hi, b.hi)}
}
func biblend(a, b, s bladeInterval) bladeInterval {
	f := func(x float64) bladeInterval { return bi(1).sub(bi(x)).mul(a).add(bi(x).mul(b)) }
	return bihull(f(s.lo), f(s.hi))
}

type bivector [3]bladeInterval

func bidot(a, b bivector) bladeInterval {
	return a[0].mul(b[0]).add(a[1].mul(b[1])).add(a[2].mul(b[2]))
}
func binorm(a bivector) bladeInterval { return a[0].square().add(a[1].square()).add(a[2].square()) }
func bicross(a, b bivector) bivector {
	return bivector{a[1].mul(b[2]).sub(a[2].mul(b[1])), a[2].mul(b[0]).sub(a[0].mul(b[2])), a[0].mul(b[1]).sub(a[1].mul(b[0]))}
}

type biCoeff struct{ n, w bladeInterval }
type biRow []biCoeff
type biChart [12]biRow

func bihalves(in biRow) (biRow, biRow) {
	p := append(biRow(nil), in...)
	a, b := make(biRow, 0, len(p)), make(biRow, 0, len(p))
	for len(p) > 0 {
		a = append(a, p[0])
		b = append(b, p[len(p)-1])
		for i := 1; i < len(p); i++ {
			p[i-1] = biCoeff{p[i-1].n.add(p[i].n).mul(bi(.5)), p[i-1].w.add(p[i].w).mul(bi(.5))}
		}
		p = p[:len(p)-1]
	}
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return a, b
}
func bilocal(q [12]bladeInterval, h bladeInterval, radius float64, t0, t1 float64, depth int) bool {
	t := bladeInterval{t0, t1}
	smooth := func(v bladeInterval) bladeInterval { return v.square().mul(bi(3).sub(bi(2).mul(v))) }
	s := bihull(smooth(bi(t0)), smooth(bi(t1)))
	derivative := func(v bladeInterval) bladeInterval { return bi(6).mul(v).mul(bi(1).sub(v)) }
	ds := bihull(derivative(bi(t0)), derivative(bi(t1)))
	if t0 <= .5 && .5 <= t1 {
		ds = bihull(ds, bi(1.5))
	}
	dd := bi(6).sub(bi(12).mul(t))
	u := bivector{biblend(q[0], q[2], s), biblend(q[6], q[8], s), bi(0)}
	v := bivector{q[4].mul(ds), q[10].mul(ds), h}
	uu := bivector{biblend(q[1], q[3], s), biblend(q[7], q[9], s), bi(0)}
	uv := bivector{q[5].mul(ds), q[11].mul(ds), bi(0)}
	vv := bivector{q[4].mul(dd), q[10].mul(dd), bi(0)}
	n := bicross(u, v)
	n2 := binorm(n)
	if n2.lo > 0 {
		length := bladeInterval{bidown(math.Sqrt(n2.lo)), biup(math.Sqrt(n2.hi))}
		k := bi(radius).div(length)
		a := binorm(u).sub(k.mul(bidot(n, uu)))
		b := bidot(u, v).sub(k.mul(bidot(n, uv)))
		c := binorm(v).sub(k.mul(bidot(n, vv)))
		if a.lo > 0 && c.lo > 0 && a.mul(c).sub(b.square()).lo > 0 {
			return true
		}
	}
	if depth == 8 {
		return false
	}
	mid := (t0 + t1) / 2
	return bilocal(q, h, radius, t0, mid, depth+1) && bilocal(q, h, radius, mid, t1, depth+1)
}
func bicheck(c biChart, h bladeInterval, radius float64, depth int) bool {
	var q [12]bladeInterval
	for k, row := range c {
		for i, v := range row {
			if v.w.lo <= 0 {
				return false
			}
			x := v.n.div(v.w)
			if i == 0 {
				q[k] = x
			} else {
				q[k] = bihull(q[k], x)
			}
		}
	}
	if bilocal(q, h, radius, 0, 1, 0) {
		return true
	}
	if depth == 10 {
		return false
	}
	var a, b biChart
	for k := range c {
		a[k], b[k] = bihalves(c[k])
	}
	return bicheck(a, h, radius, depth+1) && bicheck(b, h, radius, depth+1)
}
func brpTrim(a []*big.Rat) []*big.Rat {
	for len(a) > 1 && a[len(a)-1].Sign() == 0 {
		a = a[:len(a)-1]
	}
	return a
}
func brpAdd(a, b []*big.Rat, sign int64) []*big.Rat {
	n := max(len(a), len(b))
	out := make([]*big.Rat, n)
	for i := range out {
		out[i] = new(big.Rat)
		if i < len(a) {
			out[i].Set(a[i])
		}
		if i < len(b) {
			out[i].Add(out[i], new(big.Rat).Mul(b[i], big.NewRat(sign, 1)))
		}
	}
	return brpTrim(out)
}
func brpMul(a, b []*big.Rat) []*big.Rat {
	out := make([]*big.Rat, len(a)+len(b)-1)
	for i := range out {
		out[i] = new(big.Rat)
	}
	for i, x := range a {
		for j, y := range b {
			out[i+j].Add(out[i+j], new(big.Rat).Mul(x, y))
		}
	}
	return brpTrim(out)
}
func brpDer(a []*big.Rat) []*big.Rat {
	if len(a) < 2 {
		return []*big.Rat{new(big.Rat)}
	}
	out := make([]*big.Rat, len(a)-1)
	for i := 1; i < len(a); i++ {
		out[i-1] = new(big.Rat).Mul(a[i], big.NewRat(int64(i), 1))
	}
	return brpTrim(out)
}
func brpQuo(a, b []*big.Rat) ([]*big.Rat, bool) {
	rem := make([]*big.Rat, len(a))
	for i := range a {
		rem[i] = new(big.Rat).Set(a[i])
	}
	rem = brpTrim(rem)
	n := max(1, len(rem)-len(b)+1)
	out := make([]*big.Rat, n)
	for i := range out {
		out[i] = new(big.Rat)
	}
	for len(rem) >= len(b) && (len(rem) != 1 || rem[0].Sign() != 0) {
		k := len(rem) - len(b)
		v := new(big.Rat).Quo(rem[len(rem)-1], b[len(b)-1])
		out[k].Set(v)
		for j, x := range b {
			rem[k+j].Sub(rem[k+j], new(big.Rat).Mul(v, x))
		}
		rem = brpTrim(rem)
	}
	return brpTrim(out), len(rem) == 1 && rem[0].Sign() == 0
}
func brpBernstein(p []*big.Rat, n int) []*big.Rat {
	out := make([]*big.Rat, n+1)
	for i := range out {
		out[i] = new(big.Rat)
		for j := 0; j < len(p) && j <= i; j++ {
			ratio := new(big.Rat).SetFrac(new(big.Int).Binomial(int64(i), int64(j)), new(big.Int).Binomial(int64(n), int64(j)))
			out[i].Add(out[i], new(big.Rat).Mul(p[j], ratio))
		}
	}
	return out
}
func birow(p, q []*big.Rat) (biRow, error) {
	n := max(len(p), len(q)) - 1
	pn, qn := brpBernstein(p, n), brpBernstein(q, n)
	row := make(biRow, n+1)
	for i := range row {
		if qn[i].Sign() <= 0 {
			return nil, fmt.Errorf("nonpositive denominator coefficient %d", i)
		}
		x, _ := pn[i].Float64()
		w, _ := qn[i].Float64()
		row[i] = biCoeff{bladeInterval{bidown(x), biup(x)}, bladeInterval{bidown(w), biup(w)}}
	}
	return row, nil
}
func bladeRootCurvature(p bladeProofPatch, radius float64) error {
	if p.linear || radius <= 0 || p.z1 <= p.z0 {
		return fmt.Errorf("invalid root patch")
	}
	w2, w3 := brpMul(p.w, p.w), brpMul(brpMul(p.w, p.w), p.w)
	var chart biChart
	for axis, pair := range [][2][]*big.Rat{{p.ax, p.bx}, {p.ay, p.by}} {
		a, b := pair[0], pair[1]
		poly, ok := brpQuo(b, p.w)
		if !ok {
			return fmt.Errorf("fitted denominator does not cancel")
		}
		ap := brpAdd(brpMul(brpDer(a), p.w), brpMul(a, brpDer(p.w)), -1)
		app := brpAdd(brpMul(brpDer(ap), p.w), brpMul(brpMul(ap, brpDer(p.w)), []*big.Rat{big.NewRat(2, 1)}), -1)
		p1, p2 := brpDer(poly), brpDer(brpDer(poly))
		numer := [][]*big.Rat{ap, app, p1, p2, brpAdd(b, a, -1), brpAdd(brpMul(p1, w2), ap, -1)}
		denom := [][]*big.Rat{w2, w3, {big.NewRat(1, 1)}, {big.NewRat(1, 1)}, p.w, w2}
		for j := range numer {
			var err error
			chart[axis*6+j], err = birow(numer[j], denom[j])
			if err != nil {
				return err
			}
		}
	}
	if !bicheck(chart, bi(p.z1).sub(bi(p.z0)), radius, 0) {
		return fmt.Errorf("root patch band %d face %d: curvature not proved", p.band, p.face)
	}
	return nil
}
