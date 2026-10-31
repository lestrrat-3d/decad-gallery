package spinningwheel_test

import (
	"math"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestSpinningWheelReferenceLayout(t *testing.T) {
	a, err := spinningwheel.Build(t.Context(), decad.New(), spinningwheel.DefaultParams())
	require.NoError(t, err)
	spokes := 0
	for _, part := range a.Parts {
		if strings.HasPrefix(part.Name, "wheel-spoke-") {
			spokes++
		}
	}
	require.Equal(t, 6, spokes, "the reference wheel has six spokes")
	wheel, err := wheelPart(t, a, "wheel-rim").Bounds()
	require.NoError(t, err)
	require.InDelta(t, 580, (wheel.Min.Z+wheel.Max.Z)/2, wheel.Bound.Base()+1e-6, "raised wheel hub")
	pedal, err := wheelPart(t, a, "treadle").Bounds()
	require.NoError(t, err)
	require.Greater(t, pedal.Max.Y-pedal.Min.Y, 2*(pedal.Max.X-pedal.Min.X), "footboard extends fore and aft")
	for _, name := range []string{"heel-bearing-1", "heel-bearing-2"} {
		for _, face := range wheelPart(t, a, name).Faces() {
			if cylinder, ok := face.Surface().(decad.Cylinder); ok {
				require.InDelta(t, 1, math.Abs(cylinder.Axis.X), 1e-10, "heel axis runs across the machine")
			}
		}
	}
	front, err := wheelPart(t, a, "front-maiden").Bounds()
	require.NoError(t, err)
	whorl, err := wheelPart(t, a, "flyer-drive-whorl").Bounds()
	require.NoError(t, err)
	crossbar, err := wheelPart(t, a, "flyer-crossbar").Bounds()
	require.NoError(t, err)
	bobbin, err := wheelPart(t, a, "bobbin").Bounds()
	require.NoError(t, err)
	require.Greater(t, whorl.Min.Y, front.Max.Y, "front support precedes the whorl")
	require.Less(t, whorl.Max.Y, crossbar.Min.Y, "whorl precedes the flyer crossbar")
	require.Less(t, crossbar.Max.Y, bobbin.Min.Y, "bobbin follows the flyer crossbar")
	rod, err := wheelPart(t, a, "footman").Bounds()
	require.NoError(t, err)
	require.Greater(t, rod.Min.Y, wheel.Max.Y, "footman runs behind the wheel")
}

func TestSpinningWheelRearRailClearance(t *testing.T) {
	p := spinningwheel.DefaultParams()
	a, err := spinningwheel.Build(t.Context(), decad.New(), p)
	require.NoError(t, err)
	pose := p
	pose.WheelAngle = 3 * math.Pi / 2
	state, err := spinningwheel.Motion(pose)
	require.NoError(t, err)
	xf, err := spinningwheel.PoseTransform(a.State, state, "treadle", pose.WheelAngle-p.WheelAngle)
	require.NoError(t, err)
	rail, err := decad.Copy(t.Context(), wheelPart(t, a, "treadle-actuating-rail"), decad.WithCopyPlacement(xf))
	require.NoError(t, err)
	defer func() { require.NoError(t, decad.Remove(rail)) }()
	probe := r3.Vec{X: 201, Y: 219, Z: 34}
	legLocation, err := wheelPart(t, a, "rear-wheel-leg").LocatePoint(t.Context(), probe, units.Millimeters(.02))
	require.NoError(t, err)
	require.True(t, legLocation == decad.PointInside, "probe must be inside the structural leg, got %s", legLocation)
	railLocation, err := rail.LocatePoint(t.Context(), probe, units.Millimeters(.02))
	require.NoError(t, err)
	require.True(t, railLocation == decad.PointOutside, "the actuating rail must clear the leg's interior, got %s", railLocation)
}
