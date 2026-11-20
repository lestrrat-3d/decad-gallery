package turbinefan_test

// CI: isolated-tests

import (
	"fmt"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestEveryBodyVerifyAll tessellates every body of the default fan, the 608ZZ fan and the fan matched to the default
// clutch under VerifyAll: printed parts at 0.05 mm and bought hardware at 0.01 mm. The renderer meshes them without
// verification.
func TestEveryBodyVerifyAll(t *testing.T) {
	clutchParams, err := turbinefan.ParamsForClutch(onewayclutch.DefaultParams())
	require.NoError(t, err)
	for name, p := range map[string]turbinefan.Params{
		"default": turbinefan.DefaultParams(), "608ZZ": turbinefan.Params608ZZ(), "clutch": clutchParams,
	} {
		t.Run(name, func(t *testing.T) {
			m, err := turbinefan.Build(t.Context(), turbinefan.NewDocument(), p)
			require.NoError(t, err)
			printed := map[string]*decad.Body{"hub": m.Hub}
			hardware := map[string]*decad.Body{}
			if m.SetScrew != nil {
				hardware["setscrew-bolt"], hardware["setscrew-nut"] = m.SetScrew.Body, m.SetScrewNut.Body
			}
			for k := range m.AdapterBolts {
				if m.AdapterBolts[k] == nil {
					continue
				}
				hardware[fmt.Sprintf("adapterbolt-%d", k+1)] = m.AdapterBolts[k].Body
				hardware[fmt.Sprintf("adapternut-%d", k+1)] = m.AdapterNuts[k].Body
			}
			for k := range m.Blades {
				printed[fmt.Sprintf("blade-%d", k+1)] = m.Blades[k]
				hardware[fmt.Sprintf("bolt-%d", k+1)] = m.BladeBolts[k].Body
				hardware[fmt.Sprintf("nut-%d", k+1)] = m.BladeNuts[k].Body
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
