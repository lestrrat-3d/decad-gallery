package screwgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/models/screwgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestGearCorners applies "Corners and roots" to both ribbons, read in the print orientation the README states: each
// ribbon stands on one end, so its sections are the planes square to its own axis.
//
// MidHeights finds one height on a ribbon (its two end faces), and a twisted ribbon has the same outline kind all the
// way along, so the test cuts it at six heights spread over its length and reads the outline at each. A section is
// the ribbon's rectangle with a toothed long side. Its two back corners are 90 degrees and need no round. The two
// corners where the toothed edge meets a thickness face are tooth tips: the tooth's flank edge meets the flat face at
// as little as 34 degrees of material. The rule's gear-teeth exemption covers the flanks and tips of a tooth, so
// those corners are listed as exempt, found by their side of the ribbon (u > 0 in the gear's own section). Every
// other corner of the ribbon passes the check, and the build holds a mesh bound the check may read.
func TestGearCorners(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    screwgear.Params
	}{
		{"default", screwgear.DefaultParams()},
		{"short", shortParams()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m, _ := build(t, tc.p)
			gears, _, err := screwgear.PairForTest(tc.p)
			require.NoError(t, err)
			for gi, body := range []*decad.Body{m.GearA, m.GearB} {
				g := gears[gi]
				axis := g.Ez
				mid, err := cornercheck.MidHeights(body, axis)
				require.NoError(t, err)
				require.NotEmpty(t, mid, "gear %d has no end face level", gi)

				mesh, err := body.Tessellate(t.Context(), units.Millimeters(cornercheck.MaxMeshBound))
				require.NoError(t, err)
				require.LessOrEqual(t, mesh.Bound().Base(), cornercheck.MaxMeshBound)

				length := tc.p.Length()
				var heights []float64
				for _, f := range []float64{-0.45, -0.3, -0.1, 0.07, 0.25, 0.4} {
					heights = append(heights, g.Origin.Dot(axis)+f*length)
				}
				for _, h := range heights {
					// An empty section would report no corner, so require the ribbon's whole outline: one loop
					// that spans the ribbon's width across its axis.
					outline, err := cornercheck.Outline(t.Context(), body, axis, h)
					require.NoError(t, err)
					require.Len(t, outline.Loops, 1, "gear %d at %.2f", gi, h)
					require.Greater(t, len(outline.Loops[0]), 8, "gear %d at %.2f", gi, h)
					require.Greater(t, loopSpan(outline.Loops[0], axis), tc.p.Width-0.5, "gear %d at %.2f", gi, h)
				}
				toothSide := func(pt r3.Vec) bool {
					u, _, _ := screwgear.GearLocalForTest(tc.p, gi, pt)
					return u > 0
				}
				fs, err := cornercheck.CheckAt(t.Context(), body, axis, heights, []cornercheck.Exempt{
					{Reason: "tooth tips where the toothed edge meets a thickness face", Near: toothSide},
				})
				require.NoError(t, err)
				require.Empty(t, fs, cornercheck.Report(fs, 5))
			}
		})
	}
}

// loopSpan is the largest distance between two points of the loop in the plane square to axis.
func loopSpan(loop []r3.Vec, axis r3.Vec) float64 {
	var span float64
	for i := range loop {
		for j := i + 1; j < len(loop); j++ {
			d := loop[i].Sub(loop[j])
			d = d.Sub(axis.Scale(d.Dot(axis)))
			span = math.Max(span, d.Len())
		}
	}
	return span
}

// TestSleeveCornersRecorded records that the sleeve keeps sharp corners that "Corners and roots" requires rounded.
//
// The sleeve stands on an end face, so its sections are the planes square to the pair frame's Z axis. The two end
// faces give MidHeights one height (0), where only the windows' rims cut, so the test also reads the sleeve at the
// heights below, where the four twisted bores cut the wall. At each of them the check finds inside corners turning
// more than the limit: the bores' rectangle corners, 90 degrees in the channel and swept along the twist, and the
// places where a bore's wall meets the tube's cylinders. None is a gear tooth, an engagement edge or a mating corner,
// so none is exempt.
//
// Rounding them is a design decision. A bore is its ribbon's crest rectangle plus 0.2 mm all round. A 0.5 mm round on
// the bore's corner leaves 0.076 mm between it and the ribbon's sharp 90 degree back corner, and the largest round
// that keeps 0.2 mm is 0.2 mm. The test fails when the sleeve is rounded (the corners disappear), so the change that
// rounds the bores replaces it, and the model stays on CORNER_PENDING and CORNER_PARTIAL until then.
func TestSleeveCornersRecorded(t *testing.T) {
	t.Parallel()
	m, _ := build(t, shortParams())
	axis := r3.Vec{Z: 1}
	mid, err := cornercheck.MidHeights(m.Cage, axis)
	require.NoError(t, err)
	require.NotEmpty(t, mid)

	mesh, err := m.Cage.Tessellate(t.Context(), units.Millimeters(cornercheck.MaxMeshBound))
	require.NoError(t, err)
	require.LessOrEqual(t, mesh.Bound().Base(), cornercheck.MaxMeshBound)

	for _, h := range []float64{-13, -9, -5, -1, 1, 5, 9, 13} {
		outline, err := cornercheck.Outline(t.Context(), m.Cage, axis, h)
		require.NoError(t, err)
		require.NotEmpty(t, outline.Loops, "the sleeve's section at %.1f is empty", h)

		fs, err := cornercheck.Section(t.Context(), m.Cage, axis, h)
		require.NoError(t, err)
		var inside int
		var worst float64
		for _, f := range fs {
			if f.Kind != cornercheck.Inside {
				continue
			}
			inside++
			worst = math.Max(worst, f.Turn)
		}
		require.GreaterOrEqual(t, inside, 2, "inside corners at %.1f: %s", h, cornercheck.Report(fs, 3))
		require.Greater(t, worst, cornercheck.InsideMaxTurn, "sharpest inside corner at %.1f", h)
	}
}
