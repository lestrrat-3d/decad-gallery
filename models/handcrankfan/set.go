package handcrankfan

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad/parts"
	"github.com/lestrrat-3d/r3"
)

// hardware is the bought part each hardware group member is, which the bill of materials counts.
var hardwareSpec = map[string]string{
	"handle-bolt": "M2 x 16 screw, self-tapping",
	"crank-arm":   "M2 x 10 screw and M2 nut",
	"crank-pin":   "M2 x 10 screw and M2 nut",
	"fan-bolt":    "M2 x 16 screw and M2 nut",
	"case-bolt":   "M2 x 16 screw, self-tapping, gear case",
	"cover-bolt":  "M2 x 16 screw, self-tapping, gear case",
}

// Set returns the assembly as a parts.Set over doc, which must be the document Build used: every part with its
// material (the printed parts are PLA and the screws and nuts are bought) and the joints between them. The set feeds
// parts.Set.Explode, the bill of materials and the connection diagram.
func (a *Assembly) Set(doc *decad.Document) (*parts.Set, error) {
	set := parts.NewSet(doc)
	for _, part := range a.Parts {
		p := parts.Part{Name: part.Name, Kind: parts.Manufactured, Body: part.Body, Material: parts.Material{Name: "PLA"}}
		if part.Group == "hardware" {
			p.Kind, p.Material = parts.Purchased, parts.Material{Name: "steel"}
			for prefix, spec := range hardwareSpec {
				if len(part.Name) >= len(prefix) && part.Name[:len(prefix)] == prefix {
					p.Spec = spec
				}
			}
		}
		if err := set.Add(p); err != nil {
			return nil, fmt.Errorf("add %s to the parts set: %w", part.Name, err)
		}
	}
	for _, j := range a.joints() {
		if err := set.Join(j); err != nil {
			return nil, fmt.Errorf("join %s: %w", j.Name, err)
		}
	}
	return set, nil
}

// joints are the connections of the posed assembly, with each axis stated in the posed coordinates. A is the part
// that goes in and B the part it goes into.
func (a *Assembly) joints() []parts.Joint {
	apex := a.State.Apex
	crank := parts.Axis{Point: a.State.Crank, Dir: r3.Vec{X: 1}}
	drive := parts.Axis{Point: apex, Dir: r3.Vec{X: 1}}
	pin := parts.Axis{Point: a.State.Crank.Add(a.State.ArmDirection.Scale(crankThrow)), Dir: r3.Vec{X: 1}}
	fan := parts.Axis{Point: apex, Dir: fanAxis}
	vertical := func(dx float64) parts.Axis {
		return parts.Axis{Point: r3.Vec{X: gripX + dx, Y: -28, Z: axisHeight}, Dir: r3.Vec{Z: 1}}
	}
	var js []parts.Joint
	add := func(name string, a, b string, axis parts.Axis, motion parts.Motion) {
		js = append(js, parts.Joint{Name: name, Kind: parts.PinInBore, A: a, B: b, Axis: &axis, Motion: motion})
	}
	add("crank shaft in the plate", "crank-shaft", "gear-case", crank, parts.Turns)
	add("crank shaft in the cover", "crank-shaft", "gear-cover", crank, parts.Turns)
	add("crank shaft in the crank gear", "crank-shaft", "crank-gear", crank, parts.Fixed)
	add("drive shaft in the plate", "drive-shaft", "gear-case", drive, parts.Turns)
	add("drive shaft in the cover", "drive-shaft", "gear-cover", drive, parts.Turns)
	add("drive shaft in the drive gear", "drive-shaft", "drive-gear", drive, parts.Fixed)
	add("drive shaft in the driving gear", "drive-shaft", "driving-gear", drive, parts.Fixed)
	add("crank shaft in the arm", "crank-shaft", "crank-arm", crank, parts.Fixed)
	add("arm screw in the arm", "crank-arm-bolt", "crank-arm", crank, parts.Fixed)
	add("pin in the arm", "crank-pin", "crank-arm", pin, parts.Fixed)
	add("pin screw in the arm", "crank-pin-bolt", "crank-arm", pin, parts.Fixed)
	// The pin screw's head is on the arm's inner face, so it comes out toward the shaft.
	js[len(js)-1].Pull = r3.Vec{X: -1}
	add("handle on the pin", "crank-pin", "crank-grip", pin, parts.Turns)
	add("arm screw in the shaft", "crank-arm-bolt", "crank-shaft", crank, parts.Fixed)
	add("pin screw in the pin", "crank-pin-bolt", "crank-pin", pin, parts.Fixed)
	add("fan shaft in the pinion", "fan-shaft", "pinion", fan, parts.Fixed)
	add("fan shaft in the wall", "fan-shaft", "gearbox", fan, parts.Turns)
	add("hub on the fan shaft", "fan-shaft", "fan-hub", fan, parts.Fixed)
	for i, dx := range []float64{-gripScrewX, gripScrewX} {
		add(fmt.Sprintf("grip screw %d in the floor", i+1), fmt.Sprintf("handle-bolt-%d", i+1), "gearbox", vertical(dx), parts.Fixed)
		add(fmt.Sprintf("grip screw %d in the grip", i+1), fmt.Sprintf("handle-bolt-%d", i+1), "handle", vertical(dx), parts.Fixed)
	}
	for i, at := range caseScrews {
		bolt := parts.Axis{Point: r3.Vec{X: plateOut, Y: at.u, Z: at.v}, Dir: r3.Vec{X: 1}}
		name := fmt.Sprintf("case-bolt-%d", i+1)
		add(fmt.Sprintf("case screw %d in the plate", i+1), name, "gear-case", bolt, parts.Fixed)
		add(fmt.Sprintf("case screw %d in the gearbox", i+1), name, "gearbox", bolt, parts.Fixed)
	}
	for i, at := range coverScrews {
		bolt := parts.Axis{Point: r3.Vec{X: coverOut, Y: at.u, Z: at.v}, Dir: r3.Vec{X: 1}}
		name := fmt.Sprintf("cover-bolt-%d", i+1)
		add(fmt.Sprintf("cover screw %d in the cover", i+1), name, "gear-cover", bolt, parts.Fixed)
		add(fmt.Sprintf("cover screw %d in the case", i+1), name, "gear-case", bolt, parts.Fixed)
	}
	for k := 0; k < a.Params.Blades; k++ {
		theta := a.State.FanAngle + 2*math.Pi*float64(k)/float64(a.Params.Blades)
		radial := r3.Vec{X: math.Cos(theta), Z: math.Sin(theta)}
		stud := parts.Axis{Point: apex, Dir: radial}
		bolt := parts.Axis{Point: apex.Add(radial.Scale(boltAt)), Dir: fanAxis}
		blade, screw := fmt.Sprintf("fan-blade-%d", k+1), fmt.Sprintf("fan-bolt-%d", k+1)
		add(fmt.Sprintf("blade %d stud in the hub", k+1), blade, "fan-hub", stud, parts.Fixed)
		add(fmt.Sprintf("bolt %d in the hub", k+1), screw, "fan-hub", bolt, parts.Fixed)
		add(fmt.Sprintf("bolt %d in the blade", k+1), screw, blade, bolt, parts.Fixed)
	}
	return js
}

