package turbinefan_test

// CI: isolated-tests

import (
	"errors"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
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

// TestBladeHarnessContour requires the harness to follow the blade: below the widest line every sampled blade surface
// point is within CradleGapXY + 0.3 of the cradle's mesh, or of a brim pad's when over it, or 2.2 in the step zone and
// its lead-in. Over a pad the cradle stands back to PadClearanceTop, so the pad is the nearest support there. The bolt
// hole's band, the root step's face and the near-bed band (blade under 0.6 mm) are exempt.
func TestBladeHarnessContour(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		h := b.harness
		gs, bound := harnessGaps(t, h)
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
				d := supportDistance(t, h, gs, p)
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
			samples, worst, worstX, bound, over)
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

// TestBladePadsTouch requires the brim pads in order, stud corner first, and each to hug its blade bed patch: the pad's
// first-layer section has exactly one hole, whose area matches the blade's patch within 0.05 mm² and whose bounding box
// matches within 0.05 mm, and the pad overlaps the blade only between z 0.1 and 0.2, by more than 0 and less than
// 0.5 mm³.
func TestBladePadsTouch(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		islands := sectionAt(t, b.blade, 0.1).Loops
		require.Equal(t, 2, len(islands), "blade bed patches")
		// The stud corner's pad comes first: its patch lies nearer the stud end and is the smaller one.
		require.Equal(t, 2, len(b.harness.PadPatches))
		studArea, studCen := loopArea(b.harness.PadPatches[0])
		edgeArea, edgeCen := loopArea(b.harness.PadPatches[1])
		require.Less(t, studCen.X, edgeCen.X, "pad order along x")
		require.Less(t, studArea, edgeArea, "pad order by patch area")
		for i, pad := range b.harness.Pads {
			loops := sectionAt(t, pad, 0.1).Loops
			require.Equal(t, 2, len(loops), "pad %d loops", i)
			outer, hole := loops[0], loops[1]
			outerArea, _ := loopArea(outer)
			if a, _ := loopArea(hole); a > outerArea {
				outer, hole = hole, outer
			}
			require.True(t, insideLoop(outer, hole[0].X, hole[0].Y), "pad %d hole inside its outline", i)
			holeArea, holeCen := loopArea(hole)
			island, best := []r3.Vec(nil), math.Inf(1)
			for _, l := range islands {
				_, cen := loopArea(l)
				if d := math.Hypot(cen.X-holeCen.X, cen.Y-holeCen.Y); d < best {
					island, best = l, d
				}
			}
			islandArea, _ := loopArea(island)
			hlo, hhi := loopBox(hole)
			ilo, ihi := loopBox(island)
			t.Logf("%s: pad %d hole %.3f mm², blade patch %.3f mm²", c.name, i, holeArea, islandArea)
			require.InDelta(t, islandArea, holeArea, 0.05, "pad %d hole area", i)
			for k, pair := range [][2]float64{{ilo.X, hlo.X}, {ilo.Y, hlo.Y}, {ihi.X, hhi.X}, {ihi.Y, hhi.Y}} {
				require.InDelta(t, pair[0], pair[1], 0.05, "pad %d hole box %d", i, k)
			}
			// Intersect retires its operands, so it runs on copies.
			bladeCopy, err := decad.Copy(t.Context(), b.blade)
			require.NoError(t, err)
			padCopy, err := decad.Copy(t.Context(), pad)
			require.NoError(t, err)
			in, err := decad.Intersect(t.Context(), bladeCopy, padCopy)
			require.NoError(t, err, "pad %d", i)
			v, err := in.Volume()
			require.NoError(t, err)
			t.Logf("%s: pad %d overlaps the blade by %.4f mm³", c.name, i, v.Value.Base())
			require.Greater(t, v.Value.Base(), 0.0, "pad %d overlap", i)
			require.Less(t, v.Value.Base(), 0.5, "pad %d overlap", i)
		}
	})
}

// TestBladePadsClear requires the cradle to keep clear of each brim pad: they share no volume, and the cradle's section
// at z = 0.3 has no point inside any pad's outline grown by CradleGapXY - 0.05, sampled every 0.1 mm along its loops.
func TestBladePadsClear(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		h := b.harness
		for i, pad := range h.Pads {
			padCopy, err := decad.Copy(t.Context(), pad)
			require.NoError(t, err)
			cradleCopy, err := decad.Copy(t.Context(), h.Body)
			require.NoError(t, err)
			_, err = decad.Intersect(t.Context(), padCopy, cradleCopy)
			var be *decad.BooleanError
			require.True(t, errors.As(err, &be) && be.Code == decad.BooleanEmpty, "pad %d intersect: %v", i, err)
		}
		inside := 0
		for _, l := range sectionAt(t, h.Body, 0.3).Loops {
			for k := range l {
				p, q := l[k], l[(k+1)%len(l)]
				n := max(1, int(math.Ceil(q.Sub(p).Len()/0.1)))
				for j := range n {
					s := p.Add(q.Sub(p).Scale(float64(j) / float64(n)))
					for _, r := range h.PadOutlines {
						if inPadRect(r, turbinefan.CradleGapXY-0.05, s.X, s.Y) {
							inside++
						}
					}
				}
			}
		}
		require.Zero(t, inside, "cradle points at z 0.3 inside a pad's clearance")
	})
}

// TestBladeHarnessOverhangs requires printable faces. Every cradle face pointing down more than 45 degrees lies on the
// bed, or over a brim pad at PadClearanceTop. Every such blade face rests over the harness (its centroid within 1.0 of
// the cradle's mesh, or of a brim pad's when over it, 2.2 in the step zone and its lead-in), or lies in the bolt hole's
// band, on the stud end's chamfer, past the cradle's tip end, or in the near-bed band (every vertex under 0.6 mm).
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
			angle, down := downward(v, 0.01)
			if !down {
				continue
			}
			if v[0].Z <= bound && v[1].Z <= bound && v[2].Z <= bound {
				continue
			}
			// The cradle over a pad prints on it across the support gap.
			overPad := true
			for _, p := range v {
				onTop := math.Abs(p.Z-turbinefan.PadClearanceTop) <= bound
				inPad := false
				for _, r := range h.PadOutlines {
					inPad = inPad || inPadRect(r, turbinefan.CradleGapXY+bound, p.X, p.Y)
				}
				overPad = overPad && onTop && inPad
			}
			if overPad {
				continue
			}
			bad++
			worst = math.Max(worst, angle)
		}
		require.Zero(t, bad, "%d harness overhangs off the bed, worst %.1f°", bad, worst)

		gs, _ := harnessGaps(t, h)
		bm, err := b.blade.Tessellate(t.Context(), units.Millimeters(0.05), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		pose, err := turbinefan.PrintPoseFor(b.sz, turbinefan.DefaultBladePitch, c.hand, b.tilt)
		require.NoError(t, err)
		radial := pose.U().Cross(pose.V())
		bv := bm.Vertices()
		counts := map[string]int{}
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
			if d := supportDistance(t, h, gs, cen); d <= limit {
				counts["over the harness"]++
				farthest = math.Max(farthest, d)
				continue
			}
			bad++
			worst = math.Max(worst, angle)
		}
		t.Logf("%s: blade overhangs %v, farthest over the harness %.3f mm", c.name, counts, farthest)
		require.Zero(t, bad, "%d blade overhangs unsupported, worst %.1f°", bad, worst)
	})
}
