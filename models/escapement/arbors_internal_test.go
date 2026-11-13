package escapement

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func testBuilder(t *testing.T) *builder {
	t.Helper()
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	return newBuilder(t.Context(), NewDocument(), g)
}

// inside reports whether the point is inside the solid. It cuts the body with the horizontal plane through the point
// and tests the point against the section's polygons (even-odd), which works for every body, a chamfered Boolean
// result included.
func inside(t *testing.T, body *decad.Body, x, y, z float64) bool {
	t.Helper()
	f, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	// A faceted Boolean result refuses a chord below its own bound, so retry coarser.
	var o decad.SectionOutline
	for _, tol := range []float64{0.005, 0.02, 0.05} {
		o, err = body.SectionOutline(context.Background(), f, units.Millimeters(tol))
		if err == nil {
			break
		}
	}
	require.NoError(t, err)
	n := 0
	for _, l := range o.Loops {
		if inPoly(loopPoints(l), Pt{x, y}) {
			n++
		}
	}
	return n%2 == 1
}

func volume(t *testing.T, body *decad.Body) float64 {
	t.Helper()
	v, err := body.Volume()
	require.NoError(t, err)
	return v.Value.Base()
}

// segmentArea is the area of the circular segment of a circle of radius r beyond the chord x = flat; zero when the
// flat lies outside the circle.
func segmentArea(r, flat float64) float64 {
	if r <= flat {
		return 0
	}
	return r*r*math.Acos(flat/r) - flat*math.Sqrt(r*r-flat*flat)
}

// sectionRadius returns the radius of the meridian pts at height z.
func sectionRadius(pts meridianPoints, z float64) float64 {
	for i, p := range pts {
		q := pts[(i+1)%len(pts)]
		if p[0] == q[0] && p[0] > 0 && z > math.Min(p[1], q[1]) && z < math.Max(p[1], q[1]) {
			return p[0]
		}
	}
	return 0
}

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
			// Volume: the revolve minus, over each flat's span, the circular segment of each section beyond its flat.
			want := 0.0
			for i, p := range c.pts {
				q := c.pts[(i+1)%len(c.pts)]
				if p[0] != q[0] {
					continue
				}
				want += math.Pi * p[0] * p[0] * (q[1] - p[1])
				for _, sp := range c.spans {
					lo, hi := math.Max(p[1], sp.Z0), math.Min(q[1], sp.Z1)
					if hi > lo {
						want -= segmentArea(p[0], sp.X) * (hi - lo)
					}
				}
			}
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

// loopPoints returns the XY points of a section loop.
func loopPoints(l []r3.Vec) []Pt {
	out := make([]Pt, len(l))
	for i, v := range l {
		out[i] = Pt{v.X, v.Y}
	}
	return out
}

// outerLoop returns the XY points of the loop that reaches farthest from the origin.
func outerLoop(loops [][]r3.Vec) []Pt {
	var best []Pt
	far := -1.0
	for _, l := range loops {
		pts := loopPoints(l)
		m := 0.0
		for _, p := range pts {
			m = math.Max(m, p.Len())
		}
		if m > far {
			best, far = pts, m
		}
	}
	return best
}
