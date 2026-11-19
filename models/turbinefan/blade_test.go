package turbinefan_test

// CI: isolated-tests

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestBladePrintEdge(t *testing.T) {
	for _, d := range bladeSizes {
		sz := mustSize(t, d)
		for _, pitch := range bladePitches {
			for _, mirror := range []bool{false, true} {
				t.Run(fmt.Sprintf("%g/%g/%t", d, pitch*180/math.Pi, mirror), func(t *testing.T) {
					build := turbinefan.BuildBlade
					if mirror {
						build = turbinefan.BuildBladeMirrored
					}
					doc := turbinefan.NewDocument()
					blade, err := build(t.Context(), doc, pitch, sz)
					require.NoError(t, err)
					require.Len(t, doc.Bodies(), 1)
					mesh := meshOf(t, blade.Body, 0.05)
					requireBladeBedContact(t, mesh, sz, mirror)
					requireBladePrintEdgeCorners(t, blade, sz, mirror)
				})
			}
		}
	}
}

func TestBuildBlade(t *testing.T) {
	for _, d := range bladeSizes {
		sz := mustSize(t, d)
		for _, pitch := range bladePitches {
			doc := turbinefan.NewDocument()
			blade, err := turbinefan.BuildBlade(t.Context(), doc, pitch, sz)
			require.NoError(t, err, "rotor %g, pitch %g", d, pitch)
			require.True(t, blade.Body.IsSolid())
			require.Len(t, blade.Sketches, 14)
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
			lo, hi := math.Inf(1), math.Inf(-1)
			for _, v := range ring {
				if math.Abs(v.X+sz.BladeStudRadius) < 1e-9 {
					lo, hi = math.Min(lo, v.Z), math.Max(hi, v.Z)
				}
			}
			require.InDelta(t, 2.0, hi-lo, 1e-8, "trailing flat at radius %g", y)
			mid := r3.Vec{X: -sz.BladeStudRadius, Y: y, Z: (lo + hi) / 2}
			le := farthest(ring, mid, nil)
			sourceRadius := sz.Scale * (19 + (y/sz.Scale-26)*45/38)
			requireThickness(t, turbinefan.ThicknessRatio(sourceRadius, sz), math.Atan2(-(le.Z-mid.Z), le.X-mid.X), ring, 0.03, y)
		}
		// The root is nearly round.
		y, ring := meshRing(verts, sz.RootRadius+0.6*sz.HubScale)
		requireThickness(t, 1, 0, ring, 0.1, y)
	})
}
