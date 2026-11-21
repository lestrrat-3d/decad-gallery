package turbinefan_test

// CI: isolated-tests

import (
	"errors"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestBuildBladeHarness builds the cradle for every case: one solid standing on the bed with a first layer over
// 400 mm², two bed spots, the stud corner's first, every sketch fully constrained, sections of CradleFitPoints points
// with y strictly increasing, the step zone ending inside the block, and the left hand at 192 mirroring the right. Bad
// inputs are rejected and add no body.
func TestBuildBladeHarness(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		h := b.harness
		require.True(t, h.Body.IsSolid())
		require.Equal(t, 1, len(h.Body.Lumps()), "one piece")
		// v5's seven, two collars, four pocket trapezoids and the bed box.
		require.Equal(t, shadowSectionCount(h)+14, len(h.Sketches), "sketch count")
		for i, spot := range h.Spots {
			require.NotEmpty(t, spot, "spot %d", i)
		}
		_, stud := loopArea(h.Spots[0])
		_, edge := loopArea(h.Spots[1])
		require.Less(t, stud.X, edge.X, "stud corner spot first")
		for i, s := range h.Sketches {
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
		}
		for i, s := range h.Sections {
			require.Equal(t, turbinefan.CradleFitPoints, len(s.Outline), "section %d", i)
			for j := 1; j < len(s.Outline); j++ {
				require.Greater(t, s.Outline[j][0], s.Outline[j-1][0], "section %d point %d", i, j)
			}
		}
		require.Less(t, h.XStart, h.XEnd)
		// The block starts past the stud pocket, which at 128 reaches into the step zone.
		require.Less(t, h.XStart, h.StepZone[1])
		require.Less(t, h.StepZone[1], h.XEnd)
		require.Equal(t, 2, len(b.doc.Bodies()), "posed blade and harness")
		box, err := h.Body.Bounds()
		require.NoError(t, err)
		require.InDelta(t, 0, box.Min.Z, box.Bound.Base()+1e-6)
		// The bed face is its largest loop less the others, which are holes in it (TestBladeHarnessBedFace).
		area, largest := 0.0, 0.0
		for _, l := range sectionAt(t, h.Body, 0.1).Loops {
			a, _ := loopArea(l)
			area -= a
			largest = max(largest, a)
		}
		area += 2 * largest
		t.Logf("%s: rest tilt %.4f rad, first-layer outline area %.0f mm²", c.name, b.tilt, area)
		require.Greater(t, area, 400.0, "first layer")
	})

	right, left := cradleFor(t, cradleCases[1]).harness, cradleFor(t, cradleCases[2]).harness
	require.Equal(t, len(right.Sections), len(left.Sections))
	for i := range right.Sections {
		r, l := right.Sections[i], left.Sections[i]
		require.InDelta(t, r.X, l.X, 1e-9, "section %d x", i)
		// The left hand's outline runs the other way along y; its widest points swap sides. A widest point's z sits where
		// the outline turns vertical, so the section's 0.02 mm chords move it up to 0.32 mm between the hands (measured
		// at 192 in v4); its y holds within 0.002.
		require.InDelta(t, -r.LeftY, l.RightY, 0.05, "section %d y", i)
		require.InDelta(t, -r.RightY, l.LeftY, 0.05, "section %d y", i)
		require.InDelta(t, r.LeftZ, l.RightZ, 0.4, "section %d z", i)
		require.InDelta(t, r.RightZ, l.LeftZ, 0.4, "section %d z", i)
	}

	sz := mustSize(t, 192)
	doc, blade, tilt := posedBlade(t, 192, turbinefan.DefaultBladePitch, turbinefan.RightHand)
	_, err := turbinefan.BuildBladeHarness(t.Context(), doc, nil, sz, turbinefan.RightHand, tilt)
	require.Error(t, err)
	_, err = turbinefan.BuildBladeHarness(t.Context(), doc, blade, turbinefan.Size{}, turbinefan.RightHand, tilt)
	require.Error(t, err)
	_, err = turbinefan.BuildBladeHarness(t.Context(), doc, blade, sz, turbinefan.RightHand, turbinefan.RestTiltMax+1e-3)
	require.Error(t, err)
	require.Equal(t, 1, len(doc.Bodies()), "only the posed blade")
}

// TestBladeHarnessVerifyAll tessellates every case's harness at 0.2 mm with every mesh verification on. The cradle's
// cuts cross two faceted tools, so its held mesh bound is 0.09 to 0.14 mm.
func TestBladeHarnessVerifyAll(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		_, err := b.harness.Body.Tessellate(t.Context(), units.Millimeters(0.2), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
	})
}