// Explode pulls the assembly apart with parts.Set.Explode, which reads each part's pull direction from its joint.
// set must be this assembly's Set. The gearbox stays put. Along +X the gear case's screws and the case come off
// first, then the two shafts and the two spur gears, then the cover's screws and the cover, then the arm, the handle
// (which slides onto the pin's stub end, so it goes between the arm and the pin), the pin and the arm screw. The order
// is the result of a search over the stage orders that leaves no two parts sharing space and keeps the arm, the handle
// and the pin in that order. The driving gear slides off to the left, the pinion lifts out of the open top, the grip
// screws lift out and the grip drops, and the fan side comes off along the fan axis.
func (a *Assembly) Explode(ctx context.Context, set *parts.Set) (*parts.Exploded, error) {
	along := r3.Vec{X: 1}
	stage := func(name string, direction r3.Vec, names ...string) parts.Stage {
		return parts.Stage{Name: name, Parts: names, Clear: parts.ClearAll, Direction: direction}
	}
	var coverBolts, caseBolts []string
	for i := range coverScrews {
		coverBolts = append(coverBolts, fmt.Sprintf("cover-bolt-%d", i+1))
	}
	for i := range caseScrews {
		caseBolts = append(caseBolts, fmt.Sprintf("case-bolt-%d", i+1))
	}
	return set.Explode(ctx, parts.WithExplodeGround("gearbox"), parts.WithExplodeStages(
		stage("case screws", along, caseBolts...),
		stage("case", along, "gear-case"),
		stage("drive shaft", along, "drive-shaft"),
		stage("crank shaft", along, "crank-shaft"),
		stage("drive gear", along, "drive-gear"),
		stage("crank gear", along, "crank-gear"),
		stage("cover screws", along, coverBolts...),
		stage("cover", along, "gear-cover"),
		parts.Stage{Name: "arm", Parts: []string{"crank-arm"}, Direction: along},
		stage("handle", along, "crank-grip"),
		stage("pin", along, "crank-pin"),
		stage("arm screw", along, "crank-arm-bolt"),
		stage("pin screw", r3.Vec{X: -1}, "crank-pin-bolt"),
		stage("driving gear", r3.Vec{X: -1}, "driving-gear"),
		stage("pinion", r3.Vec{Z: 1}, "pinion"),
		stage("grip screws", r3.Vec{Z: 1}, "handle-bolt-1", "handle-bolt-2"),
		stage("grip", r3.Vec{Z: -1}, "handle"),
	))
}
