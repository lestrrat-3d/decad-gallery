package clamparm_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

const derivedTol = 1e-9

func requireVec(t *testing.T, want, got r3.Vec) {
	t.Helper()
	require.InDelta(t, want.X, got.X, derivedTol, "X of %v", got)
	require.InDelta(t, want.Y, got.Y, derivedTol, "Y of %v", got)
	require.InDelta(t, want.Z, got.Z, derivedTol, "Z of %v", got)
}

func TestClampArmParamsDefaults(t *testing.T) {
	p := clamparm.DefaultParams()
	require.Equal(t, 120.0, p.LinkLength)
	require.Equal(t, 25.0, p.Opening)
	require.Equal(t, 0.0, p.Pan)
	require.Equal(t, 70.0, p.Shoulder)
	require.Equal(t, -90.0, p.Elbow)
	require.Equal(t, 20.0, p.Wrist)
	require.Nil(t, p.Placement)
	require.NoError(t, p.Validate())
	// The default pose puts the head level.
	require.Equal(t, 0.0, p.Shoulder+p.Elbow+p.Wrist)
}

func TestClampArmParamsValidation(t *testing.T) {
	nan, inf := math.NaN(), math.Inf(1)
	tests := []struct {
		name   string
		edit   func(*clamparm.Params)
		errMsg string
	}{
		{"link length NaN", func(p *clamparm.Params) { p.LinkLength = nan }, "clamparm: LinkLength must be finite"},
		{"opening inf", func(p *clamparm.Params) { p.Opening = inf }, "clamparm: Opening must be finite"},
		{"pan NaN", func(p *clamparm.Params) { p.Pan = nan }, "clamparm: Pan must be finite"},
		{"shoulder inf", func(p *clamparm.Params) { p.Shoulder = -inf }, "clamparm: Shoulder must be finite"},
		{"elbow NaN", func(p *clamparm.Params) { p.Elbow = nan }, "clamparm: Elbow must be finite"},
		{"wrist inf", func(p *clamparm.Params) { p.Wrist = inf }, "clamparm: Wrist must be finite"},
		{"link too short", func(p *clamparm.Params) { p.LinkLength = 79.9 },
			"clamparm: LinkLength must be from 80 to 150 mm, got 79.9"},
		{"link too long", func(p *clamparm.Params) { p.LinkLength = 150.1 },
			"clamparm: LinkLength must be from 80 to 150 mm, got 150.1"},
		{"opening too thin", func(p *clamparm.Params) { p.Opening = 9 },
			"clamparm: Opening must be from 10 to 40 mm, got 9"},
		{"opening too thick", func(p *clamparm.Params) { p.Opening = 41 },
			"clamparm: Opening must be from 10 to 40 mm, got 41"},
		{"pan low", func(p *clamparm.Params) { p.Pan = -125.5 },
			"clamparm: Pan must be from -125 to 125 degrees, got -125.5"},
		{"pan high", func(p *clamparm.Params) { p.Pan = 126 },
			"clamparm: Pan must be from -125 to 125 degrees, got 126"},
		{"shoulder low", func(p *clamparm.Params) { p.Shoulder = -61 },
			"clamparm: Shoulder must be from -60 to 115 degrees, got -61"},
		{"shoulder high", func(p *clamparm.Params) { p.Shoulder = 116 },
			"clamparm: Shoulder must be from -60 to 115 degrees, got 116"},
		{"elbow low", func(p *clamparm.Params) { p.Elbow = -116 },
			"clamparm: Elbow must be from -115 to 115 degrees, got -116"},
		{"elbow high", func(p *clamparm.Params) { p.Elbow = 116 },
			"clamparm: Elbow must be from -115 to 115 degrees, got 116"},
		{"wrist low", func(p *clamparm.Params) { p.Wrist = -111 },
			"clamparm: Wrist must be from -110 to 90 degrees, got -111"},
		{"wrist high", func(p *clamparm.Params) { p.Wrist = 91 },
			"clamparm: Wrist must be from -110 to 90 degrees, got 91"},
		{"placement invalid", func(p *clamparm.Params) { p.Placement = &r3.Frame{} },
			"clamparm: Placement is not a valid frame"},
		// Order: every finite check precedes every range check, and the checks follow the field order.
		{"finite before range", func(p *clamparm.Params) { p.LinkLength = 10; p.Wrist = nan },
			"clamparm: Wrist must be finite"},
		{"finite in field order", func(p *clamparm.Params) { p.Elbow = nan; p.Opening = nan },
			"clamparm: Opening must be finite"},
		{"link before opening", func(p *clamparm.Params) { p.LinkLength = 10; p.Opening = 1 },
			"clamparm: LinkLength must be from 80 to 150 mm, got 10"},
		{"opening before pan", func(p *clamparm.Params) { p.Opening = 1; p.Pan = 200 },
			"clamparm: Opening must be from 10 to 40 mm, got 1"},
		{"pan before shoulder", func(p *clamparm.Params) { p.Pan = 200; p.Shoulder = 200 },
			"clamparm: Pan must be from -125 to 125 degrees, got 200"},
		{"shoulder before elbow", func(p *clamparm.Params) { p.Shoulder = 200; p.Elbow = 200 },
			"clamparm: Shoulder must be from -60 to 115 degrees, got 200"},
		{"elbow before wrist", func(p *clamparm.Params) { p.Elbow = 200; p.Wrist = 200 },
			"clamparm: Elbow must be from -115 to 115 degrees, got 200"},
		{"wrist before placement", func(p *clamparm.Params) { p.Wrist = 200; p.Placement = &r3.Frame{} },
			"clamparm: Wrist must be from -110 to 90 degrees, got 200"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := clamparm.DefaultParams()
			tc.edit(&p)
			_, err := p.Derive()
			require.EqualError(t, err, tc.errMsg)
			require.EqualError(t, p.Validate(), tc.errMsg)
		})
	}
}

