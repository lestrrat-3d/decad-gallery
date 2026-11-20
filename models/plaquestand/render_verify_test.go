package plaquestand_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/plaquestand"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestEveryBodyVerifyAll tessellates every body of the default, minimum and maximum stands under VerifyAll: printed
// parts and the plaque at 0.05 mm, bought hardware at 0.01 mm. The renderer meshes them without verification.
func TestEveryBodyVerifyAll(t *testing.T) {
	for _, tc := range cases() {
		t.Run(tc.name, func(t *testing.T) {
			m, err := plaquestand.Build(t.Context(), decad.New(), tc.params)
			require.NoError(t, err)
			bodies := map[string]*decad.Body{
				"bracket-1": m.Brackets[0], "bracket-2": m.Brackets[1], "rail": m.Rail, "plaque": m.Plaque,
			}
			hardware := map[string]*decad.Body{
				"bolt-1": m.Bolts[0].Body, "bolt-2": m.Bolts[1].Body, "nut-1": m.Nuts[0].Body, "nut-2": m.Nuts[1].Body,
			}
			for part, body := range bodies {
				_, err := body.Tessellate(t.Context(), units.Millimeters(.05), decad.WithVerification(decad.VerifyAll))
				require.NoError(t, err, part)
			}
			for part, body := range hardware {
				_, err := body.Tessellate(t.Context(), units.Millimeters(.01), decad.WithVerification(decad.VerifyAll))
				require.NoError(t, err, part)
			}
		})
	}
}
