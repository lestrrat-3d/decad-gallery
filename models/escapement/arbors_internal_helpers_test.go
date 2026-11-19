package escapement

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func testBuilder(t *testing.T) *builder {
	t.Helper()
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	return newBuilder(t.Context(), NewDocument(), g)
}

// inside reports whether the point is inside the solid. It cuts the body with the horizontal plane through the point
// and tests the point against the section's polygons (even-odd), which works for every body, a chamfered Boolean
// result included.
func inside(t *testing.T, body *decad.Body, x, y, z float64) bool {
	t.Helper()
	f, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	// A faceted Boolean result refuses a chord below its own bound, so retry coarser.
	var o decad.SectionOutline
	for _, tol := range []float64{0.005, 0.02, 0.05} {
		o, err = body.SectionOutline(context.Background(), f, units.Millimeters(tol))
		if err == nil {
			break
		}
	}
	require.NoError(t, err)
	n := 0
	for _, l := range o.Loops {
		if inPoly(loopPoints(l), Pt{x, y}) {
			n++
		}
	}
	return n%2 == 1
}

func volume(t *testing.T, body *decad.Body) float64 {
	t.Helper()
	v, err := body.Volume()
	require.NoError(t, err)
	return v.Value.Base()
}

// segmentArea is the area of the circular segment of a circle of radius r beyond the chord x = flat; zero when the
// flat lies outside the circle.
func segmentArea(r, flat float64) float64 {
	if r <= flat {
		return 0
	}
	return r*r*math.Acos(flat/r) - flat*math.Sqrt(r*r-flat*flat)
}

// sectionRadius returns the radius of the meridian pts at height z, along a cylinder or a cone.
func sectionRadius(pts meridianPoints, z float64) float64 {
	for i, p := range pts {
		q := pts[(i+1)%len(pts)]
		lo, hi := math.Min(p[1], q[1]), math.Max(p[1], q[1])
		if p[1] == q[1] || z <= lo || z >= hi || math.Min(p[0], q[0]) < 0 || (p[0] == 0 && q[0] == 0) {
			continue
		}
		return p[0] + (q[0]-p[0])*(z-p[1])/(q[1]-p[1])
	}
	return 0
}

// flatPlaneAt returns the least flat plane the spans put at height z, and whether any covers z. A ramp past the end of
// a span moves the plane out at 30 degrees to the axis.
func flatPlaneAt(spans []flatSpan, z float64) (float64, bool) {
	best, found := math.Inf(1), false
	for _, sp := range spans {
		x := math.Inf(1)
		switch {
		case z >= sp.Z0 && z <= sp.Z1:
			x = sp.X
		case sp.RampEnd && z > sp.Z1:
			x = sp.X + (z-sp.Z1)*math.Tan(coneHalfAngle)
		case sp.RampStart && z < sp.Z0:
			x = sp.X + (sp.Z0-z)*math.Tan(coneHalfAngle)
		}
		if x < best {
			best, found = x, true
		}
	}
	return best, found
}

// meridianVolume integrates the meridian pts, less the circular segment each flat takes off a section, over z in
// steps of 1 µm.
func meridianVolume(pts meridianPoints, spans []flatSpan) float64 {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, p := range pts {
		lo, hi = math.Min(lo, p[1]), math.Max(hi, p[1])
	}
	const step = 0.001
	sum := 0.0
	for z := lo + step/2; z < hi; z += step {
		r := sectionRadius(pts, z)
		area := math.Pi * r * r
		if x, ok := flatPlaneAt(spans, z); ok {
			area -= segmentArea(r, x)
		}
		sum += area * step
	}
	return sum
}

// loopPoints returns the XY points of a section loop.
func loopPoints(l []r3.Vec) []Pt {
	out := make([]Pt, len(l))
	for i, v := range l {
		out[i] = Pt{v.X, v.Y}
	}
	return out
}

// outerLoop returns the XY points of the loop that reaches farthest from the origin.
func outerLoop(loops [][]r3.Vec) []Pt {
	var best []Pt
	far := -1.0
	for _, l := range loops {
		pts := loopPoints(l)
		m := 0.0
		for _, p := range pts {
			m = math.Max(m, p.Len())
		}
		if m > far {
			best, far = pts, m
		}
	}
	return best
}
