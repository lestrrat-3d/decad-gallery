package turbinefan_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestBladeHarnessWallTop checks each wall's top: it stays CradleTopDrop under the widest line, it raises no fin in the
// step zone above the root's clean wall, and it does not zigzag: no two neighbouring second differences of more than
// 0.1 mm have opposite signs. The wall top follows the blade's own widest line, which turns sharply where the bolt hole
// crosses the stud's sides and where the root flares into the airfoil, so points where that line's own second
// difference exceeds 0.3 mm, the step zone's reach and the two sections at each end are not read for zigzags.
func TestBladeHarnessWallTop(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		h := b.harness
		var clean turbinefan.CradleSection
		for _, s := range h.Sections {
			if s.X >= h.StepZone[1] {
				clean = s
				break
			}
		}
		blk := h.Sections[1 : len(h.Sections)-1]
		inZone := func(x float64) bool { return x >= h.StepZone[0]-1.0-1e-9 && x < h.StepZone[1] }
		for w, top := range h.WallTop {
			require.Equal(t, len(blk), len(top), "wall %d points", w)
			pick := func(s turbinefan.CradleSection) (float64, float64) {
				if w == 0 {
					return s.LeftZ, s.LeftLimit
				}
				return s.RightZ, s.RightLimit
			}
			cleanTop := math.Inf(-1)
			for i, q := range top {
				if blk[i].X >= h.StepZone[1] && blk[i].X <= h.StepZone[1]+3 {
					cleanTop = math.Max(cleanTop, q[1])
				}
			}
			fin := math.Inf(-1)
			limits := make([]float64, len(top))
			for i, q := range top {
				src := blk[i]
				if inZone(src.X) {
					src = clean
				}
				widest, limit := pick(src)
				limits[i] = limit
				require.LessOrEqual(t, q[1], widest-turbinefan.CradleTopDrop+1e-9, "wall %d at x %.2f above the widest line",
					w, q[0])
				if inZone(blk[i].X) || blk[i].X == h.StepZone[1] {
					fin = math.Max(fin, q[1]-cleanTop)
				}
			}
			second := func(v []float64, i int) float64 {
				if i <= 0 || i >= len(v)-1 {
					return 0
				}
				return v[i+1] - 2*v[i] + v[i-1]
			}
			heights := make([]float64, len(top))
			for i, q := range top {
				heights[i] = q[1]
			}
			zigzags, skipped, worst := 0, 0, 0.0
			for i := 3; i+3 < len(top); i++ {
				kink := false
				for j := i - 1; j <= i+2; j++ {
					kink = kink || math.Abs(second(limits, j)) > 0.3
				}
				if kink || inStepReach(h, top[i][0]) || inStepReach(h, top[i+1][0]) {
					skipped++
					continue
				}
				a, n := second(heights, i), second(heights, i+1)
				worst = math.Max(worst, math.Abs(a))
				if a*n < 0 && math.Abs(a) > 0.1 && math.Abs(n) > 0.1 {
					zigzags++
					t.Logf("%s wall %d zigzag at x %.2f: %.3f then %.3f", c.name, w, top[i][0], a, n)
				}
			}
			t.Logf("%s wall %d: %d zigzags, worst second difference %.3f mm over %d points (%d skipped), step-zone height "+
				"over the clean wall %.3f mm", c.name, w, zigzags, worst, len(top)-6-skipped, skipped, fin)
			require.Zero(t, zigzags, "wall %d zigzags", w)
			require.LessOrEqual(t, fin, 0.05, "wall %d fin in the step zone", w)
		}
	})
}

// TestBladeHarnessContour requires the cradle to follow the blade: below the widest line every sampled blade surface
// point is within CradleGapXY + 0.3 of the cradle's mesh, or 2.2 in the step zone and its lead-in. The bolt hole's band,
// the root step's face and the near-bed band (blade under 0.6 mm) are exempt.
func TestBladeHarnessContour(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		h := b.harness
		hm, err := h.Body.Tessellate(t.Context(), units.Millimeters(0.2), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		g, err := meshgap.New(hm.Vertices(), hm.Triangles(), hm.Bound().Base())
		require.NoError(t, err)
		pose, err := turbinefan.PrintPoseFor(b.sz, turbinefan.DefaultBladePitch, c.hand, b.tilt)
		require.NoError(t, err)
		radial := pose.U().Cross(pose.V())
		worst, worstX, over, samples := 0.0, 0.0, 0, 0
		for _, s := range h.Sections[2 : len(h.Sections)-2] {
			loop := lowestLoop(section(t, b.blade, frame3(s.X)))
			require.NotNil(t, loop, "blade section at x %.2f", s.X)
			lo, hi := math.Inf(1), math.Inf(-1)
			for _, p := range loop {
				lo, hi = math.Min(lo, p.Y), math.Max(hi, p.Y)
			}
			limit := turbinefan.CradleGapXY + 0.3
			step := s.X >= h.StepZone[0]-1.5 && s.X <= h.StepZone[1]
			if step {
				limit = 2.2
			}
			for y := lo + 0.3; y <= hi-0.3; y += 0.25 {
				z, ok := lowestCrossing(loop, y)
				if !ok || z < 0.6 {
					continue
				}
				p := r3.Vec{X: s.X, Y: y, Z: z}
				// The bolt hole's band, and the root step's face (blade-local radius RootRadius), which faces the stud
				// end, tilted down only by the rest tilt, and needs no support.
				r := p.Sub(pose.Origin()).Dot(radial)
				if math.Abs(r-b.sz.BladeBoltAt) <= m2fastener.HoleRadius+0.5 || math.Abs(r-b.sz.RootRadius) <= 0.1 {
					continue
				}
				d, err := g.PointDistance(p)
				require.NoError(t, err)
				samples++
				if !step && d > worst {
					worst, worstX = d, s.X
				}
				if d > limit {
					over++
				}
			}
		}
		t.Logf("%s: %d samples, worst %.3f mm at x %.2f outside the step zone (mesh bound %.3f), %d over the limit", c.name,
			samples, worst, worstX, hm.Bound().Base(), over)
		require.Zero(t, over, "samples farther from the cradle than allowed")
	})
}
