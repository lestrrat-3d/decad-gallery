package m5bolt_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/m5bolt"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func requireFullyConstrained(t *testing.T, s *sketch.Sketch) {
	t.Helper()
	require.NotNil(t, s)
	report := s.Verify(t.Context())
	require.Equal(t, sketch.FullyConstrained, report.Status)
	require.Equal(t, 0, report.DOF)
	require.Empty(t, report.Redundant)
	require.Empty(t, report.Conflicts)
	require.NoError(t, report.Check())
}

// headVolume integrates the chamfered hexagonal head with the midpoint rule over the corner chamfer ring.
func headVolume() float64 {
	const steps = 20000
	s, e := m5bolt.AcrossFlats, m5bolt.AcrossCorners()
	dw := m5bolt.BearingDiameter
	slope := math.Tan(m5bolt.ChamferAngle * math.Pi / 180)
	depth := (e/2 - dw/2) * slope
	hexArea := math.Sqrt(3) / 2 * s * s
	// Removed volume of one chamfer: the part of the hexagon outside the cone, integrated over height.
	removed := 0.0
	dz := depth / steps
	for i := 0; i < steps; i++ {
		z := (float64(i) + 0.5) * dz
		radius := dw/2 + (depth-z)/slope
		removed += (hexArea - hexSection(radius, s)) * dz
	}
	return hexArea*m5bolt.HeadHeight - 2*removed
}

// hexSection returns the area of the regular hexagon of across-flats s that lies inside the circle of radius r.
func hexSection(r, s float64) float64 {
	const steps = 4000
	apothem := s / 2
	// Integrate over one 30 degree wedge from a flat's midpoint to a corner, then multiply by twelve.
	area := 0.0
	dtheta := (math.Pi / 6) / steps
	for i := 0; i < steps; i++ {
		theta := (float64(i) + 0.5) * dtheta
		rho := math.Min(r, apothem/math.Cos(theta))
		area += 0.5 * rho * rho * dtheta
	}
	return 12 * area
}
