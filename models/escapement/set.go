package escapement

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad/parts"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// explodeGap is how far past touching each part of the exploded view sits from the part it clears, in mm.
const explodeGap = 8.0

// NutsStageName names the explode stage that lifts the four nuts out of the back plate's cavities.
const NutsStageName = "nuts"

// Set returns the movement as a parts.Set over doc, which must be the document Build used with p: every part with
// its material and the joints between them. A in a joint is the part that goes in and B the part it goes into. The
// set feeds Explode.
func (m *Movement) Set(doc *decad.Document, p Params) (*parts.Set, error) {
	g, err := Layout(p)
	if err != nil {
		return nil, fmt.Errorf("escapement: layout: %w", err)
	}
	frame, err := frameTransform(p)
	if err != nil {
		return nil, fmt.Errorf("escapement: %w", err)
	}
	set := parts.NewSet(doc)
	for _, nb := range m.Named() {
		part := parts.Part{Name: nb.Name, Kind: parts.Manufactured, Body: nb.Body, Material: parts.Material{Name: "PLA"}}
		if strings.HasSuffix(nb.Name, "Bearing") || strings.HasPrefix(nb.Name, "Bolt ") || strings.HasPrefix(nb.Name, "Nut ") {
			part.Kind, part.Material = parts.Purchased, parts.Material{Name: "steel"}
		}
		if nb.Name == MainspringName {
			part.Material = parts.Material{Name: "PETG"}
		}
		if err := set.Add(part); err != nil {
			return nil, fmt.Errorf("escapement: add %s to the parts set: %w", nb.Name, err)
		}
	}
	for _, j := range joints(&g, frame) {
		if err := set.Join(j); err != nil {
			return nil, fmt.Errorf("escapement: join %s: %w", j.Name, err)
		}
	}
	return set, nil
}

