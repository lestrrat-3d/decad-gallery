package escapement

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

func TestArbors(t *testing.T) {
	b := testBuilder(t)
	cases := []struct {
		name  string
		build func() (*decad.Body, error)
		x, y  float64
		pts   meridianPoints
		spans []flatSpan
		seats [][2]float64 // z ranges of the round 5.0 bearing seats
	}{
		{"escape arbor", b.buildEscapeArbor, 0, 0, escapeArborPoints, escapeArborFlats, [][2]float64{{-5, 0}, {28, 33}}},
		{"pallet arbor", b.buildPalletArbor, 0, b.g.P.Y, palletArborPoints, palletArborFlats,
			[][2]float64{{-5, 0}, {28, 33}}},
		{"great wheel arbor", b.buildGreatWheelArbor, b.g.G.X, b.g.G.Y, greatArborPoints, greatArborFlats,
			[][2]float64{{-5, 0}, {28, 33}}},
		{"suspension pin", b.buildSuspensionPin, 0, b.g.S.Y, pinPoints, pinFlats, [][2]float64{{40.05, 45.05}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body, err := c.build()
			require.NoError(t, err)
			require.True(t, body.IsSolid())
			// Volume: the revolve minus, over each flat and its ramp, the circular segment of each section beyond the flat.
			want := meridianVolume(c.pts, c.spans)
			require.InDelta(t, want, volume(t, body), 0.01*want)
			for _, sp := range c.spans {
				mid := (sp.Z0 + sp.Z1) / 2
				r := sectionRadius(c.pts, mid)
				require.Greater(t, r, sp.X, "the flat at z %v cuts a section", mid)
				require.False(t, inside(t, body, c.x+(sp.X+r)/2, c.y, mid), "flat cut at z %v", mid)
				require.True(t, inside(t, body, c.x+sp.X-0.15, c.y, mid), "shaft kept at z %v", mid)
				require.True(t, inside(t, body, c.x-r+0.1, c.y, mid), "round side kept at z %v", mid)
				require.False(t, inside(t, body, c.x-r-0.05, c.y, mid), "section radius at z %v", mid)
			}
			// A ramp is a plane at 30 degrees to the axis: 0.3 mm past the end of the flat the cut plane stands
			// 0.3 tan 30 degrees further out, and the shaft is kept below it and cut above it.
			for _, sp := range c.spans {
				for _, ramp := range []struct {
					on bool
					z  float64
				}{{sp.RampEnd, sp.Z1 + 0.3}, {sp.RampStart, sp.Z0 - 0.3}} {
					if !ramp.on {
						continue
					}
					plane := sp.X + 0.3*math.Tan(coneHalfAngle)
					require.Greater(t, sectionRadius(c.pts, ramp.z), plane+0.05, "the ramp at z %v cuts a section", ramp.z)
					require.True(t, inside(t, body, c.x+plane-0.05, c.y, ramp.z), "shaft kept under the ramp at z %v", ramp.z)
					require.False(t, inside(t, body, c.x+plane+0.05, c.y, ramp.z), "ramp cut at z %v", ramp.z)
				}
			}
			for _, seat := range c.seats {
				mid := (seat[0] + seat[1]) / 2
				// The bearing seat is the full 5.0 mm round shaft: no flat and no clearance.
				require.True(t, inside(t, body, c.x+2.45, c.y, mid), "seat radius at z %v", mid)
				require.False(t, inside(t, body, c.x+2.55, c.y, mid), "seat radius at z %v", mid)
				require.True(t, inside(t, body, c.x, c.y+2.45, mid))
				require.False(t, inside(t, body, c.x, c.y+2.55, mid))
			}
		})
	}
	// Every arbor sketch is fully constrained (profile() enforces it).
	require.NotEmpty(t, b.sketches)
}
