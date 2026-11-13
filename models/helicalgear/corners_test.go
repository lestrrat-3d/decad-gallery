package helicalgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/models/helicalgear"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestCorners applies "Corners and roots" to the gear body in its print orientation, axis +Z.
func TestCorners(t *testing.T) {
	bored := helicalgear.DefaultParams()
	bored.BoreDiameter = 3
	for name, p := range map[string]helicalgear.Params{
		"default": helicalgear.DefaultParams(),
		"bored":   bored,
	} {
		t.Run(name, func(t *testing.T) {
			body, err := helicalgear.Build(t.Context(), decad.New(), p)
			require.NoError(t, err)
			// Gear teeth are exempt: tips and roots follow the involute and the gear train rules. The hub, web and
			// bore are outside this band.
			// The root circle lies about 0.06 mm under pitch - 1.25 m, so the band reaches 0.15 mm under it.
			pitch := p.Module * float64(p.Teeth) / 2
			teeth := cornercheck.Exempt{
				Reason: "gear teeth",
				Near:   cornercheck.Annulus(r3.Vec{}, r3.Vec{Z: 1}, pitch-1.25*p.Module-0.15, pitch+p.Module+0.05),
			}
			fs, err := cornercheck.Check(t.Context(), body, r3.Vec{Z: 1}, []cornercheck.Exempt{teeth})
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		})
	}
}
