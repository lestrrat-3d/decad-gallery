package swisslever

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

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
	b := partBuilder(t)
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

// TestBalanceInertia checks the wheel's mass against 13.1 g and its inertia, with the roller table and the safety roller
// added in closed form, against the Layout value within 2 %. The unchamfered build is held to 2 %; the finished wheel
// is lighter by its chamfers (about 0.2 g) and is held to 3 %.
func TestBalanceInertia(t *testing.T) {
	density := units.GramsPerCubicCentimeter(PLADensity * 1e3)
	for _, c := range []struct {
		name  string
		plain bool
		tol   float64
	}{{"unchamfered", true, 0.02}, {"finished", false, 0.03}} {
		t.Run(c.name, func(t *testing.T) {
			b := partBuilder(t)
			b.plain = c.plain
			wheel, err := b.buildBalanceWheel()
			require.NoError(t, err)
			mp, err := wheel.MassProperties(t.Context(), density)
			require.NoError(t, err)
			mass := mp.Mass.Value.Base() * 1e3 // kg to g
			require.InEpsilon(t, 13.1, mass, c.tol)
			require.InEpsilon(t, b.g.WheelMass, mass, c.tol)
			inertia := mp.Inertia.ZZ.Value.Base() * 1e3 // kg mm² to g mm²
			for _, d := range []struct{ r, h float64 }{{11, 2.5}, {SafetyRadius, 7.5}} {
				m := PLADensity * math.Pi * d.r * d.r * d.h
				inertia += m * d.r * d.r / 2
			}
			require.InEpsilon(t, b.g.BalanceInertia, inertia, c.tol)
		})
	}
}

func TestBalanceCorners(t *testing.T) {
	b := partBuilder(t)
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

// TestBalanceChamferSelection requires the chamfer's edge set to be the rim on both faces and the four rear-face window
// loops, and to leave sharp the four boss root edges and the 28 front-face window edges.
func TestBalanceChamferSelection(t *testing.T) {
	b := partBuilder(t)
	b.plain = true
	wheel, err := b.buildBalanceWheel()
	require.NoError(t, err)
	type tally struct{ rim, windows, sharpRoot, sharpWindows int }
	got := map[float64]*tally{balanceLow: {}, balanceHigh: {}}
	for _, f := range wheel.Faces() {
		plane, ok := f.Surface().(decad.Plane)
		if !ok {
			continue
		}
		z := plane.Frame.Origin().Z
		n := got[z]
		area, err := f.Area()
		require.NoError(t, err)
		if n == nil || area.Value.Base() < 400 {
			continue
		}
		for _, l := range f.Loops() {
			edges := l.Edges()
			if !balanceChamferLoops(l, edges) {
				continue
			}
			for _, e := range edges {
				switch {
				case bossRootEdge(e):
					n.sharpRoot++
					require.InDelta(t, balanceHigh, z, 1e-6)
					require.IsType(t, decad.FacetedCurve{}, e.Curve())
				case balanceLeftSharp(e):
					n.sharpWindows++
				case l.IsOuter():
					n.rim++
					p := e.Start().Position().Value
					require.InDelta(t, balanceRimOuter, math.Hypot(p.X, p.Y), 1e-3)
				default:
					n.windows++
				}
			}
		}
	}
	front, rear := got[balanceHigh], got[balanceLow]
	require.Equal(t, 4, front.sharpRoot, "the four boss root edges stay sharp")
	require.Equal(t, 32-4, front.sharpWindows, "the front face's window edges stay sharp")
	require.Zero(t, front.windows+rear.sharpRoot+rear.sharpWindows)
	require.Equal(t, 1, front.rim, "front rim")
	require.Equal(t, 1, rear.rim, "rear rim")
	require.Equal(t, 4*8, rear.windows, "rear face: four windows of eight edges")
	require.Equal(t, 34, front.rim+rear.rim+rear.windows, "one Chamfer of 34 edges")
}
