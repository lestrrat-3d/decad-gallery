package turbinefan_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// bladeSizes are the rotor diameters every blade test covers: the compact, default and large rotors.
var bladeSizes = []float64{128, 192, 224}

// bladePitches are the pitches every blade test covers: the least, the default and the greatest.
var bladePitches = []float64{turbinefan.MinBladePitch, turbinefan.DefaultBladePitch, turbinefan.MaxBladePitch}

func mustSize(t *testing.T, d float64) turbinefan.Size {
	t.Helper()
	sz, err := turbinefan.SizeFor(d)
	require.NoError(t, err)
	return sz
}

func TestBladePrintEdgeCorners(t *testing.T) {
	sz := mustSize(t, 128)
	blade, err := turbinefan.BuildBlade(t.Context(), turbinefan.NewDocument(), turbinefan.DefaultBladePitch, sz)
	require.NoError(t, err)
	findings, err := cornercheck.CheckAt(t.Context(), blade.Body, r3.Vec{X: 1},
		[]float64{-2.8, -2, -1, 0, 1, 2, 3}, nil)
	require.NoError(t, err)
	require.Empty(t, findings, cornercheck.Report(findings, 5))
}

func TestBladePrintEdge(t *testing.T) {
	for _, d := range bladeSizes {
		sz := mustSize(t, d)
		for _, pitch := range bladePitches {
			for _, mirror := range []bool{false, true} {
				build := turbinefan.BuildBlade
				angle := pitch - turbinefan.DefaultBladePitch
				if mirror {
					build = turbinefan.BuildBladeMirrored
					angle = -angle
				}
				blade, err := build(t.Context(), turbinefan.NewDocument(), pitch, sz)
				require.NoError(t, err)
				flatTip := false
				for _, v := range meshOf(t, blade.Body, 0.05).Vertices() {
					x := v.X*math.Cos(angle) + v.Y*math.Sin(angle)
					if mirror {
						x = -x
					}
					require.GreaterOrEqual(t, x, -sz.BladeStudRadius-0.05)
					if math.Abs(x+sz.BladeStudRadius) < 0.05 && v.Z > sz.TipRadius-1 {
						flatTip = true
					}
				}
				require.True(t, flatTip, "rotor %g, pitch %g, mirrored %t: print face must reach tip", d, pitch, mirror)
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
