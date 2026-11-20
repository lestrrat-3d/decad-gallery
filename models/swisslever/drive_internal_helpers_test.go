package swisslever

import (
	"math"
)

// socketArea returns the area of the end-family D socket.
func socketArea() float64 {
	return dArea(EndBoreRadius, EndBoreFlat)
}

// dArea returns the area of the D outline of radius r and flat x = f.
func dArea(r, f float64) float64 {
	return math.Pi*r*r - (r*r*math.Acos(f/r) - f*math.Sqrt(r*r-f*f))
}
