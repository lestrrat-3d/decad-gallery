package escapement

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Movement holds the bodies of a built movement, posed at p.Phase, and every sketch drawn to build them.
type Movement struct {
	BackPlate, FrontPlate *decad.Body

	BackEscapeBearing, BackPalletBearing, BackGreatWheelBearing    *decad.Body
	FrontEscapeBearing, FrontPalletBearing, FrontGreatWheelBearing *decad.Body
	PendulumBearing                                                *decad.Body

	EscapeArbor, PalletArbor, GreatWheelArbor *decad.Body

	EscapeWheel, Anchor                                    *decad.Body
	Crutch, Pendulum, SuspensionPin, SuspensionCap         *decad.Body
	GreatWheel, WindingKnob                                *decad.Body
	PawlCarrier, Mainspring, SpringSpacer, SpringAnchorPin *decad.Body
	Bolts, Nuts                                            []*decad.Body

	// Sketches holds every sketch of the movement's own parts and of the bearing, bolt and nut models.
	Sketches []*sketch.Sketch
	// State is the pose the bodies were built at.
	State State
}

// NamedBody is a body with its display name.
type NamedBody struct {
	Name string
	Body *decad.Body
}

// BoltName and NutName name bolt and nut i (0-based).
func BoltName(i int) string { return fmt.Sprintf("Bolt %d", i+1) }

// NutName names nut i (0-based).
func NutName(i int) string { return fmt.Sprintf("Nut %d", i+1) }

// Body names.
const (
	BackPlateName              = "Back Plate"
	FrontPlateName             = "Front Plate"
	BackEscapeBearingName      = "Back Escape Bearing"
	BackPalletBearingName      = "Back Pallet Bearing"
	BackGreatWheelBearingName  = "Back Great Wheel Bearing"
	FrontEscapeBearingName     = "Front Escape Bearing"
	FrontPalletBearingName     = "Front Pallet Bearing"
	FrontGreatWheelBearingName = "Front Great Wheel Bearing"
	PendulumBearingName        = "Pendulum Bearing"
	EscapeArborName            = "Escape Arbor"
	PalletArborName            = "Pallet Arbor"
	GreatWheelArborName        = "Great Wheel Arbor"
	EscapeWheelName            = "Escape Wheel"
	AnchorName                 = "Anchor"
	CrutchName                 = "Crutch"
	PendulumName               = "Pendulum"
	SuspensionPinName          = "Suspension Pin"
	SuspensionCapName          = "Suspension Cap"
	GreatWheelName             = "Great Wheel"
	PawlCarrierName            = "Pawl Carrier"
	MainspringName             = "Mainspring"
	SpringSpacerName           = "Spring Spacer"
	SpringAnchorPinName        = "Spring Anchor Pin"
	WindingKnobName            = "Winding Knob"
)

// slot is a body field of a Movement with its display name.
type slot struct {
	name string
	dst  **decad.Body
}

// slots lists every body field of m, in the order of the Bodies table of instructions.md.
func (m *Movement) slots() []slot {
	out := []slot{
		{BackPlateName, &m.BackPlate}, {FrontPlateName, &m.FrontPlate},
		{BackEscapeBearingName, &m.BackEscapeBearing}, {BackPalletBearingName, &m.BackPalletBearing},
		{BackGreatWheelBearingName, &m.BackGreatWheelBearing}, {FrontEscapeBearingName, &m.FrontEscapeBearing},
		{FrontPalletBearingName, &m.FrontPalletBearing}, {FrontGreatWheelBearingName, &m.FrontGreatWheelBearing},
		{PendulumBearingName, &m.PendulumBearing},
		{EscapeArborName, &m.EscapeArbor}, {PalletArborName, &m.PalletArbor}, {GreatWheelArborName, &m.GreatWheelArbor},
		{EscapeWheelName, &m.EscapeWheel}, {AnchorName, &m.Anchor}, {CrutchName, &m.Crutch}, {PendulumName, &m.Pendulum},
		{SuspensionPinName, &m.SuspensionPin}, {SuspensionCapName, &m.SuspensionCap},
		{GreatWheelName, &m.GreatWheel},
		{PawlCarrierName, &m.PawlCarrier}, {MainspringName, &m.Mainspring}, {SpringSpacerName, &m.SpringSpacer},
		{SpringAnchorPinName, &m.SpringAnchorPin}, {WindingKnobName, &m.WindingKnob},
	}
	for i := range m.Bolts {
		out = append(out, slot{BoltName(i), &m.Bolts[i]})
	}
	for i := range m.Nuts {
		out = append(out, slot{NutName(i), &m.Nuts[i]})
	}
	return out
}