func TestClampArmParamsLimitsAreValid(t *testing.T) {
	edits := []func(*clamparm.Params){
		func(p *clamparm.Params) { p.LinkLength = 80 },
		func(p *clamparm.Params) { p.LinkLength = 150 },
		func(p *clamparm.Params) { p.Opening = 10 },
		func(p *clamparm.Params) { p.Opening = 40 },
		func(p *clamparm.Params) { p.Pan = 125 },
		func(p *clamparm.Params) { p.Pan = -125 },
		func(p *clamparm.Params) { p.Shoulder = clamparm.ShoulderMin },
		func(p *clamparm.Params) { p.Shoulder = clamparm.JointMax },
		func(p *clamparm.Params) { p.Elbow = -clamparm.JointMax },
		func(p *clamparm.Params) { p.Elbow = clamparm.JointMax },
		func(p *clamparm.Params) { p.Wrist = clamparm.WristMin },
		func(p *clamparm.Params) { p.Wrist = clamparm.WristMax },
	}
	for _, edit := range edits {
		p := clamparm.DefaultParams()
		edit(&p)
		_, err := p.Derive()
		require.NoError(t, err)
	}
}

func TestClampArmParamsDerivedDefault(t *testing.T) {
	p := clamparm.DefaultParams()
	d, err := p.Derive()
	require.NoError(t, err)

	// Each link runs 120 mm along its pitch: link A at 70 degrees, link B at 70 - 90 = -20, the head level.
	a, b := 70*math.Pi/180, -20*math.Pi/180
	jointA := r3.NewVec(100, 0, 24.26)
	jointB := r3.NewVec(jointA.X+120*math.Cos(a), 0, jointA.Z+120*math.Sin(a))
	jointH := r3.NewVec(jointB.X+120*math.Cos(b), 0, jointB.Z+120*math.Sin(b))
	tip := r3.NewVec(jointH.X+40, 0, jointH.Z+10)

	requireVec(t, r3.NewVec(30, 0, 14.26), d.Joints[0])
	requireVec(t, jointA, d.Joints[1])
	requireVec(t, jointB, d.Joints[2])
	requireVec(t, jointH, d.Joints[3])
	requireVec(t, jointA, d.LinkA.Origin())
	requireVec(t, jointB, d.LinkB.Origin())
	requireVec(t, jointH, d.Head.Origin())
	requireVec(t, tip, d.Interface.Origin())
	requireVec(t, r3.NewVec(30, 0, 14.26), d.Swivel.Origin())

	// The head is level: its frame has the world's axes, so the interface faces +Z.
	requireVec(t, r3.NewVec(1, 0, 0), d.Head.U())
	requireVec(t, r3.NewVec(0, 1, 0), d.Head.V())
	requireVec(t, r3.NewVec(0, 0, 1), d.Interface.N())

	// Axes: +Z for J1, the fork's +Y (the knob side) for the others.
	for i, want := range []r3.Vec{r3.NewVec(0, 0, 1), r3.NewVec(0, 1, 0), r3.NewVec(0, 1, 0), r3.NewVec(0, 1, 0)} {
		requireVec(t, want, d.Axes[i])
	}

	// Sites.
	requireVec(t, r3.NewVec(30, 0, 0), d.PanSite.Head)
	requireVec(t, r3.NewVec(0, 0, 1), d.PanSite.Dir)
	require.Equal(t, 35.0, d.PanSite.Length)
	requireVec(t, r3.NewVec(100, -10, 24.26), d.JointSites[0].Head)
	requireVec(t, r3.NewVec(jointB.X, -10, jointB.Z), d.JointSites[1].Head)
	requireVec(t, r3.NewVec(jointH.X, -10, jointH.Z), d.JointSites[2].Head)
	for _, s := range d.JointSites {
		requireVec(t, r3.NewVec(0, 1, 0), s.Dir)
		require.Equal(t, 25.0, s.Length)
		require.NoError(t, s.Validate())
	}
	requireVec(t, r3.NewVec(jointH.X+40, 0, jointH.Z-20), d.StudSite.Head)
	requireVec(t, r3.NewVec(0, 0, 1), d.StudSite.Dir)
	require.Equal(t, 40.0, d.StudSite.Length)
	requireVec(t, r3.NewVec(25, 0, -102.5), d.ScrewSite.Head)
	requireVec(t, r3.NewVec(0, 0, 1), d.ScrewSite.Dir)
	requireVec(t, r3.NewVec(0, 1, 0), d.ScrewSite.Corner)
	require.Equal(t, 70.0, d.ScrewSite.Length)

	requireSiteTips(t, jointH, d)

	// Knob frames: each U is its site's Corner and each normal its Dir.
	sites := []struct {
		name  string
		frame r3.Frame
		U, N  r3.Vec
	}{
		{"J1", d.KnobFrames[0], d.PanSite.Corner, d.PanSite.Dir},
		{"J2", d.KnobFrames[1], d.JointSites[0].Corner, d.JointSites[0].Dir},
		{"J3", d.KnobFrames[2], d.JointSites[1].Corner, d.JointSites[1].Dir},
		{"J4", d.KnobFrames[3], d.JointSites[2].Corner, d.JointSites[2].Dir},
		{"stud", d.ScrewKnobFrames[1], d.StudSite.Corner, d.StudSite.Dir.Scale(-1)},
		{"screw", d.ScrewKnobFrames[0], d.ScrewSite.Corner, d.ScrewSite.Dir.Scale(-1)},
	}
	for _, s := range sites {
		requireVec(t, s.U, s.frame.U())
		requireVec(t, s.N, s.frame.N())
	}
	requireVec(t, r3.NewVec(30, 0, 28.26), d.KnobFrames[0].Origin())
	requireVec(t, d.JointSites[1].Head.Add(r3.NewVec(0, 20, 0)), d.KnobFrames[2].Origin())
	requireVec(t, d.PanSite.Head.Add(r3.NewVec(0, 0, 31.26)), d.KnobNutFrames[0].Origin())
	requireVec(t, d.JointSites[1].Head.Add(r3.NewVec(0, 23, 0)), d.KnobNutFrames[2].Origin())
	requireVec(t, r3.NewVec(25, 0, -92.5), d.ScrewKnobFrames[0].Origin())
	requireVec(t, r3.NewVec(jointH.X+40, 0, jointH.Z-10), d.ScrewKnobFrames[1].Origin())
	requireVec(t, r3.NewVec(25, 0, -64.6), d.JawNutFrame.Origin())
	requireVec(t, tip.Add(r3.NewVec(0, 0, 4.7)), d.AdapterNutFrame.Origin())
	requireVec(t, r3.NewVec(25, 0, -25), d.PadFrame.Origin())
	require.InDelta(t, 19.0, d.ScrewKnobGap, derivedTol)
}

