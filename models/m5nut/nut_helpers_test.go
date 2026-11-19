package m5nut_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/m5nut"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func requireFullyConstrained(t *testing.T, s *sketch.Sketch) {
	t.Helper()
	report := s.Verify(t.Context())
	require.Equal(t, sketch.FullyConstrained, report.Status)
	require.Zero(t, report.DOF)
	require.Empty(t, report.Redundant)
	require.Empty(t, report.Conflicts)
	require.Empty(t, report.Check())
}

// clippedHexArea returns the area of the disc of radius rho clipped to the hexagon.
func clippedHexArea(rho float64) float64 {
	apothem := m5nut.AcrossFlats / 2
	hexArea := math.Sqrt(3) / 2 * m5nut.AcrossFlats * m5nut.AcrossFlats
	switch {
	case rho <= apothem:
		return math.Pi * rho * rho
	case rho >= m5nut.AcrossCorners()/2:
		return hexArea
	}
	half := math.Acos(apothem / rho)
	return math.Pi*rho*rho - 6*rho*rho*(half-math.Sin(half)*math.Cos(half))
}

func expectedVolume() float64 {
	const steps = 20000
	hexArea := math.Sqrt(3) / 2 * m5nut.AcrossFlats * m5nut.AcrossFlats
	slope := math.Tan(m5nut.ChamferAngle * math.Pi / 180)
	bearingR := m5nut.BearingDiameter / 2
	depth := (m5nut.AcrossCorners()/2 - bearingR) * slope
	chamfer := 0.0
	for i := 0; i < steps; i++ {
		z := (float64(i) + .5) / steps * depth
		chamfer += (hexArea - clippedHexArea(bearingR+z/slope)) * depth / steps
	}
	d := m5nut.BoreDiameter
	return hexArea*m5nut.Thickness - math.Pi*d*d/4*m5nut.Thickness - 2*chamfer
}
