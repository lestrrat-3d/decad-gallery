package sevensegment

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/decad-gallery/models/compressionspring"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// The click indexer sits on the knob end of the camshaft. The box stands on its back wall, so model -Y points up, and the
// indexer's parts stack along -Y from the front wall. Every Y below is a world Y at digit 0 and rest; a local frame has its
// Z along -Y (frontFrame), so a part's local Z is its height above its reference plane.
const (
	frontWallOuterY = -57.5
	chamfer         = 0.5 // "Surface edge finishing": the equal-setback chamfer of a large face's rim

	// Keyed hub: pressed on the stub's D flat. A cup whose wall carries the keys of the rotor's skirt.
	hubBaseY       = -57.9
	hubBaseLength  = 3.5
	hubRimY        = -67.3 // the rotor's lower stop, 0.1 short of its plate at the full 2.0 drop
	hubRadius      = 10.5
	hubWallInner   = 9.0
	hubKeyInner    = 8.05
	hubKeyWidth    = 2.0
	hubKeyCount    = 3
	hubSpringFloor = hubBaseY - hubBaseLength // y of the base's top face, where the spring stands: -61.4

	// Rotor: skirt keyed in the hub, a plate with the inner teeth that meet the button and the outer teeth that lock in the
	// fixed crown. Local Z is height above the plate's top face at rotorPlateY.
	rotorPlateY      = -70.9
	rotorPlateThick  = 1.5
	rotorPlateRadius = 14.2
	rotorBoreRadius  = 3.0
	rotorSkirtInner  = 4.05 // spring outside radius 3.725 plus 0.325, over the Tight 0.2
	rotorSkirtOuter  = 8.45
	rotorSkirtLength = 5.5
	rotorSlotDepthTo = 7.85 // radius of a skirt slot's floor
	rotorSlotWidth   = hubKeyWidth + 2*0.2
	rotorInnerR0     = 3.2
	rotorInnerR1     = 11.0
	rotorOuterR0     = 12.8
	rotorOuterR1     = 14.0

	// Housing: base on the front wall, tube and a bezel, a fixed crown ring that holds the rotor's outer teeth.
	housingBaseY     = -57.75
	housingBaseThick = 4.0
	housingBezelY    = -75.4 // 17.9 above the wall: the button's head ends flush with it
	housingTubeInner = 14.55
	housingTubeOuter = 16.55
	housingFlangeR   = 24.8 // base flange: inside the walls (x +-24.9); 1.1 round the H head pockets after the chamfer
	housingCrownR0   = 11.55
	housingCrownY    = -72.5 // underside of the ring, above the crown's teeth
	housingSlotOuter = 12.7
	housingSlotWidth = 3.0

	// Button: head, ten teeth under it, three keys that slide in the housing's slots.
	buttonUndersideY = -78.9
	buttonHeadThick  = 2.5
	buttonRadius     = 11.0
	buttonRecessR    = 3.2
	buttonRecessDeep = 1.4
	buttonToothR0    = 3.2
	buttonKeyInner   = 11.4
	buttonKeyOuter   = 12.2
	buttonKeyWidth   = 2.0
	buttonKeyDrop    = 3.8 // a key hangs this far below the head's underside
	// ButtonStroke is how far the head goes down, from its rest to flush with the bezel.
	ButtonStroke = 6.0

	// The click indexer's ten steps.
	ClickerSteps = 10
	// RotorTravel is how far the button pushes the rotor down before it turns, mm.
	RotorTravel = 2.0
	// ClickerStep is the rotor's turn per step, degrees.
	ClickerStep = 36.0
	// RampAngle is the button's ramp over one press, degrees. The release adds ClickerStep - RampAngle.
	RampAngle = 32.5
	// ToothHeight is the height of the inner teeth, mm.
	ToothHeight = 4.0
	// CrownHeight is the height of the outer teeth, mm.
	CrownHeight = 1.6
	// CrownBack is the fixed crown's long back ramp, degrees: the rotor tips run up to it, then the short ramp turns the rotor on.
	CrownBack = 30.0
	// SpringRestLength is the spring's installed length at rest, mm.
	SpringRestLength = 8.0
)

