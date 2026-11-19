package swisslever

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func arborTestBuilder(t *testing.T) *builder {
	t.Helper()
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	return newBuilder(t.Context(), NewDocument(), g)
}

// arborSection returns the section outline of body at height z. A faceted Boolean result refuses a chord below its
// own bound, so the cut retries coarser.
func arborSection(t *testing.T, body *decad.Body, z float64) decad.SectionOutline {
	t.Helper()
	f, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	var o decad.SectionOutline
	for _, tol := range []float64{0.005, 0.02, 0.05} {
		o, err = body.SectionOutline(t.Context(), f, units.Millimeters(tol))
		if err == nil {
			break
		}
	}
	require.NoError(t, err)
	return o
}

func arborLoop(l []r3.Vec) []Pt {
	out := make([]Pt, len(l))
	for i, v := range l {
		out[i] = Pt{v.X, v.Y}
	}
	return out
}

// arborInPoly reports whether x lies inside the closed polygon (even-odd).
func arborInPoly(poly []Pt, x Pt) bool {
	in := false
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		if (a.Y > x.Y) == (b.Y > x.Y) {
			continue
		}
		if x.X < a.X+(x.Y-a.Y)/(b.Y-a.Y)*(b.X-a.X) {
			in = !in
		}
	}
	return in
}

// arborInside reports whether the point is inside the solid, by the even-odd rule on the section at its height.
func arborInside(t *testing.T, body *decad.Body, x, y, z float64) bool {
	t.Helper()
	n := 0
	for _, l := range arborSection(t, body, z).Loops {
		if arborInPoly(arborLoop(l), Pt{x, y}) {
			n++
		}
	}
	return n%2 == 1
}

func arborVolume(t *testing.T, body *decad.Body) float64 {
	t.Helper()
	v, err := body.Volume()
	require.NoError(t, err)
	return v.Value.Base()
}

// segmentArea is the area of the circular segment of a circle of radius r beyond the chord x = flat.
func segmentArea(r, flat float64) float64 {
	if r <= flat {
		return 0
	}
	return r*r*math.Acos(flat/r) - flat*math.Sqrt(r*r-flat*flat)
}

// sectionRadius returns the radius of the meridian pts at height z.
func sectionRadius(pts meridianPoints, z float64) float64 {
	for i, p := range pts {
		q := pts[(i+1)%len(pts)]
		lo, hi := math.Min(p[1], q[1]), math.Max(p[1], q[1])
		if p[1] == q[1] || z <= lo || z >= hi || (p[0] == 0 && q[0] == 0) {
			continue
		}
		return p[0] + (q[0]-p[0])*(z-p[1])/(q[1]-p[1])
	}
	return 0
}

// flatPlaneAt returns the least flat plane the spans put at height z, and whether any covers z.
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

// meridianVolume integrates the meridian pts, less the circular segment each flat takes off a section.
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

// arborPlaneAt requires a planar face of body normal to Z at height z.
func arborPlaneAt(t *testing.T, body *decad.Body, z float64, what string) {
	t.Helper()
	for _, f := range body.Faces() {
		pl, ok := f.Surface().(decad.Plane)
		if ok && math.Abs(math.Abs(pl.Frame.N().Z)-1) < 1e-9 && math.Abs(pl.Frame.Origin().Z-z) < 1e-6 {
			return
		}
	}
	require.Failf(t, "missing face", "%s has no planar face at z %v", what, z)
}

// arborRadii returns the least and greatest distance from axis to the points of the outer loop of the section at z,
// and the section's OffSurface bound.
func arborRadii(t *testing.T, body *decad.Body, z float64, axis Pt) (float64, float64, float64) {
	t.Helper()
	o := arborSection(t, body, z)
	lo, hi, far := math.Inf(1), 0.0, -1.0
	for _, l := range o.Loops {
		pts := arborLoop(l)
		m, least := 0.0, math.Inf(1)
		for _, p := range pts {
			d := p.Sub(axis).Len()
			m, least = math.Max(m, d), math.Min(least, d)
		}
		if m > far {
			lo, hi, far = least, m, m
		}
	}
	return lo, hi, o.OffSurface.Base()
}

