package swisslever

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// partBuilder returns a builder for the default movement, with its own document and sketch world.
func partBuilder(t *testing.T) *builder {
	t.Helper()
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	return newBuilder(t.Context(), NewDocument(), g)
}

// partSection returns the section outline of body at height z, retrying at coarser chords as a faceted Boolean result
// refuses a chord below its own bound.
func partSection(t *testing.T, body *decad.Body, z float64) decad.SectionOutline {
	t.Helper()
	f, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	var o decad.SectionOutline
	for _, tol := range []float64{0.005, 0.02, 0.05} {
		o, err = body.SectionOutline(t.Context(), f, units.Millimeters(tol))
		if err == nil {
			break
		}
	}
	require.NoError(t, err)
	return o
}

// partInside reports whether (x, y, z) is in the solid, by even-odd over the section's loops.
func partInside(t *testing.T, body *decad.Body, x, y, z float64) bool {
	t.Helper()
	n := 0
	for _, l := range partSection(t, body, z).Loops {
		if pointInLoop(l, x, y) {
			n++
		}
	}
	return n%2 == 1
}

func pointInLoop(l []r3.Vec, x, y float64) bool {
	in := false
	for i, a := range l {
		c := l[(i+1)%len(l)]
		if (a.Y > y) == (c.Y > y) {
			continue
		}
		if x < a.X+(y-a.Y)*(c.X-a.X)/(c.Y-a.Y) {
			in = !in
		}
	}
	return in
}

// requireBoundsZ requires the body to span z0 to z1 and to fit within radius about the axis.
func requireBoundsZ(t *testing.T, body *decad.Body, z0, z1, radius float64) {
	t.Helper()
	bb, err := body.Bounds()
	require.NoError(t, err)
	const slack = 3e-3
	require.InDelta(t, z0, bb.Min.Z, slack)
	require.InDelta(t, z1, bb.Max.Z, slack)
	require.InDelta(t, radius, math.Max(math.Max(-bb.Min.X, bb.Max.X), math.Max(-bb.Min.Y, bb.Max.Y)), slack)
}

// requireSketchesConstrained requires every sketch the builder recorded to verify fully constrained, with DOF 0.
func requireSketchesConstrained(t *testing.T, b *builder, want int) {
	t.Helper()
	require.Len(t, b.sketches, want)
	for _, s := range b.sketches {
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s))
	}
}

// requireCorners runs cornercheck on body in its print orientation with exemptions for the mating D corners.
func requireCorners(t *testing.T, body *decad.Body, exempt ...cornercheck.Exempt) {
	t.Helper()
	fs, err := cornercheck.Check(t.Context(), body, r3.Vec{Z: 1}, exempt)
	require.NoError(t, err)
	require.Empty(t, fs, cornercheck.Report(fs, 5))
}
