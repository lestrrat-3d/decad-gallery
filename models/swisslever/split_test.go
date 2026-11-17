package swisslever

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/splitcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// splitTolerance is the chord tolerance of the split check's mesh.
const splitTolerance = 0.01

// otherPhase is the second pose the split test reads: the arbors turn by other angles, so the flats face another way
// and the sections cut other parts of every cone and ramp.
const otherPhase = 0.6

// radialBand returns a test for the points of a section within 0.06 mm of height z whose distance from the line
// through axis along Z lies in [rLo, rHi].
func radialBand(axis Pt, z, rLo, rHi float64) func(r3.Vec) bool {
	return func(p r3.Vec) bool {
		r := math.Hypot(p.X-axis.X, p.Y-axis.Y)
		return math.Abs(p.Z-z) <= 0.06 && r >= rLo && r <= rHi
	}
}

// bearingStep exempts the inside corner of a step that faces a purchased bearing across a gap: the corner lies at the
// seat radius 2.5, and the bearing's inner-ring corner sits in it (docs/model-pipeline.md Split shafts; instructions.md
// Retained sharp corners).
func bearingStep(axis Pt, z float64) cornercheck.Exempt {
	return cornercheck.Exempt{Reason: "bearing-facing step", Near: radialBand(axis, z, ArborRadius-0.2, ArborRadius+0.2)}
}

// chordStop exempts the corner where a flat ends under a D bore's chord, which stops on the end of the flat: the pair
// of corners, the flat's end and the part's chord, stay sharp.
func chordStop(axis Pt, z float64) cornercheck.Exempt {
	return cornercheck.Exempt{Reason: "chord stop on a flat's end", Near: radialBand(axis, z, 0, CollarRadius)}
}

// posedArbor is an arbor built in its local frame and turned about its axis to a pose.
type posedArbor struct {
	name   string
	body   *decad.Body
	at     Pt
	angle  float64 // math angle the pose turned it by
	rMin   float64 // smallest radius that meets a step or a cone, which sets the heights read
	exempt []cornercheck.Exempt
}

// posedArbors builds the three arbors and turns each about its axis as Build does at phase t: the escape arbor by
// -(phi - 0.05 deg), the pallet arbor by -psi and the balance staff by gamma_f + theta.
func posedArbors(t *testing.T, phase float64) []posedArbor {
	t.Helper()
	p := DefaultParams()
	p.Phase = phase
	g, err := Layout(p)
	require.NoError(t, err)
	s, err := g.StateAt(phase)
	require.NoError(t, err)
	b := newBuilder(t.Context(), NewDocument(), g)
	var out []posedArbor
	for _, c := range []struct {
		name   string
		build  func() (*decad.Body, error)
		at     Pt
		angle  float64
		rMin   float64
		exempt []cornercheck.Exempt
	}{
		{"escape arbor", b.buildEscapeArbor, g.O, -(s.WheelAngle - BuildBackoff), endRadius, []cornercheck.Exempt{
			bearingStep(g.O, 0.25), bearingStep(g.O, 26.75), chordStop(g.O, 33.25)}},
		{"pallet arbor", b.buildPalletArbor, g.P, -s.LeverAngle, ArborRadius, []cornercheck.Exempt{
			bearingStep(g.P, 0.25), bearingStep(g.P, 26.75)}},
		{"balance staff", b.buildBalanceStaff, g.B, g.GammaF + s.BalanceAngle, ArborRadius, []cornercheck.Exempt{
			bearingStep(g.B, 0.25), bearingStep(g.B, 26.75)}},
	} {
		body, err := c.build()
		require.NoError(t, err)
		turn, err := r3.RotationAround(r3.NewVec(c.at.X, c.at.Y, 0), r3.Vec{Z: 1}, units.Radians(c.angle))
		require.NoError(t, err)
		body, err = decad.Transform(t.Context(), body, turn)
		require.NoError(t, err)
		out = append(out, posedArbor{c.name, body, c.at, c.angle, c.rMin, c.exempt})
	}
	return out
}

// requireSplitShafts checks the three arbors against "Split shafts" in docs/model-pipeline.md: the split check passes
// with the split plane instructions.md lists, and the halves read on their cut face show no inside corner other than
// the listed ones.
func requireSplitShafts(t *testing.T, arbors []posedArbor) {
	t.Helper()
	for _, c := range arbors {
		t.Run(c.name, func(t *testing.T) {
			point := r3.Vec{X: c.at.X, Y: c.at.Y}
			r, err := splitcheck.Check(t.Context(), splitcheck.Shaft{Body: c.body, Point: point, Axis: r3.Vec{Z: 1}},
				splitTolerance)
			require.NoError(t, err)
			require.Empty(t, r.Violations)
			// The flats face along +X in the shaft's own frame, so the split plane's normal is +Y turned by the pose.
			want := r3.Vec{X: -math.Sin(c.angle), Y: math.Cos(c.angle)}
			require.InDelta(t, 1, math.Abs(r.Normal.Dot(want)), 1e-6, "split normal %v", r.Normal)
			for i, h := range r.HalfHeights {
				require.InDelta(t, CollarRadius, h, splitTolerance, "half %d", i+1)
			}

			// MidHeights finds no height on a round shaft, so the heights are explicit: from the seam to 0.8 of the
			// smallest radius that meets a step or a cone.
			heights, err := cornercheck.SplitHeights(point, r.Normal, c.rMin)
			require.NoError(t, err)
			fs, err := cornercheck.CheckAt(t.Context(), c.body, r.Normal, heights, c.exempt)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 6))
		})
	}
}

// TestSplitShafts reads the arbors as halves on their cut face, at the default pose and at another phase.
func TestSplitShafts(t *testing.T) {
	t.Run("default", func(t *testing.T) { requireSplitShafts(t, posedArbors(t, DefaultParams().Phase)) })
	t.Run("phase 0.6", func(t *testing.T) { requireSplitShafts(t, posedArbors(t, otherPhase)) })
}

// builtArbors returns the three arbors of the built movement at phase, with the pose angle Build turned each by.
func builtArbors(t *testing.T, phase float64) []posedArbor {
	t.Helper()
	_, m := sharedAt(t, phase)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	table, err := poses(&g, m.State)
	require.NoError(t, err)
	return []posedArbor{
		{EscapeArborName, m.EscapeArbor, g.O, table[EscapeArborName].angle, endRadius, []cornercheck.Exempt{
			bearingStep(g.O, 0.25), bearingStep(g.O, 26.75), chordStop(g.O, 33.25)}},
		{PalletArborName, m.PalletArbor, g.P, table[PalletArborName].angle, ArborRadius, []cornercheck.Exempt{
			bearingStep(g.P, 0.25), bearingStep(g.P, 26.75)}},
		{BalanceStaffName, m.BalanceStaff, g.B, table[BalanceStaffName].angle, ArborRadius, []cornercheck.Exempt{
			bearingStep(g.B, 0.25), bearingStep(g.B, 26.75)}},
	}
}

// TestSplitShaftsBuilt reads the arbors of the built, posed movement as halves on their cut face, at the default pose
// and at another phase, so the check follows the pose Build gives each arbor (the lever and wheel backed off their
// last contact), not a pose the test computes for itself.
func TestSplitShaftsBuilt(t *testing.T) {
	t.Run("default", func(t *testing.T) { requireSplitShafts(t, builtArbors(t, DefaultParams().Phase)) })
	t.Run("phase 0.6", func(t *testing.T) { requireSplitShafts(t, builtArbors(t, otherPhase)) })
}
