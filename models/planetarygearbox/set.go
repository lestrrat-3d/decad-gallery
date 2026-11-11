package planetarygearbox

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad/parts"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// Spec labels of the bought parts, which the bill of materials groups on.
const (
	bearingSpec = "605ZZ ball bearing"
	nutSpec     = "M2 hex nut"
)

// explodeGap is how far past touching each part of the exploded view sits from the part it clears.
const explodeGap = 6.0

// Set returns the gearbox as a parts.Set over doc, which must be the document Build used with p: every part with its
// material (the printed parts are PLA, the bearings and fasteners are bought) and the joints between them. A in a
// joint is the part that goes in and B the part it goes into. The set feeds Gearbox.Explode.
func (g Gearbox) Set(doc *decad.Document, p Params) (*parts.Set, error) {
	frame, err := frameTransform(p)
	if err != nil {
		return nil, fmt.Errorf("planetarygearbox: %w", err)
	}
	set := parts.NewSet(doc)
	for _, nb := range g.Named() {
		part := parts.Part{Name: nb.Name, Kind: parts.Manufactured, Body: nb.Body, Material: parts.Material{Name: "PLA"}}
		switch {
		case isBearing(g, nb.Body):
			part.Kind, part.Material, part.Spec = parts.Purchased, parts.Material{Name: "steel"}, bearingSpec
		case isFastener(g, nb.Body):
			part.Kind, part.Material = parts.Purchased, parts.Material{Name: "steel"}
			part.Spec = fmt.Sprintf("M2 x %g bolt", fastenerLength)
			if isNut(g, nb.Body) {
				part.Spec = nutSpec
			}
		}
		if err := set.Add(part); err != nil {
			return nil, fmt.Errorf("planetarygearbox: add %s to the parts set: %w", nb.Name, err)
		}
	}
	for _, j := range g.joints(p, frame) {
		if err := set.Join(j); err != nil {
			return nil, fmt.Errorf("planetarygearbox: join %s: %w", j.Name, err)
		}
	}
	return set, nil
}

func isBearing(g Gearbox, b *decad.Body) bool {
	if b == g.FrontBearing || b == g.RearBearing {
		return true
	}
	for _, s := range g.Stages {
		for _, bearing := range s.Bearings {
			if b == bearing {
				return true
			}
		}
	}
	return false
}

func isNut(g Gearbox, b *decad.Body) bool {
	for _, nut := range g.Nuts {
		if b == nut {
			return true
		}
	}
	return false
}

func isFastener(g Gearbox, b *decad.Body) bool {
	if isNut(g, b) {
		return true
	}
	for _, bolt := range g.Bolts {
		if b == bolt {
			return true
		}
	}
	return false
}

// joints lists the connections of the assembled gearbox with each axis stated in world coordinates. The order of the
// list matters: Decad assigns each part the first joint that reaches it from the ground, the input shaft, as its parent
// joint. Each shaft is joined to the cover and bearing it passes through before they are joined to anything else, so
// the cover and the bearing take the shaft as their parent and clear its end.
func (g Gearbox) joints(p Params, frame r3.Transform) []parts.Joint {
	d := derive(p)
	axisAt := func(x, y float64) parts.Axis {
		return parts.Axis{Point: frame.Apply(r3.Vec{X: x, Y: y}), Dir: frame.ApplyDir(r3.Vec{Z: 1})}
	}
	centre := axisAt(0, 0)
	var js []parts.Joint
	add := func(name string, kind parts.JointKind, a, b string, axis *parts.Axis) {
		js = append(js, parts.Joint{Name: name, Kind: kind, A: a, B: b, Axis: axis})
	}
	pin := func(name, a, b string, axis parts.Axis) { add(name, parts.PinInBore, a, b, &axis) }

	// The input shaft passes through the front bearing and the front cover and fills the first sun's bore. Its collar
	// is wider than the bearing's bore and rests on the bearing's inner side, so the bearing and the cover leave over
	// the shaft's outer end and the shaft goes in from the sun's side.
	pin("input shaft in the front bearing", InputShaftName, FrontBearingName, centre)
	pin("input shaft in the front cover", InputShaftName, FrontCoverName, centre)
	pin("input shaft in the first sun", InputShaftName, SunName(0), centre)
	pin("front bearing in the front cover", FrontBearingName, FrontCoverName, centre)
	add("front cover on the ring", parts.Contact, FrontCoverName, RingName, nil)

	last := len(g.Stages) - 1
	for k, s := range g.Stages {
		shaft := ShaftName(k, len(g.Stages))
		pin(shaft+" in "+CarrierName(k), shaft, CarrierName(k), centre)
		if k < last {
			pin(shaft+" in "+SunName(k+1), shaft, SunName(k+1), centre)
		}
		for i := range s.Planets {
			phi := PlanetOrbitAngle(p, i)
			at := axisAt(d.orbit*math.Cos(phi), d.orbit*math.Sin(phi))
			planet, bearing, carrierPin := PlanetName(k, i), BearingName(k, i), PinName(k, i)
			add(fmt.Sprintf("sun %d meshes planet %d.%d", k+1, k+1, i+1), parts.Contact, planet, SunName(k), nil)
			add(fmt.Sprintf("planet %d.%d meshes the ring", k+1, i+1), parts.Contact, planet, RingName, nil)
			pin(fmt.Sprintf("bearing %d.%d in its planet", k+1, i+1), bearing, planet, at)
			pin(fmt.Sprintf("pin %d.%d in its bearing", k+1, i+1), carrierPin, bearing, at)
			pin(fmt.Sprintf("pin %d.%d in the carrier", k+1, i+1), carrierPin, CarrierName(k), at)
		}
	}

	// The output shaft passes through the rear cover and the rear bearing, and its collar rests on the bearing's inner
	// side like the input shaft's. The ring slides over the shaft's outer end after the rear cover is off, so the ring
	// keeps clear of the shaft.
	output := ShaftName(last, len(g.Stages))
	pin("output shaft in the rear bearing", output, RearBearingName, centre)
	pin("output shaft in the rear cover", output, RearCoverName, centre)
	pin("rear bearing in the rear cover", RearBearingName, RearCoverName, centre)
	add("rear cover on the ring", parts.Contact, RearCoverName, RingName, nil)
	js = append(js, parts.Joint{Name: "the ring clears the output shaft", Kind: parts.Keepout, A: output, B: RingName,
		MinGap: units.Millimeters(1)})

	// A bolt passes through its cover's hole and screws into a nut that the ring holds in a closed cavity.
	half := len(g.Bolts) / 2
	for i := range g.Bolts {
		cover, name, nut := FrontCoverName, FrontBoltName(i), FrontNutName(i)
		if i >= half {
			cover, name, nut = RearCoverName, RearBoltName(i-half), RearNutName(i-half)
		}
		angle := boltAngles()[i%half]
		at := axisAt(d.boltCircle*math.Cos(angle), d.boltCircle*math.Sin(angle))
		pin(name+" in the cover", name, cover, at)
		add(nut+" in the ring", parts.Contact, nut, RingName, nil)
		pin(name+" in "+nut, name, nut, at)
	}
	return js
}

