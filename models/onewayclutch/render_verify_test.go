package onewayclutch_test

// CI: isolated-tests

import (
	"fmt"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestEveryBodyVerifyAll tessellates every body of the clutch for each shaft preset and for both freewheel directions,
// and of the default clutch's adapter, under VerifyAll: printed parts at 0.04 mm and bought hardware at 0.01 mm. The
// renderer meshes them without verification.
func TestEveryBodyVerifyAll(t *testing.T) {
	type tc struct {
		name    string
		p       onewayclutch.Params
		adapter bool
	}
	ccw := onewayclutch.DefaultParams()
	ccw.Freewheel = onewayclutch.CounterClockwise
	cases := []tc{{"default-adapter", onewayclutch.DefaultParams(), true}, {"counterclockwise", ccw, false}}
	for _, shaft := range onewayclutch.Shafts() {
		p := onewayclutch.DefaultParams()
		p.Shaft = shaft.Name
		cases = append(cases, tc{"shaft-" + shaft.Name, p, false})
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc := onewayclutch.NewDocument()
			m, err := onewayclutch.Build(t.Context(), doc, c.p)
			require.NoError(t, err)
			printed := map[string]*decad.Body{"hub": m.Hub, "ring": m.Ring, "back plate": m.BackPlate, "cover": m.Cover}
			hardware := map[string]*decad.Body{}
			for i := range m.Bolts {
				hardware[fmt.Sprintf("bolt %d", i+1)] = m.Bolts[i].Body
			}
			for i := range m.Nuts {
				hardware[fmt.Sprintf("nut %d", i+1)] = m.Nuts[i].Body
			}
			if c.adapter {
				a, err := onewayclutch.BuildAdapter(t.Context(), doc, c.p)
				require.NoError(t, err)
				printed["adapter"] = a.Body
				for i := range a.Bolts {
					hardware[fmt.Sprintf("mount bolt %d", i+1)] = a.Bolts[i].Body
				}
				for i := range a.Nuts {
					hardware[fmt.Sprintf("mount nut %d", i+1)] = a.Nuts[i].Body
				}
			}
			for part, body := range printed {
				_, err := body.Tessellate(t.Context(), units.Millimeters(.04), decad.WithVerification(decad.VerifyAll))
				require.NoError(t, err, part)
			}
			for part, body := range hardware {
				_, err := body.Tessellate(t.Context(), units.Millimeters(.01), decad.WithVerification(decad.VerifyAll))
				require.NoError(t, err, part)
			}
		})
	}
}
