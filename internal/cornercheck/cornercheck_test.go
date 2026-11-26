package cornercheck_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestInsideCorners(t *testing.T) {
	lShape := []xy{{5, 5}, {25, 5}, {25, 15}, {15, 15}, {15, 25}, {5, 25}}
	round := func(r float64) []float64 { return []float64{0, 0, 0, r, 0, 0} }

	t.Run("sharp notch", func(t *testing.T) {
		fs := requireFindings(t, axisPolygon(t, lShape, round(0)), cornercheck.Inside, 1)
		require.Len(t, fs, 1)
		require.InDelta(t, 15, fs[0].At.X, 0.01)
		require.InDelta(t, 15, fs[0].At.Y, 0.01)
		require.InDelta(t, 90, fs[0].Turn, 1)
	})
	t.Run("round 0.5", func(t *testing.T) {
		requireFindings(t, axisPolygon(t, lShape, round(0.5)), cornercheck.Inside, 0)
	})
	t.Run("round 0.4", func(t *testing.T) {
		requireFindings(t, axisPolygon(t, lShape, round(0.4)), cornercheck.Inside, 1)
	})
	t.Run("45 degree turn", func(t *testing.T) {
		body := polygon(t, []xy{{5, 5}, {25, 5}, {25, 15}, {15, 15}, {5, 25}})
		fs, err := cornercheck.Check(t.Context(), body, axisZ, nil)
		require.NoError(t, err)
		for _, f := range fs {
			require.NotEqual(t, cornercheck.Inside, f.Kind, cornercheck.Report(fs, 5))
		}
	})
	t.Run("slot ending in a line", func(t *testing.T) {
		requireFindings(t, slotPlate(t, false), cornercheck.Inside, 2)
	})
	t.Run("slot ending in a round", func(t *testing.T) {
		requireFindings(t, slotPlate(t, true), cornercheck.Inside, 0)
	})
	t.Run("D bore", func(t *testing.T) {
		requireFindings(t, dBorePlate(t), cornercheck.Inside, 0)
	})
}

func TestTips(t *testing.T) {
	t.Run("60 degrees", func(t *testing.T) {
		requireFindings(t, wedge(t, 60, 0), cornercheck.Tip, 0)
	})
	t.Run("45 degrees", func(t *testing.T) {
		fs := requireFindings(t, wedge(t, 45, 0), cornercheck.Tip, 1)
		require.InDelta(t, 15, fs[0].At.X, 0.01)
		require.InDelta(t, 135, fs[0].Turn, 1)
	})
	t.Run("45 degrees blunted", func(t *testing.T) {
		requireFindings(t, wedge(t, 45, 0.4), cornercheck.Tip, 0)
	})
}

func TestFacetedBody(t *testing.T) {
	doc := decad.New()
	box := func(x0, y0, x1, y1 float64) *decad.Body {
		s := newSketch(t)
		r := s.CreateRectangle(x0, y0, x1, y1)
		s.AddConstraint(
			sketch.NewHorizontalDistance(s.Origin(), r.A, x0),
			sketch.NewVerticalDistance(s.Origin(), r.A, y0),
			sketch.NewHorizontalDistance(r.A, r.B, x1-x0),
			sketch.NewVerticalDistance(r.B, r.C, y1-y0),
		)
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s))
		_, err := s.Solve(t.Context())
		require.NoError(t, err)
		body, err := decad.Extrude(doc, s, s.Profiles()[0],
			decad.Distance{D: units.Millimeters(plateHeight), Dir: decad.Along}).Do(t.Context())
		require.NoError(t, err)
		return body
	}
	union, err := decad.Union(box(5, 5, 25, 15), box(5, 5, 15, 25)).Do(t.Context())
	require.NoError(t, err)
	// The union's only inside corner is the notch at (15, 15).
	fs := requireFindings(t, union, cornercheck.Inside, 1)
	require.InDelta(t, 15, fs[0].At.X, 0.01)
	require.InDelta(t, 15, fs[0].At.Y, 0.01)
}

func TestExemptions(t *testing.T) {
	body := axisPolygon(t, []xy{{5, 5}, {25, 5}, {25, 15}, {15, 15}, {15, 25}, {5, 25}}, make([]float64, 6))
	notch := cornercheck.Exempt{Reason: "notch", Near: cornercheck.Disc(r3.Vec{X: 15, Y: 15}, axisZ, 0.5)}
	away := cornercheck.Exempt{Reason: "unrelated spot", Near: cornercheck.Disc(r3.Vec{X: 5, Y: 5}, axisZ, 0.5)}

	fs, err := cornercheck.Check(t.Context(), body, axisZ, []cornercheck.Exempt{notch})
	require.NoError(t, err)
	require.Empty(t, fs, cornercheck.Report(fs, 5))

	_, err = cornercheck.Check(t.Context(), body, axisZ, []cornercheck.Exempt{notch, away})
	require.ErrorContains(t, err, "unrelated spot")

	ring := cornercheck.Annulus(r3.Vec{X: 15, Y: 15}, axisZ, 0, 0.5)
	require.True(t, ring(r3.Vec{X: 15.2, Y: 15, Z: 99}))
	require.False(t, ring(r3.Vec{X: 16, Y: 15}))
}

func TestSplitHeights(t *testing.T) {
	hs, err := cornercheck.SplitHeights(r3.Vec{X: 5, Y: 2}, r3.Vec{Y: 2}, 2.0)
	require.NoError(t, err)
	// 0.1, 0.35, ..., 1.6 on each side of the seam at y 2.
	require.Len(t, hs, 14)
	require.InDelta(t, 1.9, hs[0], 1e-12)
	require.InDelta(t, 2.1, hs[1], 1e-12)
	require.InDelta(t, 0.4, hs[12], 1e-12)
	require.InDelta(t, 3.6, hs[13], 1e-12)
	_, err = cornercheck.SplitHeights(r3.Vec{}, r3.Vec{}, 2)
	require.Error(t, err)
	_, err = cornercheck.SplitHeights(r3.Vec{}, r3.Vec{Z: 1}, 0)
	require.Error(t, err)
}
