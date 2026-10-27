package screwgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/screwgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// shortParams keeps the ribbons to 16 teeth, the least the bores' range check admits, so
// a build takes about 15 s instead of 40 s.
func shortParams() screwgear.Params {
	p := screwgear.DefaultParams()
	p.ToothCount = 16
	return p
}

func build(t *testing.T, p screwgear.Params) (*screwgear.Mechanism, *decad.Document) {
	t.Helper()
	doc := screwgear.NewDocument()
	m, err := screwgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 3)
	for name, b := range map[string]*decad.Body{"gear A": m.GearA, "gear B": m.GearB, "cage": m.Cage} {
		require.True(t, b.IsSolid(), name)
	}
	return m, doc
}

// ribbonVolume integrates, without decad, the section area of a ribbon over its length.
// The twist turns the section rigidly and so leaves its area alone; the toothed edge
// u = W/2 - H/2 + H/2*cos(2*pi*(s + tan(slant)*v - phase)/P) - bow*v^2 sets it.
func ribbonVolume(p screwgear.Params, phase float64) float64 {
	const nv, ns = 200, 20000
	length := p.Length()
	total := 0.0
	for i := range ns {
		s := -length/2 + phase + (float64(i)+0.5)*length/ns
		area := 0.0
		for j := range nv {
			v := -p.Thickness/2 + (float64(j)+0.5)*p.Thickness/nv
			cell := (s + math.Tan(p.ToothSlant)*v - phase) / p.ToothPitch
			edge := p.Width/2 - p.ToothHeight/2 + p.ToothHeight/2*math.Cos(2*math.Pi*cell) - p.ToothBow*v*v
			area += (edge + p.Width/2) * p.Thickness / nv
		}
		total += area * length / ns
	}
	return total
}

func TestBuildDefaultMechanism(t *testing.T) {
	p := screwgear.DefaultParams()
	m, _ := build(t, p)

	for name, b := range map[string]*decad.Body{"gear A": m.GearA, "gear B": m.GearB} {
		v, err := b.Volume()
		require.NoError(t, err)
		want := ribbonVolume(p, map[string]float64{"gear A": 0, "gear B": p.AssemblyPhase}[name])
		require.InDelta(t, want, v.Value.Base(), v.Bound.Base(), name)
		require.Less(t, v.Bound.Base(), 0.03*want, name)
	}

	// The unmarked sleeve holds 17,088 mm^3 in the Fusion proof's voxel count.
	v, err := m.Cage.Volume()
	require.NoError(t, err)
	require.InDelta(t, 17088, v.Value.Base(), v.Bound.Base())
	require.Less(t, v.Bound.Base(), 0.01*17088)

	// The tube spans CageRise either side of the middle plane and Ro round its axis.
	box, err := m.Cage.Bounds()
	require.NoError(t, err)
	require.InDelta(t, -p.CageRise, box.Min.Z, box.Bound.Base()+1e-6)
	require.InDelta(t, p.CageRise, box.Max.Z, box.Bound.Base()+1e-6)
	require.InDelta(t, p.CageRadius+p.CollarHalf, box.Max.X, box.Bound.Base()+1e-6)
}

