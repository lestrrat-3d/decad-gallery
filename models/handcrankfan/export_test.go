package handcrankfan

import "github.com/lestrrat-3d/r3"

// Beta exposes the blade angle for the external tests.
var Beta = beta

// ThicknessRatio is the table's thickness ratio at radius r, interpolated linearly between stations.
func ThicknessRatio(r float64) float64 {
	for i := 1; i < len(stations); i++ {
		a, b := stations[i-1], stations[i]
		if r <= b.r {
			return a.ratio + (b.ratio-a.ratio)*(r-a.r)/(b.r-a.r)
		}
	}
	return stations[len(stations)-1].ratio
}

// PosePoint applies motion's pose for p (fixed, crank, drive or fan) to a build-frame point.
func PosePoint(p Params, motion string, v r3.Vec) (r3.Vec, error) {
	state, err := Motion(p)
	if err != nil {
		return r3.Vec{}, err
	}
	xf, err := motionTransform(state, motion)
	if err != nil {
		return r3.Vec{}, err
	}
	return xf.Apply(v), nil
}

// PoseVector applies motion's pose for p to a build-frame direction.
func PoseVector(p Params, motion string, v r3.Vec) (r3.Vec, error) {
	state, err := Motion(p)
	if err != nil {
		return r3.Vec{}, err
	}
	xf, err := motionTransform(state, motion)
	if err != nil {
		return r3.Vec{}, err
	}
	return xf.ApplyDir(v), nil
}