func TestClampArmParamsScrewKnobGap(t *testing.T) {
	for opening, want := range map[float64]float64{10: 4, 25: 19, 40: 34} {
		p := clamparm.DefaultParams()
		p.Opening = opening
		d, err := p.Derive()
		require.NoError(t, err)
		require.InDelta(t, want, d.ScrewKnobGap, derivedTol, "opening %g", opening)
		// The knob's inner face stands ScrewKnobGap below the bottom jaw's underside at z = -73.5.
		require.InDelta(t, -73.5-want, d.ScrewKnobFrames[0].Origin().Z, derivedTol)
		require.GreaterOrEqual(t, d.ScrewKnobGap, 0.5)
	}
}

func TestClampArmParamsPan(t *testing.T) {
	p := clamparm.DefaultParams()
	p.Pan = 90
	d, err := p.Derive()
	require.NoError(t, err)
	// J1 stays on its axis; the swivel's +X turns onto +Y and its fork axis onto -X.
	requireVec(t, r3.NewVec(30, 0, 14.26), d.Joints[0])
	requireVec(t, r3.NewVec(0, 1, 0), d.Swivel.U())
	requireVec(t, r3.NewVec(-1, 0, 0), d.Axes[1])
	requireVec(t, r3.NewVec(30, 70, 24.26), d.Joints[1])
}

