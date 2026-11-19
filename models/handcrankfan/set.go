package handcrankfan

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/decad/parts"
	"github.com/lestrrat-3d/r3"
)

// Set builds a parts set for this assembly and the document used by Build.
func (a *Assembly) Set(doc *decad.Document) (*parts.Set, error) {
	set := parts.NewSet(doc)
	for _, part := range a.Parts {
		item := parts.Part{
			Name:     part.Name,
			Body:     part.Body,
			Kind:     parts.Manufactured,
			Material: parts.Material{Name: "PLA"},
		}
		switch part.Group {
		case "bearing":
			item.Kind = parts.Purchased
			item.Material = parts.Material{Name: "steel"}
			item.Spec = "608-2RS ball bearing"
		case "hardware":
			item.Kind = parts.Purchased
			item.Material = parts.Material{Name: "steel"}
			item.Spec = fastenerSpec(part.Name)
		}
		if err := set.Add(item); err != nil {
			return nil, fmt.Errorf("add %s to the parts set: %w", part.Name, err)
		}
	}
	for _, joint := range a.joints() {
		if err := set.Join(joint); err != nil {
			return nil, fmt.Errorf("join %s: %w", joint.Name, err)
		}
	}
	return set, nil
}

func (a *Assembly) joints() []parts.Joint {
	var joints []parts.Joint
	add := func(name, first, second string, axis parts.Axis, motion parts.Motion) {
		axisCopy := axis
		joints = append(joints, parts.Joint{
			Name: name, Kind: parts.PinInBore, A: first, B: second, Axis: &axisCopy, Motion: motion,
		})
	}
	addPull := func(name, first, second string, axis parts.Axis, motion parts.Motion, pull r3.Vec) {
		add(name, first, second, axis, motion)
		joints[len(joints)-1].Pull = pull
	}
	nutJoints := func(bolt, nut, host string, axis parts.Axis) {
		add(nut+" in "+host, nut, host, axis, parts.Fixed)
		add(bolt+" in "+nut, bolt, nut, axis, parts.Fixed)
	}
	crank := parts.Axis{Point: a.State.Crank, Dir: r3.Vec{X: 1}}
	drive := parts.Axis{Point: a.State.Apex, Dir: r3.Vec{X: 1}}
	pin := parts.Axis{
		Point: a.State.Crank.Add(a.State.ArmDirection.Scale(crankThrow)),
		Dir:   r3.Vec{X: 1},
	}
	fan := parts.Axis{Point: a.State.Apex, Dir: fanAxis}

	for _, site := range bearingSites {
		var shaft string
		var axis parts.Axis
		switch site.axis {
		case crankSpline:
			shaft, axis = "crank-shaft", crank
		case driveSpline:
			shaft, axis = "drive-shaft", drive
		case fanSpline:
			shaft, axis = "fan-shaft", fan
		default:
			continue
		}
		add(shaft+" in "+site.name, shaft, site.name, axis, parts.Turns)
		add(site.name+" in the gearbox", site.name, site.holder, axis, parts.Fixed)
	}

	add("crank shaft in the crank gear", "crank-shaft", "crank-gear", crank, parts.Fixed)
	add("drive shaft in the drive gear", "drive-shaft", "drive-gear", drive, parts.Fixed)
	add("drive shaft in the driving gear", "drive-shaft", "driving-gear", drive, parts.Fixed)
	add("crank shaft in the arm", "crank-shaft", "crank-arm", crank, parts.Fixed)
	add("pin in the arm", "crank-pin", "crank-arm", pin, parts.Fixed)
	addPull("pin bolt in the arm", "crank-pin-bolt", "crank-arm", pin, parts.Fixed, r3.Vec{X: -1})
	add("handle on the pin", "crank-pin", "crank-grip", pin, parts.Turns)
	nutJoints("crank-pin-bolt", "crank-pin-nut", "crank-pin", pin)
	add("fan shaft in the pinion", "fan-shaft", "pinion", fan, parts.Fixed)
	add("hub on the fan shaft", "fan-shaft", "fan-hub", fan, parts.Fixed)

	for i, dx := range []float64{-gripScrewX, gripScrewX} {
		index := i + 1
		axis := parts.Axis{Point: r3.Vec{X: gripX + dx, Y: gripY, Z: axisHeight}, Dir: r3.Vec{Z: 1}}
		bolt, nut := fmt.Sprintf("handle-bolt-%d", index), fmt.Sprintf("handle-nut-%d", index)
		add(fmt.Sprintf("grip bolt %d in the floor", index), bolt, "gearbox-base", axis, parts.Fixed)
		add(fmt.Sprintf("grip bolt %d in the grip", index), bolt, "handle", axis, parts.Fixed)
		nutJoints(bolt, nut, "handle", axis)
	}

	for i, ear := range gearboxEars {
		index := i + 1
		axis := parts.Axis{Point: ear.centre(), Dir: ear.dir}
		bolt, nut := fmt.Sprintf("gearbox-bolt-%d", index), fmt.Sprintf("gearbox-nut-%d", index)
		addPull(fmt.Sprintf("ear bolt %d in %s", index, ear.head), bolt, ear.head, axis, parts.Fixed, ear.dir.Scale(-1))
		add(fmt.Sprintf("ear bolt %d in %s", index, ear.nut), bolt, ear.nut, axis, parts.Fixed)
		nutJoints(bolt, nut, ear.nut, axis)
	}

	for i, wall := range wallScrews {
		index := i + 1
		axis := parts.Axis{Point: wall.at(wall.seam), Dir: wall.dir}
		bolt := fmt.Sprintf("gearbox-screw-%d", index)
		addPull(fmt.Sprintf("wall bolt %d in %s", index, wall.head), bolt, wall.head, axis, parts.Fixed, wall.dir.Scale(-1))
		add(fmt.Sprintf("wall bolt %d in %s", index, wall.recv), bolt, wall.recv, axis, parts.Fixed)
	}

	if xf, err := motionTransform(a.State, "fan"); err == nil {
		socketCenter := a.State.Apex.Add(fanAxis.Scale(hubBack + fanSize.BladeSocketZ()))
		for k := range turbinefan.BladeCount {
			index := k + 1
			phi := 2 * math.Pi * float64(k) / turbinefan.BladeCount
			radial := xf.ApplyDir(r3.Vec{X: math.Cos(phi), Z: math.Sin(phi)})
			studAxis := parts.Axis{Point: socketCenter, Dir: radial}
			boltAxis := parts.Axis{Point: a.State.Apex.Add(radial.Scale(boltAt)), Dir: fanAxis}
			blade, bolt, nut := fmt.Sprintf("fan-blade-%d", index), fmt.Sprintf("fan-bolt-%d", index), fmt.Sprintf("fan-nut-%d", index)
			add(fmt.Sprintf("blade %d stud in the hub", index), blade, "fan-hub", studAxis, parts.Fixed)
			add(fmt.Sprintf("bolt %d in the hub", index), bolt, "fan-hub", boltAxis, parts.Fixed)
			add(fmt.Sprintf("bolt %d in the blade", index), bolt, blade, boltAxis, parts.Fixed)
			add(fmt.Sprintf("nut %d in the hub", index), nut, "fan-hub", boltAxis, parts.Fixed)
			add(fmt.Sprintf("bolt %d in the nut", index), bolt, nut, boltAxis, parts.Fixed)
		}
		for _, site := range sitesNamed("fan-set-screw") {
			axis := parts.Axis{Point: xf.Apply(site.Head), Dir: xf.ApplyDir(site.Dir)}
			add("set screw in the hub", site.bolt, "fan-hub", axis, parts.Fixed)
			nutJoints(site.bolt, site.nut, "fan-hub", axis)
		}
	}
	return joints
}

