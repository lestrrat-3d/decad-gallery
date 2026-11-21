package turbinefan_test

// CI: isolated-tests

import (
	"errors"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestBuildBladeHarness builds the nose rib for every case: one solid lump standing on the bed, two fully constrained
// sketches, two bed spots, the rib's x range on the section planes from BrimClearance past the leading-edge spot to
// RibTipMargin before the tip, one top point per plane, the blade's lowest point near the track, a volume of at most a
// quarter of the blade's, and the left hand at 192 mirroring the right. Bad inputs are rejected and add no body.
func TestBuildBladeHarness(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		h := b.harness
		require.True(t, h.Body.IsSolid())
		require.Equal(t, 1, len(h.Body.Lumps()), "one piece")
		require.Equal(t, 2, len(h.Sketches), "the wedge's and the profile slab's sketches")
		for i, s := range h.Sketches {
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
		}
		for i, spot := range h.Spots {
			require.NotEmpty(t, spot, "spot %d", i)
		}
		require.Equal(t, 2, len(b.doc.Bodies()), "posed blade and rib")
		box, err := h.Body.Bounds()
		require.NoError(t, err)
		require.InDelta(t, 0, box.Min.Z, 1e-6, "on the bed")

		// The range rule: the first plane at or past the leading-edge spot's end + BrimClearance, the last at or before
		// the blade's greatest x - RibTipMargin.
		blade, err := b.blade.Bounds()
		require.NoError(t, err)
		edge := math.Inf(-1)
		for _, p := range h.Spots[1] {
			edge = math.Max(edge, p.X)
		}
		onPlane := func(x float64) bool {
			return math.Abs(x/turbinefan.SectionStep-math.Round(x/turbinefan.SectionStep)) < 1e-9
		}
		require.True(t, onPlane(h.XStart) && onPlane(h.XEnd), "range on the section planes")
		require.GreaterOrEqual(t, h.XStart, edge+turbinefan.BrimClearance-1e-9, "start")
		require.Less(t, h.XStart-turbinefan.SectionStep, edge+turbinefan.BrimClearance, "start")
		require.LessOrEqual(t, h.XEnd, blade.Max.X-turbinefan.RibTipMargin+1e-9, "end")
		require.Greater(t, h.XEnd+turbinefan.SectionStep, blade.Max.X-turbinefan.RibTipMargin, "end")
		require.Equal(t, int(math.Round((h.XEnd-h.XStart)/turbinefan.SectionStep))+1, len(h.Top), "one top point per plane")

		// The blade's lowest point stays within the band the top reads.
		worst := 0.0
		for x := h.XStart; x <= h.XEnd; x++ {
			low := r3Lowest(section(t, b.blade, frame3(x)))
			worst = math.Max(worst, math.Abs(low.Y-(h.TrackY0+h.TrackSlope*x)))
		}
		require.LessOrEqual(t, worst, turbinefan.RibTopWidth/2+turbinefan.RibTopBand, "nose off the track")

		rv, err := h.Body.Volume()
		require.NoError(t, err)
		bv, err := b.blade.Volume()
		require.NoError(t, err)
		t.Logf("%s: rib x %.2f..%.2f, top z %.2f..%.2f, volume %.0f mm³ (%.3f of the blade's), nose off the track %.2f mm",
			c.name, h.XStart, h.XEnd, h.Top[0][1], maxTop(h.Top), rv.Value.Base(), rv.Value.Base()/bv.Value.Base(), worst)
		require.LessOrEqual(t, rv.Value.Base(), 0.25*bv.Value.Base(), "rib volume")
	})

	right, err := cradleFor(t, cradleCases[1]).harness.Body.Bounds()
	require.NoError(t, err)
	left, err := cradleFor(t, cradleCases[2]).harness.Body.Bounds()
	require.NoError(t, err)
	requireBoxNear(t, right, left, 0.05, true, "left hand")

	sz := mustSize(t, 192)
	doc, blade, tilt := posedBlade(t, 192, turbinefan.DefaultBladePitch, turbinefan.RightHand)
	_, err = turbinefan.BuildBladeHarness(t.Context(), doc, nil, sz, turbinefan.RightHand, tilt)
	require.Error(t, err)
	_, err = turbinefan.BuildBladeHarness(t.Context(), doc, blade, turbinefan.Size{}, turbinefan.RightHand, tilt)
	require.Error(t, err)
	_, err = turbinefan.BuildBladeHarness(t.Context(), doc, blade, sz, turbinefan.RightHand, turbinefan.RestTiltMax+1e-3)
	require.Error(t, err)
	require.Equal(t, 1, len(doc.Bodies()), "only the posed blade")
}

// TestBladeHarnessVerifyAll tessellates every case's nose rib at 0.2 mm with every mesh verification on.
func TestBladeHarnessVerifyAll(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		_, err := b.harness.Body.Tessellate(t.Context(), units.Millimeters(0.2), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
	})
}

// TestBladeHarnessSeparate requires the nose rib and the blade to share no volume: Intersect reports an empty result.
// The least gap (0.15 mm) is near the meshes' bounds, so a mesh distance cannot prove it; Intersect is exact.
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

// TestBladePrintBedContact requires the posed blade to touch the bed at two patches: its stud flat, by the cut-away
// stud end corner, and its leading edge near stations 4 and 5, each 0.5 mm² or more in the first layer.
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

// TestCradleFirstLayer checks the nose rib's first layer, its section at z = 0.1: one loop with no hole, of 60 mm² or
// more, at least RibTopWidth wide across the track at every 1 mm from XStart + 0.5. That width holds only because the
// rib keeps its bed edges sharp: a 0.5 mm chamfer each side would take its low end under 1.2 mm.
func TestCradleFirstLayer(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		h := b.harness
		loops := sectionAt(t, h.Body, 0.1).Loops
		require.Equal(t, 1, len(loops), "one loop")
		area, _ := loopArea(loops[0])
		across := math.Cos(math.Atan(h.TrackSlope))
		narrowest := math.Inf(1)
		for x := h.XStart + 0.5; x <= h.XEnd; x++ {
			lo, hi := loopSpanAtX(loops[0], x)
			narrowest = math.Min(narrowest, (hi-lo)*across)
		}
		t.Logf("%s: rib first layer %.1f mm², narrowest %.3f mm across the track", c.name, area, narrowest)
		require.GreaterOrEqual(t, area, 60.0, "first layer")
		require.GreaterOrEqual(t, narrowest, turbinefan.RibTopWidth, "first layer width")
	})
}

// TestBladeBrimClearance requires BrimClearance of free bed round each blade bed spot over the first two layers: at
// z = 0.1 and 0.3 the rib's section stays at least BrimClearance from the spot's outline in the XY plane.
func TestBladeBrimClearance(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		h := b.harness
		for _, z := range []float64{0.1, 0.3} {
			loops := sectionAt(t, h.Body, z).Loops
			for i, spot := range h.Spots {
				d := math.Inf(1)
				for _, l := range loops {
					d = math.Min(d, loopDistance(spot, l))
				}
				t.Logf("%s: spot %d to the rib at z %.1f: %.3f mm", c.name, i, z, d)
				require.GreaterOrEqual(t, d, turbinefan.BrimClearance, "spot %d at z %.1f", i, z)
			}
		}
	})
}
