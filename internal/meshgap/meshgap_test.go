package meshgap_test

// CI: isolated-tests

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

func TestDistantSliverDoesNotHideClearance(t *testing.T) {
	// The sliver and the first triangle came from a verified cube hub and nut.
	// The other triangle makes the whole-mesh boxes overlap in X and Y.
	a := []r3.Vec{
		{X: 5.30542337425491, Y: 10.203062423601535},
		{X: 4.965175051892669, Y: 10.367106543423194, Z: 0.3466101864841946},
		{X: 5.30542337425491, Y: 10.19736466348172, Z: 0.3409350962925137},
	}
	b := []r3.Vec{
		{X: -8.700000000000001, Y: 1.74, Z: -1.0650903202395404e-16},
		{X: -9.028743864943385, Y: 2.3094010767585034, Z: 1.831469322699649e-28},
		{X: -8.700000000000001, Y: 1.74, Z: 1.3799060949475048e-28},
		{X: 5, Y: 10, Z: 2}, {X: 6, Y: 10, Z: 2}, {X: 5, Y: 11, Z: 2},
	}
	first, err := meshgap.New(a, [][3]int{{0, 1, 2}}, .026)
	require.NoError(t, err)
	second, err := meshgap.New(b, [][3]int{{0, 1, 2}, {3, 4, 5}}, .0006)
	require.NoError(t, err)
	for _, pair := range [][2]meshgap.Mesh{{first, second}, {second, first}} {
		gap, separated, err := pair[0].Exceeds(pair[1], .05)
		require.NoError(t, err)
		require.True(t, separated, "the distant sliver must not hide clearance")
		require.Greater(t, gap, .05)
	}
	copy(b[3:], a)
	touching, err := meshgap.New(b, [][3]int{{0, 1, 2}, {3, 4, 5}}, .0006)
	require.NoError(t, err)
	_, separated, err := first.Exceeds(touching, .05)
	require.NoError(t, err)
	require.False(t, separated, "a distant sliver must not hide contact")
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