// Explode returns a staged exploded view with the gearbox base grounded.
func (a *Assembly) Explode(ctx context.Context, set *parts.Set) (*parts.Exploded, error) {
	stage := func(name string, dir r3.Vec, names ...string) parts.Stage {
		return parts.Stage{Name: name, Direction: dir, Parts: names, Clear: parts.ClearAll}
	}
	together := func(name string, dir r3.Vec, names ...string) parts.Stage {
		result := stage(name, dir, names...)
		result.Together = true
		return result
	}
	stages := []parts.Stage{
		stage("ear bolts", r3.Vec{Y: -1}, "gearbox-bolt-1", "gearbox-bolt-2", "gearbox-bolt-3", "gearbox-bolt-4"),
		stage("post bolts", r3.Vec{X: -1}, "gearbox-screw-1", "gearbox-screw-2"),
		stage("crank-side wall bolts", r3.Vec{X: 1}, "gearbox-screw-3", "gearbox-screw-4"),
		{Name: "arm", Direction: r3.Vec{X: 1}, Parts: []string{"crank-arm"}},
		stage("handle", r3.Vec{X: 1}, "crank-grip"),
		together("pin", r3.Vec{X: 1}, "crank-pin", "crank-pin-nut"),
		stage("pin bolt", r3.Vec{X: -1}, "crank-pin-bolt"),
		stage("drive shaft", r3.Vec{X: -1}, "drive-shaft"),
		stage("crank shaft", r3.Vec{X: -1}, "crank-shaft"),
		stage("pinion", r3.Vec{X: -1}, "pinion"),
		stage("grip bolts", r3.Vec{Z: 1}, "handle-bolt-1", "handle-bolt-2"),
		stage("crank gear", r3.Vec{Z: 1}, "crank-gear"),
		stage("drive gear", r3.Vec{Z: 1}, "drive-gear"),
		stage("driving gear", r3.Vec{Z: 1}, "driving-gear"),
		stage("crank block bearing", r3.Vec{X: 1}, "bearing-crank-block"),
		stage("wall bearings", r3.Vec{X: -1}, "bearing-crank-wall", "bearing-drive-wall"),
		stage("post bearings", r3.Vec{X: -1}, "bearing-crank-post", "bearing-drive-post"),
		stage("fan wall bearing", r3.Vec{Y: 1}, "bearing-fan-wall"),
		stage("fan block bearing", r3.Vec{Y: -1}, "bearing-fan-block"),
	}
	bladeBolts := make([]string, turbinefan.BladeCount)
	blades := make([]string, turbinefan.BladeCount)
	hubParts := []string{"fan-hub"}
	for k := range turbinefan.BladeCount {
		index := fmt.Sprintf("%d", k+1)
		bladeBolts[k], blades[k] = "fan-bolt-"+index, "fan-blade-"+index
		hubParts = append(hubParts, "fan-nut-"+index)
	}
	hubParts = append(hubParts, "fan-set-screw", "fan-set-screw-nut")
	stages = append(stages,
		stage("blade bolts", r3.Vec{Y: -1}, bladeBolts...),
		stage("blades", r3.Vec{}, blades...),
		together("hub", r3.Vec{Y: -1}, hubParts...),
		stage("fan shaft", r3.Vec{Y: -1}, "fan-shaft"),
		together("post", r3.Vec{X: -1}, "gearbox-post", "gearbox-nut-1", "gearbox-nut-3"),
		together("crank-side wall", r3.Vec{X: 1}, "gearbox-crank-wall", "gearbox-nut-2", "gearbox-nut-4"),
		stage("fan-side wall", r3.Vec{X: -1}, "gearbox-fan-wall"),
		together("grip", r3.Vec{Z: -1}, "handle", "handle-nut-1", "handle-nut-2"),
	)
	return set.Explode(ctx, parts.WithExplodeGround("gearbox-base"), parts.WithExplodeStages(stages...))
}
