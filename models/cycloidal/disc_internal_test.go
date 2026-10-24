package cycloidal

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestBuildDefaultDisc(t *testing.T) {
	doc := decad.New(decad.WithChainTolerance(units.Millimeters(0.1)))
	r, err := begin(t.Context(), doc, DefaultParams())
	require.NoError(t, err)
	body, err := r.buildDisc(t.Context(), 0)
	require.NoError(t, err)
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	volume, err := body.Volume()
	require.NoError(t, err)
	box, err := body.Bounds()
	require.NoError(t, err)
	t.Logf("volume %.1f ± %.2f, bounds %+v", volume.Value.Base(), volume.Bound.Base(), box)
	// The rotor reaches the peak of the housing contour, R - Rr + 2E - c, from its own center E away.
	d := r.d
	require.Less(t, math.Max(box.Max.X, -box.Min.X), d.pinCircleRadius)
	require.InDelta(t, r.p.DiscThickness, box.Max.Z-box.Min.Z, 1e-6)
}

// outlinePerimeter returns the length of the rotor outline of d, summed over a fine sampling of its lobes.
func outlinePerimeter(p Params, d dimensions) float64 {
	const steps = 200000
	span := 2 * math.Pi / float64(d.lobes)
	var length float64
	prev := discPoint(p, d, 0, 0, 0, 0)
	for i := 1; i <= steps; i++ {
		cur := discPoint(p, d, span*float64(i)/steps, 0, 0, 0)
		length += math.Hypot(cur.x-prev.x, cur.y-prev.y)
		prev = cur
	}
	return length * float64(d.lobes)
}

// A 45° chamfer c on a rim of perimeter P removes P·c²/2 - π·c³/3 of volume, because the section of a parallel
// outline loses P·s - π·s² of area at inset s. Both rims of the disc are chamfered.
func TestChamferedDiscVolume(t *testing.T) {
	volume := func(chamfer float64) (*run, float64, float64) {
		p := DefaultParams()
		p.ChamferSize = chamfer
		doc := decad.New(decad.WithChainTolerance(units.Millimeters(0.1)))
		r, err := begin(t.Context(), doc, p)
		require.NoError(t, err)
		body, err := r.chamferedDisc(t.Context(), 0)
		require.NoError(t, err)
		mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		require.True(t, mesh.BoundaryVerified())
		require.True(t, mesh.VolumeVerified())
		v, err := body.Volume()
		require.NoError(t, err)
		return r, v.Value.Base(), v.Bound.Base()
	}
	_, sharp, _ := volume(0)
	r, chamfered, bound := volume(0.5)
	c := r.p.ChamferSize
	want := 2 * (outlinePerimeter(r.p, r.d)*c*c/2 - math.Pi*c*c*c/3)
	// Decad's volume bound is the tolerance: the outline is a fit spline, and the mesh is only certified to it.
	require.InDelta(t, want, sharp-chamfered, bound)
	require.Greater(t, sharp-chamfered, 0.5*want)
}
