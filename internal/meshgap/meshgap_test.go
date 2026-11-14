package meshgap_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func bearingMesh(t *testing.T, p ballbearing.Params, origin r3.Vec) meshgap.Mesh {
	t.Helper()
	f, err := r3.NewFrame(origin, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	p.Placement = &f
	m, err := ballbearing.Build(t.Context(), decad.New(), p)
	require.NoError(t, err)
	mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified() && mesh.VolumeVerified())
	result, err := meshgap.New(mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err)
	return result
}

func TestTouchingBearingCaps(t *testing.T) {
	p := ballbearing.DefaultParams()
	a := bearingMesh(t, p, r3.Vec{})
	b := bearingMesh(t, p, r3.Vec{Z: p.Width})
	gap, separated, err := a.Exceeds(b, 0)
	require.NoError(t, err)
	require.False(t, separated)
	require.Zero(t, gap)
}

func TestBearingBoundaryGaps(t *testing.T) {
	a := bearingMesh(t, ballbearing.DefaultParams(), r3.Vec{})
	for _, test := range []struct {
		name                string
		inner, outer, width float64
		origin              r3.Vec
		clearance, minimum  float64
		separated           bool
	}{
		{"axial_gap", 8, 22, 7, r3.Vec{Z: 8}, .1, .8, true},
		{"permitted_gap", 8, 22, 7, r3.Vec{Z: 7.1}, .2, 0, false},
		{"inside_empty_bore", 2, 4, 1, r3.Vec{Z: 3}, .1, 1.8, true},
		{"contained_in_material", 10, 12, 1, r3.Vec{Z: 3}, .1, .8, true},
		{"crossing_wall", 4, 12, 1, r3.Vec{Z: 3}, .1, 0, false},
		{"far_sideways", 8, 22, 7, r3.Vec{X: 40}, .1, 17, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := bearingMesh(t, ballbearing.Params{BoreDiameter: test.inner, OuterDiameter: test.outer, Width: test.width}, test.origin)
			gap, separated, err := a.Exceeds(b, test.clearance)
			require.NoError(t, err)
			require.True(t, separated == test.separated, "separated=%t, want %t; lower gap %g mm", separated, test.separated, gap)
			if test.separated {
				require.Greater(t, gap, test.minimum)
			} else {
				require.LessOrEqual(t, gap, test.clearance+1e-6)
			}
			_, reverse, err := b.Exceeds(a, test.clearance)
			require.NoError(t, err)
			require.True(t, reverse == separated, "boundary separation must be symmetric")
		})
	}
}

func TestMeshSnapshotAndBounds(t *testing.T) {
	m, err := ballbearing.Build(t.Context(), decad.New(), ballbearing.DefaultParams())
	require.NoError(t, err)
	mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	vertices, triangles, bound := mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base()
	a, err := meshgap.New(vertices, triangles, bound)
	require.NoError(t, err)
	b := bearingMesh(t, ballbearing.DefaultParams(), r3.Vec{Z: 7})
	for i := range vertices {
		vertices[i].Z += 1000
	}
	triangles[0] = [3]int{}
	gap, separated, err := a.Exceeds(b, 0)
	require.NoError(t, err)
	require.False(t, separated, "mutating source slices must not change the captured geometry")
	require.Zero(t, gap)

	// A short gap inside the verified displacement envelopes cannot be declared loose.
	c := bearingMesh(t, ballbearing.DefaultParams(), r3.Vec{Z: 7 + bound/2})
	_, separated, err = a.Exceeds(c, 0)
	require.NoError(t, err)
	require.False(t, separated, "both real mesh bounds must be charged")
}

func TestInvalidMeshGapInputs(t *testing.T) {
	_, err := meshgap.New(nil, nil, 0)
	require.ErrorContains(t, err, "empty mesh")
	_, err = meshgap.New([]r3.Vec{{}}, [][3]int{{0, 0, 0}}, math.NaN())
	require.ErrorContains(t, err, "displacement bound")
	_, err = meshgap.New([]r3.Vec{{}}, [][3]int{{-1, 0, 0}}, 0)
	require.ErrorContains(t, err, "face 0")
	_, err = meshgap.New([]r3.Vec{{X: math.NaN()}}, [][3]int{{0, 0, 0}}, 0)
	require.ErrorContains(t, err, "vertex 0")
	a := bearingMesh(t, ballbearing.DefaultParams(), r3.Vec{})
	_, _, err = a.Exceeds(a, -1)
	require.ErrorContains(t, err, "clearance")
	_, _, err = a.Exceeds(meshgap.Mesh{}, 0)
	require.ErrorContains(t, err, "empty mesh")
}

func TestDegenerateTriangleHasFiniteGap(t *testing.T) {
	// A triangle with two coincident vertices has zero area; its distance to another mesh stays finite.
	sliver, err := meshgap.New([]r3.Vec{{}, {X: 1}}, [][3]int{{0, 0, 1}}, 0)
	require.NoError(t, err)
	other, err := meshgap.New([]r3.Vec{{X: 0.5, Z: 2}, {X: 3, Z: 2}, {X: 0.5, Y: 3, Z: 2}}, [][3]int{{0, 1, 2}}, 0)
	require.NoError(t, err)
	gap, separated, err := sliver.Exceeds(other, 3)
	require.NoError(t, err)
	require.False(t, separated)
	require.InDelta(t, 2, gap, 1e-5)
}

func TestPointDistanceToBearingBoundary(t *testing.T) {
	p := ballbearing.DefaultParams()
	m := bearingMesh(t, p, r3.Vec{})
	require.Positive(t, m.Bound())
	for _, test := range []struct {
		name  string
		point r3.Vec
		want  float64
	}{
		{"above_cap", r3.Vec{X: p.OuterDiameter / 2, Z: p.Width + 3}, 3},
		{"outside_rim", r3.Vec{X: p.OuterDiameter/2 + 2, Z: p.Width / 2}, 2},
		{"in_bore", r3.Vec{Z: p.Width / 2}, p.BoreDiameter / 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := m.PointDistance(test.point)
			require.NoError(t, err)
			require.InDelta(t, test.want, got, m.Bound()+1e-6)
		})
	}
	_, err := m.PointDistance(r3.Vec{X: math.Inf(1)})
	require.ErrorContains(t, err, "nonfinite")
	_, err = meshgap.Mesh{}.PointDistance(r3.Vec{})
	require.ErrorContains(t, err, "empty mesh")
}