// Named lists every body with its display name, in the order of the Bodies table of instructions.md.
func (m *Movement) Named() []NamedBody {
	var out []NamedBody
	for _, s := range m.slots() {
		out = append(out, NamedBody{s.name, *s.dst})
	}
	return out
}

// frameTransform returns the motion from the model frame onto p's placement, or the identity.
func frameTransform(p Params) (r3.Transform, error) {
	if p.Placement == nil {
		return r3.Identity(), nil
	}
	t, err := r3.FromFrame(*p.Placement)
	if err != nil {
		return r3.Transform{}, fmt.Errorf("placement frame: %w", err)
	}
	return t, nil
}

// poser moves bodies from the model frame to their pose and onto the placement.
type poser struct {
	ctx   context.Context
	frame r3.Transform
	place bool
}

// about turns body by the math angle about the vertical line through c, then places it. A fixed body with no
// placement stays as it is.
func (ps poser) about(body *decad.Body, c Pt, angle float64) (*decad.Body, error) {
	if angle == 0 && !ps.place {
		return body, nil
	}
	turn, err := r3.RotationAround(r3.NewVec(c.X, c.Y, 0), zAxis, units.Radians(angle))
	if err != nil {
		return nil, fmt.Errorf("rotation: %w", err)
	}
	return ps.apply(body, turn)
}

// aboutOrigin turns a body built about the origin by the math angle and moves it to c, then places it.
func (ps poser) aboutOrigin(body *decad.Body, c Pt, angle float64) (*decad.Body, error) {
	turn, err := r3.Rotation(zAxis, units.Radians(angle))
	if err != nil {
		return nil, fmt.Errorf("rotation: %w", err)
	}
	move, err := r3.Translation(r3.NewVec(c.X, c.Y, 0))
	if err != nil {
		return nil, fmt.Errorf("translation: %w", err)
	}
	t, err := turn.Then(move)
	if err != nil {
		return nil, fmt.Errorf("compose pose: %w", err)
	}
	return ps.apply(body, t)
}

func (ps poser) apply(body *decad.Body, t r3.Transform) (*decad.Body, error) {
	placed, err := t.Then(ps.frame)
	if err != nil {
		return nil, fmt.Errorf("compose placement: %w", err)
	}
	out, err := decad.Transform(ps.ctx, body, placed)
	if err != nil {
		return nil, fmt.Errorf("move: %w", err)
	}
	return out, nil
}

// Build constructs the movement in doc, posed at p.Phase. Use a document made by NewDocument.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Movement, error) {
	if ctx == nil {
		return nil, errors.New("escapement: nil context")
	}
	if doc == nil {
		return nil, errors.New("escapement: nil document")
	}
	g, err := Layout(p)
	if err != nil {
		return nil, err
	}
	state, err := g.StateAt(p.Phase)
	if err != nil {
		return nil, fmt.Errorf("escapement: pose: %w", err)
	}
	frame, err := frameTransform(p)
	if err != nil {
		return nil, fmt.Errorf("escapement: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("escapement: %w", err)
	}
	b := newBuilder(ctx, doc, g)
	m := &Movement{State: state}
	if err := b.buildAll(m); err != nil {
		return nil, fmt.Errorf("escapement: %w", err)
	}
	ps := poser{ctx: ctx, frame: frame, place: p.Placement != nil}
	if err := ps.poseAll(m, &g, state); err != nil {
		return nil, fmt.Errorf("escapement: pose parts: %w", err)
	}
	if err := moveBodies(doc, m); err != nil {
		return nil, fmt.Errorf("escapement: %w", err)
	}
	m.Sketches = b.sketches
	return m, nil
}