// fUp is the frame of a part whose Z is the height above y0 toward -Y (up): its angle about +Z runs backward against the
// shaft's forward turn.
func fUp(y0 float64) r3.Frame { return frontFrame(y0) }

// fDown is the frame of a part whose Z runs toward +Y (the front wall): its angle about +Z is the shaft's forward angle.
func fDown(y0 float64) r3.Frame {
	f, _ := r3.NewFrame(r3.Vec{Y: y0, Z: ShaftZ}, r3.Vec{Z: 1}, r3.Vec{X: 1})
	return f
}

// The teeth of the four rings. Angles are in the part's own frame (see fUp, fDown); shift turns each ring so its tips lie
// where the design puts them in forward angle: the rotor's inner tips and the button's tips at 0, the crown's tips and the
// rotor's outer notches at ClickerStep - CrownBack.
func rotorInnerTeeth() faceTeeth {
	return faceTeeth{Name: "rotor inner", R0: rotorInnerR0, R1: rotorInnerR1, H: ToothHeight, Rise: ClickerStep - RampAngle, Fall: RampAngle,
		Tip: 0.05, Bot: 0.01, Plate: 0.03, Overhang: 0.3}
}

func rotorOuterTeeth() faceTeeth {
	return faceTeeth{Name: "rotor outer", R0: rotorOuterR0, R1: rotorOuterR1, H: CrownHeight, Rise: CrownBack,
		Fall: ClickerStep - CrownBack, Tip: 0.05, Bot: 0.01, Plate: 0.03, Overhang: 0.3}
}

func crownTeeth() faceTeeth {
	return faceTeeth{Name: "crown", R0: rotorOuterR0, R1: housingTubeInner + 0.3, H: CrownHeight, Rise: CrownBack,
		Fall: ClickerStep - CrownBack, Tip: 0.05, Bot: 0.01, Plate: 0.03, Overhang: 0.3}
}

func buttonTeeth() faceTeeth {
	return faceTeeth{Name: "button", R0: buttonRecessR + 0.2, R1: buttonRadius - 0.15, H: ToothHeight, Rise: ClickerStep - RampAngle, Fall: RampAngle,
		Tip: 0.05, Bot: 0.01, Plate: 0.03, Overhang: 0.3}
}

// ringSolid is a full turn about the shaft axis of the polygon through the (radius, height) vertices, placed at world Y y
// with its height along -Y.
func (r *run) ringSolid(y float64, vertices [][2]float64) (*decad.Body, error) {
	body, err := r.meridian(vertices)
	if err != nil {
		return nil, fmt.Errorf("revolve: %w", err)
	}
	f := frontFrame(y)
	return placement.Place(r.ctx, body, &f)
}

// radialBox is a box whose inner and outer faces lie at radii r0 and r1 on the +X axis, across the width w, over world Y
// from yLow to yHigh (yLow < yHigh), turned by angle degrees about the shaft axis.
func (r *run) radialBox(r0, r1, w, yLow, yHigh, angle float64) (*decad.Body, error) {
	s, err := r.sketchOn(frontFrame(yHigh))
	if err != nil {
		return nil, err
	}
	polygon(s, [][2]float64{{r0, -w / 2}, {r1, -w / 2}, {r1, w / 2}, {r0, w / 2}})
	body, err := r.extrude(s, 0, yHigh-yLow, decad.Along)
	if err != nil {
		return nil, err
	}
	turn, err := shaftTurn(angle)
	if err != nil {
		return nil, err
	}
	return decad.Transform(r.ctx, body, turn)
}

// shaftTurn is the rotation by deg degrees about the shaft axis, right-handed about +Y.
func shaftTurn(deg float64) (r3.Transform, error) {
	return r3.RotationAround(r3.Vec{Z: ShaftZ}, r3.Vec{Y: 1}, units.Degrees(deg))
}