func TestClampArmParamsPlacementIsNotApplied(t *testing.T) {
	// Dimensions are in the world frame before Placement.
	f, err := r3.NewFrame(r3.NewVec(10, -5, 7), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	p := clamparm.DefaultParams()
	p.Placement = &f
	placed, err := p.Derive()
	require.NoError(t, err)
	plain, err := clamparm.DefaultParams().Derive()
	require.NoError(t, err)
	requireVec(t, plain.Interface.Origin(), placed.Interface.Origin())
}

// requireTip requires the bolt's tip point, built from the head, the direction and an independent distance.
func requireTip(t *testing.T, want r3.Vec, head, dir r3.Vec, dist float64) {
	t.Helper()
	requireVec(t, want, head.Add(dir.Scale(dist)))
}

// requireSiteTips checks every bolt's tip against the design values: each stays past its nut and inside the outer face.
func requireSiteTips(t *testing.T, jointH r3.Vec, d clamparm.Dimensions) {
	t.Helper()
	// J1: head at z = 0, tip z 38.5 (past the nut's top at 35.96, inside the knob's face at 40.26).
	requireTip(t, r3.NewVec(30, 0, 38.5), d.PanSite.Head, d.PanSite.Dir, 38.5)
	require.Equal(t, 38.5, d.PanSite.Head.Z+d.PanSite.Tip())
	// J2 to J4: heads at y = -10, tips at y 18.5 (past the nut's top at 17.7, inside the outer face at 22).
	for i, s := range d.JointSites {
		requireVec(t, r3.NewVec(0, 1, 0), s.Dir)
		require.InDelta(t, 18.5, s.Head.Y+s.Dir.Y*28.5, derivedTol, "J%d", i+2)
		require.InDelta(t, 28.5, s.Tip(), derivedTol, "J%d", i+2)
	}
	// Stud: 23.5 above the head's origin plane z, 13.5 above the interface face.
	requireTip(t, r3.NewVec(jointH.X+40, 0, jointH.Z+23.5), d.StudSite.Head, d.StudSite.Dir, 43.5)
	require.InDelta(t, 43.5, d.StudSite.Tip(), derivedTol)
}

func TestClampArmParamsScrewTip(t *testing.T) {
	for _, o := range []float64{10, 25, 40} {
		p := clamparm.DefaultParams()
		p.Opening = o
		d, err := p.Derive()
		require.NoError(t, err)
		requireTip(t, r3.NewVec(25, 0, -(o+4)), d.ScrewSite.Head, d.ScrewSite.Dir, 73.5)
		require.InDelta(t, 73.5, d.ScrewSite.Tip(), derivedTol, "opening %g", o)
	}
}

// requireNoResidue requires that every entry of v within 1e-9 of 0 or of an integer is exactly it.
func requireNoResidue(t *testing.T, what string, v r3.Vec) {
	t.Helper()
	for i, x := range []float64{v.X, v.Y, v.Z} {
		near := math.Round(x)
		if math.Abs(x-near) < derivedTol {
			require.Equal(t, near, x, "%s entry %d is a rounding residue", what, i)
		}
	}
}

// TestClampArmParamsDefaultPoseExact requires the default pose's head, interface, stud knob and adapter nut frames and
// their transforms to carry no rounding residue: a level head (70 - 90 + 20) is the identity rotation.
func TestClampArmParamsDefaultPoseExact(t *testing.T) {
	d, err := clamparm.DefaultParams().Derive()
	require.NoError(t, err)
	frames := map[string]r3.Frame{
		"Head": d.Head, "Interface": d.Interface, "StudKnob": d.ScrewKnobFrames[1], "AdapterNut": d.AdapterNutFrame,
	}
	for name, f := range frames {
		t.Run(name, func(t *testing.T) {
			requireNoResidue(t, "origin", f.Origin())
			requireNoResidue(t, "U", f.U())
			requireNoResidue(t, "V", f.V())
			xf, err := r3.FromFrame(f)
			require.NoError(t, err)
			b := xf.Basis()
			requireNoResidue(t, "EX", b.EX)
			requireNoResidue(t, "EY", b.EY)
			requireNoResidue(t, "EZ", b.EZ)
			requireNoResidue(t, "translation", xf.Translation())
		})
	}
	// The level head's axes are exactly the world axes.
	require.Equal(t, r3.NewVec(1, 0, 0), d.Head.U())
	require.Equal(t, r3.NewVec(0, 1, 0), d.Head.V())
}
