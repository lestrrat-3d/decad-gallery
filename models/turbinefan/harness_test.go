package turbinefan_test

// CI: isolated-tests

import (
	"errors"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestBuildBladeHarness builds the cradle for every case: a solid on the bed, every sketch fully constrained, sections
// of CradleFitPoints points with y strictly increasing, the step zone inside the block's range, and the left hand at 192
// mirroring the right. Bad inputs are rejected and add no body.
func TestBuildBladeHarness(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		h := b.harness
		require.True(t, h.Body.IsSolid())
		require.Equal(t, shadowSectionCount(h)+8, len(h.Sketches), "sketch count")
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
		require.Less(t, h.XStart, h.StepZone[0])
		require.Less(t, h.StepZone[1], h.XEnd)
		require.Equal(t, 2, len(b.doc.Bodies()), "posed blade and harness")
		box, err := h.Body.Bounds()
		require.NoError(t, err)
		require.InDelta(t, 0, box.Min.Z, box.Bound.Base()+1e-6)
	})

	right, left := cradleFor(t, cradleCases[1]).harness, cradleFor(t, cradleCases[2]).harness
	require.Equal(t, len(right.Sections), len(left.Sections))
	for i := range right.Sections {
		r, l := right.Sections[i], left.Sections[i]
		require.InDelta(t, r.X, l.X, 1e-9, "section %d x", i)
		// The left hand's outline runs the other way along y; its widest points swap sides. A widest point's z sits where
		// the outline turns vertical, so the section's 0.02 mm chords move it up to 0.32 mm between the hands (measured
		// at 192); its y holds within 0.002.
		require.InDelta(t, -r.LeftY, l.RightY, 0.05, "section %d y", i)
		require.InDelta(t, -r.RightY, l.LeftY, 0.05, "section %d y", i)
		require.InDelta(t, r.LeftZ, l.RightZ, 0.4, "section %d z", i)
		require.InDelta(t, r.RightZ, l.LeftZ, 0.4, "section %d z", i)
	}

	sz := mustSize(t, 192)
	doc, blade := posedBlade(t, 192, turbinefan.DefaultBladePitch, turbinefan.RightHand)
	_, err := turbinefan.BuildBladeHarness(t.Context(), doc, nil, sz, turbinefan.RightHand)
	require.Error(t, err)
	_, err = turbinefan.BuildBladeHarness(t.Context(), doc, blade, turbinefan.Size{}, turbinefan.RightHand)
	require.Error(t, err)
	require.Equal(t, 1, len(doc.Bodies()), "only the posed blade")
}

// TestBladeHarnessVerifyAll tessellates every case's harness at 0.1 mm with every mesh verification on. The cradle's
// cuts cross two faceted tools, so its held mesh is coarser than 0.05.
func TestBladeHarnessVerifyAll(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		_, err := b.harness.Body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
	})
}

// TestBladeHarnessSeparate requires the cradle and the blade to share no volume: Intersect reports an empty result, and
// the meshes are proven apart.
func TestBladeHarnessSeparate(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		// Intersect retires its operands, so it runs on copies, filed in one container.
		bladeCopy, err := decad.Copy(t.Context(), b.blade)
		require.NoError(t, err)
		harnessCopy, err := decad.Copy(t.Context(), b.harness.Body)
		require.NoError(t, err)
		_, err = decad.Intersect(t.Context(), bladeCopy, harnessCopy)
		var be *decad.BooleanError
		require.True(t, errors.As(err, &be) && be.Code == decad.BooleanEmpty, "intersect: %v", err)

		bm, err := b.blade.Tessellate(t.Context(), units.Millimeters(0.05), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		hm, err := b.harness.Body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		bg, err := meshgap.New(bm.Vertices(), bm.Triangles(), bm.Bound().Base())
		require.NoError(t, err)
		hg, err := meshgap.New(hm.Vertices(), hm.Triangles(), hm.Bound().Base())
		require.NoError(t, err)
		gap, apart, err := hg.Exceeds(bg, 0)
		require.NoError(t, err)
		t.Logf("%s: separation lower bound %.3f mm (blade bound %.3f, harness bound %.3f)", c.name, gap,
			bm.Bound().Base(), hm.Bound().Base())
		require.True(t, apart, "separation not proven")
	})
}
