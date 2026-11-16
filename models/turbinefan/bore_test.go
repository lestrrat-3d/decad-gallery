package turbinefan_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestTurbineFanBoreFit measures the D bore against the nominal shaft: every point of the bore lies BoreClearance
// outside the shaft's D section, the bore's floor is 1 mm thick apart from the slit's round end, and the wall round
// the bore is at least 1.6 mm (1.65 mm through the socket floors at the 608ZZ bore). The slit joins the bore to the
// outline, so the bore is read as the points within r_b of the axis outside the slit.
func TestTurbineFanBoreFit(t *testing.T) {
	eachCase(t, boreCases, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		p := c.params()
		rs, fs := p.ShaftDiameter/2, p.ShaftDiameter/2-p.FlatDepth
		half := turbinefan.SlitWidth / 2
		for _, z := range []float64{5, 10, 15} {
			o := section(t, m.Hub, atZ(t, z))
			bound := o.OffSurface.Base() + 1e-6
			inSlit := func(v r3.Vec) bool { return math.Abs(v.X) <= half+bound && v.Y < 0 }
			onBore := func(v r3.Vec) bool { return math.Hypot(v.X, v.Y) < m.Dimensions.BoreRadius+0.1 && !inSlit(v) }
			var bore []r3.Vec
			for _, l := range o.Loops {
				for _, v := range l {
					if onBore(v) {
						bore = append(bore, v)
					}
				}
			}
			require.NotEmpty(t, bore, "z %g: bore", z)
			lo, hi := math.Inf(1), math.Inf(-1)
			for _, v := range bore {
				// Outside the shaft's convex D section, the distance is at least the larger of the distances outside the
				// round and outside the flat's half plane.
				gap := math.Max(math.Hypot(v.X, v.Y)-rs, v.Y-fs)
				lo, hi = math.Min(lo, gap), math.Max(hi, gap)
			}
			require.GreaterOrEqual(t, lo, p.BoreClearance-bound, "z %g", z)
			require.LessOrEqual(t, hi, p.BoreClearance+bound, "z %g", z)
			// The wall: from the bore to every edge of the section that is neither the bore nor the slit.
			wall := math.Inf(1)
			for _, l := range o.Loops {
				for i := range l {
					a, b := l[i], l[(i+1)%len(l)]
					if onBore(a) || onBore(b) || inSlit(a) || inSlit(b) {
						continue
					}
					for _, v := range bore {
						wall = math.Min(wall, distanceToLoop(v, []r3.Vec{a, b}))
					}
				}
			}
			require.GreaterOrEqual(t, wall, turbinefan.PressWall-bound, "z %g: wall round the bore", z)
			if z == 10 {
				// Through the socket floors, at radius 5.7 from the axis.
				want := handcrankfan.BladeSocketStart - m.Dimensions.BoreRadius
				require.GreaterOrEqual(t, want, turbinefan.PressWall-1e-9, "designed wall to the socket floors")
				require.InDelta(t, want, wall, 0.05, "z 10: the least wall is the one to the socket floors")
			}
		}
		// At the front face the floor closes the bore everywhere but the slit's round end.
		o := section(t, m.Hub, atZ(t, turbinefan.HubLength-0.25))
		bound := o.OffSurface.Base() + 1e-3
		for _, l := range o.Loops {
			for _, v := range l {
				if math.Abs(v.X) <= half+bound {
					continue
				}
				require.Greater(t, math.Hypot(v.X, v.Y), m.Dimensions.BoreRadius+0.5, "bore through the floor at %v", v)
			}
		}
	})
}

// TestTurbineFanSlit checks that the slit closes the bore: low in the hub one loop of its section runs from the bore
// down the slit's two walls, SlitWidth apart, to the tail's end arc, so tightening the clamp bolt narrows the bore; and
// in the floor the slit ends in its full round SlitInBore past the bore's wall.
func TestTurbineFanSlit(t *testing.T) {
	eachCase(t, boreCases, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		dm := m.Dimensions
		half := turbinefan.SlitWidth / 2
		o := section(t, m.Hub, atZ(t, 1))
		bound := o.OffSurface.Base() + 1e-3
		joined := false
		for _, l := range o.Loops {
			bore, end := false, false
			for _, v := range l {
				bore = bore || math.Hypot(v.X, v.Y) < dm.BoreRadius+0.1
				end = end || v.Y < -dm.Reach+1
			}
			joined = joined || bore && end
		}
		require.True(t, joined, "one loop holds the bore and the end arc")
		gap := math.Inf(1)
		for _, l := range o.Loops {
			for _, v := range l {
				if v.Y < -dm.Reach+1.5 || v.Y > -dm.BoreRadius-0.3 {
					continue
				}
				gap = math.Min(gap, math.Abs(v.X))
			}
		}
		require.InDelta(t, half, gap, bound, "the slit's walls")
		floor := section(t, m.Hub, atZ(t, turbinefan.HubLength-0.5))
		top := math.Inf(-1)
		for _, l := range floor.Loops {
			for _, v := range l {
				if math.Abs(v.X) <= half+bound && v.Y < 0 {
					top = math.Max(top, v.Y)
				}
			}
		}
		require.InDelta(t, -dm.BoreRadius+turbinefan.SlitInBore, top, floor.OffSurface.Base()+1e-3, "the slit's round end")
	})
}

// TestTurbineFanEdgeFinishing checks the 0.5 mm chamfer on both faces' outlines: a section 0.25 mm inside a face is
// 0.25 mm short of the disc, of both tab sides and of the tail's end arc.
func TestTurbineFanEdgeFinishing(t *testing.T) {
	eachCase(t, []string{"605ZZ", "608ZZ"}, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		dm := m.Dimensions
		for _, z := range []float64{0.25, turbinefan.HubLength - 0.25} {
			o := section(t, m.Hub, atZ(t, z))
			bound := o.OffSurface.Base() + 1e-3
			lo, hi := math.Inf(1), math.Inf(-1)
			left, right := math.Inf(1), math.Inf(-1)
			for _, l := range o.Loops {
				for _, v := range l {
					lo, hi = math.Min(lo, v.Y), math.Max(hi, v.Y)
					if math.Abs(v.Y-dm.ClampY) <= 0.5 {
						left, right = math.Min(left, v.X), math.Max(right, v.X)
					}
				}
			}
			require.InDelta(t, -dm.Reach+0.25, lo, bound, "z %g: end arc", z)
			require.InDelta(t, turbinefan.HubRadius-0.25, hi, bound, "z %g: disc at +Y", z)
			require.InDelta(t, -turbinefan.TabHalfWidth+0.25, left, bound, "z %g: -X tab side", z)
			require.InDelta(t, turbinefan.TabHalfWidth-0.25, right, bound, "z %g: +X tab side", z)
		}
	})
}
