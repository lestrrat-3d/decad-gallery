package fastener_test

import (
	"math"

	"github.com/lestrrat-3d/decad-gallery/models/fastener"
)

// nutVolume is the hexagonal prism, less the bore and the two chamfers. Each chamfer removes, at depth z

// from its face, the part of the hexagon outside the circle of radius dw/2 + z/tan(30 degrees).

func nutVolume(n fastener.NutSize) float64 {
	hexArea := math.Sqrt(3) / 2 * n.AcrossFlats * n.AcrossFlats
	bore := math.Pi * n.BoreDiameter * n.BoreDiameter / 4 * n.Thickness
	slope := math.Tan(fastener.HexChamferAngle * math.Pi / 180)
	bearingR := n.BearingDiameter / 2
	depth := (n.AcrossCorners()/2 - bearingR) * slope
	const steps = 20000
	chamfer := 0.0
	for i := range steps {
		z := (float64(i) + .5) / steps * depth
		chamfer += (hexArea - clippedHexArea(n, bearingR+z/slope)) * depth / steps
	}
	return hexArea*n.Thickness - bore - 2*chamfer
}

// clippedHexArea is the area of the nut's hexagon inside the circle of radius rho about its center.

func clippedHexArea(n fastener.NutSize, rho float64) float64 {
	apothem := n.AcrossFlats / 2
	if rho <= apothem {
		return math.Pi * rho * rho
	}
	if rho >= n.AcrossCorners()/2 {
		return math.Sqrt(3) / 2 * n.AcrossFlats * n.AcrossFlats
	}
	// Each of the six flats cuts a circular segment off the circle.
	half := math.Acos(apothem / rho)
	segment := rho * rho * (half - math.Sin(half)*math.Cos(half))
	return math.Pi*rho*rho - 6*segment
}
