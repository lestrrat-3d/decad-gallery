package escapement

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// minPressWall is the thinnest wall round a press-fit bore: docs/model-pipeline.md Corners and roots.
const minPressWall = 1.6

// minCollar is the shortest collar: instructions.md "Arbor collars", docs/model-pipeline.md Arbor collars.
const minCollar = 3.0

// sectionRadii returns the least and greatest distance from axis to the points of the section of body at z, over the
// loop that reaches farthest from the axis, and the section's OffSurface bound.
func sectionRadii(t *testing.T, body *decad.Body, z float64, axis Pt) (float64, float64, float64) {
	t.Helper()
	o := sectionAt(t, body, z)
	loop := outerLoop2(o.Loops, axis)
	lo, hi := math.Inf(1), 0.0
	for _, p := range loop {
		lo, hi = math.Min(lo, p.Len()), math.Max(hi, p.Len())
	}
	return lo, hi, o.OffSurface.Base()
}

// arborCone is a cone of an arbor's meridian: its two end points (radius, z).
type arborCone struct{ a, b [2]float64 }

// coneBetween returns the cone of pts that joins the points at index i and i+1.
func coneBetween(pts meridianPoints, i int) arborCone { return arborCone{pts[i], pts[i+1]} }

// gapTo returns the normal distance from the meridian point (r, z) to the cone, measured to the cone's segment.
func (c arborCone) gapTo(r, z float64) float64 {
	return distPointSeg(Pt{r, z}, Pt{c.a[0], c.a[1]}, Pt{c.b[0], c.b[1]})
}

// coneFaces returns the cone faces of body with a half angle of 30 degrees.
func coneFaces(t *testing.T, body *decad.Body) []decad.Cone {
	t.Helper()
	var out []decad.Cone
	for _, f := range body.Faces() {
		c, ok := f.Surface().(decad.Cone)
		if !ok {
			continue
		}
		// A chamfer is a cone too, at 45 degrees; only the 30 degree cones are the arbor's and its mates'.
		if math.Abs(c.HalfAngle.Base()-math.Pi/6) < 1e-6 {
			out = append(out, c)
		}
	}
	return out
}

// coneZ returns the z at radius r on the cone surface c.
func coneZ(c decad.Cone, r float64) float64 {
	return c.Origin.Z + c.Axis.Z*(r-c.Radius.Base())/math.Tan(c.HalfAngle.Base())
}

// requireCone requires a cone face on body, of 30 degrees half angle, that passes through both end points of want.
func requireCone(t *testing.T, body *decad.Body, want arborCone, what string) {
	t.Helper()
	for _, c := range coneFaces(t, body) {
		if math.Abs(coneZ(c, want.a[0])-want.a[1]) < 1e-6 && math.Abs(coneZ(c, want.b[0])-want.b[1]) < 1e-6 {
			return
		}
	}
	require.Failf(t, "missing cone", "%s has no 30 degree cone from (r %.4f, z %.4f) to (r %.4f, z %.4f)", what,
		want.a[0], want.a[1], want.b[0], want.b[1])
}

// coneFloorGap returns the least normal distance from the cone face of body that passes the cone c's line, sampled
// over radii lo to hi, to the segment of c.
func coneFloorGap(t *testing.T, body *decad.Body, c arborCone, lo, hi float64) float64 {
	t.Helper()
	least := math.Inf(1)
	for _, f := range coneFaces(t, body) {
		for i := 0; i <= 100; i++ {
			r := lo + (hi-lo)*float64(i)/100
			least = math.Min(least, c.gapTo(r, coneZ(f, r)))
		}
	}
	require.False(t, math.IsInf(least, 1), "the body has a cone face")
	return least
}

// requireConeCollar requires a collar of the shaft's own body from its end face at far to where its cone meets the
// smaller diameter at near: the end face and the cone are on the arbor, the whole length is at least 3.0 mm, and a
// round section of radius 3.7 at z round.
func requireConeCollar(t *testing.T, arbor *decad.Body, axis Pt, far, near, round float64, cone arborCone, what string) {
	t.Helper()
	hasPlane(t, arbor, far, what+": end face")
	requireCone(t, arbor, cone, what)
	require.GreaterOrEqual(t, math.Abs(far-near), minCollar-1e-9, "%s: whole length to the cone's end", what)
	lo, hi, off := sectionRadii(t, arbor, round, axis)
	require.InDelta(t, CollarRadius, lo, 0.01+off, "%s: radius", what)
	require.InDelta(t, CollarRadius, hi, 0.01+off, "%s: radius", what)
}

// boreChamferReach is the radius where a 605ZZ inner ring's bore chamfer ends on its face. The repository holds no
// chamfer figure, so this is the design's worst case: a chamfer that reaches radius 3.0 (0.5 mm on the 2.5 mm bore).
const boreChamferReach = 3.0

// minThrustLand is the narrowest ring of an arbor step that must reach the inner ring's flat face beyond the chamfer.
const minThrustLand = 0.3

// requireBodyStep requires a body step of the arbor at z: a planar face there, a round section of radius 3.3 beside it, a
// land of at least minThrustLand on the inner ring's face beyond the bore chamfer, and the carried part's face at least
// 1.0 mm from the flush side.
func requireBodyStep(t *testing.T, arbor *decad.Body, axis Pt, z, mid, partFace float64, what string) {
	t.Helper()
	hasPlane(t, arbor, z, what+": step face")
	lo, hi, off := sectionRadii(t, arbor, mid, axis)
	require.InDelta(t, bodyRadius, hi, 0.01+off, "%s: round radius", what)
	require.Greater(t, lo, ArborRadius, "%s: the flat stays clear of the seat", what)
	require.GreaterOrEqual(t, bodyRadius-boreChamferReach, minThrustLand-1e-9, "%s: land beyond the bore chamfer", what)
	require.GreaterOrEqual(t, math.Abs(partFace-z), 1.0, "%s: carried part's face to the flush side", what)
}
