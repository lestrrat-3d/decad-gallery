package compressionspring_test

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
