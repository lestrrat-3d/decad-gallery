package compressionspring_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/compressionspring"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestInstalledLeeSpring(t *testing.T) {
	p := compressionspring.DefaultParams()
	m, err := compressionspring.Build(t.Context(), decad.New(), p)
	require.NoError(t, err)
	require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), m.Profile))
	box, err := m.Body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, p.InstalledLength, box.Max.Y-box.Min.Y, 0.02)
	require.InDelta(t, p.Spec.OutsideDiameter, box.Max.X-box.Min.X, 0.02)
	require.InDelta(t, 7*p.Spec.Rate, m.Force, 1e-10)
	require.False(t, math.IsNaN(m.Pitch) || math.IsInf(m.Pitch, 0))
	mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(0.05))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func TestButtonReturnSpring(t *testing.T) {
	for _, tc := range []struct {
		name          string
		length, force float64
	}{
		{"home", 55.50, 2.40}, {"pressed", 18.94, 13.368},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := compressionspring.LeeLC026D16S()
			m, err := compressionspring.Build(t.Context(), decad.New(), compressionspring.Params{Spec: spec, InstalledLength: tc.length})
			require.NoError(t, err)
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), m.Profile))
			box, err := m.Body.Bounds()
			require.NoError(t, err)
			require.InDelta(t, tc.length, box.Max.Y-box.Min.Y, .02)
			require.InDelta(t, 7.62, box.Max.X-box.Min.X, .02)
			require.InDelta(t, tc.force, m.Force, 1e-10)
			mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(.05))
			require.NoError(t, err)
			require.True(t, mesh.BoundaryVerified())
			require.True(t, mesh.VolumeVerified())
		})
	}
}

func TestReturnSpringLimits(t *testing.T) {
	spec := compressionspring.LeeLC026D16S()
	p := compressionspring.Params{Spec: spec, InstalledLength: spec.FreeLength}
	m, err := compressionspring.Build(t.Context(), decad.New(), p)
	require.NoError(t, err)
	require.Zero(t, m.Force)
	require.InDelta(t, (63.50-.66)/18.33, m.Pitch, 1e-12)

	for _, bad := range []float64{math.NaN(), math.Inf(1), spec.FreeLength + 1, spec.SolidHeight, 12.2} {
		p.InstalledLength = bad
		_, err := compressionspring.Build(t.Context(), decad.New(), p)
		require.Error(t, err)
	}
}

func TestClickIndexerSpring(t *testing.T) {
	spec := compressionspring.LeeLC022D03S()
	require.Equal(t, "Lee Spring LC 022D 03 S", spec.Name)
	for _, tc := range []struct {
		name          string
		length, force float64
	}{
		{"rest", 8.0, 0.736 * (15.88 - 8.0)}, {"pressed", 6.0, 0.736 * (15.88 - 6.0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, err := compressionspring.Build(t.Context(), decad.New(), compressionspring.Params{Spec: spec, InstalledLength: tc.length})
			require.NoError(t, err)
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), m.Profile))
			box, err := m.Body.Bounds()
			require.NoError(t, err)
			require.InDelta(t, tc.length, box.Max.Y-box.Min.Y, .02)
			require.InDelta(t, 7.62, box.Max.X-box.Min.X, .02)
			require.InDelta(t, tc.force, m.Force, 1e-10)
		})
	}
}

func TestLesjoforsStock2353Values(t *testing.T) {
	spec := compressionspring.LesjoforsStock2353()
	require.Equal(t, "Lesjofors Cat.no 2353", spec.Name)
	require.Equal(t, 0.60, spec.WireDiameter)
	require.Equal(t, 15.0, spec.FreeLength)
	require.Equal(t, 6.3, spec.TotalTurns)
	require.Equal(t, 0.86, spec.Rate)
	require.Zero(t, spec.ActiveTurns, "the catalogue does not publish active coils")
	// Computed, not published: Dm + wire, wire x total coils, rate x (free length - solid height).
	require.InDelta(t, 6.85+0.60, spec.OutsideDiameter, 1e-12)
	require.InDelta(t, 0.60*6.3, spec.SolidHeight, 1e-12)
	require.InDelta(t, 0.86*(15-3.78), spec.LoadAtSolid, 5e-3)
}

func TestLesjoforsStock2353Installed(t *testing.T) {
	spec := compressionspring.LesjoforsStock2353()
	for _, tc := range []struct {
		name          string
		length, force float64
	}{
		{"rest", 8.0, 0.86 * (15 - 8.0)}, {"pressed", 6.0, 0.86 * (15 - 6.0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, err := compressionspring.Build(t.Context(), decad.New(), compressionspring.Params{Spec: spec, InstalledLength: tc.length})
			require.NoError(t, err)
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), m.Profile))
			box, err := m.Body.Bounds()
			require.NoError(t, err)
			require.InDelta(t, tc.length, box.Max.Y-box.Min.Y, .02)
			require.InDelta(t, 7.45, box.Max.X-box.Min.X, .02)
			require.InDelta(t, tc.force, m.Force, 1e-10)
		})
	}
}

func TestSpecActiveTurnsRule(t *testing.T) {
	for _, tc := range []struct {
		name    string
		active  float64
		wantErr bool
	}{
		{"not published", 0, false},
		{"equal to total", 6.3, false},
		{"below total", 4.3, false},
		{"above total", 6.4, true},
		{"negative", -1, true},
		{"NaN", math.NaN(), true},
		{"infinite", math.Inf(1), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := compressionspring.LesjoforsStock2353()
			spec.ActiveTurns = tc.active
			_, err := compressionspring.Build(t.Context(), decad.New(), compressionspring.Params{Spec: spec, InstalledLength: 8})
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
