package turbinefan_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestBuildBlade(t *testing.T) {
	doc := turbinefan.NewDocument()
	blade, err := turbinefan.BuildBlade(t.Context(), doc, turbinefan.DefaultBladePitch)
	require.NoError(t, err)
	require.True(t, blade.Body.IsSolid())
	require.Len(t, blade.Sketches, 12)
	for i, s := range blade.Sketches {
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
	}
	box, err := blade.Body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, turbinefan.BladeStudStart, box.Min.Z, 0.01)
	require.InDelta(t, turbinefan.BladeTipRadius(), box.Max.Z, 0.5)
	require.Len(t, doc.Bodies(), 1)

	for _, pitch := range []float64{9 * math.Pi / 180, 41 * math.Pi / 180, math.NaN()} {
		bad := turbinefan.NewDocument()
		_, err := turbinefan.BuildBlade(t.Context(), bad, pitch)
		require.Error(t, err)
		require.Empty(t, bad.Bodies())
	}
}

// TestBuildBladeMirrored builds the other hand: a solid with the same volume as the right-hand blade and bounds that are
// the right-hand blade's reflected in x = 0.
func TestBuildBladeMirrored(t *testing.T) {
	for _, pitch := range []float64{turbinefan.MinBladePitch, turbinefan.DefaultBladePitch, turbinefan.MaxBladePitch} {
		doc := turbinefan.NewDocument()
		right, err := turbinefan.BuildBlade(t.Context(), doc, pitch)
		require.NoError(t, err)
		left, err := turbinefan.BuildBladeMirrored(t.Context(), doc, pitch)
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

// TestBladeThickness requires the blade's thickness ratio at each ring to match the station table: the ring's extent
// across the chord against its extent along it.
func TestBladeThickness(t *testing.T) {
	eachCase(t, []string{"605ZZ"}, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		verts := meshOf(t, m.Blades[0], 0.1).Vertices()
		for _, r := range []float64{30, 40, 50, 60} {
			y, ring := meshRing(verts, r)
			require.NotEmpty(t, ring, "ring near radius %g", r)
			stack := r3.Vec{Y: y, Z: turbinefan.HubMid}
			te1 := farthest(ring, stack, nil)
			le := farthest(ring, te1, nil)
			te2 := farthest(ring, le, func(v r3.Vec) bool { return v.Sub(te1).Len() < 0.2 })
			mid := te1.Add(te2).Scale(0.5)
			requireThickness(t, turbinefan.ThicknessRatio(y), math.Atan2(-(le.Z-mid.Z), le.X-mid.X), ring, 0.03, y)
		}
		// The root is nearly round.
		y, ring := meshRing(verts, 15)
		requireThickness(t, 1, 0, ring, 0.1, y)
	})
}

// requireThickness compares the ring's extent across the chord direction with its extent along it. The chord runs at
// angle from +X toward -Z.
func requireThickness(t *testing.T, want, angle float64, ring []r3.Vec, tol, y float64) {
	t.Helper()
	along := r3.Vec{X: math.Cos(angle), Z: -math.Sin(angle)}
	across := r3.Vec{X: math.Sin(angle), Z: math.Cos(angle)}
	lo, hi := [2]float64{math.Inf(1), math.Inf(1)}, [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, v := range ring {
		for i, axis := range []r3.Vec{along, across} {
			d := v.X*axis.X + v.Z*axis.Z
			lo[i], hi[i] = math.Min(lo[i], d), math.Max(hi[i], d)
		}
	}
	require.InDelta(t, want, (hi[1]-lo[1])/(hi[0]-lo[0]), tol, "thickness ratio at radius %g", y)
}
