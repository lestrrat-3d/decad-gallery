package screwgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/screwgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestRibbonDefaultIsOneVerifiedLoft lofts gear A's default ribbon, 68 teeth at ten sections a
// pitch (681 sections, a step of Length()/680 = 0.2625 mm), as one solid and checks it the way
// decad checks a twisted fitted-edge loft: the volume against the denoted solid, the verified
// mesh, the mesh volume, and a rigid motion of the body. The ribbon is the unplaced loft the
// build places, so the real producer runs into the real consumer.
func TestRibbonDefaultIsOneVerifiedLoft(t *testing.T) {
	p := screwgear.DefaultParams()
	doc := decad.New()
	body, sections, step, err := screwgear.RibbonForTest(t.Context(), doc, p, 0)
	require.NoError(t, err)
	require.Len(t, sections, p.ToothCount*10+1)
	require.Equal(t, decad.BodySolid, body.Kind())
	require.Len(t, body.Faces(), 6)

	reference := denotedRibbonReference(t, sections, step)
	volume, err := body.Volume()
	require.NoError(t, err)
	t.Logf("volume %.3f reference %.3f bound %.3f", volume.Value.Base(), reference, volume.Bound.Base())
	require.LessOrEqual(t, math.Abs(volume.Value.Base()-reference), volume.Bound.Base())
	require.Less(t, volume.Bound.Base(), 0.03*reference)

	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	require.LessOrEqual(t, mesh.Bound().Base(), 0.05)
	closed, signed := meshIsWatertight(mesh)
	require.True(t, closed)
	require.LessOrEqual(t, math.Abs(signed-reference), volume.Bound.Base())
	t.Logf("mesh: %d triangles, bound %.5f", len(mesh.Triangles()), mesh.Bound().Base())

	// A rigid motion keeps the body usable and its readings.
	motion, err := r3.RotationAround(r3.NewVec(1, 2, 3), r3.NewVec(1, 1, 0), units.Radians(0.7))
	require.NoError(t, err)
	moved, err := decad.Copy(t.Context(), body, decad.WithCopyPlacement(motion))
	require.NoError(t, err)
	movedVolume, err := moved.Volume()
	require.NoError(t, err)
	require.LessOrEqual(t, math.Abs(movedVolume.Value.Base()-volume.Value.Base()),
		movedVolume.Bound.Base()+volume.Bound.Base())
	movedMesh, err := moved.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, movedMesh.BoundaryVerified())
	require.True(t, movedMesh.VolumeVerified())
}

// denotedRibbonReference integrates the section areas of the denoted solid without decad: each
// boundary sample moves along the C1 Hermite curve through its counterparts in every section,
// and the shoelace area of the sampled loop is integrated over height by Gauss-Legendre
// quadrature. It reads the sections through the public sketch API only.
func denotedRibbonReference(t *testing.T, sections []decad.LoftSection, step float64) float64 {
	t.Helper()
	const lineSamples, splineSamples = 24, 96
	samples := make([][][2]float64, len(sections))
	for i, section := range sections {
		var perEdge [][][2]float64
		fitAt := -1
		for e, edge := range section.Profile.Outer {
			var pts [][2]float64
			switch curve := edge.Entity.(type) {
			case *sketch.Line:
				x0, y0, x1, y1 := curve.Start.X(), curve.Start.Y(), curve.End.X(), curve.End.Y()
				for j := range lineSamples {
					f := float64(j) / lineSamples
					pts = append(pts, [2]float64{x0 + f*(x1-x0), y0 + f*(y1-y0)})
				}
			case *sketch.FitSpline:
				fitAt = e
				// Spans are the chord-length intervals between fit points.
				chord := make([]float64, len(curve.Fit))
				for j := 1; j < len(chord); j++ {
					chord[j] = chord[j-1] + math.Hypot(curve.Fit[j].X()-curve.Fit[j-1].X(),
						curve.Fit[j].Y()-curve.Fit[j-1].Y())
				}
				for j := range len(chord) - 1 {
					for k := range splineSamples {
						f := float64(k) / splineSamples
						x, y := curve.Eval((chord[j] + f*(chord[j+1]-chord[j])) / chord[len(chord)-1])
						pts = append(pts, [2]float64{x, y})
					}
				}
			default:
				t.Fatalf("unexpected entity %T", edge.Entity)
			}
			require.False(t, edge.Reversed, "the reference samples assume a forward walk")
			perEdge = append(perEdge, pts)
		}
		require.GreaterOrEqual(t, fitAt, 0)
		for e := range perEdge {
			samples[i] = append(samples[i], perEdge[(fitAt+e)%len(perEdge)]...)
		}
	}
	n := len(sections)
	count := len(samples[0])
	for i := range samples {
		require.Len(t, samples[i], count)
	}
	tangent := func(i, k, axis int) float64 {
		switch i {
		case 0:
			return samples[1][k][axis] - samples[0][k][axis]
		case n - 1:
			return samples[n-1][k][axis] - samples[n-2][k][axis]
		}
		return (samples[i+1][k][axis] - samples[i-1][k][axis]) / 2
	}
	// point is boundary sample k at height span i and parameter s.
	point := func(i, k int, s float64) [2]float64 {
		h00, h10 := 2*s*s*s-3*s*s+1, s*s*s-2*s*s+s
		h01, h11 := -2*s*s*s+3*s*s, s*s*s-s*s
		var pt [2]float64
		for axis := range 2 {
			pt[axis] = h00*samples[i][k][axis] + h10*tangent(i, k, axis) +
				h01*samples[i+1][k][axis] + h11*tangent(i+1, k, axis)
		}
		return pt
	}
	shoelace := func(pts [][2]float64) float64 {
		twice := 0.0
		for k, pt := range pts {
			q := pts[(k+1)%len(pts)]
			twice += pt[0]*q[1] - pt[1]*q[0]
		}
		return twice / 2
	}
	nodes := [5]float64{0, -0.5384693101056831, 0.5384693101056831, -0.9061798459386640, 0.9061798459386640}
	weights := [5]float64{0.5688888888888889, 0.4786286704993665, 0.4786286704993665, 0.2369268850561891, 0.2369268850561891}
	volume := 0.0
	ring := make([][2]float64, count)
	for i := range n - 1 {
		for g := range nodes {
			for k := range ring {
				ring[k] = point(i, k, (nodes[g]+1)/2)
			}
			volume += weights[g] / 2 * step * shoelace(ring)
		}
	}
	return volume
}

// meshIsWatertight reports whether every undirected edge has exactly one triangle in each
// direction, and returns the signed volume.
func meshIsWatertight(mesh *decad.Mesh) (bool, float64) {
	type edge struct{ a, b int }
	directed := make(map[edge]int)
	vertices := mesh.Vertices()
	volume := 0.0
	for _, tri := range mesh.Triangles() {
		for k := range 3 {
			directed[edge{tri[k], tri[(k+1)%3]}]++
		}
		a, b, c := vertices[tri[0]], vertices[tri[1]], vertices[tri[2]]
		volume += a.Dot(b.Cross(c)) / 6
	}
	for e, count := range directed {
		if count != 1 || directed[edge{e.b, e.a}] != 1 {
			return false, volume
		}
	}
	return true, volume
}
