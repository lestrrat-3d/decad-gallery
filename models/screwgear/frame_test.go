package screwgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/screwgear"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func requireVec(t *testing.T, want, got r3.Vec, tol float64) {
	t.Helper()
	require.InDelta(t, want.X, got.X, tol)
	require.InDelta(t, want.Y, got.Y, tol)
	require.InDelta(t, want.Z, got.Z, tol)
}

// TestFramePairAtTheDefaults pins the section 1 construction in the pair frame, where C is
// the origin, e is X, k is Y and n is Z.
func TestFramePairAtTheDefaults(t *testing.T) {
	p := screwgear.DefaultParams()
	gears, world, err := screwgear.PairForTest(p)
	require.NoError(t, err)
	a, b := gears[0], gears[1]
	half := deg(40)

	// The axes sit +-A/2 along n, and open by +-Sigma/2 about n from e.
	requireVec(t, r3.NewVec(0, 0, -6.925), a.Origin, 1e-12)
	requireVec(t, r3.NewVec(0, 0, 6.925), b.Origin, 1e-12)
	requireVec(t, r3.NewVec(math.Cos(half), math.Sin(half), 0), a.Ez, 1e-12)
	requireVec(t, r3.NewVec(math.Cos(half), -math.Sin(half), 0), b.Ez, 1e-12)

	// u points at the other gear and v = dir x u, so the two gears are the same part.
	requireVec(t, r3.NewVec(0, 0, 1), a.Ex, 1e-12)
	requireVec(t, r3.NewVec(0, 0, -1), b.Ex, 1e-12)
	requireVec(t, a.Ez.Cross(a.Ex), a.Ey, 1e-12)
	requireVec(t, b.Ez.Cross(b.Ex), b.Ey, 1e-12)
	require.InDelta(t, 1, a.Ey.Len(), 1e-12)
	require.InDelta(t, 0, a.Ey.Dot(a.Ez), 1e-12)

	// The angle between the axes is Sigma.
	require.InDelta(t, p.CrossAngle, math.Acos(a.Ez.Dot(b.Ez)), 1e-12)

	// The assembly phase: gear A at zero and gear B at -1.31 mm.
	require.Equal(t, 0.0, a.Phase)
	require.Equal(t, -1.31, b.Phase)

	// With no placement the pair frame is the world frame.
	pt := r3.NewVec(1, 2, 3)
	requireVec(t, pt, world(pt), 1e-12)
}

func TestFrameAngleAndSectionCoordinates(t *testing.T) {
	p := screwgear.DefaultParams()
	p.MountAngleA, p.MountAngleB = deg(7), deg(-5)
	// theta = s/Lambda + Phi_g.
	require.InDelta(t, 12/p.Lambda()+deg(7), screwgear.GearAngleForTest(p, 0, 12), 1e-12)
	require.InDelta(t, -12/p.Lambda()+deg(-5), screwgear.GearAngleForTest(p, 1, -12), 1e-12)

	for gi := range 2 {
		for _, q := range [][3]float64{{0, 0, 0}, {7.5, -1.9, 12.3}, {-7.5, 1.9, -18}, {3, 0.5, 40}} {
			pt := screwgear.GearWorldForTest(p, gi, q[0], q[1], q[2])
			u, v, s := screwgear.GearLocalForTest(p, gi, pt)
			require.InDelta(t, q[0], u, 1e-9)
			require.InDelta(t, q[1], v, 1e-9)
			require.InDelta(t, q[2], s, 1e-9)
		}
	}
	// At station zero with no mount, (u, v) is plain (u, v): A's u is +n.
	pt := screwgear.GearWorldForTest(screwgear.DefaultParams(), 0, 2, 0, 0)
	requireVec(t, r3.NewVec(0, 0, -6.925+2), pt, 1e-12)
}

// TestFramePlacementCarriesThePair places the mechanism on a side plane and checks the
// section 1 relations in world coordinates.
func TestFramePlacementCarriesThePair(t *testing.T) {
	c := r3.NewVec(10, -5, 7)
	f, err := r3.NewFrame(c, r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1)) // e = +Y, k = +Z, n = +X
	require.NoError(t, err)
	p := screwgear.DefaultParams()
	p.Placement = &f
	require.NoError(t, p.Validate())

	gears, world, err := screwgear.PairForTest(p)
	require.NoError(t, err)
	n, e, k := f.N(), f.U(), f.V()
	requireVec(t, r3.NewVec(1, 0, 0), n, 1e-12)
	requireVec(t, n.Cross(e), k, 1e-12)

	a, b := gears[0], gears[1]
	requireVec(t, c.Sub(n.Scale(6.925)), world(a.Origin), 1e-12)
	requireVec(t, c.Add(n.Scale(6.925)), world(b.Origin), 1e-12)
	dir := func(v r3.Vec) r3.Vec { return world(v).Sub(c) }
	// dirA = rotate(e, +Sigma/2 about n) = cos*e + sin*(n x e).
	requireVec(t, e.Scale(math.Cos(deg(40))).Add(k.Scale(math.Sin(deg(40)))), dir(a.Ez), 1e-12)
	requireVec(t, e.Scale(math.Cos(deg(40))).Sub(k.Scale(math.Sin(deg(40)))), dir(b.Ez), 1e-12)
	requireVec(t, n, dir(a.Ex), 1e-12)
	requireVec(t, n.Scale(-1), dir(b.Ex), 1e-12)
}

func TestFrameBoreRoofsAtTheDefaults(t *testing.T) {
	p := screwgear.DefaultParams()
	// Gear A's -R bore has its +v face up and gear B's -R bore its -v face up, and each
	// gear's +R bore is the other way: at every bore the roof carries the 0.60 mm allowance
	// and the opening is 4.75 mm through.
	roof := map[[2]float64]float64{{0, -1}: 1, {0, 1}: -1, {1, -1}: -1, {1, 1}: 1}
	for key, want := range roof {
		gi, sign := int(key[0]), key[1]
		require.Equal(t, want, screwgear.RoofSideForTest(p, gi, sign), "gear %d sign %v", gi, sign)
		hw, vLo, vHi := screwgear.BoreOpeningForTest(p, gi, sign)
		require.InDelta(t, 7.7, hw, 1e-12)
		require.InDelta(t, 4.75, vHi-vLo, 1e-12)
		if want > 0 {
			require.InDelta(t, -2.075, vLo, 1e-12)
			require.InDelta(t, 2.675, vHi, 1e-12)
			continue
		}
		require.InDelta(t, -2.675, vLo, 1e-12)
		require.InDelta(t, 2.075, vHi, 1e-12)
	}
}
