package turbinefan_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestBuildBlade(t *testing.T) {
	for _, d := range bladeSizes {
		sz := mustSize(t, d)
		for _, pitch := range bladePitches {
			doc := turbinefan.NewDocument()
			blade, err := turbinefan.BuildBlade(t.Context(), doc, pitch, sz)
			require.NoError(t, err, "rotor %g, pitch %g", d, pitch)
			require.True(t, blade.Body.IsSolid())
			// Ten sections, the stud, the bolt hole and the stud flat's cutter.
			require.Len(t, blade.Sketches, 13)
			for i, s := range blade.Sketches {
				require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
			}
			box, err := blade.Body.Bounds()
			require.NoError(t, err)
			require.InDelta(t, sz.BladeStudStart, box.Min.Z, 0.01, "rotor %g, pitch %g", d, pitch)
			require.InDelta(t, sz.TipRadius, box.Max.Z, 0.5, "rotor %g, pitch %g", d, pitch)
			require.Len(t, doc.Bodies(), 1)
		}
	}

	sz := mustSize(t, 192)
	for _, pitch := range []float64{9 * math.Pi / 180, 41 * math.Pi / 180, math.NaN()} {
		bad := turbinefan.NewDocument()
		_, err := turbinefan.BuildBlade(t.Context(), bad, pitch, sz)
		require.Error(t, err)
		require.Empty(t, bad.Bodies())
	}
	bad := turbinefan.NewDocument()
	_, err := turbinefan.BuildBlade(t.Context(), bad, turbinefan.DefaultBladePitch, turbinefan.Size{})
	require.Error(t, err, "a Size that SizeFor did not make")
	require.Empty(t, bad.Bodies())
}

// TestBuildBladeMirrored builds the other hand: a solid with the same volume as the right-hand blade and bounds that are
// the right-hand blade's reflected in x = 0.
func TestBuildBladeMirrored(t *testing.T) {
	for _, d := range bladeSizes {
		sz := mustSize(t, d)
		for _, pitch := range bladePitches {
			doc := turbinefan.NewDocument()
			right, err := turbinefan.BuildBlade(t.Context(), doc, pitch, sz)
			require.NoError(t, err, "rotor %g, pitch %g", d, pitch)
			left, err := turbinefan.BuildBladeMirrored(t.Context(), doc, pitch, sz)
			require.NoError(t, err, "rotor %g, pitch %g", d, pitch)
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
}

// TestBladeThickness requires the blade's thickness ratio at each ring to match the station table: the ring's extent
// across the chord against its extent along it.
func TestBladeThickness(t *testing.T) {
	eachCase(t, []string{"605ZZ", "compact", "large"}, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		sz := m.Dimensions.Size
		verts := meshOf(t, m.Blades[0], 0.1).Vertices()
		for _, r := range []float64{30, 40, 50, 60} {
			y, ring := meshRing(verts, r*sz.Scale)
			require.NotEmpty(t, ring, "ring near radius %g", r)
			stack := r3.Vec{Y: y, Z: sz.HubMid}
			te1 := farthest(ring, stack, nil)
			le := farthest(ring, te1, nil)
			te2 := farthest(ring, le, func(v r3.Vec) bool { return v.Sub(te1).Len() < 0.2 })
			mid := te1.Add(te2).Scale(0.5)
			requireThickness(t, turbinefan.ThicknessRatio(y, sz), math.Atan2(-(le.Z-mid.Z), le.X-mid.X), ring, 0.03, y)
		}
		// The root is nearly round.
		y, ring := meshRing(verts, sz.RootRadius+0.6*sz.HubScale)
		requireThickness(t, 1, 0, ring, 0.1, y)
	})
}
