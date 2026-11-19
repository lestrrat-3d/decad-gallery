package turbinefan

import (
	"fmt"
	"math"
	"math/big"
)

type bedCertificateChart [18]biRow
type bedCertificateBox [18]bladeInterval

func bedChartBounds(chart bedCertificateChart) (bedCertificateBox, bool) {
	var out bedCertificateBox
	for j, row := range chart {
		for k, p := range row {
			if p.w.lo <= 0 {
				return out, false
			}
			v := p.n.div(p.w)
			if k == 0 {
				out[j] = v
			} else {
				out[j] = bihull(out[j], v)
			}
		}
	}
	return out, true
}

// bedSectionRadius checks the planar curve (Y(t),Z0+h*t) on a constant-X section.
// Its absolute curvature is |h*Y”| / (h² + Y'²)^(3/2).
func bedSectionRadius(h, yp, ypp bladeInterval, minRadius float64) bool {
	speed := h.square().add(yp.square())
	if speed.lo <= 0 {
		return false
	}
	cubed := speed.mul(bladeInterval{bidown(math.Sqrt(speed.lo)), biup(math.Sqrt(speed.hi))})
	k := bi(minRadius).mul(h).mul(ypp)
	return k.lo > -cubed.lo && k.hi < cubed.lo
}
func bedSignedQuotient(n, d bladeInterval) bladeInterval {
	if d.lo > 0 {
		return n.div(d)
	}
	return bi(0).sub(n).div(bi(0).sub(d))
}