// buildAll builds every body in its local frame.
func (b *builder) buildAll(m *Movement) error {
	var err error
	steps := []struct {
		what string
		run  func() error
	}{
		{"back plate", func() (e error) { m.BackPlate, e = b.buildBackPlate(); return }},
		{"front plate", func() (e error) { m.FrontPlate, e = b.buildFrontPlate(); return }},
		{"arbors", func() (e error) {
			if m.EscapeArbor, e = b.buildEscapeArbor(); e != nil {
				return
			}
			if m.PalletArbor, e = b.buildPalletArbor(); e != nil {
				return
			}
			m.GreatWheelArbor, e = b.buildGreatWheelArbor()
			return
		}},
		{"suspension pin", func() (e error) { m.SuspensionPin, e = b.buildSuspensionPin(); return }},
		{"escape wheel", func() (e error) { m.EscapeWheel, e = b.buildEscapeWheel(); return }},
		{"anchor", func() (e error) { m.Anchor, e = b.buildAnchor(); return }},
		{"crutch", func() (e error) { m.Crutch, e = b.buildCrutch(); return }},
		{"pendulum", func() (e error) { m.Pendulum, e = b.buildPendulum(); return }},
		{"suspension cap", func() (e error) { m.SuspensionCap, e = b.buildCap(); return }},
		{"great wheel", func() (e error) { m.GreatWheel, e = b.buildGreatWheel(); return }},
		{"pawl carrier", func() (e error) { m.PawlCarrier, e = b.buildPawlCarrier(); return }},
		{"mainspring", func() (e error) { m.Mainspring, e = b.buildMainspring(); return }},
		{"spring spacer", func() (e error) { m.SpringSpacer, e = b.buildSpringSpacer(); return }},
		{"spring anchor pin", func() (e error) { m.SpringAnchorPin, e = b.buildSpringAnchorPin(); return }},
		{"winding knob", func() (e error) { m.WindingKnob, e = b.buildKnob(); return }},
		{"hardware", func() (e error) {
			var h *Hardware
			if h, e = b.buildHardware(); e != nil {
				return
			}
			m.BackEscapeBearing, m.BackPalletBearing, m.BackGreatWheelBearing = h.BackEscape, h.BackPallet, h.BackGreat
			m.FrontEscapeBearing, m.FrontPalletBearing, m.FrontGreatWheelBearing = h.FrontEscape, h.FrontPallet, h.FrontGreat
			m.PendulumBearing, m.Bolts, m.Nuts = h.Pendulum, h.Bolts, h.Nuts
			return
		}},
	}
	for _, s := range steps {
		if err = s.run(); err != nil {
			return fmt.Errorf("%s: %w", s.what, err)
		}
	}
	return nil
}

// bodyPose is one body's pose: a turn by the math angle about the vertical line through about. A body built about the
// origin (local) is turned about the origin and then moved to about.
type bodyPose struct {
	about Pt
	angle float64
	local bool
}

// poses returns the pose of every body by name at state s; a body not listed is fixed.
func poses(g *Geometry, s State) map[string]bodyPose {
	wheel := s.WheelAngle - BuildBackoff
	great := g.GreatPhase + wheel*PinionTeeth/GreatTeeth
	arbor := great + DrumOffset + g.CarrierOffset
	return map[string]bodyPose{
		PendulumBearingName: {g.S, -s.PendulumAngle, false},
		EscapeArborName:     {g.O, -wheel, false}, PalletArborName: {g.P, -s.AnchorAngle, false},
		GreatWheelArborName: {g.G, arbor, false},
		EscapeWheelName:     {g.O, -wheel, false}, AnchorName: {g.P, -s.AnchorAngle, false},
		CrutchName: {g.P, -s.AnchorAngle, false}, PendulumName: {g.S, -s.PendulumAngle, false},
		GreatWheelName: {g.G, great, true}, PawlCarrierName: {g.G, arbor, true}, SpringSpacerName: {g.G, arbor, true},
		WindingKnobName: {g.G, arbor, true},
	}
}

// poseAll turns every moving body to the pose and places every body.
func (ps poser) poseAll(m *Movement, g *Geometry, s State) error {
	table := poses(g, s)
	for _, sl := range m.slots() {
		pose := table[sl.name]
		var out *decad.Body
		var err error
		if pose.local {
			out, err = ps.aboutOrigin(*sl.dst, pose.about, pose.angle)
		} else {
			out, err = ps.about(*sl.dst, pose.about, pose.angle)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", sl.name, err)
		}
		*sl.dst = out
	}
	return nil
}
