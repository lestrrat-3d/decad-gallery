package spinningwheel_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func appendDistinctCentre(centres []r3.Vec, centre r3.Vec) []r3.Vec {
	for _, existing := range centres {
		if existing.Sub(centre).Len() < 1e-6 {
			return centres
		}
	}
	return append(centres, centre)
}

// Read the actual uncut rivet's circles; neither its orientation nor its centre
// comes from a duplicated renderer pose or a guessed spring bounding box.
func brakePinGeometry(t *testing.T, body *decad.Body) (r3.Vec, r3.Vec) {
	t.Helper()
	var shank, heads []r3.Vec
	for _, face := range body.Faces() {
		for _, loop := range face.Loops() {
			for _, edge := range loop.Edges() {
				circle, ok := edge.Curve().(decad.Circle3)
				if !ok {
					continue
				}
				radius := circle.Radius.Base()
				switch {
				case math.Abs(radius-1.5) < 1e-6:
					shank = appendDistinctCentre(shank, circle.Center)
				case math.Abs(radius-2.3) < 1e-6:
					heads = appendDistinctCentre(heads, circle.Center)
				default:
					t.Fatalf("unexpected brake rivet circle radius %g mm", radius)
				}
			}
		}
	}
	require.Equal(t, 2, len(shank), "both ends of the actual 3 mm shank must be present")
	require.Equal(t, 4, len(heads), "both 4.6 mm heads must have two circular ends")
	direction := shank[1].Sub(shank[0])
	require.InDelta(t, 6, direction.Len(), 1e-6, "actual rivet shank length")
	axis, ok := direction.Normalize()
	require.True(t, ok, "rivet axis must have nonzero length")
	centre := shank[0].Add(shank[1]).Scale(.5)
	ends := 0
	for _, point := range heads {
		delta := point.Sub(centre)
		axial := delta.Dot(axis)
		require.Less(t, delta.Sub(axis.Scale(axial)).Len(), 1e-6, "rivet heads must be coaxial")
		if math.Abs(math.Abs(axial)-4) < 1e-6 {
			ends++
			continue
		}
		require.InDelta(t, 3, math.Abs(axial), 1e-6, "actual head inner shoulder")
	}
	require.Equal(t, 2, ends, "both heads must extend past the spring eye")
	volume, err := body.Volume()
	require.NoError(t, err)
	require.InDelta(t, 24.08*math.Pi, volume.Value.Base(), volume.Bound.Base()+1e-6, "solid peened rivet")
	return centre, axis
}

func requireEyeMaterial(t *testing.T, name string, mesh surfaceTriangles, centre, u, v r3.Vec, radius float64) {
	t.Helper()
	require.False(t, mesh.contains(centre), "%s eye must have an open centre", name)
	for i := range 16 {
		angle := float64(i) * math.Pi / 8
		point := centre.Add(u.Scale(radius * math.Cos(angle))).Add(v.Scale(radius * math.Sin(angle)))
		require.True(t, mesh.contains(point), "%s eye lacks wire at sample %d", name, i)
	}
}

func requireBrakeCaptivity(t *testing.T, a *spinningwheel.Assembly, p spinningwheel.Params, meshes map[string]surfaceTriangles) {
	t.Helper()
	band := meshes["brake-band"]
	require.Equal(t, -2, meshEulerCharacteristic(band.triangles), "the one-piece cord needs two closed terminal eyes")
	for _, side := range []string{"left", "right"} {
		name := "brake-spring-" + side
		spring := meshes[name]
		require.Equal(t, -2, meshEulerCharacteristic(spring.triangles), "%s needs two closed eyes", name)
		front, pinAxis := brakePinGeometry(t, wheelPart(t, a, "brake-front-pin-"+side))
		y := r3.Vec{Y: 1}
		require.InDelta(t, 0, pinAxis.Dot(y), 1e-6, "pin axis must be transverse to spring travel")
		v, ok := pinAxis.Cross(y).Normalize()
		require.True(t, ok, "rivet axis must define the eye plane")
		rear := front.Add(y.Scale(25*p.Scale + p.BrakeTravel/2 + 6))
		requireEyeMaterial(t, name+" front", spring, front, y, v, 2.1)
		requireEyeMaterial(t, name+" rear", spring, rear, y, v, 2.1)
		// The perpendicular terminal circle crosses the rear eye's plane twice.
		// Their 3 mm centre offset and radii put exactly one crossing inside the
		// other eye. The sampled material and closed handles anchor those circles
		// to the actual verified bodies; the separate overlap screen checks wire clearance.
		requireEyeMaterial(t, "brake-band "+side, band, rear.Add(y.Scale(3)), y, pinAxis, 2.4)
	}
}
