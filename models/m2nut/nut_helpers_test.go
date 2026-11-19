package m2nut_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

// expectedVolume is the hexagonal prism, less the bore and the two chamfers. Each chamfer removes, at depth z
// from its face, the part of the hexagon outside the circle of radius dw/2 + z/tan(30 degrees).
func expectedVolume() float64 {
	hexArea := math.Sqrt(3) / 2 * m2nut.AcrossFlats * m2nut.AcrossFlats
	bore := math.Pi * m2nut.BoreDiameter * m2nut.BoreDiameter / 4 * m2nut.Thickness
	slope := math.Tan(m2nut.ChamferAngle * math.Pi / 180)
	bearingR := m2nut.BearingDiameter / 2
	depth := (m2nut.AcrossCorners()/2 - bearingR) * slope
	const steps = 20000
	chamfer := 0.0
	for i := range steps {
		z := (float64(i) + .5) / steps * depth
		chamfer += (hexArea - clippedHexArea(bearingR+z/slope)) * depth / steps
	}
	return hexArea*m2nut.Thickness - bore - 2*chamfer
}

// clippedHexArea is the area of the nut's hexagon inside the circle of radius rho about its center.
func clippedHexArea(rho float64) float64 {
	apothem := m2nut.AcrossFlats / 2
	if rho <= apothem {
		return math.Pi * rho * rho
	}
	if rho >= m2nut.AcrossCorners()/2 {
		return math.Sqrt(3) / 2 * m2nut.AcrossFlats * m2nut.AcrossFlats
	}
	// Each of the six flats cuts a circular segment off the circle.
	half := math.Acos(apothem / rho)
	segment := rho * rho * (half - math.Sin(half)*math.Cos(half))
	return math.Pi*rho*rho - 6*segment
}

func requireFullyConstrained(t *testing.T, s *sketch.Sketch) {
	t.Helper()
	report := s.Verify(t.Context())
	require.True(t, report.Status == sketch.FullyConstrained, "status %s", report.Status)
	require.Zero(t, report.DOF)
	require.Zero(t, len(report.Redundant))
	require.Zero(t, len(report.Conflicts))
	require.NoError(t, report.Check())
}
