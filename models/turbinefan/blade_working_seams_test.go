package turbinefan

import (
	"fmt"
	"math/big"
)

func bladePolyEndpoint(p []*big.Rat, end bool) *big.Rat {
	x := new(big.Rat)
	for i, q := range p {
		if !end && i > 0 {
			break
		}
		x.Add(x, q)
	}
	return x
}
func bladeEndpointJet(p, w []*big.Rat, end bool) (*big.Rat, *big.Rat) {
	n, den := bladePolyEndpoint(p, end), bladePolyEndpoint(w, end)
	dn, dw := bladePolyEndpoint(brpDer(p), end), bladePolyEndpoint(brpDer(w), end)
	return bprDiv(n, den), bprDiv(bprSub(bprMul(dn, den), bprMul(n, dw)), bprMul(den, den))
}
func bcCross(p, q bcPoint) *big.Rat { return bprSub(bprMul(p.x, q.y), bprMul(p.y, q.x)) }
func bcDot(p, q bcPoint) *big.Rat   { return bprAdd(bprMul(p.x, q.x), bprMul(p.y, q.y)) }
func bcZero(p bladePolynomial) bool {
	for _, x := range p {
		if x.Sign() != 0 {
			return false
		}
	}
	return true
}
func bcConstant(p, w bladePolynomial, expected *big.Rat) bool {
	return bcZero(brpAdd(p, bpScaled(w, expected), -1))
}

// bladeWorkingSeams checks each complete cyclic loop of patches. The determinant
// and scalar product of its two angular tangents are quadratic in the axial blend
// weight; their three exact Bernstein coefficients cover every intermediate height.
func bladeWorkingSeams(rows []bladeCertificateRow, poleX, bedX float64) error {
	bands := make(map[int]map[int]bladeCertificateRow)
	for _, r := range rows {
		if r.Band < 3 {
			continue
		}
		if bands[r.Band] == nil {
			bands[r.Band] = make(map[int]bladeCertificateRow)
		}
		if _, ok := bands[r.Band][r.Chart]; ok {
			return fmt.Errorf("duplicate chart %d/%d", r.Band, r.Chart)
		}
		bands[r.Band][r.Chart] = r
	}
	if len(bands) == 0 {
		return fmt.Errorf("no working bands")
	}
	tangentX := bprSub(bprFloat(bedX), bprFloat(poleX))
	for k := 3; k < 3+len(bands); k++ {
		ring := bands[k]
		if len(ring) != 23 {
			return fmt.Errorf("band %d has %d charts", k, len(ring))
		}
		for j := 0; j < 23; j++ {
			l, lok := ring[j]
			r, rok := ring[(j+1)%23]
			if !lok || !rok || l.Linear || r.Linear || l.Z0 != r.Z0 || l.Z1 != r.Z1 || l.Z0 >= l.Z1 {
				return fmt.Errorf("invalid band structure %d/%d", k, j)
			}
			if j < 3 {
				if !bcConstant(l.AX, l.W, tangentX) || !bcConstant(l.BX, l.W, tangentX) {
					return fmt.Errorf("flat chart %d/%d leaves bed plane", k, j)
				}
				if k == 3 && !bcConstant(l.AY, l.W, bprDiv(bladePolyEndpoint(l.AY, false), bladePolyEndpoint(l.W, false))) {
					return fmt.Errorf("flat source chart %d is not a point", j)
				}
			}
			leftX := [2][]*big.Rat{l.AX, l.BX}
			rightX := [2][]*big.Rat{r.AX, r.BX}
			leftY := [2][]*big.Rat{l.AY, l.BY}
			rightY := [2][]*big.Rat{r.AY, r.BY}
			var a, b [2]bcPoint
			for side := range 2 {
				lx, dx := bladeEndpointJet(leftX[side], l.W, true)
				ly, dy := bladeEndpointJet(leftY[side], l.W, true)
				rx, ex := bladeEndpointJet(rightX[side], r.W, false)
				ry, ey := bladeEndpointJet(rightY[side], r.W, false)
				if lx.Cmp(rx) != 0 || ly.Cmp(ry) != 0 {
					return fmt.Errorf("disjoint section endpoints band %d chart %d", k, j)
				}
				a[side], b[side] = bcPoint{dx, dy}, bcPoint{ex, ey}
			}
			// The middle coefficient is twice its Bernstein value; only its sign matters.
			quadratic := func(op func(bcPoint, bcPoint) *big.Rat) [3]int {
				return [3]int{op(a[0], b[0]).Sign(), bprAdd(op(a[0], b[1]), op(a[1], b[0])).Sign(), op(a[1], b[1]).Sign()}
			}
			cross, dot := quadratic(bcCross), quadratic(bcDot)
			if j == 2 || j == 22 {
				if k == 3 {
					if cross != [3]int{0, 0, 1} {
						return fmt.Errorf("root flat edge %d is not strictly outward for every positive height", j)
					}
				} else if cross != [3]int{1, 1, 1} {
					return fmt.Errorf("working flat edge %d/%d lacks strictly convex turn", k, j)
				}
			} else {
				wantDot := [3]int{1, 1, 1}
				if k == 3 && j < 2 {
					wantDot = [3]int{0, 0, 1}
				}
				if cross != [3]int{} || dot != wantDot {
					return fmt.Errorf("G1 join failed on band %d after chart %d", k, j)
				}
			}
		}
	}
	return nil
}
