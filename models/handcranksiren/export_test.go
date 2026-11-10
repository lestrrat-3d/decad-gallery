package handcranksiren

import "github.com/lestrrat-3d/r3"

// PosePoint applies motion's pose for p (fixed, crank, intermediate or rotor) to a build-frame point.
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
