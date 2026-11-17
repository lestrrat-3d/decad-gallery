package handcrankfan

import (
	"fmt"

	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
)

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
	// Rotor marks a site whose nut sits in a slot open to a face; SetScrew marks the fan's set screw.
	Rotor, SetScrew       bool
	Head, Dir, Corner     r3.Vec  // posed head top centre, unit shank direction, unit corner direction
	Length                float64 // under the head
	CavityNear, CavityFar float64 // from Head along Dir
}

// FanParams is the turbinefan parameters the rotor is built from.
func FanParams(p Params) turbinefan.Params { return fanParams(p) }

// Fasteners returns every fastener of a build with p, posed as Build poses it.
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
			Rotor: s.headHost == "fan-hub", SetScrew: s.bolt == "fan-set-screw",
			Head: xf.Apply(s.Head), Dir: xf.ApplyDir(s.Dir), Corner: xf.ApplyDir(s.Corner),
			Length: s.Length, CavityNear: s.Cavity, CavityFar: s.CavityFar(),
		})
	}
	return out, nil
}

// FanSize is the rotor's turbinefan sizes.
func FanSize() turbinefan.Size { return fanSize }
