package screwgear_test

import (
	"github.com/lestrrat-3d/r3"
)

// onPlane is how far p stands from the plane, along the plane's normal.
func onPlane(plane r3.Frame, p r3.Vec) float64 { return p.Sub(plane.Origin()).Dot(plane.N()) }