// arborCone is a cone of an arbor's meridian: its two end points (radius, z).
type arborCone struct{ a, b [2]float64 }

// gapTo returns the distance from the meridian point (r, z) to the cone's segment.
func (c arborCone) gapTo(r, z float64) float64 {
	a, b := Pt{c.a[0], c.a[1]}, Pt{c.b[0], c.b[1]}
	d := b.Sub(a)
	k := math.Max(0, math.Min(1, Pt{r, z}.Sub(a).Dot(d)/d.Dot(d)))
	return Pt{r, z}.Sub(a.Add(d.Scale(k))).Len()
}

// requireCone requires a face of body that is a cone of 30 degrees half angle through both end points of want.
func requireCone(t *testing.T, body *decad.Body, want arborCone, what string) {
	t.Helper()
	for _, f := range body.Faces() {
		c, ok := f.Surface().(decad.Cone)
		if !ok || math.Abs(c.HalfAngle.Base()-coneHalfAngle) > 1e-6 {
			continue
		}
		zAt := func(r float64) float64 { return c.Origin.Z + c.Axis.Z*(r-c.Radius.Base())/math.Tan(c.HalfAngle.Base()) }
		if math.Abs(zAt(want.a[0])-want.a[1]) < 1e-6 && math.Abs(zAt(want.b[0])-want.b[1]) < 1e-6 {
			return
		}
	}
	require.Failf(t, "missing cone", "%s has no 30 degree cone from (r %.4f, z %.4f) to (r %.4f, z %.4f)", what,
		want.a[0], want.a[1], want.b[0], want.b[1])
}

// minCollar is the shortest collar; minThrustLand the narrowest ring of a step on the inner ring's flat face beyond its
// bore chamfer, which reaches radius 3.0 at worst (docs/model-pipeline.md Arbor collars).
const (
	minCollar        = 3.0
	boreChamferReach = 3.0
	minThrustLand    = 0.3
)

// requireConeCollar requires a collar of the shaft's own body from its end face at far to the end of its cone at near:
// the end face and the cone are on the arbor, the whole length is at least 3.0 mm, and a round section of radius 3.7
// stands at z round.
func requireConeCollar(t *testing.T, arbor *decad.Body, axis Pt, far, near, round float64, cone arborCone, what string) {
	t.Helper()
	arborPlaneAt(t, arbor, far, what+": end face")
	requireCone(t, arbor, cone, what)
	require.GreaterOrEqual(t, math.Abs(far-near), minCollar-1e-9, "%s: whole length to the cone's end", what)
	lo, hi, off := arborRadii(t, arbor, round, axis)
	require.InDelta(t, CollarRadius, lo, 0.01+off, "%s: radius", what)
	require.InDelta(t, CollarRadius, hi, 0.01+off, "%s: radius", what)
}

// requireBodyStep requires a body step of the arbor at z: a planar face there, a round section of radius 3.3 beside it
// with its flat clear of the seat, the land beyond the bore chamfer, and the carried part's face at least 1.0 mm from
// the flush side, which stands flushSide.
func requireBodyStep(t *testing.T, arbor *decad.Body, axis Pt, z, mid, partFace, flushSide float64, what string) {
	t.Helper()
	arborPlaneAt(t, arbor, z, what+": step face")
	lo, hi, off := arborRadii(t, arbor, mid, axis)
	require.InDelta(t, bodyRadius, hi, 0.01+off, "%s: round radius", what)
	require.Greater(t, lo, ArborRadius, "%s: the flat stays clear of the seat", what)
	require.GreaterOrEqual(t, bodyRadius-boreChamferReach, minThrustLand-1e-9, "%s: land beyond the bore chamfer", what)
	require.GreaterOrEqual(t, math.Abs(partFace-flushSide), 1.0, "%s: carried part's face to the flush side", what)
}

func arborCollarCone(pts meridianPoints, i int) arborCone { return arborCone{pts[i], pts[i+1]} }
