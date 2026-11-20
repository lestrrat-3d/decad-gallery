package escapement

import (
	"math"

	"github.com/lestrrat-3d/r3"
)

func anchorCases() map[string]Params {
	p := DefaultParams()
	wide, narrow := p, p
	wide.LockAngle, wide.LiftAngle, wide.PendulumLength = 1.25*deg, 3*deg, 120
	narrow.LiftAngle, narrow.PendulumLength = 2*deg, 250
	return map[string]Params{"default": p, "wide": wide, "narrow": narrow}
}

// outerLoop2 returns the outline loop with the most extent, relative to the point origin.
func outerLoop2(loops [][]r3.Vec, origin Pt) []Pt {
	var best []Pt
	far := -1.0
	for _, l := range loops {
		pts := loopPoints(l)
		m := 0.0
		for i := range pts {
			pts[i] = pts[i].Sub(origin)
			m = math.Max(m, pts[i].Len())
		}
		if m > far {
			best, far = pts, m
		}
	}
	return best
}