func TestBuildPlacementMovesTheWholeMechanism(t *testing.T) {
	t.Parallel()
	frame, err := r3.NewFrame(r3.NewVec(10, -5, 7), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	p := shortParams()
	p.Placement = &frame
	m, _ := build(t, p)

	// U = +Y and V = +Z make the normal U x V = +X, so the tube's axis is world X.
	box, err := m.Cage.Bounds()
	require.NoError(t, err)
	tol := box.Bound.Base() + 1e-6
	require.InDelta(t, 10-p.CageRise, box.Min.X, tol)
	require.InDelta(t, 10+p.CageRise, box.Max.X, tol)
	ro := p.CageRadius + p.CollarHalf
	require.InDelta(t, -5-ro, box.Min.Y, tol)
	require.InDelta(t, -5+ro, box.Max.Y, tol)
	require.InDelta(t, 7-ro, box.Min.Z, tol)
	require.InDelta(t, 7+ro, box.Max.Z, tol)

	v, err := m.Cage.Volume()
	require.NoError(t, err)
	require.InDelta(t, 17126, v.Value.Base(), v.Bound.Base()+50)
}

// Measured from the real cage, the bore faces stand about 0.19 mm from the ribbon's back edge,
// crest and one thickness face, and about 0.48 mm from the other, which is the roof face that
// carries the 0.6 mm allowance. The readings are distances to the twisted wall, shorter than the
// section's own 0.2 mm and 0.8 mm offsets because the wall tilts against the section plane.
// Which face is the roof alternates with the gear and the side, as roofSide says. decad
// reports a lower bound of zero for these distances, so the test checks the measured values
// and does not prove clearance.
func TestBuildBoresFollowTheRibbonSurface(t *testing.T) {
	t.Parallel()
	p := shortParams()
	m, _ := build(t, p)
	for gi := range 2 {
		for _, sign := range []float64{-1, 1} {
			s := sign * p.CageRadius
			roof := screwgear.RoofSideForTest(p, gi, sign)
			for _, c := range []struct {
				name string
				u, v float64
				roof bool
			}{
				{"crest", p.Width / 2, 0, false},
				{"back edge", -p.Width / 2, 0, false},
				{"upper face", 0, p.Thickness / 2, roof > 0},
				{"lower face", 0, -p.Thickness / 2, roof < 0},
			} {
				pt := screwgear.GearWorldForTest(p, gi, c.u, c.v, s)
				d, err := m.Cage.DistanceToPoint(t.Context(), pt, units.Millimeters(0.5))
				require.NoError(t, err)
				want, tol := 0.19, 0.03
				if c.roof {
					want, tol = 0.48, 0.05
				}
				require.InDelta(t, want, d.Value.Base(), tol, "gear %d side %+.0f %s", gi, sign, c.name)
			}
		}
	}
}

// Each case changes one construction branch of the sleeve. A build that succeeds has drawn
// every bore, window and tube sketch fully constrained (buildSleeve verifies each one).
func TestBuildBranches(t *testing.T) {
	t.Parallel()
	deg := math.Pi / 180
	for _, c := range []struct {
		name      string
		change    func(*screwgear.Params)
		facing    r3.Vec  // the first window's direction
		low, high float64 // cage volume range in mm^3, wide enough for the branch's own bores
	}{
		{"no roof allowance", func(p *screwgear.Params) { p.RoofAllowance = 0 }, r3.NewVec(0, 1, 0), 15500, 19500},
		{"windows face the sleeve axes", func(p *screwgear.Params) { p.CrossAngle = 100 * deg }, r3.NewVec(1, 0, 0), 15500, 19500},
		{"mounted at plus and minus five degrees", func(p *screwgear.Params) {
			p.MountAngleA, p.MountAngleB = 5*deg, -5*deg
		}, r3.NewVec(0, 1, 0), 15500, 19500},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := shortParams()
			c.change(&p)
			windows, err := screwgear.WindowsForTest(p)
			require.NoError(t, err)
			require.Equal(t, c.facing, windows[0].Facing)
			require.Empty(t, windows[0].Room)
			require.Empty(t, windows[1].Room)
			m, _ := build(t, p)
			v, err := m.Cage.Volume()
			require.NoError(t, err)
			require.Greater(t, v.Value.Base(), c.low)
			require.Less(t, v.Value.Base(), c.high)
		})
	}
}

func TestBuildRejectsInvalidParams(t *testing.T) {
	t.Parallel()
	p := screwgear.DefaultParams()
	p.Clearance = 0
	_, err := screwgear.Build(t.Context(), screwgear.NewDocument(), p)
	require.ErrorContains(t, err, "clearance")
	_, err = screwgear.Build(nil, screwgear.NewDocument(), screwgear.DefaultParams()) //nolint:staticcheck // the nil contract
	require.Error(t, err)
}
