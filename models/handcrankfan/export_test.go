package handcrankfan

import (
	"fmt"

	"github.com/lestrrat-3d/r3"
)

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

// Fastener is one bolt and its nut in posed coordinates.
type Fastener struct {
	Bolt, Nut, HeadHost, NutHost string
	Head, Dir, Corner            r3.Vec  // posed head top centre, unit shank direction, unit corner direction
	Length                       float64 // under the head
	CavityNear, CavityFar        float64 // from Head along Dir
}

// Fasteners returns every fastener of a build with p, posed as Build poses it (blade sites patterned).
func Fasteners(p Params) ([]Fastener, error) {
	state, err := Motion(p)
	if err != nil {
		return nil, err
	}
	blades, err := bladeSites(p.Blades)
	if err != nil {
		return nil, err
	}
	var out []Fastener
	for _, s := range append(blades, fastenerSites()[1:]...) {
		xf, err := motionTransform(state, s.motion)
		if err != nil {
			return nil, fmt.Errorf("pose %s: %w", s.bolt, err)
		}
		out = append(out, Fastener{
			Bolt: s.bolt, Nut: s.nut, HeadHost: s.headHost, NutHost: s.nutHost,
			Head: xf.Apply(s.Head), Dir: xf.ApplyDir(s.Dir), Corner: xf.ApplyDir(s.Corner),
			Length: s.Length, CavityNear: s.Cavity, CavityFar: s.CavityFar(),
		})
	}
	return out, nil
}
