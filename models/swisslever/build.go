package swisslever

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

	BackEscapeBearing, BackPalletBearing, BackBalanceBearing    *decad.Body
	FrontEscapeBearing, FrontPalletBearing, FrontBalanceBearing *decad.Body

	EscapeArbor, PalletArbor, BalanceStaff *decad.Body

	EscapeWheel, Lever, EntryStone, ExitStone *decad.Body
	SafetyRoller, RollerTable, BalanceWheel   *decad.Body
	Hairspring, HairspringRetainer            *decad.Body
	WindingKnob, KnobCollar                   *decad.Body
	Bolts, Nuts                               []*decad.Body

	// Sketches holds every sketch of the movement's own parts and of the bearing, bolt, nut and spring models.
	Sketches []*sketch.Sketch
	// State is the pose the bodies were built at.
	State State
}

// NamedBody is a body with its display name.
type NamedBody struct {
	Name string
	Body *decad.Body
}

// BoltName names bolt i (0-based).
func BoltName(i int) string { return fmt.Sprintf("Bolt %d", i+1) }

// NutName names nut i (0-based).
func NutName(i int) string { return fmt.Sprintf("Nut %d", i+1) }

// Body names, as in the Bodies table of instructions.md.
const (
	BackPlateName           = "Back Plate"
	FrontPlateName          = "Front Plate"
	BackEscapeBearingName   = "Back Escape Bearing"
	BackPalletBearingName   = "Back Pallet Bearing"
	BackBalanceBearingName  = "Back Balance Bearing"
	FrontEscapeBearingName  = "Front Escape Bearing"
	FrontPalletBearingName  = "Front Pallet Bearing"
	FrontBalanceBearingName = "Front Balance Bearing"
	EscapeArborName         = "Escape Arbor"
	PalletArborName         = "Pallet Arbor"
	BalanceStaffName        = "Balance Staff"
	EscapeWheelName         = "Escape Wheel"
	LeverName               = "Lever"
	EntryStoneName          = "Entry Stone"
	ExitStoneName           = "Exit Stone"
	SafetyRollerName        = "Safety Roller"
	RollerTableName         = "Roller Table"
	BalanceWheelName        = "Balance Wheel"
	HairspringName          = "Hairspring"
	HairspringRetainerName  = "Hairspring Retainer"
	WindingKnobName         = "Winding Knob"
	KnobCollarName          = "Knob Collar"
)

// bodySlot is a body field of a Movement with its display name.
type bodySlot struct {
	name string
	dst  **decad.Body
}

