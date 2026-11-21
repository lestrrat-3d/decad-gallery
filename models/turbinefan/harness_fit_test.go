package turbinefan_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestBladeHarnessOverhangs requires the nose rib to print without support: every face pointing down more than 45
// degrees lies on the bed.
func TestBladeHarnessOverhangs(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		mesh, err := b.harness.Body.Tessellate(t.Context(), units.Millimeters(0.2), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		vs := mesh.Vertices()
		bad, worst := 0, 0.0
		for _, tri := range mesh.Triangles() {
			v := [3]r3.Vec{vs[tri[0]], vs[tri[1]], vs[tri[2]]}
			angle, down := downward(v, 1e-3)
			if !down || (v[0].Z <= 1e-6 && v[1].Z <= 1e-6 && v[2].Z <= 1e-6) {
				continue
			}
			bad++
			worst = math.Max(worst, angle)
		}
		require.Zero(t, bad, "%d rib faces overhang off the bed, worst %.1f°", bad, worst)
	})
}

// TestCradleLayers slices the nose rib at the middle of every 0.08 mm layer and requires each layer to rest on the one
// below: no loop with nothing under it, and no outline point more than 0.1 mm past the layer below.
func TestCradleLayers(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		r := checkLayers(t, b.harness.Body, 0.08)
		t.Logf("%s: %d layers, %d islands %v, worst reach past the layer below %.3f mm at %.2f", c.name, r.layers,
			r.islands, r.islandAt, r.worst, r.worstAt)
		require.Zero(t, r.islands, "islands with nothing below")
		require.LessOrEqual(t, r.worst, 0.1, "reach past the layer below at %.2f", r.worstAt)
	})
}

// TestRibGap requires the rib's top to follow the nose: at every 0.1 mm of x from XStart + 0.05 to XEnd, the blade's
// lowest point over the rib's section stands RibGapZ over the rib's highest point (within -0.005 and +0.01), and the
// rib's top is RibTopWidth wide (within -0.01 and +0.15).
func TestRibGap(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		h := b.harness
		gapLo, gapHi, widthLo, widthHi := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
		for x := h.XStart + 0.05; x <= h.XEnd; x += 0.1 {
			rib := section(t, h.Body, frame3(x)).Loops
			require.Equal(t, 1, len(rib), "rib section at x %.2f", x)
			blade := section(t, b.blade, frame3(x)).Loops
			ylo, yhi, top := math.Inf(1), math.Inf(-1), math.Inf(-1)
			for _, p := range rib[0] {
				ylo, yhi, top = math.Min(ylo, p.Y), math.Max(yhi, p.Y), math.Max(top, p.Z)
			}
			nose := math.Inf(1)
			for y := ylo; y <= yhi+1e-9; y += 0.02 {
				for _, l := range blade {
					if z, ok := lowestCrossing(l, y); ok {
						nose = math.Min(nose, z)
					}
				}
			}
			tlo, thi := math.Inf(1), math.Inf(-1)
			for _, p := range rib[0] {
				if p.Z >= top-0.02 {
					tlo, thi = math.Min(tlo, p.Y), math.Max(thi, p.Y)
				}
			}
			gap, width := nose-top, thi-tlo
			gapLo, gapHi = math.Min(gapLo, gap), math.Max(gapHi, gap)
			widthLo, widthHi = math.Min(widthLo, width), math.Max(widthHi, width)
			require.InDelta(t, turbinefan.RibGapZ+0.0025, gap, 0.0075, "gap at x %.2f", x)
			require.GreaterOrEqual(t, width, turbinefan.RibTopWidth-0.01, "top width at x %.2f", x)
			require.LessOrEqual(t, width, turbinefan.RibTopWidth+0.15, "top width at x %.2f", x)
		}
		t.Logf("%s: gap %.4f..%.4f mm, top width %.3f..%.3f mm", c.name, gapLo, gapHi, widthLo, widthHi)
	})
}

// TestRibLayerGap requires the rib and the blade to stay 0.4 mm apart in every 0.08 mm layer they share: their
// sections at the middle of each layer up to the rib's top.
func TestRibLayerGap(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		box, err := b.harness.Body.Bounds()
		require.NoError(t, err)
		least, at := math.Inf(1), 0.0
		for z := 0.04; z < box.Max.Z; z += 0.08 {
			rib := sectionAt(t, b.harness.Body, z).Loops
			blade := sectionAt(t, b.blade, z).Loops
			for _, r := range rib {
				for _, l := range blade {
					if d := loopDistance(r, l); d < least {
						least, at = d, z
					}
				}
			}
		}
		t.Logf("%s: rib to blade in a shared layer at least %.3f mm (z %.2f)", c.name, least, at)
		require.GreaterOrEqual(t, least, 0.4, "rib to blade at z %.2f", at)
	})
}
