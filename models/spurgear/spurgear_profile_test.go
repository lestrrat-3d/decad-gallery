package spurgear

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func TestDefaultGearProfileKeepsUnroundedRootCorners(t *testing.T) {
	t.Parallel()
	p := DefaultParams()
	d := derive(p)
	w := sketch.NewWorld()
	s, _, roots, err := buildGearProfile(t.Context(), w, w.XY(), p, d)
	require.NoError(t, err)
	require.Len(t, roots, 2*p.Teeth)
	for _, root := range roots {
		require.InDelta(t, d.root, math.Hypot(root.x, root.y), 1e-7)
	}
	rootArcs, tipArcs := 0, 0
	for _, entity := range s.Entities() {
		arc, ok := entity.(*sketch.Arc)
		if !ok {
			continue
		}
		switch {
		case math.Abs(arc.R()-d.root) < 1e-7:
			rootArcs++
		case math.Abs(arc.R()-d.tip) < 1e-7:
			tipArcs++
		default:
			t.Fatalf("unexpected profile arc radius %g", arc.R())
		}
	}
	require.Equal(t, p.Teeth, rootArcs)
	require.Equal(t, p.Teeth, tipArcs)
}

func TestGearProfilesAreFullyConstrained(t *testing.T) {
	cases := []struct {
		name string
		p    Params
	}{
		{"ordinary", DefaultParams()},
		{"embedded", Params{
			Module: 0.5, Teeth: 30, PressureAngle: 30 * math.Pi / 180,
			Thickness: 2, Samples: 5,
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := sketch.NewWorld()
			s, _, err := BuildProfile(t.Context(), world, world.XY(), tc.p)
			require.NoError(t, err)
			requireSpurSketchFullyConstrained(t, s)
		})
	}
}

func TestEmbeddedProfilesKeepCertifiedRootTrims(t *testing.T) {
	for _, teeth := range []int{24, 30} {
		p := Params{
			Module: 0.5, Teeth: teeth, PressureAngle: 30 * math.Pi / 180,
			Thickness: 2, Samples: 5,
		}
		d := derive(p)
		world := sketch.NewWorld()
		s, profile, roots, err := buildGearProfile(t.Context(), world, world.XY(), p, d)
		require.NoError(t, err)
		requireSpurSketchFullyConstrained(t, s)
		require.True(t, profile.Valid)
		require.Empty(t, profile.Holes)
		require.Len(t, profile.Outer, 4*p.Teeth)
		require.Len(t, roots, 2*p.Teeth)
		for _, root := range roots {
			require.InDelta(t, d.root, math.Hypot(root.x, root.y), 1e-8)
		}
		trimmed := 0
		for _, edge := range profile.Outer {
			flank, ok := edge.Entity.(*sketch.FitSpline)
			if !ok {
				continue
			}
			require.True(t, edge.Partial)
			require.True(t, edge.TExact)
			x, y := flank.Eval(edge.TStart)
			require.InDelta(t, d.root, math.Hypot(x, y), 1e-8)
			trimmed++
		}
		require.Equal(t, 2*p.Teeth, trimmed)
	}
}

func TestBoreProfileIsFullyConstrained(t *testing.T) {
	p := DefaultParams()
	p.BoreDiameter = 3
	s, _, err := buildBoreProfile(t.Context(), sketch.NewWorld(), p, 1)
	require.NoError(t, err)
	requireSpurSketchFullyConstrained(t, s)
}

func requireSpurSketchFullyConstrained(t *testing.T, s *sketch.Sketch) {
	t.Helper()
	report := s.Verify(t.Context())
	require.Equal(t, sketch.FullyConstrained, report.Status)
	require.Zero(t, report.DOF)
	require.Empty(t, report.Redundant)
	require.Empty(t, report.Conflicts)
	require.NoError(t, report.Check())
}

// The ordinary gear built from a root disc and N lofted teeth must enclose the
// same volume as a loft of the whole-gear outlines, filleted the same way.
func TestOrdinaryJoinMatchesOutlineLoft(t *testing.T) {
	p := DefaultParams()
	d := derive(p)
	ctx := t.Context()

	gear, err := Build(ctx, decad.New(), p)
	require.NoError(t, err)

	world := sketch.NewWorld()
	upperFrame, err := r3.NewFrame(r3.NewVec(0, 0, p.Thickness), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	upperPlane, err := world.CreatePlaneFromFrame(upperFrame)
	require.NoError(t, err)
	lowerSketch, lowerProfile, roots, err := buildGearProfile(ctx, world, world.XY(), p, d)
	require.NoError(t, err)
	upperSketch, upperProfile, _, err := buildGearProfile(ctx, world, upperPlane, p, d)
	require.NoError(t, err)
	reference, err := decad.Loft(ctx, decad.New(), lowerSketch, lowerProfile, upperSketch, upperProfile)
	require.NoError(t, err)
	frame, err := placement(p)
	require.NoError(t, err)
	reference, err = filletRoots(ctx, reference, frame, roots, filletRadius(p, d))
	require.NoError(t, err)

	got, err := gear.Volume()
	require.NoError(t, err)
	want, err := reference.Volume()
	require.NoError(t, err)
	require.InDelta(t, want.Value.Base(), got.Value.Base(), got.Bound.Base()+want.Bound.Base()+0.02)
}