// slots lists every body field of m, in the order of the Bodies table of instructions.md.
func (m *Movement) slots() []bodySlot {
	out := []bodySlot{
		{BackPlateName, &m.BackPlate}, {FrontPlateName, &m.FrontPlate},
		{BackEscapeBearingName, &m.BackEscapeBearing}, {BackPalletBearingName, &m.BackPalletBearing},
		{BackBalanceBearingName, &m.BackBalanceBearing}, {FrontEscapeBearingName, &m.FrontEscapeBearing},
		{FrontPalletBearingName, &m.FrontPalletBearing}, {FrontBalanceBearingName, &m.FrontBalanceBearing},
		{EscapeArborName, &m.EscapeArbor}, {PalletArborName, &m.PalletArbor}, {BalanceStaffName, &m.BalanceStaff},
		{EscapeWheelName, &m.EscapeWheel}, {LeverName, &m.Lever}, {EntryStoneName, &m.EntryStone},
		{ExitStoneName, &m.ExitStone}, {SafetyRollerName, &m.SafetyRoller}, {RollerTableName, &m.RollerTable},
		{BalanceWheelName, &m.BalanceWheel}, {HairspringName, &m.Hairspring},
		{HairspringRetainerName, &m.HairspringRetainer}, {WindingKnobName, &m.WindingKnob},
		{KnobCollarName, &m.KnobCollar},
	}
	for i := range m.Bolts {
		out = append(out, bodySlot{BoltName(i), &m.Bolts[i]})
	}
	for i := range m.Nuts {
		out = append(out, bodySlot{NutName(i), &m.Nuts[i]})
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

// Build constructs the movement in doc, posed at p.Phase. Use a document made by NewDocument. It validates p before it
// touches doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Movement, error) {
	if ctx == nil {
		return nil, errors.New("swisslever: nil context")
	}
	if doc == nil {
		return nil, errors.New("swisslever: nil document")
	}
	g, err := Layout(p)
	if err != nil {
		return nil, err
	}
	state, err := g.StateAt(p.Phase)
	if err != nil {
		return nil, fmt.Errorf("swisslever: pose: %w", err)
	}
	frame, err := frameTransform(p)
	if err != nil {
		return nil, fmt.Errorf("swisslever: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("swisslever: %w", err)
	}
	b := newBuilder(ctx, doc, g)
	m := &Movement{State: state}
	if err := b.buildAll(m); err != nil {
		return nil, fmt.Errorf("swisslever: %w", err)
	}
	table, err := poses(&g, state)
	if err != nil {
		return nil, fmt.Errorf("swisslever: pose: %w", err)
	}
	ps := poser{ctx: ctx, frame: frame, place: p.Placement != nil}
	if err := ps.poseAll(m, table); err != nil {
		return nil, fmt.Errorf("swisslever: pose parts: %w", err)
	}
	if err := moveBodies(doc, m); err != nil {
		return nil, fmt.Errorf("swisslever: %w", err)
	}
	m.Sketches = b.sketches
	return m, nil
}

// buildAll builds every body in its local frame.
func (b *builder) buildAll(m *Movement) error {
	steps := []struct {
		what string
		run  func() error
	}{
		{"back plate", func() (err error) { m.BackPlate, err = b.buildBackPlate(); return err }},
		{"front plate", func() (err error) { m.FrontPlate, err = b.buildFrontPlate(); return err }},
		{"escape arbor", func() (err error) { m.EscapeArbor, err = b.buildEscapeArbor(); return err }},
		{"pallet arbor", func() (err error) { m.PalletArbor, err = b.buildPalletArbor(); return err }},
		{"balance staff", func() (err error) { m.BalanceStaff, err = b.buildBalanceStaff(); return err }},
		{"escape wheel", func() (err error) { m.EscapeWheel, err = b.buildEscapeWheel(); return err }},
		{"lever", func() (err error) { m.Lever, err = b.buildLever(); return err }},
		{"stones", func() (err error) { m.EntryStone, m.ExitStone, err = b.buildStones(); return err }},
		{"safety roller", func() (err error) { m.SafetyRoller, err = b.buildSafetyRoller(); return err }},
		{"roller table", func() (err error) { m.RollerTable, err = b.buildRollerTable(); return err }},
		{"balance wheel", func() (err error) { m.BalanceWheel, err = b.buildBalanceWheel(); return err }},
		{"hairspring", func() (err error) { m.Hairspring, err = b.buildHairspring(); return err }},
		{"hairspring retainer", func() (err error) { m.HairspringRetainer, err = b.buildRetainer(); return err }},
		{"winding knob", func() (err error) { m.WindingKnob, err = b.buildKnob(); return err }},
		{"knob collar", func() (err error) { m.KnobCollar, err = b.buildKnobCollar(); return err }},
		{"hardware", func() error {
			h, err := b.buildHardware()
			if err != nil {
				return err
			}
			m.BackEscapeBearing, m.BackPalletBearing, m.BackBalanceBearing = h.BackEscape, h.BackPallet, h.BackBalance
			m.FrontEscapeBearing, m.FrontPalletBearing, m.FrontBalanceBearing = h.FrontEscape, h.FrontPallet, h.FrontBalance
			m.Bolts, m.Nuts = h.Bolts, h.Nuts
			return nil
		}},
	}
	for _, s := range steps {
		if err := s.run(); err != nil {
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

// buildPoseAngles returns the lever angle ψ_b and the wheel angle φ_b the build pose uses: the lever is backed off its
// last contact by BuildBackoff, and the wheel stands at the wheel angle of that lever angle.
func buildPoseAngles(g *Geometry, s State) (float64, float64, error) {
	sign := 1.0
	if s.Phase > 0.5 {
		sign = -1
	}
	psi := s.LeverAngle
	switch s.LeverStage {
	case LeverBanking:
		// Back off toward the centre line.
		if psi > 0 {
			psi -= BuildBackoff
		} else {
			psi += BuildBackoff
		}
	case LeverUnlock:
		psi += BuildBackoff * sign
	case LeverImpulse:
		psi -= BuildBackoff * sign
	default:
		return 0, 0, fmt.Errorf("unknown lever stage %q", s.LeverStage)
	}
	phi, _, err := g.WheelAt(psi, s.Phase <= 0.5)
	if err != nil {
		return 0, 0, fmt.Errorf("wheel angle at the build pose: %w", err)
	}
	return psi, phi, nil
}

// poses returns the pose of every body by name at state s; a body not listed is fixed.
func poses(g *Geometry, s State) (map[string]bodyPose, error) {
	psi, phi, err := buildPoseAngles(g, s)
	if err != nil {
		return nil, err
	}
	wheel := -(phi - BuildBackoff)
	lever := -psi
	balance := g.GammaF + s.BalanceAngle
	return map[string]bodyPose{
		EscapeArborName: {g.O, wheel, false}, EscapeWheelName: {g.O, wheel, false},
		WindingKnobName: {g.O, wheel, true}, KnobCollarName: {g.O, wheel, true},
		PalletArborName: {g.P, lever, false}, LeverName: {g.P, lever, false},
		EntryStoneName: {g.P, lever, false}, ExitStoneName: {g.P, lever, false},
		BalanceStaffName: {g.B, balance, false},
		SafetyRollerName: {g.B, balance, true}, RollerTableName: {g.B, balance, true},
		BalanceWheelName: {g.B, balance, true}, HairspringRetainerName: {g.B, balance, true},
	}, nil
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

// poseAll turns every moving body to its pose and places every body.
func (ps poser) poseAll(m *Movement, table map[string]bodyPose) error {
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
