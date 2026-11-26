package sevensegment

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// groovePinSegments are the sliders that take the separate groove pin.
func groovePinSegments() []Segment { return []Segment{SegA, SegG, SegD} }

// pinSection cuts the groove pin with the plane y = y and returns its loops as (x, z - ShaftZ). The pin's faceted wings hold
// a mesh bound of 0.086, so the section is read at 0.09.
func pinSection(t *testing.T, body *decad.Body, y float64) [][]r3.Vec {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{Y: y, Z: ShaftZ}, r3.Vec{X: 1}, r3.Vec{Z: 1})
	require.NoError(t, err)
	out, err := body.SectionOutline(t.Context(), f, units.Millimeters(0.09))
	require.NoError(t, err)
	loops := make([][]r3.Vec, len(out.Loops))
	for i, l := range out.Loops {
		for _, p := range l {
			loops[i] = append(loops[i], r3.Vec{X: p.X, Y: p.Z - ShaftZ})
		}
	}
	return loops
}

// largestRadius returns the largest distance of a loop's points from the axis at (0, OffRadius) in section coordinates.
func largestRadius(loops [][]r3.Vec) float64 {
	rad := 0.0
	for _, l := range loops {
		for _, p := range l {
			rad = math.Max(rad, math.Hypot(p.X, p.Y-OffRadius))
		}
	}
	return rad
}

// holeLoops splits a slider section into the keyed hole's loop, the one round the pin's axis, and the others.
func holeLoops(loops [][]r3.Vec) (hole, other [][]r3.Vec) {
	for _, l := range loops {
		if c, e := centroidExtent(l); math.Abs(c.X) < 0.5 && math.Abs(c.Y-OffRadius) < 0.5 && e < 8 {
			hole = append(hole, l)
		} else {
			other = append(other, l)
		}
	}
	return hole, other
}

// overhangOf reports the steepest lean from vertical, in degrees, of the downward faces of body above the bed (a vertex 0.02 mm
// or more above it) with up the print direction, and the area in mm2 of those that lean more than 45 degrees. Faces that start
// at the bed are skipped: the stem's 0.5 chamfers lie on it.
func overhangOf(t *testing.T, body *decad.Body, up r3.Vec) (worst, steepArea float64) {
	t.Helper()
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.035))
	require.NoError(t, err)
	vs := mesh.Vertices()
	bed := math.Inf(1)
	for _, v := range vs {
		bed = math.Min(bed, v.Dot(up))
	}
	for _, tri := range mesh.Triangles() {
		a, b, c := vs[tri[0]], vs[tri[1]], vs[tri[2]]
		if math.Min(a.Dot(up), math.Min(b.Dot(up), c.Dot(up)))-bed < 0.02 {
			continue
		}
		n := b.Sub(a).Cross(c.Sub(a))
		if n.Len() < 1e-12 {
			continue
		}
		nu := n.Scale(1 / n.Len()).Dot(up)
		if nu >= 0 {
			continue
		}
		lean := math.Asin(math.Min(1, -nu)) * 180 / math.Pi
		if lean > 45.01 {
			steepArea += n.Len() / 2
		}
		if n.Len()/2 >= 0.002 {
			worst = math.Max(worst, lean)
		}
	}
	return worst, steepArea
}
