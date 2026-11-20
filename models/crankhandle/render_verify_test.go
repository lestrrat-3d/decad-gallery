package crankhandle_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestEveryBodyVerifyAll tessellates every body of the default and the 6082RS crank handle under VerifyAll: the printed
// parts at 0.05 mm and the bought bolts and nuts at 0.01 mm. The renderer meshes them without verification.
func TestEveryBodyVerifyAll(t *testing.T) {
	for name, p := range map[string]crankhandle.Params{
		"default": crankhandle.DefaultParams(), "6082RS": crankhandle.Params6082RS(),
	} {
		t.Run(name, func(t *testing.T) {
			m, err := crankhandle.Build(t.Context(), crankhandle.NewDocument(), p)
			require.NoError(t, err)
			printed := map[string]*decad.Body{"arm": m.Arm, "pin": m.Pin, "handle": m.Handle}
			hardware := map[string]*decad.Body{"bolt": m.Bolt.Body, "nut": m.Nut.Body}
			for i := range m.ClampBolts {
				hardware["clamp-bolt"+string(rune('1'+i))] = m.ClampBolts[i].Body
				hardware["clamp-nut"+string(rune('1'+i))] = m.ClampNuts[i].Body
			}
			for i := range m.CrossBolts {
				hardware["cross-bolt"+string(rune('1'+i))] = m.CrossBolts[i].Body
				hardware["cross-nut"+string(rune('1'+i))] = m.CrossNuts[i].Body
			}
			for part, body := range printed {
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
