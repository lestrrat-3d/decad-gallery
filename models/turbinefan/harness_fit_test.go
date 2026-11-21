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

// TestBladeHarnessWallTop checks each wall's top over the block's sections: it stays CradleTopDrop under the widest
// line, it raises no fin in the step zone above the root's clean wall (0.05 mm; 0.1 within CradleTopSmooth of the
// block's start when the block starts inside the zone or its lead-in, where the smoothing sees one side only), and it
// does not zigzag: no two neighbouring second differences of more than 0.1 mm have opposite signs. The wall top follows
// the blade's own widest line, which turns sharply where the bolt hole crosses the stud's sides and where the root
// flares into the airfoil, so points where that line's own second difference exceeds 0.3 mm, the step zone's reach and
// the two sections at each end are not read for zigzags.
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
		// The block's sections run from the first one past the stud pocket, whose x moved on by CradleStep/2 to XStart.
		var blk []turbinefan.CradleSection
		for _, s := range h.Sections[1 : len(h.Sections)-1] {
			if s.X >= h.XStart-turbinefan.CradleStep/2-1e-9 {
				blk = append(blk, s)
			}
		}
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
			// fin is read past the smoothing's reach from the block's start, startFin within it; at 128 the block starts
			// in the step zone, and its first section stands 0.067 over the clean wall.
			fin, finX, startFin, read := math.Inf(-1), 0.0, math.Inf(-1), 0
			startsInZone := inZone(blk[0].X)
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
				if !inZone(blk[i].X) && blk[i].X != h.StepZone[1] {
					continue
				}
				read++
				if startsInZone && q[0] <= h.XStart+turbinefan.CradleTopSmooth+1e-9 {
					startFin = math.Max(startFin, q[1]-cleanTop)
					continue
				}
				if q[1]-cleanTop > fin {
					fin, finX = q[1]-cleanTop, q[0]
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
				"over the clean wall %.3f mm at x %.2f, %.3f mm near the block's start at %.2f", c.name, w, zigzags, worst,
				len(top)-6-skipped, skipped, fin, finX, startFin, h.XStart)
			require.Zero(t, zigzags, "wall %d zigzags", w)
			require.Positive(t, read, "wall %d step-zone points read", w)
			require.LessOrEqual(t, fin, 0.05, "wall %d fin in the step zone", w)
			require.LessOrEqual(t, startFin, 0.1, "wall %d fin in the step zone near the block's start", w)
		}
	})
}

// TestBladeHarnessContour requires the cradle to follow the blade: below the widest line every sampled blade surface
// point is within CradleGapXY + 0.3 of the cradle's mesh, or 2.2 in the step zone and its lead-in. The bolt hole's band,
// the root step's face, the near-bed band (blade under 0.6 mm) and the brim pockets' bed rectangles are exempt.
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
				if inRect(h.Pockets[0], p.X, p.Y) || inRect(h.Pockets[1], p.X, p.Y) {
					continue
				}
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

// TestBladeHarnessNothingAbove requires no cradle beyond either widest line to stand above it: every harness vertex
// outside the step zone and its lead-in that lies past a widest y, interpolated between the sections around its x,
// stays at or under that widest point's z, both within the harness mesh's bound and 0.05.
func TestBladeHarnessNothingAbove(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		h := b.harness
		hm, err := h.Body.Tessellate(t.Context(), units.Millimeters(0.2), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		tol := 0.05 + hm.Bound().Base()
		bad, worst := 0, 0.0
		for _, v := range hm.Vertices() {
			if v.X >= h.StepZone[0]-1.5 && v.X <= h.StepZone[1] {
				continue
			}
			s, ok := sectionAround(h, v.X)
			if !ok {
				continue
			}
			for _, side := range []struct{ y, z, sign float64 }{{s.LeftY, s.LeftZ, -1}, {s.RightY, s.RightZ, 1}} {
				if side.sign*(v.Y-side.y) > tol && v.Z > side.z+tol {
					bad++
					worst = math.Max(worst, v.Z-side.z)
				}
			}
		}
		require.Zero(t, bad, "%d harness vertices beyond a widest line stand above it, worst by %.3f mm", bad, worst)
	})
}