// boreD cuts the stub's D bore through body from the plane at Y y over length.
func (r *run) boreD(body *decad.Body, y, length float64) (*decad.Body, error) {
	s, err := r.sketchOn(frontFrame(y + 1))
	if err != nil {
		return nil, err
	}
	dBoreLoop(s, boreStubRadius, boreStubFlat)
	tool, err := r.extrude(s, 0, length+2, decad.Along)
	if err != nil {
		return nil, fmt.Errorf("bore tool: %w", err)
	}
	return decad.Cut(r.ctx, body, tool)
}

// hub builds the keyed hub: the collar ring's D-bored base with a cup on it, three keys inside the cup wall.
func (r *run) hub() (*decad.Body, error) {
	top := hubBaseY - hubRimY // height of the rim over the base's underside
	body, err := r.ringSolid(hubBaseY, [][2]float64{{0, 0}, {hubRadius, 0}, {hubRadius, top}, {hubWallInner, top},
		{hubWallInner, hubBaseLength}, {0, hubBaseLength}})
	if err != nil {
		return nil, err
	}
	var keys []*decad.Body
	for k := range hubKeyCount {
		key, err := r.radialBox(hubKeyInner, hubWallInner+0.5, hubKeyWidth, hubRimY, hubBaseY-hubBaseLength+0.5,
			360/float64(hubKeyCount)*float64(k))
		if err != nil {
			return nil, fmt.Errorf("key %d: %w", k+1, err)
		}
		keys = append(keys, key)
	}
	for _, k := range keys {
		if body, err = decad.Union(r.ctx, body, k); err != nil {
			return nil, err
		}
	}
	return r.boreD(body, hubBaseY, hubBaseLength)
}

// rotor builds the rotor in the world at its rest and digit 0, in a frame whose Z is the height above the plate's nominal
// top face.
func (r *run) rotor() (*decad.Body, error) {
	in, out := rotorInnerTeeth(), rotorOuterTeeth()
	const skirtTop = -rotorPlateThick
	body, err := r.meridian([][2]float64{{rotorBoreRadius, skirtTop}, {rotorSkirtInner, skirtTop}, {rotorSkirtInner, skirtTop - rotorSkirtLength},
		{rotorSkirtOuter, skirtTop - rotorSkirtLength}, {rotorSkirtOuter, skirtTop}, {rotorPlateRadius, skirtTop},
		{rotorPlateRadius, in.Plate}, {rotorBoreRadius, in.Plate}}) // one root land under both rings: in.Plate == out.Plate
	if err != nil {
		return nil, err
	}
	// The plate's underside rim carries the 0.5 chamfer of "Surface edge finishing".
	if body, _, err = edgefinish.Chamfer(r.ctx, body, r3.Vec{Z: 1}, []float64{skirtTop}, edgefinish.Long(edgefinish.Outer), chamfer); err != nil {
		return nil, fmt.Errorf("chamfer rim: %w", err)
	}
	// The skirt's slots take the hub's keys, with a Tight fit each side.
	for k := range hubKeyCount {
		slot, err := r.localBox(rotorSlotDepthTo, rotorSkirtOuter+0.5, rotorSlotWidth, skirtTop-rotorSkirtLength-0.5, skirtTop,
			360/float64(hubKeyCount)*float64(k))
		if err != nil {
			return nil, fmt.Errorf("slot %d: %w", k+1, err)
		}
		if body, err = decad.Cut(r.ctx, body, slot); err != nil {
			return nil, fmt.Errorf("cut slot %d: %w", k+1, err)
		}
	}
	for _, ring := range []struct {
		f     faceTeeth
		shift float64
	}{{in, -(ClickerStep - RampAngle)}, {out, -CrownBack}} {
		teeth, err := r.faceTeethBodies(ring.f, ring.shift)
		if err != nil {
			return nil, fmt.Errorf("%s teeth: %w", ring.f.Name, err)
		}
		if body, err = r.joinTeeth(body, teeth); err != nil {
			return nil, fmt.Errorf("%s: %w", ring.f.Name, err)
		}
	}
	f := fUp(rotorPlateY)
	return placement.Place(r.ctx, body, &f)
}