// TestBladeHarnessSeparate requires the cradle and the blade to share no volume: Intersect reports an empty result.
// At 224 the harness's mesh bound exceeds the smallest gap, so a mesh distance cannot prove it; Intersect is exact.
func TestBladeHarnessSeparate(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		// Intersect retires its operands, so it runs on copies.
		bladeCopy, err := decad.Copy(t.Context(), b.blade)
		require.NoError(t, err)
		harnessCopy, err := decad.Copy(t.Context(), b.harness.Body)
		require.NoError(t, err)
		_, err = decad.Intersect(t.Context(), bladeCopy, harnessCopy)
		var be *decad.BooleanError
		require.True(t, errors.As(err, &be) && be.Code == decad.BooleanEmpty, "intersect: %v", err)
	})
}

// TestBladePrintBedContact requires the posed blade to touch the bed at two patches: its stud end corner and its
// leading edge near stations 4 and 5, each 0.5 mm² or more in the first layer.
func TestBladePrintBedContact(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		corner, le4, le5 := turbinefan.PosedEdgePoints(b.sz, b.hand, b.tilt)
		loops := sectionAt(t, b.blade, 0.1).Loops
		require.Equal(t, 2, len(loops), "bed patches")
		lo, hi := min(le4.X, le5.X)-2, max(le4.X, le5.X)+2
		var stud, edge int
		for i, l := range loops {
			a, cen := loopArea(l)
			t.Logf("%s: patch %d at x %.2f, %.2f mm²", c.name, i, cen.X, a)
			require.GreaterOrEqual(t, a, 0.5, "patch %d area", i)
			switch {
			case math.Abs(cen.X-corner.X) <= 3:
				stud++
			case cen.X >= lo && cen.X <= hi:
				edge++
			}
		}
		require.Equal(t, 1, stud, "stud corner patch")
		require.Equal(t, 1, edge, "leading-edge patch")
	})
}

// TestCradleFirstLayer checks the cradle's first layer, its section at z = 0.1: one outline with exactly two holes, the
// brim pockets, each holding its blade bed spot, and an area of at least cradleFirstLayerMin.
func TestCradleFirstLayer(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		h := b.harness
		loops := sectionAt(t, h.Body, 0.1).Loops
		require.Equal(t, 3, len(loops), "outline and two pockets")
		outer, best := -1, 0.0
		for i, l := range loops {
			if a, _ := loopArea(l); a > best {
				outer, best = i, a
			}
		}
		area := 2 * best
		var holes [][]r3.Vec
		for i, l := range loops {
			a, _ := loopArea(l)
			area -= a
			if i == outer {
				continue
			}
			require.True(t, insideLoop(loops[outer], l[0].X, l[0].Y), "loop %d is a second outline", i)
			holes = append(holes, l)
		}
		pocketOf := [2]int{-1, -1}
		for i, spot := range h.Spots {
			_, cen := loopArea(spot)
			held := 0
			for k, hole := range holes {
				if insideLoop(hole, cen.X, cen.Y) {
					held++
					pocketOf[i] = k
				}
			}
			require.Equal(t, 1, held, "spot %d in a pocket", i)
		}
		require.NotEqual(t, pocketOf[0], pocketOf[1], "each spot in its own pocket")
		t.Logf("%s: cradle first layer %.0f mm²", c.name, area)
		require.GreaterOrEqual(t, area, cradleFirstLayerMin[c.rotor], "first layer")
	})
}

// TestBladeBrimClearance requires BrimClearance of free bed round each blade bed spot over the first two layers: at
// z = 0.1 and 0.3 the cradle's section stays at least BrimClearance from the spot's outline in the XY plane.
func TestBladeBrimClearance(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		h := b.harness
		for _, z := range []float64{0.1, 0.3} {
			loops := sectionAt(t, h.Body, z).Loops
			for i, spot := range h.Spots {
				require.NotEmpty(t, spot, "spot %d", i)
				d := math.Inf(1)
				for _, l := range loops {
					d = math.Min(d, loopDistance(spot, l))
				}
				t.Logf("%s: spot %d to the cradle at z %.1f: %.3f mm", c.name, i, z, d)
				require.GreaterOrEqual(t, d, turbinefan.BrimClearance, "spot %d at z %.1f", i, z)
			}
		}
	})
}
