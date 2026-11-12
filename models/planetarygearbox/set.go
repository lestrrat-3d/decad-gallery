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
// joint. The input shaft is joined to the front bearing and the front cover before they are joined to anything else,
// so both take the input shaft as their parent and clear its outer end. The rear end is reached from the ring, which
// the front cover reaches first: the rear cover takes the ring as its parent, and the output shaft and the rear bearing
// take the rear cover.
//
// Each shaft turns in its cover's bearing and passes the cover's lip bore, and each planet turns on its carrier pin
// through its bearing and its lip bore; every other joint holds its parts fixed. A pin-in-bore joint declares the
// radial clearance of its fit: zero for a shaft or pin in a bearing bore and for a bolt in its nut, Params.
// BearingClearance for a bearing in its seat, pressFitClearance for a shaft or pin pressed into a sun or carrier, and
// headClearance for a bolt in its hole. A lip bore is far wider than the shaft or pin in it and is not a fit, so those
// joints declare no clearance.
func (g Gearbox) joints(p Params, frame r3.Transform) []parts.Joint {
	d := derive(p)
	axisAt := func(x, y float64) parts.Axis {
		return parts.Axis{Point: frame.Apply(r3.Vec{X: x, Y: y}), Dir: frame.ApplyDir(r3.Vec{Z: 1})}
	}
	centre := axisAt(0, 0)
	var js []parts.Joint
	contact := func(name, a, b string) {
		js = append(js, parts.Joint{Name: name, Kind: parts.Contact, A: a, B: b})
	}
	pin := func(name, a, b string, axis parts.Axis, motion parts.Motion, clearance units.Value) {
		js = append(js, parts.Joint{Name: name, Kind: parts.PinInBore, A: a, B: b, Axis: &axis, Motion: motion,
			Clearance: clearance})
	}
	none := units.Value{}
	zero := units.Millimeters(0)
	seat := units.Millimeters(p.BearingClearance)
	press := units.Millimeters(pressFitClearance)
	hole := units.Millimeters(headClearance)

	// The input shaft passes through the front bearing and the front cover and fills the first sun's bore. Its collar
	// is wider than the bearing's bore and rests on the bearing's inner side, so the bearing and the cover leave over
	// the shaft's outer end and the shaft goes in from the sun's side.
	pin("input shaft in the front bearing", InputShaftName, FrontBearingName, centre, parts.Turns, zero)
	pin("input shaft in the front cover", InputShaftName, FrontCoverName, centre, parts.Turns, none)
	pin("input shaft in the first sun", InputShaftName, SunName(0), centre, parts.Fixed, press)
	pin("front bearing in the front cover", FrontBearingName, FrontCoverName, centre, parts.Fixed, seat)
	contact("front cover on the ring", FrontCoverName, RingName)

	last := len(g.Stages) - 1
	for k, s := range g.Stages {
		shaft := ShaftName(k, len(g.Stages))
		pin(shaft+" in "+CarrierName(k), shaft, CarrierName(k), centre, parts.Fixed, press)
		if k < last {
			pin(shaft+" in "+SunName(k+1), shaft, SunName(k+1), centre, parts.Fixed, press)
		}
		for i := range s.Planets {
			phi := PlanetOrbitAngle(p, i)
			at := axisAt(d.orbit*math.Cos(phi), d.orbit*math.Sin(phi))
			planet, bearing, carrierPin := PlanetName(k, i), BearingName(k, i), PinName(k, i)
			contact(fmt.Sprintf("sun %d meshes planet %d.%d", k+1, k+1, i+1), planet, SunName(k))
			contact(fmt.Sprintf("planet %d.%d meshes the ring", k+1, i+1), planet, RingName)
			pin(fmt.Sprintf("bearing %d.%d in its planet", k+1, i+1), bearing, planet, at, parts.Fixed, seat)
			pin(fmt.Sprintf("pin %d.%d in its bearing", k+1, i+1), carrierPin, bearing, at, parts.Turns, zero)
			pin(fmt.Sprintf("pin %d.%d in the carrier", k+1, i+1), carrierPin, CarrierName(k), at, parts.Fixed, press)
			pin(fmt.Sprintf("pin %d.%d through its planet's lip", k+1, i+1), carrierPin, planet, at, parts.Turns, none)
		}
	}

	// The output shaft passes through the rear cover and the rear bearing, and its collar rests on the bearing's inner
	// side like the input shaft's.
	output := ShaftName(last, len(g.Stages))
	pin("output shaft in the rear bearing", output, RearBearingName, centre, parts.Turns, zero)
	pin("output shaft in the rear cover", output, RearCoverName, centre, parts.Turns, none)
	pin("rear bearing in the rear cover", RearBearingName, RearCoverName, centre, parts.Fixed, seat)
	contact("rear cover on the ring", RearCoverName, RingName)

	// A bolt passes through its cover's hole and the ring's hole and screws into a nut that the ring holds in a
	// closed cavity.
	half := len(g.Bolts) / 2
	for i := range g.Bolts {
		cover, name, nut := FrontCoverName, FrontBoltName(i), FrontNutName(i)
		if i >= half {
			cover, name, nut = RearCoverName, RearBoltName(i-half), RearNutName(i-half)
		}
		angle := boltAngles()[i%half]
		at := axisAt(d.boltCircle*math.Cos(angle), d.boltCircle*math.Sin(angle))
		pin(name+" in the cover", name, cover, at, parts.Fixed, hole)
		pin(name+" in the ring", name, RingName, at, parts.Fixed, hole)
		contact(nut+" in the ring", nut, RingName)
		pin(name+" in "+nut, name, nut, at, parts.Fixed, zero)
	}
	return js
}

// Explode pulls the gearbox apart along its axis with parts.Set.Explode. set must be this gearbox's Set over the
// document Build used with p. The input shaft stays put and every other part moves along the axis in a stage of its
// own, each stage clearing the parts it is joined to; the ring stage clears every other part. Decad has no
// rule for the order in which parts can be inserted, so the stages state it.
//
// The stages run outward from the input shaft, and each moves its parts past the ones moved before, so the stage
// order is the order of the parts along the axis, not the order in which they come off the gearbox. Toward the front
// the front cover moves first, then the front bearing and then the front bolts beyond it. A bearing is wider than the
// cover's 10 mm lip bore, so to take the gearbox apart the bearing comes off the shaft before the cover does.
// Toward the rear each stage of the gearbox moves as the first sun and its planets, the planets' bearings, the
// carrier's pins, the carrier and the shaft pressed into it, and the next stage follows the same way. The ring, the
// rear cover, the rear bearing and the rear bolts follow beyond the output shaft. A shaft's collar is wider than the
// bore of the bearing it passes through, so each shaft stays on the carrier side of that bearing and its cover.
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
	// The ring is joined to the planets but not to their bearings or pins, which it passes on its way, so it clears
	// every other part, not only the parts it is joined to.
	ring := together("ring", rear, append([]string{RingName}, nuts...)...)
	ring.Clear = parts.ClearAll
	stages = append(stages,
		ring,
		stage("rear cover", rear, RearCoverName),
		stage("rear bearing", rear, RearBearingName),
		stage("rear bolts", rear, rearBolts...))
	return set.Explode(ctx, parts.WithExplodeGround(InputShaftName), parts.WithExplodeGap(units.Millimeters(explodeGap)),
		parts.WithExplodeStages(stages...))
}
