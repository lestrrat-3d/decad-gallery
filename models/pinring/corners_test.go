package pinring_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/models/pinring"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestCorners applies "Corners and roots" to every parameter branch. The ring prints with its axis along +Z.
func TestCorners(t *testing.T) {
	for name, p := range map[string]pinring.Params{
		"default":       pinring.DefaultParams(),
		"no clearance":  {RodRadius: 2.5, Clearance: 0, Width: 5, Thickness: 5},
		"wide and thin": {RodRadius: 4, Clearance: 0.3, Width: 7, Thickness: 2},
	} {
		t.Run(name, func(t *testing.T) {
			m, err := pinring.Build(t.Context(), decad.New(), p)
			require.NoError(t, err)
			fs, err := cornercheck.Check(t.Context(), m.Body, r3.Vec{Z: 1}, nil)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		})
	}
}