// joints lists the connections of the assembled movement, each axis in world coordinates along the movement's
// Z axis. A turning joint lets A turn in B; a fixed joint holds A in B.
func joints(g *Geometry, frame r3.Transform) []parts.Joint {
	axisAt := func(at Pt) parts.Axis {
		return parts.Axis{Point: frame.Apply(r3.NewVec(at.X, at.Y, 0)),
			Dir: frame.ApplyDir(r3.NewVec(0, 0, 1))}
	}
	var js []parts.Joint
	pin := func(name, a, b string, at Pt, motion parts.Motion) {
		axis := axisAt(at)
		js = append(js, parts.Joint{Name: name, Kind: parts.PinInBore, A: a, B: b, Axis: &axis, Motion: motion})
	}
	contact := func(name, a, b string) {
		js = append(js, parts.Joint{Name: name, Kind: parts.Contact, A: a, B: b})
	}

	// Each bearing sits in a seat of its plate.
	pin("back escape bearing in the back plate", BackEscapeBearingName, BackPlateName, g.O, parts.Fixed)
	pin("back pallet bearing in the back plate", BackPalletBearingName, BackPlateName, g.P, parts.Fixed)
	pin("back great wheel bearing in the back plate", BackGreatWheelBearingName, BackPlateName, g.G, parts.Fixed)
	pin("front escape bearing in the front plate", FrontEscapeBearingName, FrontPlateName, g.O, parts.Fixed)
	pin("front pallet bearing in the front plate", FrontPalletBearingName, FrontPlateName, g.P, parts.Fixed)
	pin("front great wheel bearing in the front plate", FrontGreatWheelBearingName, FrontPlateName, g.G, parts.Fixed)

	// Each arbor turns in its two bearings, and its wheels sit on it.
	pin("escape arbor in the back bearing", EscapeArborName, BackEscapeBearingName, g.O, parts.Turns)
	pin("escape arbor in the front bearing", EscapeArborName, FrontEscapeBearingName, g.O, parts.Turns)
	pin("pallet arbor in the back bearing", PalletArborName, BackPalletBearingName, g.P, parts.Turns)
	pin("pallet arbor in the front bearing", PalletArborName, FrontPalletBearingName, g.P, parts.Turns)
	pin("great wheel arbor in the back bearing", GreatWheelArborName, BackGreatWheelBearingName, g.G, parts.Turns)
	pin("great wheel arbor in the front bearing", GreatWheelArborName, FrontGreatWheelBearingName, g.G, parts.Turns)
	pin("escape wheel on its arbor", EscapeWheelName, EscapeArborName, g.O, parts.Fixed)
	pin("anchor on the pallet arbor", AnchorName, PalletArborName, g.P, parts.Fixed)
	pin("escape rear collar on the escape arbor", EscapeRearCollarName, EscapeArborName, g.O, parts.Fixed)
	pin("pallet front collar on the pallet arbor", PalletFrontCollarName, PalletArborName, g.P, parts.Fixed)
	pin("crutch collar on the pallet arbor", CrutchCollarName, PalletArborName, g.P, parts.Fixed)
	pin("knob collar on the great wheel arbor", KnobCollarName, GreatWheelArborName, g.G, parts.Fixed)
	pin("crutch on the pallet arbor", CrutchName, PalletArborName, g.P, parts.Fixed)
	pin("pawl carrier on the great wheel arbor", PawlCarrierName, GreatWheelArborName, g.G, parts.Fixed)
	pin("mainspring on the great wheel arbor", MainspringName, GreatWheelArborName, g.G, parts.Fixed)
	pin("spring spacer on the great wheel arbor", SpringSpacerName, GreatWheelArborName, g.G, parts.Fixed)
	pin("winding knob on the great wheel arbor", WindingKnobName, GreatWheelArborName, g.G, parts.Fixed)
	pin("great wheel on its arbor", GreatWheelName, GreatWheelArborName, g.G, parts.Turns)
	contact("great wheel meshes the escape wheel's pinion", GreatWheelName, EscapeWheelName)
	contact("anchor meets the escape wheel", AnchorName, EscapeWheelName)
	contact("carrier pawls on the great wheel", PawlCarrierName, GreatWheelName)
	pin("spring anchor pin in the front plate", SpringAnchorPinName, FrontPlateName, g.SpringAnchor, parts.Fixed)
	pin("mainspring eye on the spring anchor pin", MainspringName, SpringAnchorPinName, g.SpringAnchor, parts.Turns)

	// The pendulum hangs on the suspension pin through its bearing, and its slot takes the crutch's pin.
	pin("pendulum bearing in the pendulum", PendulumBearingName, PendulumName, g.S, parts.Fixed)
	pin("suspension pin in the pendulum bearing", SuspensionPinName, PendulumBearingName, g.S, parts.Turns)
	pin("suspension pin in the front plate", SuspensionPinName, FrontPlateName, g.S, parts.Fixed)
	pin("suspension cap on the suspension pin", SuspensionCapName, SuspensionPinName, g.S, parts.Fixed)
	contact("crutch pin in the pendulum slot", CrutchName, PendulumName)

	// The front plate rests on the back plate's pillars, and a bolt through the front plate and the pillar screws
	// into a nut that the pillar holds in a closed cavity.
	contact("front plate on the back plate's pillars", FrontPlateName, BackPlateName)
	for i, at := range pillars {
		bolt, nut := BoltName(i), NutName(i)
		pin(bolt+" in the front plate", bolt, FrontPlateName, at, parts.Fixed)
		pin(bolt+" in the back plate", bolt, BackPlateName, at, parts.Fixed)
		contact(nut+" in the back plate", nut, BackPlateName)
		pin(bolt+" in "+nut, bolt, nut, at, parts.Fixed)
	}
	return js
}

// Explode pulls the movement apart along its axis with parts.Set.Explode in Decad's derived mode. set must be this
// movement's Set over the document Build used. The back plate stays put. Decad reads from the tessellated bodies
// which parts stand in the way of which, takes the parts off in rounds, and lays the rounds out with the round removed
// last nearest the back plate. The movement is thin and its layers do not share an axis, so Decad also slides each
// layer sideways into its own cell and records each part's guide line back to its joint partner in Exploded.Guides.
// The four nuts, which the derived search finds no direction for, leave in a stage of their own.
// The layout is a picture, not a proven assembly sequence.
func (m *Movement) Explode(ctx context.Context, set *parts.Set) (*parts.Exploded, error) {
	var nuts []string
	for i := range m.Nuts {
		nuts = append(nuts, NutName(i))
	}
	return set.Explode(ctx, parts.WithDerivedExplode(), parts.WithExplodeGround(BackPlateName),
		parts.WithExplodeGap(units.Millimeters(explodeGap)),
		parts.WithExplodeStages(parts.Stage{Name: NutsStageName, Parts: nuts}))
}
