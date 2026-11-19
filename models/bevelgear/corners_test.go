package bevelgear_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/rootcheck"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
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

// TestToothRootCorners records that the default 31/31 gear and the spiral member keep a sharp root.
//
// The default gear's fillet does not fit: its radial flank ends below the fillet's tangent point, so the tooth sketch
// runs the flank line to the blank's root and nothing rounds that join. The spiral member shares that tooth and
// draws no fillet by design: its bed-parallel sections meet the root obliquely, 31 corners remain at 107 to 116
// degrees with a fillet, and the build takes 3.6 times as long. So every tooth leaves one sharp inside corner on the
// root cone, in every section from the toe to the heel. The rule's gear-teeth exemption names "root fillets" that
// follow the generators; these gears have none, so these corners are known findings of "Corners and roots", not
// exempt ones, and the model stays on CORNER_PENDING and CORNER_PARTIAL until they are rounded. The fan's pinion and
// every radial-flank gear whose fillet fits are rounded: TestFilletedPinionRootCorners.
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

// TestFilletedPinionRootCorners reads the root fillet of the fan's pinion (module 2, 14 teeth against 28) from the
// toe to the heel. At Heights sections along the root cone it requires that
//
//   - the section cut the teeth, not a hollow: the outline crosses the pitch circle twice per tooth;
//   - no inside corner remains, by the corner check proper, with no exemption;
//   - the fillet is there, so an empty or sharp root cannot pass: every tooth's flank-to-root turn spreads over a
//     0.4 mm window as 20 to 50 degrees, and no 0.05 mm window turns more than 15 degrees. A sharp root turns 80 to 90
//     degrees in one vertex, and fails the second;
//   - the joined mesh holds a bound the corner check may read, bored or not.
//
// Sections behind the pitch point of the heel cone no longer cut the pitch circle, so only the corner check reads
// them.
func TestFilletedPinionRootCorners(t *testing.T) {
	bored := filletPinion()
	bored.EnableBore = true
	for _, tc := range []struct {
		name string
		p    bevelgear.Params
	}{{"unbored", filletPinion()}, {"bored", bored}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			band, err := tc.p.ToothBand()
			require.NoError(t, err)
			require.Positive(t, band.RootFillet, "the pinion's fillet fits")
			body, err := bevelgear.Build(t.Context(), decad.New(), tc.p)
			require.NoError(t, err)
			mesh, err := body.Tessellate(t.Context(), units.Millimeters(cornercheck.MaxMeshBound))
			require.NoError(t, err)
			require.LessOrEqual(t, mesh.Bound().Base(), cornercheck.MaxMeshBound)

			axis := r3.Vec{X: 1}
			secs, err := rootcheck.Read(t.Context(), body, band, r3.Vec{}, axis, axis)
			require.NoError(t, err)
			require.Len(t, secs, rootcheck.Heights)
			fs, err := cornercheck.CheckAt(t.Context(), body, axis, rootcheck.SectionHeights(secs), nil)
			require.NoError(t, err)
			require.Zero(t, len(fs), cornercheck.Report(fs, 5))
			inTeeth := 0
			for _, sec := range secs {
				if !sec.InTeeth {
					continue
				}
				inTeeth++
				require.Equal(t, 2*tc.p.Teeth, sec.Crossings, "the section at %.2f cuts every tooth twice", sec.Station)
				// At the toe the two fillets of a gap share one window.
				require.GreaterOrEqual(t, len(sec.Quarter), tc.p.Teeth, "fillet turns at %.2f", sec.Station)
				for _, turn := range sec.Quarter {
					require.GreaterOrEqual(t, turn, rootcheck.QuarterMinTurn)
					require.LessOrEqual(t, turn, cornercheck.InsideMaxTurn, "station %.2f", sec.Station)
				}
				require.Zero(t, sec.FineMax, "a sharp turn at station %.2f", sec.Station)
			}
			require.GreaterOrEqual(t, inTeeth, 5)
		})
	}
}
