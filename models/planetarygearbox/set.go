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

// wideGap is the gap of the ring and the front bearing in the exploded view. The README image looks at the gearbox
// from above, and from there the ring or a cover's rim hides the top of a part that sits explodeGap below it. With
// wideGap the input shaft's collar and flat end show below the ring, and the front bearing below the front cover.
const wideGap = 20.0

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

// joints lists the connections of the assembled gearbox with each axis stated in world coordinates. Each shaft turns
// in its cover's bearing and passes the cover's lip bore, and each planet turns on its carrier pin through its bearing
// and its lip bore; every other joint holds its parts fixed. A pin-in-bore joint declares the radial clearance of its
// fit: zero for a shaft or pin in a bearing bore and for a bolt in its nut, Params.BearingClearance for a bearing in
// its seat, pressFitClearance for a shaft or pin pressed into a sun or carrier, and headClearance for a bolt in its
// hole. A lip bore is far wider than the shaft or pin in it and is not a fit, so those joints declare no clearance.
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
	// is wider than the bearing's bore and rests on the bearing's inner side.
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

// Explode pulls the gearbox apart along its axis with parts.Set.Explode in Decad's derived mode. set must be this
// gearbox's Set over the document Build used. The front cover stays put. Decad reads from the tessellated bodies which
// parts stand in the way of which along each direction, takes the parts off in rounds, each round holding the parts
// nothing still present blocks, and lays the rounds out with the round removed last nearest the front cover. Every
// part leaves along the gearbox's axis, toward the front or the rear, and ends explodeGap beyond the parts it clears,
// or wideGap for the ring and the front bearing.
// The layout is a picture, not a proven assembly sequence: Decad counts a pair it cannot settle from the meshes as
// free.
//
// The input shaft's collar is narrower than the front cover's lip bore, so the shaft could leave toward the front as
// soon as the front bearing is off. Explode makes it leave toward the rear instead, and only after the ring, so the
// layout draws it on the inner side of the front cover, next to it, as the output shaft sits on the inner side of the
// rear cover: front bearing, front cover, input shaft, from the outside in.
func (g Gearbox) Explode(ctx context.Context, set *parts.Set) (*parts.Exploded, error) {
	var rear r3.Vec
	for _, j := range set.Joints() {
		if j.A == InputShaftName && j.B == FrontBearingName && j.Axis != nil {
			rear = j.Axis.Dir
		}
	}
	if rear == (r3.Vec{}) {
		return nil, fmt.Errorf("planetarygearbox: the set has no axis joining %s and %s", InputShaftName,
			FrontBearingName)
	}
	return set.Explode(ctx, parts.WithDerivedExplode(), parts.WithExplodeGround(FrontCoverName),
		parts.WithExplodeGap(units.Millimeters(explodeGap)),
		parts.WithRemoval(InputShaftName, parts.Removal{Direction: rear, After: []string{RingName}}),
		parts.WithPartGap(RingName, units.Millimeters(wideGap)),
		parts.WithPartGap(FrontBearingName, units.Millimeters(wideGap)))
}