// housing builds the housing: the base flange with its two bolt pockets, the tube, the bezel and the fixed crown.
func (r *run) housing() (*decad.Body, error) {
	cr := crownTeeth()
	zOf := func(y float64) float64 { return y - housingCrownY } // height toward the wall above the crown's valley plane
	zBez, zTop, zBot := zOf(housingBezelY), zOf(housingBaseY-housingBaseThick), zOf(housingBaseY)
	// The 0.5 chamfers of "Surface edge finishing": the bezel's two rims and the flange's outer rim on both faces.
	body, err := r.meridian([][2]float64{{housingCrownR0 + chamfer, zBez}, {housingTubeOuter - chamfer, zBez},
		{housingTubeOuter, zBez + chamfer}, {housingTubeOuter, zTop}, {housingFlangeR - chamfer, zTop},
		{housingFlangeR, zTop + chamfer}, {housingFlangeR, zBot - chamfer}, {housingFlangeR - chamfer, zBot},
		{housingTubeInner, zBot}, {housingTubeInner, cr.Plate}, {housingCrownR0, cr.Plate}, {housingCrownR0, zBez + chamfer}})
	if err != nil {
		return nil, err
	}
	for k := range hubKeyCount {
		slot, err := r.localBox(housingCrownR0-0.5, housingSlotOuter, housingSlotWidth, zOf(housingBezelY)-1, 1,
			360/float64(hubKeyCount)*float64(k))
		if err != nil {
			return nil, fmt.Errorf("slot %d: %w", k+1, err)
		}
		if body, err = decad.Cut(r.ctx, body, slot); err != nil {
			return nil, fmt.Errorf("cut slot %d: %w", k+1, err)
		}
	}
	teeth, err := r.faceTeethBodies(cr, 0)
	if err != nil {
		return nil, fmt.Errorf("crown teeth: %w", err)
	}
	if body, err = r.joinTeeth(body, teeth); err != nil {
		return nil, fmt.Errorf("crown: %w", err)
	}
	f := fDown(housingCrownY)
	if body, err = placement.Place(r.ctx, body, &f); err != nil {
		return nil, err
	}
	for i, site := range r.dm.Sites[8:] {
		hole, sk, err := m2fastener.HoleTool(r.ctx, r.doc, site)
		if err != nil {
			return nil, fmt.Errorf("hole %d: %w", i+1, err)
		}
		pocket, sk2, err := m2fastener.PocketTool(r.ctx, r.doc, site)
		if err != nil {
			return nil, fmt.Errorf("head pocket %d: %w", i+1, err)
		}
		r.sketches = append(r.sketches, sk, sk2)
		for _, tool := range []*decad.Body{hole, pocket} {
			if body, err = decad.Cut(r.ctx, body, tool); err != nil {
				return nil, fmt.Errorf("cut bolt %d: %w", i+1, err)
			}
		}
	}
	return body, nil
}

// buttonKeyAngle is the angle of button key k about the shaft axis, degrees.
func buttonKeyAngle(k int) float64 { return 360 / float64(hubKeyCount) * float64(k) }

// button builds the button at its rest, in a frame whose Z runs toward the wall from the head's underside.
func (r *run) button() (*decad.Body, error) {
	bt := buttonTeeth()
	body, err := r.meridian([][2]float64{{0, -buttonHeadThick}, {buttonRadius, -buttonHeadThick}, {buttonRadius, bt.Plate},
		{buttonRecessR, bt.Plate}, {buttonRecessR, bt.Plate - buttonRecessDeep}, {0, bt.Plate - buttonRecessDeep}})
	if err != nil {
		return nil, err
	}
	// Each key is a flange on the head's side with a lug hanging under it, which slides in a slot of the crown ring. The keys
	// join the plain head: Decad refuses a union on the toothed one.
	for k := range hubKeyCount {
		for _, box := range [][4]float64{{10.0, buttonKeyOuter, -buttonHeadThick + 0.3, 0}, {buttonKeyInner, buttonKeyOuter, -0.3, buttonKeyDrop}} {
			b, err := r.localBox(box[0], box[1], buttonKeyWidth, box[2], box[3], buttonKeyAngle(k))
			if err != nil {
				return nil, fmt.Errorf("key %d: %w", k+1, err)
			}
			if body, err = decad.Union(r.ctx, body, b); err != nil {
				return nil, fmt.Errorf("union key %d: %w", k+1, err)
			}
		}
	}
	teeth, err := r.faceTeethBodies(bt, ClickerStep-RampAngle)
	if err != nil {
		return nil, fmt.Errorf("teeth: %w", err)
	}
	if body, err = r.joinTeeth(body, teeth); err != nil {
		return nil, err
	}
	f := fDown(buttonUndersideY)
	return placement.Place(r.ctx, body, &f)
}