// Explode pulls the gearbox apart along its axis with parts.Set.Explode. set must be this gearbox's Set over the
// document Build used with p. The input shaft stays put and every other part moves along the axis in a stage of its
// own, each stage clearing the part it is joined to. Decad has no rule for the order in which parts can be inserted, so
// the stages state it. From the input shaft's outer end the front cover, then the front bearing, then the front bolts
// slide off it toward the front. Toward the rear the stack rises in the order the parts come off the carrier: the first
// sun and its planets leave the shaft's inner end, then the planets' bearings slide off the carrier's pins, then the
// pins leave the carrier, then the carrier, then the link shaft that was pressed into it, and the next stage follows
// the same way. The output shaft leaves its carrier last, and the ring, the rear cover, its bearing and its bolts
// slide off the shaft's outer end in that order. A shaft's collar is wider than the bore of the bearing it passes
// through, so each shaft stays on the carrier side of that bearing and its cover, and the bearing leaves first.
func (g Gearbox) Explode(ctx context.Context, set *parts.Set, p Params) (*parts.Exploded, error) {
	frame, err := frameTransform(p)
	if err != nil {
		return nil, fmt.Errorf("planetarygearbox: %w", err)
	}
	rear, front := frame.ApplyDir(r3.Vec{Z: 1}), frame.ApplyDir(r3.Vec{Z: -1})
	stage := func(name string, direction r3.Vec, names ...string) parts.Stage {
		return parts.Stage{Name: name, Parts: names, Direction: direction}
	}
	// A rigid group moves by one offset.
	together := func(name string, direction r3.Vec, names ...string) parts.Stage {
		st := stage(name, direction, names...)
		st.Together = true
		return st
	}
	names := func(n int, name func(int) string) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = name(i)
		}
		return out
	}
	var frontBolts, rearBolts, nuts []string
	for i := range g.Bolts {
		if i < len(g.Bolts)/2 {
			frontBolts = append(frontBolts, FrontBoltName(i))
			nuts = append(nuts, FrontNutName(i))
			continue
		}
		rearBolts = append(rearBolts, RearBoltName(i-len(g.Bolts)/2))
		nuts = append(nuts, RearNutName(i-len(g.Bolts)/2))
	}
	stages := []parts.Stage{
		stage("front cover", front, FrontCoverName),
		stage("front bearing", front, FrontBearingName),
		stage("front bolts", front, frontBolts...),
	}
	for k, s := range g.Stages {
		n := len(s.Planets)
		layer := append([]string{SunName(k)}, names(n, func(i int) string { return PlanetName(k, i) })...)
		stages = append(stages,
			together(fmt.Sprintf("sun and planets %d", k+1), rear, layer...),
			stage(fmt.Sprintf("planet bearings %d", k+1), rear, names(n, func(i int) string { return BearingName(k, i) })...),
			stage(fmt.Sprintf("carrier pins %d", k+1), rear, names(n, func(i int) string { return PinName(k, i) })...),
			stage(fmt.Sprintf("carrier %d", k+1), rear, CarrierName(k)),
			stage(fmt.Sprintf("shaft %d", k+1), rear, ShaftName(k, len(g.Stages))))
	}
	stages = append(stages,
		together("ring", rear, append([]string{RingName}, nuts...)...),
		stage("rear cover", rear, RearCoverName),
		stage("rear bearing", rear, RearBearingName),
		stage("rear bolts", rear, rearBolts...))
	return set.Explode(ctx, parts.WithExplodeGround(InputShaftName), parts.WithExplodeGap(units.Millimeters(explodeGap)),
		parts.WithExplodeStages(stages...))
}