// TestBladeHarnessOverhangs requires printable faces. Every harness face pointing down more than 45 degrees lies on the
// bed; a brim pocket's walls slope at exactly 45, and triangles under 1e-3 mm² are not read. Every such blade face
// rests over the cradle (its centroid within 1.0 of the harness mesh, 2.2 in the step zone and its lead-in), or lies in
// a brim pocket's bed rectangle (after the near-bed band and the bolt hole's band), in the bolt
// hole's band, on the stud end's chamfer, past the cradle's tip end, or in the near-bed band (every vertex under
// 0.6 mm). It logs the support the pockets give up: the area and height of the blade faces in each pocket farther than
// CradleGapXY + 0.3 from the cradle.
func TestBladeHarnessOverhangs(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		h := b.harness
		hm, err := h.Body.Tessellate(t.Context(), units.Millimeters(0.2), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		bound := hm.Bound().Base()
		hv := hm.Vertices()
		bad, worst := 0, 0.0
		for _, tri := range hm.Triangles() {
			v := [3]r3.Vec{hv[tri[0]], hv[tri[1]], hv[tri[2]]}
			// A brim pocket's walls slope at exactly 45 degrees; slivers on them read up to 45.6, so the slack is v5's 0.01.
			angle, down := downward(v, 0.01)
			if !down {
				continue
			}
			// A degenerate triangle's normal is noise: one on a pocket wall at 224 reads 69.7 degrees.
			if v[1].Sub(v[0]).Cross(v[2].Sub(v[0])).Len()/2 < 1e-3 {
				continue
			}
			if v[0].Z <= bound && v[1].Z <= bound && v[2].Z <= bound {
				continue
			}
			bad++
			worst = math.Max(worst, angle)
			if bad <= 3 {
				t.Logf("%s: harness overhang %.1f° at %.2f, %.2f, %.2f", c.name, angle, v[0], v[1], v[2])
			}
		}
		require.Zero(t, bad, "%d harness overhangs off the bed, worst %.1f°", bad, worst)

		g, err := meshgap.New(hm.Vertices(), hm.Triangles(), bound)
		require.NoError(t, err)
		bm, err := b.blade.Tessellate(t.Context(), units.Millimeters(0.05), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		pose, err := turbinefan.PrintPoseFor(b.sz, turbinefan.DefaultBladePitch, c.hand, b.tilt)
		require.NoError(t, err)
		radial := pose.U().Cross(pose.V())
		bv := bm.Vertices()
		counts := map[string]int{}
		var given [2]struct{ area, z float64 }
		bad, worst = 0, 0.0
		farthest := 0.0
		for _, tri := range bm.Triangles() {
			v := [3]r3.Vec{bv[tri[0]], bv[tri[1]], bv[tri[2]]}
			angle, down := downward(v, 1e-3)
			if !down {
				continue
			}
			cen := v[0].Add(v[1]).Add(v[2]).Scale(1.0 / 3)
			switch {
			case v[0].Z < 0.6 && v[1].Z < 0.6 && v[2].Z < 0.6:
				counts["near bed"]++
				continue
			case math.Abs(cen.Sub(pose.Origin()).Dot(radial)-b.sz.BladeBoltAt) <= m2fastener.HoleRadius+0.1:
				counts["bolt hole"]++
				continue
			case inRect(h.Pockets[0], cen.X, cen.Y) || inRect(h.Pockets[1], cen.X, cen.Y):
				counts["brim pocket"]++
				k := 0
				if !inRect(h.Pockets[0], cen.X, cen.Y) {
					k = 1
				}
				d, err := g.PointDistance(cen)
				require.NoError(t, err)
				if d > turbinefan.CradleGapXY+0.3 {
					given[k].area += v[1].Sub(v[0]).Cross(v[2].Sub(v[0])).Len() / 2
					given[k].z = math.Max(given[k].z, cen.Z)
				}
				continue
			case cen.X > h.XEnd:
				counts["tip"]++
				continue
			case cen.Sub(pose.Origin()).Dot(radial) <= b.sz.BladeStudStart+0.35:
				// The stud end's 0.3 chamfer, tilted by the rest tilt, reads 45 to 51 degrees within 0.9 mm of the bed.
				counts["stud chamfer"]++
				continue
			}
			limit := 1.0
			if cen.X >= h.StepZone[0]-1.5 && cen.X <= h.StepZone[1] {
				limit = 2.2
			}
			d, err := g.PointDistance(cen)
			require.NoError(t, err)
			if d <= limit {
				counts["over the cradle"]++
				farthest = math.Max(farthest, d)
				continue
			}
			bad++
			worst = math.Max(worst, angle)
		}
		t.Logf("%s: blade overhangs %v, farthest over the cradle %.3f mm", c.name, counts, farthest)
		t.Logf("%s: support given up in the stud pocket %.1f mm² up to z %.2f, leading-edge pocket %.1f mm² up to z %.2f",
			c.name, given[0].area, given[0].z, given[1].area, given[1].z)
		require.Zero(t, bad, "%d blade overhangs unsupported, worst %.1f°", bad, worst)
	})
}
