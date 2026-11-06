package planetarygearbox

import "math"

// PlanetOrbitAngle returns the orbit angle of planet i about the gearbox axis.
func PlanetOrbitAngle(p Params, i int) float64 {
	return 2 * math.Pi * float64(i) / float64(p.PlanetCount)
}

// PlanetToothAngle returns the turn of planet i's tooth zero about its own axis.
// It puts the planet's teeth into the sun's gaps and the ring's gaps on the line
// between their centers. It is consistent for every planet because the sun and
// ring tooth counts sum to a multiple of the planet count.
func PlanetToothAngle(p Params, i int) float64 {
	zs, zp := float64(p.SunTeeth), float64(p.PlanetTeeth)
	return ((zs+zp)*PlanetOrbitAngle(p, i) + math.Pi*(zp-1)) / zp
}

// RingToothAngle returns the turn of the ring's tooth-space tool about the axis,
// reduced to one tooth pitch.
func RingToothAngle(p Params) float64 {
	pitch := 2 * math.Pi / float64(p.RingTeeth)
	angle := math.Mod(math.Pi*float64(p.PlanetTeeth-1)/float64(p.RingTeeth), pitch)
	if angle < 0 {
		angle += pitch
	}
	if angle < 1e-12 || pitch-angle < 1e-12 {
		return 0
	}
	return angle
}