func bedCurvatureCell(q bedCertificateBox, h, bandX bladeInterval, u0, u1, t0, t1 float64, apex bool, minRadius float64) bool {
	t := bladeInterval{t0, t1}
	smooth := func(v bladeInterval) bladeInterval { return v.square().mul(bi(3).sub(bi(2).mul(v))) }
	s := bihull(smooth(bi(t0)), smooth(bi(t1)))
	x := biblend(q[0], q[3], s)
	if x.hi < bandX.lo || x.lo > bandX.hi {
		return true
	}
	derivative := func(v bladeInterval) bladeInterval { return bi(6).mul(v).mul(bi(1).sub(v)) }
	ds := bihull(derivative(bi(t0)), derivative(bi(t1)))
	if t0 <= .5 && t1 >= .5 {
		ds = bihull(ds, bi(1.5))
	}
	dd := bi(6).sub(bi(12).mul(t))
	xu, yu := biblend(q[1], q[4], s), biblend(q[9], q[12], s)
	xuu, yuu := biblend(q[2], q[5], s), biblend(q[10], q[13], s)
	xt, yt := q[6].mul(ds), q[14].mul(ds)
	xut, yut := q[7].mul(ds), q[15].mul(ds)
	xtt, ytt := q[6].mul(dd), q[14].mul(dd)

	// Near the circular tangent, the derivative of X with respect to the
	// outward chart coordinate v vanishes like v+t². Divide it out before
	// differentiating the constant-X condition. The two exact cancellations
	// used by A and E are checked while constructing the chart below.
	if apex && u1 <= .25 && t1 <= .25 {
		v := bladeInterval{u0, u1}
		a := q[16]
		b := bi(0).sub(q[4])
		e := q[17]
		g := bi(3).sub(bi(2).mul(t))
		all := true
		for i := 0; i < 32; i++ {
			// eta=v/(v+t²) and omega=1-eta decompose both the v and the t² terms,
			// including all directions approaching the singular endpoint.
			eta := bladeInterval{float64(i) / 32, float64(i+1) / 32}
			omega := bi(1).sub(eta)
			den := biblend(g.mul(b), bi(1).sub(s).mul(a), eta)
			if den.lo <= 0 {
				all = false
				break
			}
			l := bi(-6).mul(bi(1).sub(t)).mul(eta).mul(e).div(den)
			du := t.mul(l)
			n := dd.mul(eta).mul(e).add(bi(12).mul(bi(1).sub(t)).mul(omega).mul(b.sub(v.mul(a))).mul(l)).sub(xuu.mul(omega).mul(l.square()))
			ddu := bi(0).sub(n).div(den)
			yp := yt.add(yu.mul(du))
			ypp := ytt.add(bi(2).mul(yut).mul(du)).add(yuu.mul(du.square())).add(yu.mul(ddu))
			if !bedSectionRadius(h, yp, ypp, minRadius) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	if xu.lo <= 0 && xu.hi >= 0 {
		return false
	}
	du := bedSignedQuotient(bi(0).sub(xt), xu)
	ddu := bedSignedQuotient(bi(0).sub(xtt.add(bi(2).mul(xut).mul(du)).add(xuu.mul(du.square()))), xu)
	yp := yt.add(yu.mul(du))
	ypp := ytt.add(bi(2).mul(yut).mul(du)).add(yuu.mul(du.square())).add(yu.mul(ddu))
	return bedSectionRadius(h, yp, ypp, minRadius)
}

func visitBedChart(chart bedCertificateChart, q bedCertificateBox, h, x bladeInterval, u0, u1, t0, t1 float64, du, dt int, apex bool, minRadius float64, budget *int) bool {
	if *budget <= 0 {
		return false
	}
	*budget--
	if bedCurvatureCell(q, h, x, u0, u1, t0, t1, apex, minRadius) {
		return true
	}
	if du+dt >= 50 {
		return false
	}
	if du <= dt+2 {
		var a, b bedCertificateChart
		for j := range chart {
			a[j], b[j] = bihalves(chart[j])
		}
		qa, ok := bedChartBounds(a)
		if !ok {
			return false
		}
		qb, ok := bedChartBounds(b)
		if !ok {
			return false
		}
		mid := (u0 + u1) / 2
		return visitBedChart(a, qa, h, x, u0, mid, t0, t1, du+1, dt, apex, minRadius, budget) && visitBedChart(b, qb, h, x, mid, u1, t0, t1, du+1, dt, apex, minRadius, budget)
	}
	mid := (t0 + t1) / 2
	return visitBedChart(chart, q, h, x, u0, u1, t0, mid, du, dt+1, apex, minRadius, budget) && visitBedChart(chart, q, h, x, u0, u1, mid, t1, du, dt+1, apex, minRadius, budget)
}

// bladeNearBedCurvature covers each complete working patch and all fixed-X sections
// within 0.01 mm of the actual bed. The coplanar trailing faces are handled separately.
func bladeNearBedCurvature(p bladeCertificateRow, poleX, bedX, minRadius float64) error {
	if minRadius <= 0 || p.Linear || p.Z1 <= p.Z0 || p.Band < 3 || p.Face >= 4 {
		return fmt.Errorf("unexpected near-bed chart")
	}
	a, b, c, d, w := p.AX, p.AY, p.BX, p.BY, p.W
	apex := p.Band == 3 && (p.Chart == 3 || p.Chart == 22)
	if apex && p.Chart == 22 {
		// Use local v=1-u on the last working chart, so both singular endpoints are at v=0.
		reverse := func(p bladePolynomial) bladePolynomial {
			var r bladePolynomial
			for i := len(p) - 1; i >= 0; i-- {
				r = bpAdd(bpMul(r, bladePolynomial{bprInt(1), bprInt(-1)}), bladePolynomial{p[i]})
			}
			return r
		}
		a, b, c, d, w = reverse(a), reverse(b), reverse(c), reverse(d), reverse(w)
	}
	one := []*big.Rat{bprInt(1)}
	w2 := brpMul(w, w)
	w3 := brpMul(w2, w)
	var chart bedCertificateChart
	for axis, pair := range [][2][]*big.Rat{{a, c}, {b, d}} {
		left, right := pair[0], pair[1]
		fit, ok := brpQuo(right, w)
		if !ok {
			return fmt.Errorf("uncancelled fitted coordinate")
		}
		deriv := brpAdd(brpMul(brpDer(left), w), brpMul(left, brpDer(w)), -1)
		second := brpAdd(brpMul(brpDer(deriv), w), brpMul(brpMul(deriv, brpDer(w)), []*big.Rat{bprInt(2)}), -1)
		fp := brpDer(fit)
		numer := [][]*big.Rat{left, deriv, second, fit, fp, brpDer(fp), brpAdd(right, left, -1), brpAdd(brpMul(fp, w2), deriv, -1)}
		denom := [][]*big.Rat{w, w2, w3, one, one, one, w, w2}
		for i := range numer {
			r, err := birow(numer[i], denom[i])
			if err != nil {
				return err
			}
			chart[8*axis+i] = r
		}
		if axis == 0 && apex {
			delta := brpAdd(left, right, -1)
			if len(deriv) < 2 || len(delta) < 2 || deriv[0].Sign() != 0 || delta[0].Sign() != 0 {
				return fmt.Errorf("emerging edge lacks exact first-order zero")
			}
			var err error
			chart[16], err = birow(bpScaled(deriv[1:], bprInt(-1)), w2)
			if err != nil {
				return err
			}
			chart[17], err = birow(delta[1:], w)
			if err != nil {
				return err
			}
		}
	}
	if !apex {
		chart[16] = biRow{{bi(0), bi(1)}}
		chart[17] = chart[16]
	}
	q, ok := bedChartBounds(chart)
	if !ok {
		return fmt.Errorf("denominator interval not positive")
	}
	hi := bi(bedX).sub(bi(poleX))
	width := bi(1).div(bi(100))
	x := bladeInterval{hi.sub(width).lo, hi.hi}
	budget := 65536
	if !visitBedChart(chart, q, bi(p.Z1).sub(bi(p.Z0)), x, 0, 1, 0, 1, 0, 0, apex, minRadius, &budget) {
		return fmt.Errorf("continuous near-bed radius not proved on band %d chart %d", p.Band, p.Chart)
	}
	return nil
}
