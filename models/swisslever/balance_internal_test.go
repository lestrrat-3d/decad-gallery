package swisslever

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// partBuilder returns a builder for the default movement, with its own document and sketch world.
func partBuilder(t *testing.T) *builder {
	t.Helper()
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	return newBuilder(t.Context(), NewDocument(), g)
}

// partSection returns the section outline of body at height z, retrying at coarser chords as a faceted Boolean result
// refuses a chord below its own bound.
func partSection(t *testing.T, body *decad.Body, z float64) decad.SectionOutline {
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

// partInside reports whether (x, y, z) is in the solid, by even-odd over the section's loops.
func partInside(t *testing.T, body *decad.Body, x, y, z float64) bool {
	t.Helper()
	n := 0
	for _, l := range partSection(t, body, z).Loops {
		if pointInLoop(l, x, y) {
			n++
		}
	}
	return n%2 == 1
}

func pointInLoop(l []r3.Vec, x, y float64) bool {
	in := false
	for i, a := range l {
		c := l[(i+1)%len(l)]
		if (a.Y > y) == (c.Y > y) {
			continue
		}
		if x < a.X+(y-a.Y)*(c.X-a.X)/(c.Y-a.Y) {
			in = !in
		}
	}
	return in
}

// requireBoundsZ requires the body to span z0 to z1 and to fit within radius about the axis.
func requireBoundsZ(t *testing.T, body *decad.Body, z0, z1, radius float64) {
	t.Helper()
	bb, err := body.Bounds()
	require.NoError(t, err)
	const slack = 3e-3
	require.InDelta(t, z0, bb.Min.Z, slack)
	require.InDelta(t, z1, bb.Max.Z, slack)
	require.InDelta(t, radius, math.Max(math.Max(-bb.Min.X, bb.Max.X), math.Max(-bb.Min.Y, bb.Max.Y)), slack)
}

// requireSketchesConstrained requires every sketch the builder recorded to verify fully constrained, with DOF 0.
func requireSketchesConstrained(t *testing.T, b *builder, want int) {
	t.Helper()
	require.Len(t, b.sketches, want)
	for _, s := range b.sketches {
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s))
	}
}

// requireCorners runs cornercheck on body in its print orientation with exemptions for the mating D corners.
func requireCorners(t *testing.T, body *decad.Body, exempt ...cornercheck.Exempt) {
	t.Helper()
	fs, err := cornercheck.Check(t.Context(), body, r3.Vec{Z: 1}, exempt)
	require.NoError(t, err)
	require.Empty(t, fs, cornercheck.Report(fs, 5))
}

// balanceBuilder returns a builder that leaves out the balance wheel's chamfer only while decad refuses it.
func balanceBuilder(t *testing.T) *builder {
	t.Helper()
	b := partBuilder(t)
	if configure := movementSwitches(t).configure(); configure != nil {
		configure(b)
	}
	return b
}

// TestBalanceDisc checks the rim, the four arms and the windows before the boss: every sketch fully constrained.
func TestBalanceDisc(t *testing.T) {
	b := partBuilder(t)
	disc, err := b.balanceDisc()
	require.NoError(t, err)
	require.True(t, disc.IsSolid())
	requireSketchesConstrained(t, b, 1)
	requireBoundsZ(t, disc, balanceLow, balanceHigh, balanceRimOuter)
	require.Len(t, partSection(t, disc, 15.9).Loops, 1+balanceArms, "rim and four windows")
	for k := 0; k < balanceArms; k++ {
		a := 90*deg + float64(k)*90*deg
		mid := Polar(20, a)
		require.False(t, partInside(t, disc, mid.X, mid.Y, 15.9), "window %d", k)
		arm := Polar(20, a+45*deg)
		require.True(t, partInside(t, disc, arm.X, arm.Y, 15.9), "arm %d", k)
	}
	require.True(t, partInside(t, disc, 0, 39, 15.9), "rim")
	require.False(t, partInside(t, disc, 0, 43, 15.9), "outside the rim")
}

func TestBalanceWheel(t *testing.T) {
	b := balanceBuilder(t)
	wheel, err := b.buildBalanceWheel()
	require.NoError(t, err)
	require.True(t, wheel.IsSolid())
	// Disc, bore tool, boss and the boss's cut: every sketch verified fully constrained.
	requireSketchesConstrained(t, b, 2)
	requireBoundsZ(t, wheel, balanceLow, balanceHigh+balanceBossHeight, balanceRimOuter)

	// Four windows at mid-height between the arms at 45°, 135°, 225° and 315°.
	o := partSection(t, wheel, 15.9)
	require.Len(t, o.Loops, 1+balanceArms+1, "rim, four windows and the bore")
	for k := 0; k < balanceArms; k++ {
		a := 90*deg + float64(k)*90*deg
		mid := Polar(20, a)
		require.False(t, partInside(t, wheel, mid.X, mid.Y, 15.9), "window %d", k)
		arm := Polar(20, a+45*deg)
		require.True(t, partInside(t, wheel, arm.X, arm.Y, 15.9), "arm %d", k)
	}
	require.True(t, partInside(t, wheel, 0, 39, 15.9), "rim")
	require.False(t, partInside(t, wheel, 0, 43, 15.9), "outside the rim")
	// The boss stands on the front face and the D bore runs through it, flat on +X.
	require.True(t, partInside(t, wheel, -6, 0, 19.0), "boss")
	require.False(t, partInside(t, wheel, -8, 0, 19.0), "outside the boss")
	require.False(t, partInside(t, wheel, 2.4, 0, 19.0), "bore")
	require.True(t, partInside(t, wheel, 2.8, 0, 19.0), "bore flat")
	require.False(t, partInside(t, wheel, -3.2, 0, 15.9), "bore arc")
}

