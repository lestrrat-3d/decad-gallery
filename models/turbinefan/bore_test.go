package turbinefan_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// boreLoop returns the loop of o whose vertices all lie within limit of the shaft axis.
func boreLoop(t *testing.T, loops [][]r3.Vec, limit float64) []r3.Vec {
	t.Helper()
	var found [][]r3.Vec
	for _, l := range loops {
		inside := true
		for _, v := range l {
			inside = inside && math.Hypot(v.X, v.Y) <= limit
		}
		if inside {
			found = append(found, l)
		}
	}
	require.Len(t, found, 1, "one loop is the bore")
	return found[0]
}

// TestTurbineFanBoreFit measures the D bore against the nominal shaft: every point of the bore lies BoreClearance outside
// the shaft's D section, the wall round the bore is at least 1.6 mm (1.65 mm through the socket floors at the 608ZZ bore,
// and exactly 1.6 to the set screw's nut slot), and the bore runs through both faces.
func TestTurbineFanBoreFit(t *testing.T) {
	eachCase(t, boreCases, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		p := c.params()
		dm := m.Dimensions
		rs, fs := p.ShaftDiameter/2, p.ShaftDiameter/2-p.FlatDepth
		for _, z := range []float64{8, 10, 15} {
			o := section(t, m.Hub, atZ(t, z))
			bound := o.OffSurface.Base() + 1e-6
			bore := boreLoop(t, o.Loops, dm.BoreRadius+0.1)
			lo, hi := math.Inf(1), math.Inf(-1)
			for _, v := range bore {
				// Outside the shaft's convex D section, the distance is at least the larger of the distances outside the
				// round and outside the flat's half plane (the flat faces -Y).
				gap := math.Max(math.Hypot(v.X, v.Y)-rs, -v.Y-fs)
				lo, hi = math.Min(lo, gap), math.Max(hi, gap)
			}
			require.GreaterOrEqual(t, lo, p.BoreClearance-bound, "z %g", z)
			require.LessOrEqual(t, hi, p.BoreClearance+bound, "z %g", z)
			// The wall: from the bore to every edge of every other loop.
			wall := math.Inf(1)
			for _, l := range o.Loops {
				if len(l) == len(bore) && l[0] == bore[0] {
					continue
				}
				for i := range l {
					a, b := l[i], l[(i+1)%len(l)]
					for _, v := range bore {
						wall = math.Min(wall, distanceToLoop(v, []r3.Vec{a, b}))
					}
				}
			}
			require.GreaterOrEqual(t, wall, turbinefan.PressWall-bound, "z %g: wall round the bore", z)
			if z == 10 {
				require.InDelta(t, handcrankfan.BladeSocketStart-dm.BoreRadius, wall, 0.05, "z 10: the least wall is the one to the socket floors")
			}
		}
		// The wall between the bore's flat and the set screw's nut slot, read across the flat beside the screw's hole.
		for _, z := range []float64{1.2, 3.0, turbinefan.SetScrewZ} {
			o := section(t, m.Hub, atZ(t, z))
			bound := o.OffSurface.Base() + 1e-3
			for _, x := range []float64{-1.45, 1.45} {
				ys := crossingsAtX(o, x, -(dm.BoreFlat + 2.5), -(dm.BoreFlat - 0.5))
				require.Len(t, ys, 2, "z %g, x %g", z, x)
				require.InDelta(t, -(dm.BoreFlat + turbinefan.PressWall), ys[0], bound, "z %g: the slot's inner face", z)
				require.InDelta(t, -dm.BoreFlat, ys[1], bound, "z %g: the bore's flat", z)
				require.GreaterOrEqual(t, ys[1]-ys[0], turbinefan.PressWall-2*bound, "z %g: wall to the nut slot", z)
			}
		}
		// The bore runs through: both faces' sections hold the bore loop.
		for _, z := range []float64{0.75, turbinefan.HubLength - 0.75} {
			bore := boreLoop(t, section(t, m.Hub, atZ(t, z)).Loops, dm.BoreRadius+0.6)
			require.NotEmpty(t, bore, "z %g", z)
		}
	})
}

// TestTurbineFanEdgeFinishing checks the 0.5 mm chamfer on both faces: a section 0.25 mm inside a face is 0.25 mm short
// of the outer circle, and at the 608ZZ bore, whose D loop is long, 0.25 mm wider than the bore.
func TestTurbineFanEdgeFinishing(t *testing.T) {
	eachCase(t, []string{"605ZZ", "608ZZ"}, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		dm := m.Dimensions
		chamfered := dm.BoreLoop >= 20
		for _, z := range []float64{0.25, turbinefan.HubLength - 0.25} {
			o := section(t, m.Hub, atZ(t, z))
			bound := o.OffSurface.Base() + 1e-3
			outer := 0
			for _, l := range o.Loops {
				far := 0.0
				for _, v := range l {
					far = math.Max(far, math.Hypot(v.X, v.Y))
				}
				if far <= 13 {
					continue
				}
				outer++
				for _, v := range l {
					require.InDelta(t, turbinefan.HubRadius-0.25, math.Hypot(v.X, v.Y), bound, "z %g: outer circle", z)
				}
			}
			require.Equal(t, 1, outer, "z %g: one outer loop", z)
			bore := boreLoop(t, o.Loops, dm.BoreRadius+0.6)
			largest, least := 0.0, math.Inf(1)
			for _, v := range bore {
				largest = math.Max(largest, math.Hypot(v.X, v.Y))
				least = math.Min(least, v.Y)
			}
			grow := 0.0
			if chamfered {
				grow = 0.25
			}
			require.InDelta(t, dm.BoreRadius+grow, largest, bound, "z %g: bore radius", z)
			require.InDelta(t, -(dm.BoreFlat + grow), least, bound, "z %g: bore flat", z)
		}
	})
}
