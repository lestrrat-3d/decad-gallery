package escapement

import (
	"math"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/splitcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// splitTolerance is the chord tolerance of the split check's mesh.
const splitTolerance = 0.01

// otherPhase is the second pose the split test reads: the pose turns each arbor by another angle, so the flats face
// another way and the sections cut other parts of every cone and ramp.
const otherPhase = 0.6

var (
	otherOnce  sync.Once
	otherValue sharedBuild
)

// sharedOtherPhase builds the movement at otherPhase once for the whole test binary.
func sharedOtherPhase(t *testing.T) (*Movement, error) {
	t.Helper()
	otherOnce.Do(func() {
		p := DefaultParams()
		p.Phase = otherPhase
		doc := NewDocument()
		m, err := Build(t.Context(), doc, p)
		otherValue = sharedBuild{doc, m, err}
	})
	return otherValue.m, otherValue.err
}

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
// Retained sharp corners R1).
func bearingStep(axis Pt, z float64) cornercheck.Exempt {
	return cornercheck.Exempt{
		Reason: "bearing-facing step",
		Near:   radialBand(axis, z, ArborRadius-0.2, ArborRadius+0.2),
	}
}

// chordStop exempts the corner where a flat ends under a D bore's chord, which stops on the end of the flat (R3a): the
// pair of corners, the flat's end and the part's chord, stay sharp.
func chordStop(axis Pt, z float64) cornercheck.Exempt {
	return cornercheck.Exempt{
		Reason: "chord stop on a flat's end",
		Near:   radialBand(axis, z, 0, CollarRadius),
	}
}

// requireSplitShafts checks the four split shafts of m against "Split shafts" in docs/model-pipeline.md: the split
// check passes with the split plane that instructions.md lists, and the halves read on their cut face show no
// inside corner other than the listed ones.
func requireSplitShafts(t *testing.T, m *Movement) {
	t.Helper()
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	table := poses(&g, m.State)
	for _, c := range []struct {
		name string
		body *decad.Body
		at   Pt
		// rMin is the smallest radius that meets a step or a cone on the part, which sets the heights read.
		rMin   float64
		exempt []cornercheck.Exempt
	}{
		{EscapeArborName, m.EscapeArbor, g.O, ArborRadius, []cornercheck.Exempt{
			bearingStep(g.O, 0.25), bearingStep(g.O, 27.75)}},
		{PalletArborName, m.PalletArbor, g.P, endRadius, []cornercheck.Exempt{
			bearingStep(g.P, 0.25), bearingStep(g.P, 27.75), chordStop(g.P, 34.25)}},
		{GreatWheelArborName, m.GreatWheelArbor, g.G, endRadius, []cornercheck.Exempt{
			bearingStep(g.G, 0.25), bearingStep(g.G, 27.75), chordStop(g.G, 17.45), chordStop(g.G, 34.25)}},
		{SuspensionPinName, m.SuspensionPin, g.S, endRadius, []cornercheck.Exempt{
			bearingStep(g.S, 39.80), chordStop(g.S, 45.30)}},
	} {
		t.Run(c.name, func(t *testing.T) {
			point := r3.Vec{X: c.at.X, Y: c.at.Y}
			r, err := splitcheck.Check(t.Context(), splitcheck.Shaft{Body: c.body, Point: point, Axis: r3.Vec{Z: 1}},
				splitTolerance)
			require.NoError(t, err)
			require.Empty(t, r.Violations)
			// The flats face along +X in the shaft's own frame, so the split plane's normal is +Y turned by the pose.
			a := table[c.name].angle
			want := r3.Vec{X: -math.Sin(a), Y: math.Cos(a)}
			require.InDelta(t, 1, math.Abs(r.Normal.Dot(want)), 1e-6, "split normal %v", r.Normal)
			for i, h := range r.HalfHeights {
				require.InDelta(t, CollarRadius, h, splitTolerance, "half %d", i+1)
			}

			// MidHeights finds no height on a round shaft, so the heights are explicit: from the seam to 0.8 of the
			// smallest radius that meets a step or a cone. Above that a section cuts a cone so obliquely that a 30
			// degree cone reads over 50 degrees.
			heights, err := cornercheck.SplitHeights(point, r.Normal, c.rMin)
			require.NoError(t, err)
			fs, err := cornercheck.CheckAt(t.Context(), c.body, r.Normal, heights, c.exempt)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 6))
		})
	}
}
