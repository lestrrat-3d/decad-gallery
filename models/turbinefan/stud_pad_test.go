package turbinefan_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/stretchr/testify/require"
)

// TestStudPad requires the stud pad of every harness case to be one solid disc on the bed, StudPadThickness high with
// StudPadRadius, whose first layer is one loop that holds the blade's stud-flat spot, clear of the nose rib and under
// the round root's 0.2 mm clearance.
func TestStudPad(t *testing.T) {
	eachCradle(t, func(t *testing.T, c cradleCase, b *builtCradle) {
		pad := b.harness.Pad
		require.True(t, pad.IsSolid())
		require.Equal(t, 1, len(pad.Lumps()), "one piece")
		box, err := pad.Bounds()
		require.NoError(t, err)
		require.InDelta(t, 0, box.Min.Z, 1e-6, "on the bed")
		require.InDelta(t, turbinefan.StudPadThickness, box.Max.Z, 1e-6, "thickness")
		require.Less(t, turbinefan.StudPadThickness, turbinefan.StudFlatRootClearance, "under the root's clearance")

		loops := sectionAt(t, pad, 0.1).Loops
		require.Equal(t, 1, len(loops), "one first-layer loop")
		area, centre := loopArea(loops[0])
		require.InDelta(t, math.Pi*turbinefan.StudPadRadius*turbinefan.StudPadRadius, area, 0.5, "disc area")
		spotArea, spotCentre := loopArea(b.harness.Spots[0])
		for _, p := range b.harness.Spots[0] {
			require.LessOrEqual(t, p.Sub(centre).Len(), turbinefan.StudPadRadius-1, "the pad holds the stud flat's spot")
		}
		rib := sectionAt(t, b.harness.Body, 0.1).Loops
		d := math.Inf(1)
		for _, l := range rib {
			d = math.Min(d, loopDistance(loops[0], l))
		}
		t.Logf("%s: pad %.1f mm² round a %.1f mm² spot (centres %.2f mm apart), %.1f mm from the rib", c.name, area,
			spotArea, spotCentre.Sub(centre).Len(), d)
		require.GreaterOrEqual(t, d, turbinefan.BrimClearance, "rib clear of the pad's brim")

		// The pad overlaps the blade, so Intersect keeps a volume: the stud flat and the stud's slope over the pad.
		bladeCopy, err := decad.Copy(t.Context(), b.blade)
		require.NoError(t, err)
		padCopy, err := decad.Copy(t.Context(), pad)
		require.NoError(t, err)
		overlap, err := decad.Intersect(t.Context(), bladeCopy, padCopy)
		require.NoError(t, err, "pad meets the blade")
		v, err := overlap.Volume()
		require.NoError(t, err)
		t.Logf("%s: overlap with the blade %.3f mm³", c.name, v.Value.Base())
		// Only the stud flat's spot reaches into the pad's two layers, so the overlap is about the spot's area times the
		// pad's height.
		prism := spotArea * turbinefan.StudPadThickness
		require.InDelta(t, prism, v.Value.Base(), 0.5*prism, "overlap")
	})
}
