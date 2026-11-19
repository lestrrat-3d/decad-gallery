package handcrankfan_test

// CI: isolated-tests

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestPosedBladeJointAxes(t *testing.T) {
	p := handcrankfan.DefaultParams()
	p.CrankAngle = 0.37
	doc, bodies, assembly := buildFan(t, p)
	require.NotZero(t, assembly.State.FanAngle)
	set, err := assembly.Set(doc)
	require.NoError(t, err)
	joints := set.Joints()
	for k := 1; k <= turbinefan.BladeCount; k++ {
		name := fmt.Sprintf("blade %d stud in the hub", k)
		var axisPoint, axisDir r3.Vec
		found := false
		for _, joint := range joints {
			if joint.Name != name {
				continue
			}
			require.NotNil(t, joint.Axis, name)
			axisPoint, axisDir, found = joint.Axis.Point, joint.Axis.Dir, true
			break
		}
		require.True(t, found, name)
		theta := assembly.State.FanAngle + 2*math.Pi*float64(k-1)/turbinefan.BladeCount
		radial := r3.Vec{X: math.Cos(theta), Z: math.Sin(theta)}
		centre := assembly.State.Apex.Add(r3.Vec{Y: -88.5})
		require.InDelta(t, 1, axisDir.Dot(radial), 1e-12, name)
		require.LessOrEqual(t, axisPoint.Sub(centre).Cross(radial).Len(), 1e-9, name)

		point := centre.Add(radial.Scale(12))
		across := r3.Vec{X: -radial.Z, Z: radial.X}
		plane := frameAt(t, point, r3.Vec{Y: 1}, across)
		for _, check := range []struct {
			part   string
			radius float64
		}{
			{fmt.Sprintf("fan-blade-%d", k), 3},
			{"fan-hub", 3.405},
		} {
			outline, err := bodies[check.part].Body.SectionOutline(t.Context(), plane, units.Millimeters(0.05))
			require.NoError(t, err)
			found := false
			for _, loop := range outline.Loops {
				if len(loop) < 8 {
					continue
				}
				matches := true
				for _, vertex := range loop {
					if math.Abs(vertex.Sub(point).Len()-check.radius) > 0.06 {
						matches = false
						break
					}
				}
				found = found || matches
			}
			require.True(t, found, "%s: %s section centered on joint", name, check.part)
		}
	}

	// The assembly's fan transform rotates each original radial vector separately.
	// Adding a small blade offset directly to a large angle loses the offset.
	p.CrankAngle = 1e16
	assembly.State, err = handcrankfan.Motion(p)
	require.NoError(t, err)
	pose, err := r3.RotationAround(assembly.State.Apex, r3.Vec{Y: -1}, units.Radians(assembly.State.FanAngle))
	require.NoError(t, err)
	set, err = assembly.Set(doc)
	require.NoError(t, err)
	var directions []r3.Vec
	for k := 1; k <= turbinefan.BladeCount; k++ {
		name := fmt.Sprintf("blade %d stud in the hub", k)
		var direction r3.Vec
		found := false
		for _, joint := range set.Joints() {
			if joint.Name == name {
				require.NotNil(t, joint.Axis, name)
				direction, found = joint.Axis.Dir, true
				break
			}
		}
		require.True(t, found, name)
		angle := 2 * math.Pi * float64(k-1) / turbinefan.BladeCount
		want := pose.ApplyDir(r3.Vec{X: math.Cos(angle), Z: math.Sin(angle)})
		require.LessOrEqual(t, direction.Sub(want).Len(), 1e-12, name)
		for _, prior := range directions {
			require.Greater(t, direction.Sub(prior).Len(), 1.0, "joint directions must stay distinct")
		}
		directions = append(directions, direction)
	}
}
