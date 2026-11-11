package pinring_test

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/pinring"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func requireFullyConstrained(t *testing.T, s *sketch.Sketch) {
	t.Helper()
	report := s.Verify(t.Context())
	require.True(t, report.Status == sketch.FullyConstrained, "status %s", report.Status)
	require.Zero(t, report.DOF)
	require.Zero(t, len(report.Redundant))
	require.Zero(t, len(report.Conflicts))
	require.NoError(t, report.Check())
}

func TestPinRingDefaultDimensions(t *testing.T) {
	p := pinring.DefaultParams()
	require.InDelta(t, 3.0, p.InnerRadius(), 1e-12)
	require.InDelta(t, 8.0, p.OuterRadius(), 1e-12)
	require.InDelta(t, 5.0, p.Thickness, 1e-12)
}

func TestPinRingBuild(t *testing.T) {
	tests := []struct {
		name   string
		params pinring.Params
	}{
		{"default", pinring.DefaultParams()},
		{"no clearance", pinring.Params{RodRadius: 2.5, Clearance: 0, Width: 5, Thickness: 5}},
		{"wide and thin", pinring.Params{RodRadius: 4, Clearance: 0.3, Width: 7, Thickness: 2}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := decad.New()
			m, err := pinring.Build(t.Context(), doc, tc.params)
			require.NoError(t, err)
			require.Equal(t, 1, len(doc.Bodies()))
			require.True(t, m.Body.IsSolid())
			requireFullyConstrained(t, m.Profile)

			inner, outer := tc.params.InnerRadius(), tc.params.OuterRadius()
			box, err := m.Body.Bounds()
			require.NoError(t, err)
			tol := box.Bound.Base() + 1e-8
			require.InDelta(t, -outer, box.Min.X, tol)
			require.InDelta(t, outer, box.Max.X, tol)
			require.InDelta(t, -outer, box.Min.Y, tol)
			require.InDelta(t, outer, box.Max.Y, tol)
			require.InDelta(t, 0, box.Min.Z, tol)
			require.InDelta(t, tc.params.Thickness, box.Max.Z, tol)

			volume, err := m.Body.Volume()
			require.NoError(t, err)
			want := math.Pi * (outer*outer - inner*inner) * tc.params.Thickness
			require.InDelta(t, want, volume.Value.Base(), volume.Bound.Base()+1e-6)

			counts := map[decad.SurfaceKind]int{}
			for _, face := range m.Body.Faces() {
				counts[face.Surface().Kind()]++
			}
			require.Equal(t, 2, counts[decad.KindPlane], "two flat faces")
			require.Equal(t, 2, counts[decad.KindCylinder], "bore and outer wall")

			mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, mesh.BoundaryVerified())
			require.True(t, mesh.VolumeVerified())
		})
	}
}

func TestPinRingClearsRod(t *testing.T) {
	p := pinring.DefaultParams()
	require.InDelta(t, p.RodRadius+0.5, p.InnerRadius(), 1e-12, "0.5 mm radial clearance on the rod")
}

func TestPinRingPlacement(t *testing.T) {
	frame, err := r3.NewFrame(r3.NewVec(10, 5, 0), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	p := pinring.DefaultParams()
	p.Placement = &frame
	m, err := pinring.Build(t.Context(), decad.New(), p)
	require.NoError(t, err)
	box, err := m.Body.Bounds()
	require.NoError(t, err)
	tol := box.Bound.Base() + 1e-8
	// The ring's +Z maps to world +X.
	require.InDelta(t, 10, box.Min.X, tol)
	require.InDelta(t, 15, box.Max.X, tol)
	require.InDelta(t, 5-8, box.Min.Y, tol)
	require.InDelta(t, 5+8, box.Max.Y, tol)
}

func TestPinRingRejectsInvalidInputs(t *testing.T) {
	doc := decad.New()
	for name, p := range map[string]pinring.Params{
		"rod":       {RodRadius: 0, Clearance: .5, Width: 5, Thickness: 5},
		"clearance": {RodRadius: 2.5, Clearance: -1, Width: 5, Thickness: 5},
		"width":     {RodRadius: 2.5, Clearance: .5, Width: 0, Thickness: 5},
		"thickness": {RodRadius: 2.5, Clearance: .5, Width: 5, Thickness: 0},
		"placement": {RodRadius: 2.5, Clearance: .5, Width: 5, Thickness: 5, Placement: &r3.Frame{}},
	} {
		_, err := pinring.Build(t.Context(), doc, p)
		require.Error(t, err, name)
	}
	require.Zero(t, len(doc.Bodies()))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := pinring.Build(ctx, doc, pinring.DefaultParams())
	require.ErrorIs(t, err, context.Canceled)
	_, err = pinring.Build(t.Context(), nil, pinring.DefaultParams())
	require.ErrorContains(t, err, "document")
	_, err = pinring.Build(nil, doc, pinring.DefaultParams()) //nolint:staticcheck
	require.ErrorContains(t, err, "context")
}
