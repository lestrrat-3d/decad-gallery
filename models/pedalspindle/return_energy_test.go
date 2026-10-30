package pedalspindle_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/pedalspindle"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// This is an energy screening under stated loads, not a driven-body simulation.
// Full-cycle potential ranges and a full turn of drag overestimate one return.
func requireReturnEnergy(t *testing.T, a *pedalspindle.Assembly) {
	t.Helper()
	const steelDensity = 7.8e-6 // kg/mm3 = 7800 kg/m3.
	const timberDensity = 6e-7  // kg/mm3 = 600 kg/m3.
	const gravity = 9.81
	const footForce = 5.0 // Residual downward N at 150 mm from the heel.
	const crankDrag = .05 // Total load and friction torque referred to input, Nm.
	rim := partByName(t, a, "flywheel-rim")
	properties, err := rim.MassProperties(t.Context(), units.KilogramsPerCubicMillimeter(steelDensity))
	require.NoError(t, err)
	inertia := (properties.Inertia.YY.Value.Base() - properties.Inertia.YY.Bound.Base()) * 1e-6 // kg m2.
	require.InDelta(t, 2.5583, properties.Mass.Value.Base(), .001, "specified steel rim mass")
	require.Greater(t, inertia, .053)
	entryOmega := 60.0 * 2 * math.Pi / 60
	minimumOmega := 30.0 * 2 * math.Pi / 60
	available := .5 * inertia * (entryOmega*entryOmega - minimumOmega*minimumOmega)
	potential := 0.0
	for _, part := range a.Parts {
		_, _, _, moving := linkagePlacement(part.Name, a.State, a.State, 0)
		if !moving {
			continue
		}
		density := steelDensity
		if part.Name == "treadle" || part.Name == "footman" {
			density = timberDensity
		}
		volume, err := part.Body.Volume()
		require.NoError(t, err)
		mass := (volume.Value.Base() + volume.Bound.Base()) * density
		centroid, err := part.Body.Centroid()
		require.NoError(t, err)
		minZ, maxZ := math.Inf(1), math.Inf(-1)
		for i := range 721 {
			p := pedalspindle.DefaultParams()
			p.CrankAngle = float64(i) * 2 * math.Pi / 720
			state, err := pedalspindle.Motion(p)
			require.NoError(t, err)
			center, angle, offset, _ := linkagePlacement(part.Name, a.State, state, p.CrankAngle-math.Pi/3)
			delta := centroid.Value.Sub(center)
			z := center.Z + delta.X*math.Sin(angle) + delta.Z*math.Cos(angle) + offset.Z
			minZ, maxZ = math.Min(minZ, z), math.Max(maxZ, z)
		}
		potential += mass * gravity * (maxZ - minZ + 2*centroid.Bound.Base()) / 1000
	}
	minFootZ, maxFootZ := math.Inf(1), math.Inf(-1)
	for i := range 721 {
		p := pedalspindle.DefaultParams()
		p.CrankAngle = float64(i) * 2 * math.Pi / 720
		state, err := pedalspindle.Motion(p)
		require.NoError(t, err)
		z := state.Heel.Z + 150*math.Sin(state.PedalAngle)
		minFootZ, maxFootZ = math.Min(minFootZ, z), math.Max(maxFootZ, z)
	}
	footWork := footForce * (maxFootZ - minFootZ) / 1000
	dragWork := crankDrag * 2 * math.Pi
	budget := potential + footWork + dragWork
	require.Greater(t, available, 1.25*budget, "return energy: available %.4f J, potential %.4f J, foot %.4f J, drag %.4f J",
		available, potential, footWork, dragWork)
	// A lightweight printed rim of identical shape must not satisfy this same load case.
	require.Less(t, available*1200/7800, budget, "the screening must distinguish a light rim from the specified steel rim")
	t.Logf("steel rim %.4f kg, inertia %.5f kg m2; available %.4f J, budget %.4f J, factor %.3f",
		properties.Mass.Value.Base(), inertia, available, budget, available/budget)
}
