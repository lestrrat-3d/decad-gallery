package meshgap

import (
	"math"
	"math/big"

	"github.com/lestrrat-3d/r3"
)

type exactPoint [3]*big.Rat
type exactTriangle [3]exactPoint

// thinTriangle only selects candidates for the exact fallback. A mistaken
// selection cannot accept a pair: subdivision still proves separation.
func thinTriangle(t [3]r3.Vec) bool {
	a, b := t[1].Sub(t[0]), t[2].Sub(t[0])
	c := t[2].Sub(t[1])
	longest := max(a.Dot(a), b.Dot(b), c.Dot(c))
	n := a.Cross(b)
	areaSquared := n.Dot(n)
	return areaSquared > 0 && areaSquared <= 1e-24*longest*longest
}

// subdividedBoxGap proves a lower bound by bisecting the longest edge of
// either triangle and comparing exact rational boxes. Every child pair covers
// its parent pair. Work is capped; exhaustion returns no claim.
func subdividedBoxGap(a, b [3]r3.Vec, threshold float64) (float64, bool) {
	if !finite(threshold) || threshold < 0 {
		return 0, false
	}
	convert := func(p [3]r3.Vec) (exactTriangle, bool) {
		var out exactTriangle
		for i, point := range p {
			for j, value := range [3]float64{point.X, point.Y, point.Z} {
				if !finite(value) {
					return exactTriangle{}, false
				}
				out[i][j] = new(big.Rat).SetFloat64(value)
			}
		}
		return out, true
	}
	x, ok := convert(a)
	if !ok {
		return 0, false
	}
	y, ok := convert(b)
	if !ok {
		return 0, false
	}
	type pair struct{ a, b exactTriangle }
	pending := []pair{{x, y}}
	limit := new(big.Rat).SetFloat64(threshold)
	var least *big.Rat
	for visited := 0; len(pending) > 0; visited++ {
		if visited >= 128 {
			return 0, false
		}
		item := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		gap := exactTriangleBoxGap(item.a, item.b)
		if gap.Cmp(limit) > 0 {
			if least == nil || gap.Cmp(least) < 0 {
				least = gap
			}
			continue
		}
		aa, ia := exactLongestEdge(item.a)
		bb, ib := exactLongestEdge(item.b)
		if aa.Sign() == 0 && bb.Sign() == 0 {
			return 0, false
		}
		if aa.Cmp(bb) >= 0 {
			left, right := exactBisect(item.a, ia)
			pending = append(pending, pair{left, item.b}, pair{right, item.b})
		} else {
			left, right := exactBisect(item.b, ib)
			pending = append(pending, pair{item.a, left}, pair{item.a, right})
		}
	}
	if least == nil {
		return 0, false
	}
	lower, _ := least.Float64()
	if !finite(lower) || lower < 0 {
		return 0, false
	}
	if new(big.Rat).SetFloat64(lower).Cmp(least) > 0 {
		lower = math.Nextafter(lower, 0)
	}
	return lower, true
}

func exactTriangleBoxGap(a, b exactTriangle) *big.Rat {
	out := new(big.Rat)
	for k := range 3 {
		alo, ahi, blo, bhi := a[0][k], a[0][k], b[0][k], b[0][k]
		for i := 1; i < 3; i++ {
			if a[i][k].Cmp(alo) < 0 {
				alo = a[i][k]
			}
			if a[i][k].Cmp(ahi) > 0 {
				ahi = a[i][k]
			}
			if b[i][k].Cmp(blo) < 0 {
				blo = b[i][k]
			}
			if b[i][k].Cmp(bhi) > 0 {
				bhi = b[i][k]
			}
		}
		d := new(big.Rat)
		if alo.Cmp(bhi) > 0 {
			d.Sub(alo, bhi)
		} else if blo.Cmp(ahi) > 0 {
			d.Sub(blo, ahi)
		}
		out.Add(out, new(big.Rat).Mul(d, d))
	}
	return out
}

func exactLongestEdge(t exactTriangle) (*big.Rat, int) {
	best, index := new(big.Rat), 0
	for i := range 3 {
		squared := new(big.Rat)
		for k := range 3 {
			d := new(big.Rat).Sub(t[i][k], t[(i+1)%3][k])
			squared.Add(squared, new(big.Rat).Mul(d, d))
		}
		if squared.Cmp(best) > 0 {
			best, index = squared, i
		}
	}
	return best, index
}

func exactBisect(t exactTriangle, edge int) (exactTriangle, exactTriangle) {
	other := (edge + 1) % 3
	var mid exactPoint
	for k := range 3 {
		mid[k] = new(big.Rat).Quo(new(big.Rat).Add(t[edge][k], t[other][k]), big.NewRat(2, 1))
	}
	a, b := t, t
	a[edge], b[other] = mid, mid
	return a, b
}
