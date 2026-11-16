package handcrankfan_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/stretchr/testify/require"
)

func TestBuildBlade(t *testing.T) {
	doc := handcrankfan.NewDocument()
	blade, err := handcrankfan.BuildBlade(t.Context(), doc, handcrankfan.DefaultBladePitch)
	require.NoError(t, err)
	require.True(t, blade.Body.IsSolid())
	require.Len(t, blade.Sketches, 12)
	for i, s := range blade.Sketches {
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
	}
	box, err := blade.Body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, handcrankfan.BladeStudStart, box.Min.Z, 0.01)
	require.InDelta(t, handcrankfan.BladeTipRadius(), box.Max.Z, 0.5)
	require.Len(t, doc.Bodies(), 1)

	for _, pitch := range []float64{9 * math.Pi / 180, 41 * math.Pi / 180, math.NaN()} {
		bad := handcrankfan.NewDocument()
		_, err := handcrankfan.BuildBlade(t.Context(), bad, pitch)
		require.Error(t, err)
		require.Empty(t, bad.Bodies())
	}
}

// TestBuildBladeMirrored builds the other hand: a solid with the same volume as the right-hand blade and bounds that are
// the right-hand blade's reflected in x = 0.
func TestBuildBladeMirrored(t *testing.T) {
	for _, pitch := range []float64{handcrankfan.MinBladePitch, handcrankfan.DefaultBladePitch, handcrankfan.MaxBladePitch} {
		doc := handcrankfan.NewDocument()
		right, err := handcrankfan.BuildBlade(t.Context(), doc, pitch)
		require.NoError(t, err)
		left, err := handcrankfan.BuildBladeMirrored(t.Context(), doc, pitch)
		require.NoError(t, err)
		require.True(t, left.Body.IsSolid())
		require.Len(t, left.Sketches, len(right.Sketches))
		for i, s := range left.Sketches {
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
		}
		rv, err := right.Body.Volume()
		require.NoError(t, err)
		lv, err := left.Body.Volume()
		require.NoError(t, err)
		require.InDelta(t, rv.Value.Base(), lv.Value.Base(), 2e-3*rv.Value.Base())
		rb, err := right.Body.Bounds()
		require.NoError(t, err)
		lb, err := left.Body.Bounds()
		require.NoError(t, err)
		tol := rb.Bound.Base() + lb.Bound.Base() + 1e-3
		require.InDelta(t, -rb.Max.X, lb.Min.X, tol)
		require.InDelta(t, -rb.Min.X, lb.Max.X, tol)
		require.InDelta(t, rb.Min.Y, lb.Min.Y, tol)
		require.InDelta(t, rb.Max.Y, lb.Max.Y, tol)
		require.InDelta(t, rb.Min.Z, lb.Min.Z, tol)
		require.InDelta(t, rb.Max.Z, lb.Max.Z, tol)
	}
}
