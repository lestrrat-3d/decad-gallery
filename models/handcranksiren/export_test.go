package handcranksiren

import (
	"fmt"

	"github.com/lestrrat-3d/r3"
)

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

// BearingSite is one bearing: its part name, its axis point and direction (+Y), the Y of its start face and the holder.
type BearingSite struct {
	Name, Holder string
	Axis, Dir    r3.Vec
	Start        float64
}

// BearingSites returns the six bearings of the default pose.
func BearingSites() []BearingSite {
	state, err := Motion(DefaultParams())
	if err != nil {
		panic(err)
	}
	var out []BearingSite
	for _, s := range bearingSites {
		out = append(out, BearingSite{Name: s.name, Holder: s.holder, Axis: s.axis(state), Dir: r3.Vec{Y: 1}, Start: s.start})
	}
	return out
}

// Fastener is one bolt and its nut in posed coordinates.
type Fastener struct {
	Bolt, Nut, HeadHost, NutHost string
	Head, Dir, Corner            r3.Vec  // posed head top centre, unit shank direction, unit corner direction
	Length                       float64 // under the head
	CavityNear, CavityFar        float64 // from Head along Dir
}

// Fasteners returns every fastener site of a build with p, in table order, posed as Build poses it.
func Fasteners(p Params) ([]Fastener, error) {
	state, err := Motion(p)
	if err != nil {
		return nil, err
	}
	var out []Fastener
	for _, s := range fastenerSites() {
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
