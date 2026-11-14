package escapement

import (
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestCorners applies "Corners and roots" to every printed part in its print orientation, axis +Z. Bearings, bolts
// and nuts are purchased parts. The exemptions match "Retained sharp corners" in instructions.md.
func TestCorners(t *testing.T) {
	_, m, err := sharedDefault(t)
	require.NoError(t, err)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	z := r3.Vec{Z: 1}
	exempt := map[string][]cornercheck.Exempt{
		// The escape wheel's teeth: the root fillets and the tips follow the tooth generator.
		EscapeWheelName: {{
			Reason: "escape wheel teeth",
			Near:   cornercheck.Annulus(r3.Vec{}, z, RootRadius-0.2, TipRadius+0.1),
		}},
		// The two pallet corners that meet the escape wheel's teeth are engagement edges.
		AnchorName: {
			{Reason: "entry pallet corner", Near: cornercheck.Disc(r3.Vec{X: g.P.X + g.FE.X, Y: g.P.Y + g.FE.Y}, z, 0.5)},
			{Reason: "exit pallet corner", Near: cornercheck.Disc(r3.Vec{X: g.P.X + g.FX.X, Y: g.P.Y + g.FX.Y}, z, 0.5)},
		},
	}
	for _, nb := range m.Named() {
		if strings.HasSuffix(nb.Name, "Bearing") || strings.HasPrefix(nb.Name, "Bolt ") || strings.HasPrefix(nb.Name, "Nut ") {
			continue
		}
		t.Run(nb.Name, func(t *testing.T) {
			fs, err := cornercheck.Check(t.Context(), nb.Body, z, exempt[nb.Name])
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		})
	}
}
