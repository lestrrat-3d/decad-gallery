package swisslever

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// Fit class minimums of docs/model-pipeline.md Part clearances.
const (
	tightFit   = 0.2
	regularFit = 0.35
	looseFit   = 0.5
)

// planeHeights returns the heights of the planar faces of body whose normal is along Z.
func planeHeights(body *decad.Body) []float64 {
	var out []float64
	for _, f := range body.Faces() {
		pl, ok := f.Surface().(decad.Plane)
		if ok && math.Abs(math.Abs(pl.Frame.N().Z)-1) < 1e-9 {
			out = append(out, pl.Frame.Origin().Z)
		}
	}
	return out
}

func hasPlane(t *testing.T, body *decad.Body, z float64, what string) {
	t.Helper()
	for _, h := range planeHeights(body) {
		if math.Abs(h-z) < 1e-6 {
			return
		}
	}
	require.Failf(t, "missing face", "%s has no planar face at z %v", what, z)
}

// axial checks a gap between a planar face of a at za and one of b at zb.
func axial(t *testing.T, a *decad.Body, za float64, b *decad.Body, zb, want float64, what string) {
	t.Helper()
	hasPlane(t, a, za, what)
	hasPlane(t, b, zb, what)
	require.InDelta(t, want, math.Abs(zb-za), 1e-9, what)
}

// zRange returns the lowest and highest point of body.
func zRange(t *testing.T, body *decad.Body) (float64, float64) {
	t.Helper()
	b, err := body.Bounds()
	require.NoError(t, err)
	return b.Min.Z, b.Max.Z
}

// radial requires the section gap between a and b at z: at least min (a class minimum), or, for a press or seat fit,
// want within the measurement bound when min is zero. The gap's lower bound carries the section's mesh bound, so a
// body that reads only Approximate still proves its class.
func radial(t *testing.T, a, b *decad.Body, z, want, min float64, what string) {
	t.Helper()
	oa, ob := partSection(t, a, z), partSection(t, b, z)
	c, err := oa.Clearance(ob)
	require.NoError(t, err, what)
	require.False(t, c.Overlaps, what)
	bound := c.Distance.Bound.Base()
	if min > 0 {
		require.GreaterOrEqual(t, c.Lower().Base(), min, "%s: gap %v ± %v", what, c.Distance.Value.Base(), bound)
		return
	}
	require.InDelta(t, want, c.Distance.Value.Base(), bound+0.01, "%s: gap ± %v", what, bound)
}

// seatRadii returns the least and greatest distance from axis to the points of the outer loop of the section of body
// at z, and the section's mesh bound.
func seatRadii(t *testing.T, body *decad.Body, z float64, axis Pt) (float64, float64, float64) {
	t.Helper()
	return arborRadii(t, body, z, axis)
}
