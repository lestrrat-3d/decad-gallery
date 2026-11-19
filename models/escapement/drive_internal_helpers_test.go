package escapement

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// sectionAt returns the section outline of body at height z.
func sectionAt(t *testing.T, body *decad.Body, z float64) decad.SectionOutline {
	t.Helper()
	f, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	o, err := outlineAt(t, body, f)
	require.NoError(t, err)
	return o
}

// outlineAt cuts body with the plane f at the finest chord tolerance the body allows. A faceted Boolean result
// refuses a chord below its own bound, so the cut retries coarser.
func outlineAt(t *testing.T, body *decad.Body, f r3.Frame) (decad.SectionOutline, error) {
	t.Helper()
	var o decad.SectionOutline
	var err error
	for _, tol := range []float64{0.02, 0.03, 0.05} {
		o, err = body.SectionOutline(t.Context(), f, units.Millimeters(tol))
		if err == nil {
			break
		}
	}
	return o, err
}

// loopsBy returns the loops of o as XY polygons, largest extent first.
func loopsBy(o decad.SectionOutline) [][]Pt {
	var out [][]Pt
	for _, l := range o.Loops {
		out = append(out, loopPoints(l))
	}
	extent := func(p []Pt) float64 {
		m := 0.0
		for _, v := range p {
			m = math.Max(m, v.Len())
		}
		return m
	}
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if extent(out[j]) > extent(out[i]) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func rotPoly(p []Pt, a float64) []Pt {
	out := make([]Pt, len(p))
	for i, v := range p {
		out[i] = v.Rot(a)
	}
	return out
}

// fitsIn reports whether polygon p lies inside polygon hole with no crossing.
func fitsIn(p, hole []Pt) bool {
	for _, v := range p {
		if !inPoly(hole, v) {
			return false
		}
	}
	for i := range p {
		for j := range hole {
			if segCross(p[i], p[(i+1)%len(p)], hole[j], hole[(j+1)%len(hole)]) {
				return false
			}
		}
	}
	return true
}
