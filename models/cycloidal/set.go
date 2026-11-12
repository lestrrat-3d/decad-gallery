package cycloidal

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad/parts"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// explodeGap is how far past touching each part of the exploded view sits from the part it clears.
const explodeGap = 12.0

// Set returns the drive as a parts.Set over doc, which must be the document Build used with p: every part with its
// material (all printed in PLA) and the joints between them. A in a joint is the part that goes in and B the part it
// goes into. The set feeds Drive.Explode.
func (d Drive) Set(doc *decad.Document, p Params) (*parts.Set, error) {
	frame, err := placement(p)
	if err != nil {
		return nil, fmt.Errorf("cycloidal: placement: %w", err)
	}
	set := parts.NewSet(doc)
	for _, nb := range d.Named() {
		part := parts.Part{Name: nb.Name, Kind: parts.Manufactured, Body: nb.Body, Material: parts.Material{Name: "PLA"}}
		if err := set.Add(part); err != nil {
			return nil, fmt.Errorf("cycloidal: add %s to the parts set: %w", nb.Name, err)
		}
	}
	for _, j := range d.joints(p, frame) {
		if err := set.Join(j); err != nil {
			return nil, fmt.Errorf("cycloidal: join %s: %w", j.Name, err)
		}
	}
	return set, nil
}

// joints lists the connections of the assembled drive with each axis stated in world coordinates, all along the
// placement normal. Each disc orbits in the housing and on the cam, and each output pin is pressed into its socket in
// the output plate and passes through every disc's output hole. A disc's fits are wider than a press fit and declare
// no clearance; the pin's socket declares socketClearance.
func (d Drive) joints(p Params, frame r3.Frame) []parts.Joint {
	r := &run{p: p, d: resolve(p), frame: frame}
	axisAt := func(x, y float64) parts.Axis {
		point := frame.Origin().Add(frame.U().Scale(x)).Add(frame.V().Scale(y))
		return parts.Axis{Point: point, Dir: frame.N()}
	}
	var js []parts.Joint
	pin := func(name, a, b string, axis parts.Axis, motion parts.Motion, clearance units.Value) {
		js = append(js, parts.Joint{Name: name, Kind: parts.PinInBore, A: a, B: b, Axis: &axis, Motion: motion,
			Clearance: clearance})
	}
	none := units.Value{}
	for i := range d.Discs {
		disc := DiscName(i)
		cx, _ := r.discGeometry(i)
		pin(disc+" in the housing", disc, HousingName, axisAt(0, 0), parts.Turns, none)
		pin(CamName+" in "+disc, CamName, disc, axisAt(cx, 0), parts.Turns, none)
	}
	for j := range d.OutputPins {
		c := r.outputPinCenter(j)
		at := axisAt(c.x, c.y)
		name := OutputPinName(j)
		pin(name+" in the output plate", name, OutputPlateName, at, parts.Fixed, units.Millimeters(socketClearance))
		for i := range d.Discs {
			pin(fmt.Sprintf("%s through %s", name, DiscName(i)), name, DiscName(i), at, parts.Turns, none)
		}
	}
	return js
}

// Explode pulls the drive apart along its axis with parts.Set.Explode in Decad's derived mode. set must be this
// drive's Set over the document Build used. The housing stays put. Decad reads from the tessellated bodies which
// parts stand in the way of which along each direction, takes the parts off in rounds, and lays the rounds out
// with the round removed last nearest the housing. Every part leaves along the drive's axis and ends explodeGap
// beyond the parts it clears. The layout is a picture, not a proven assembly sequence.
func (d Drive) Explode(ctx context.Context, set *parts.Set) (*parts.Exploded, error) {
	return set.Explode(ctx, parts.WithDerivedExplode(), parts.WithExplodeGround(HousingName),
		parts.WithExplodeGap(units.Millimeters(explodeGap)))
}