// spring builds the click indexer's return spring, a Lesjofors Cat.no 2353 of the given installed length: its axis the shaft's, its
// first end on the hub's top face and its second end length away toward the rotor (-Y).
func (r *run) spring(length float64) (*decad.Body, error) {
	m, err := compressionspring.Build(r.ctx, r.doc, compressionspring.Params{Spec: compressionspring.LesjoforsStock2353(),
		InstalledLength: length})
	if err != nil {
		return nil, err
	}
	// The coil winds about +Y from half a wire below Y = 0, round the origin. Turn it half a turn about X so it runs along
	// -Y, then carry its axis onto the shaft and its first end onto the hub's top face.
	flip, err := r3.RotationAround(r3.Vec{}, r3.Vec{X: 1}, units.Degrees(180))
	if err != nil {
		return nil, err
	}
	body, err := decad.Transform(r.ctx, m.Body, flip)
	if err != nil {
		return nil, err
	}
	lift, err := r3.Translation(r3.Vec{Y: hubSpringFloor - compressionspring.LesjoforsStock2353().WireDiameter/2, Z: ShaftZ})
	if err != nil {
		return nil, err
	}
	return decad.Transform(r.ctx, body, lift)
}

// Clicker is the click indexer's bodies at digit 0 and rest, in the world frame, without the front wall's changes.
type Clicker struct {
	Hub, Rotor, Housing, Button, Spring *decad.Body
}

// BuildClicker adds the click indexer's five bodies to doc. ctx and doc must not be nil.
func BuildClicker(ctx context.Context, doc *decad.Document) (*Clicker, error) {
	if ctx == nil {
		return nil, errors.New("sevensegment: context must not be nil")
	}
	if doc == nil {
		return nil, errors.New("sevensegment: document must not be nil")
	}
	dm, err := DefaultParams().Derive()
	if err != nil {
		return nil, err
	}
	r := &run{ctx: ctx, doc: doc, p: DefaultParams(), dm: dm}
	out := &Clicker{}
	for _, st := range []struct {
		name  string
		build func() (*decad.Body, error)
		dst   **decad.Body
	}{
		{"hub", r.hub, &out.Hub}, {"rotor", r.rotor, &out.Rotor}, {"housing", r.housing, &out.Housing},
		{"button", r.button, &out.Button}, {"spring", func() (*decad.Body, error) { return r.spring(SpringRestLength) }, &out.Spring},
	} {
		if *st.dst, err = st.build(); err != nil {
			return nil, fmt.Errorf("sevensegment: %s: %w", st.name, err)
		}
	}
	return out, nil
}

// BuildSpring adds the return spring at the given installed length, from 3.78 (solid) to 15 (free) mm, to doc.
func BuildSpring(ctx context.Context, doc *decad.Document, length float64) (*decad.Body, error) {
	dm, err := DefaultParams().Derive()
	if err != nil {
		return nil, err
	}
	r := &run{ctx: ctx, doc: doc, p: DefaultParams(), dm: dm}
	return r.spring(length)
}

// BuildClickerPart builds one named click-indexer body (hub, rotor, housing or button) in doc.
func BuildClickerPart(ctx context.Context, doc *decad.Document, name string) (*decad.Body, error) {
	dm, err := DefaultParams().Derive()
	if err != nil {
		return nil, err
	}
	r := &run{ctx: ctx, doc: doc, p: DefaultParams(), dm: dm}
	switch name {
	case "hub":
		return r.hub()
	case "rotor":
		return r.rotor()
	case "housing":
		return r.housing()
	case "button":
		return r.button()
	}
	return nil, fmt.Errorf("sevensegment: no click-indexer part %q", name)
}
