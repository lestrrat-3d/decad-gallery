package bevelgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestCorners applies "Corners and roots" to the bevel gear body about its shaft, +X in the build frame.
//
// It reads only the hub: cornercheck.MidHeights returns one height on the default gear (17.44), which cuts the hub
// behind the teeth. The teeth are not read here. TestToothRootCorners records what the teeth sections hold.
//
// The spiral member joins its teeth on the serial trim path, which decad.Refine does not rebuild. It builds only in
// a document with a 0.1 mm chain tolerance, and its sections still read at or below cornercheck.MaxMeshBound, so
// the hub is checked like the straight member's and nothing is exempt.
func TestCorners(t *testing.T) {
	unbored := bevelgear.DefaultParams()
	unbored.EnableBore = false
	spiral := spiralDefaults()
	spiralUnbored := spiral
	spiralUnbored.EnableBore = false
	for _, tc := range []struct {
		name string
		p    bevelgear.Params
		doc  func() *decad.Document
	}{
		{"straight_bored", bevelgear.DefaultParams(), func() *decad.Document { return decad.New() }},
		{"straight_unbored", unbored, func() *decad.Document { return decad.New() }},
		{"spiral_bored", spiral, chainDocument},
		{"spiral_unbored", spiralUnbored, chainDocument},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body, err := bevelgear.Build(t.Context(), tc.doc(), tc.p)
			require.NoError(t, err)
			axis := r3.Vec{X: 1}
			// A silent empty list would check nothing, so require a section height.
			heights, err := cornercheck.MidHeights(body, axis)
			require.NoError(t, err)
			require.NotEmpty(t, heights)
			// The hub sections hold no corner, so no exemption is listed. The teeth are outside these sections.
			fs, err := cornercheck.Check(t.Context(), body, axis, nil)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		})
	}
}

// rootCone is the default gear's root cone about the shaft: the radius at axial position x, in mm. Both members' roots
// lie on it, from 10.14 at x 11.36 to 14.60 at x 16.36.
func rootCone(x float64) float64 { return 10.14 + 0.892*(x-11.36) }

// TestToothRootCorners records that the bevel teeth have no root fillet.
//
// The tooth profile runs a radial line to the base of the involute and nothing rounds that join, so every tooth
// leaves one sharp inside corner on the root cone, in every section from the toe to the heel. The rule's gear-teeth
// exemption names "root fillets" that follow the generators; this gear has none, so these corners are known findings
// of "Corners and roots", not exempt ones, and the model stays on CORNER_PENDING until decad can round them.
//
// The test cuts the teeth region at five heights. At each it requires exactly one inside corner per tooth, on the
// root cone, and exactly the expected tooth tips. It fails when the roots get a fillet (the corners disappear or
// soften), when the geometry moves, and when a section is empty, since a count of zero cannot pass.
func TestToothRootCorners(t *testing.T) {
	spiral := spiralDefaults()
	for _, tc := range []struct {
		name string
		p    bevelgear.Params
		doc  func() *decad.Document
		// rootTurn is the range of the root corners' turn in degrees across the sections.
		rootTurnMin, rootTurnMax float64
	}{
		{"straight", bevelgear.DefaultParams(), func() *decad.Document { return decad.New() }, 83, 87},
		{"spiral", spiral, chainDocument, 105, 125},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body, err := bevelgear.Build(t.Context(), tc.doc(), tc.p)
			require.NoError(t, err)
			axis := r3.Vec{X: 1}
			// Heights from the toe side to the heel side. The tooth tips reach a section only up to about x 14.4.
			for _, h := range []struct {
				x    float64
				tips int
			}{{11.5, tc.p.Teeth}, {12.5, tc.p.Teeth}, {13.5, tc.p.Teeth}, {15, 0}, {16, 0}} {
				fs, err := cornercheck.Section(t.Context(), body, axis, h.x)
				require.NoError(t, err)
				var roots, tips int
				for _, f := range fs {
					r := math.Hypot(f.At.Y, f.At.Z)
					switch f.Kind {
					case cornercheck.Inside:
						roots++
						require.InDelta(t, rootCone(h.x), r, 0.2, "root corner off the root cone at x %.2f", h.x)
						require.GreaterOrEqual(t, f.Turn, tc.rootTurnMin, "x %.2f", h.x)
						require.LessOrEqual(t, f.Turn, tc.rootTurnMax, "x %.2f", h.x)
					case cornercheck.Tip:
						// Tooth tips are exempt from the rule as gear teeth, at a material angle near 30°.
						tips++
						require.Greater(t, r, rootCone(h.x)+1, "tip below the root cone at x %.2f", h.x)
						require.GreaterOrEqual(t, f.Turn, 140.0, "x %.2f", h.x)
					}
				}
				// An empty section would report zero corners, so require the tooth count itself.
				require.Equal(t, tc.p.Teeth, roots, "root corners at x %.2f: %s", h.x, cornercheck.Report(fs, 3))
				require.Equal(t, h.tips, tips, "tip corners at x %.2f: %s", h.x, cornercheck.Report(fs, 3))
			}
		})
	}
}