// TestBalanceInertia checks the printed wheel's mass against 13.1 g and its inertia, with the roller table and the
// safety roller added in closed form, against the Layout value within 2 %.
func TestBalanceInertia(t *testing.T) {
	b := balanceBuilder(t)
	wheel, err := b.buildBalanceWheel()
	require.NoError(t, err)
	mp, err := wheel.MassProperties(t.Context(), units.GramsPerCubicCentimeter(PLADensity*1e3))
	require.NoError(t, err)
	mass := mp.Mass.Value.Base() * 1e3 // kg to g
	require.InEpsilon(t, 13.1, mass, 0.02)
	require.InEpsilon(t, b.g.WheelMass, mass, 0.02)

	inertia := mp.Inertia.ZZ.Value.Base() * 1e3 // kg mm² to g mm²
	for _, d := range []struct{ r, h float64 }{{11, 2.5}, {SafetyRadius, 7.5}} {
		m := PLADensity * math.Pi * d.r * d.r * d.h
		inertia += m * d.r * d.r / 2
	}
	require.InEpsilon(t, b.g.BalanceInertia, inertia, 0.02)
}

func TestBalanceCorners(t *testing.T) {
	b := balanceBuilder(t)
	wheel, err := b.buildBalanceWheel()
	require.NoError(t, err)
	requireCorners(t, wheel)
}

func TestHairspring(t *testing.T) {
	b := partBuilder(t)
	g := b.g
	dm, err := SpringParams().Derive()
	require.NoError(t, err)
	require.Equal(t, 8, dm.Turns)
	require.InDelta(t, 2.6667, dm.Pitch, 1e-4)
	require.InDelta(t, 1.1667, dm.Gap, 1e-4)
	require.InDelta(t, 3.55, dm.BoreRadius, 1e-4)
	require.InDelta(t, 3.4, dm.EyeRadius, 1e-4)
	require.InDelta(t, 3.8, dm.EyeHoleDiameter, 1e-4)
	require.InDelta(t, 967.611, dm.ActiveLength, 0.01)
	require.InDelta(t, 1.8312, g.SpringRate, 1e-3)
	require.InDelta(t, -32.0106, g.EyeCentre.X, 1e-4)
	require.InDelta(t, 92.5714, g.EyeCentre.Y, 1e-4)

	spring, err := b.buildHairspring()
	require.NoError(t, err)
	require.True(t, spring.IsSolid())
	requireSketchesConstrained(t, b, 1)
	bb, err := spring.Bounds()
	require.NoError(t, err)
	require.InDelta(t, springLow, bb.Min.Z, 3e-3)
	require.InDelta(t, springLow+SpringParams().Height, bb.Max.Z, 3e-3)

	// The eye's hole is a loop of the section at mid-height 0.40 round the stud (r 1.5): radius 1.9.
	o := partSection(t, spring, springLow+1.5)
	found := false
	for _, l := range o.Loops {
		lo, hi := math.Inf(1), 0.0
		for _, v := range l {
			d := math.Hypot(v.X-g.Stud.X, v.Y-g.Stud.Y)
			lo, hi = math.Min(lo, d), math.Max(hi, d)
		}
		if hi >= 3 {
			continue
		}
		require.InDelta(t, 1.9, lo, 0.02+o.OffSurface.Base())
		require.InDelta(t, 1.9, hi, 0.02+o.OffSurface.Base())
		found = true
	}
	require.True(t, found, "the eye's hole round the stud")
}

func TestRetainer(t *testing.T) {
	b := partBuilder(t)
	ring, err := b.buildRetainer()
	require.NoError(t, err)
	require.True(t, ring.IsSolid())
	requireSketchesConstrained(t, b, 2)
	requireBoundsZ(t, ring, retainerLow, retainerHigh, retainerRadius)
	require.False(t, partInside(t, ring, 2.4, 0, 24.4), "bore")
	require.True(t, partInside(t, ring, 2.8, 0, 24.4), "bore flat")
	require.True(t, partInside(t, ring, -4.5, 0, 24.4), "wall")
	// Wall round the bore: 5.0 - 3.35 = 1.65 at the arc and 5.0 - 2.60 = 2.4 at the flat.
	require.GreaterOrEqual(t, retainerRadius-BodyBoreRadius, 1.6)
	requireCorners(t, ring)
}
