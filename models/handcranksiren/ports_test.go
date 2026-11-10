package handcranksiren_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// requirePorts checks the rotor and the stator at the build frame (the pose undone through PosePoint): both are empty at
// every port centre and solid at every land centre of radius 36, the rotor is solid at each vane's mid-point, and at
// the posed state each rotor port lines up with a stator port.
func requirePorts(t *testing.T, p handcranksiren.Params, parts map[string]handcranksiren.Part) {
	n := p.Slots
	rotorAxis := r3.Vec{Z: 84}
	// solid reports whether the part's section at build-frame height y holds the build-frame point at (radius, angle).
	solid := func(name, motion string, y, radius, angle float64) bool {
		plane := frameAt(t, r3.Vec{Y: y}, r3.Vec{X: 1}, r3.Vec{Z: 1})
		outline, err := parts[name].Body.SectionOutline(t.Context(), plane, units.Millimeters(sectionTolerance))
		require.NoError(t, err)
		build := rotorAxis.Add(r3.Vec{X: radius * math.Cos(angle), Z: radius * math.Sin(angle)})
		posed, err := handcranksiren.PosePoint(p, motion, build)
		require.NoError(t, err)
		return sectionAt{outline: outline, origin: plane.Origin(), u: r3.Vec{X: 1}, v: r3.Vec{Z: 1}}.
			contains(posed.Sub(plane.Origin()))
	}
	for k := range n {
		port := 2 * math.Pi * float64(k) / float64(n)
		land := port + math.Pi/float64(n)
		require.False(t, solid("rotor", "rotor", -26.0, 36, port), "rotor port %d is open", k)
		require.True(t, solid("rotor", "rotor", -26.0, 36, land), "rotor land %d is solid", k)
		require.False(t, solid("stator", "fixed", -30.0, 36, port), "stator port %d is open", k)
		require.True(t, solid("stator", "fixed", -30.0, 36, land), "stator land %d is solid", k)
		require.True(t, solid("rotor", "rotor", -20.5, 26, land), "vane %d is solid at its mid-point", k)
		require.False(t, solid("rotor", "rotor", -20.5, 26, port), "no vane stands at port %d", k)
	}
	state, err := handcranksiren.Motion(p)
	require.NoError(t, err)
	pitch := 2 * math.Pi / float64(n)
	turns := math.Mod(state.RotorAngle, pitch) / pitch
	turns = math.Min(turns, 1-turns)
	require.InDelta(t, 0, turns*pitch, 1e-6, "a rotor port lines up with a stator port")
}
