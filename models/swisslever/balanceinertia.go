package swisslever

import "math"

// balanceInertia returns the balance wheel's mass in g and the balance group's inertia about the staff in g·mm², for
// the rim, arms, hub, boss, roller table and safety roller, all in PLA. It treats the arms as plain bars from the hub
// to the rim and leaves out the window rounds, the crescent and the impulse pin, which move it by under 0.5 %.
func balanceInertia() (float64, float64) {
	var m, i, wheel float64
	disc := func(r, h float64) { // solid cylinder
		mm := PLADensity * math.Pi * r * r * h
		m += mm
		i += mm * r * r / 2
	}
	ring := func(r0, r1, h float64) {
		mm := PLADensity * math.Pi * (r1*r1 - r0*r0) * h
		m += mm
		i += mm * (r0*r0 + r1*r1) / 2
	}
	ring(36, 42, 5)
	// Four arms, 4 wide, from the hub to the rim.
	armLen := 36.0 - 7.0
	am := PLADensity * 4 * armLen * 5
	for k := 0; k < 4; k++ {
		m += am
		i += am * (4*4/12.0 + armLen*armLen/12.0 + (7+armLen/2)*(7+armLen/2))
	}
	disc(7, 5)
	disc(7, 1)
	wheel = m
	disc(11, 2.5)
	disc(SafetyRadius, 7.5)
	return wheel, i
}
