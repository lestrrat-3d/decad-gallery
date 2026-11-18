package clamparm

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// Joint constants, in mm.
const (
	Section         = 20.0 // square section of swivel, links and head
	EndRadius       = 10.0 // round end about each joint axis
	TongueThickness = 8.0  // thickness of a tongue
	JointFit        = 0.26 // tongue-in-fork clearance per side
	ForkGap         = 8.52 // TongueThickness + 2 JointFit
	FinThickness    = 5.74 // (Section - ForkGap) / 2
	TongueShoulder  = 20.0 // joint axis to the tongue's shoulder face
	ShoulderRound   = 4.0  // shoulder inside corners
	SlotDepth       = 26.0 // joint axis to the fork slot's deepest point
	SlotRound       = 4.26 // ForkGap / 2
	FrameWidth      = 30.0 // clamp frame width along Y
	PanAxisX        = 30.0 // J1 axis x on the frame
	ScrewX          = 25.0 // clamp screw axis x on the frame
	SwivelLength    = 70.0 // J1 axis to J2 axis
	StudOffset      = 40.0 // J4 axis to the stud axis
	HeadLength      = 55.0 // J4 axis to the head's free end
	KnobHeight      = 12.0 // height of a joint knob
	ScrewKnobHeight = 10.0 // height of a screw knob
	PocketRound     = 0.6  // round on a hex pocket floor's outline
	MarkRadius      = 8.0  // radius the part-mark holes sit on
)

// Range constants, in degrees and mm.
const (
	J1Max         = 125.0  // J1 pan limit
	JointMax      = 115.0  // J2 to J4 limit
	ShoulderMin   = -60.0  // J2 lower limit
	WristMin      = -110.0 // J4 lower limit
	WristMax      = 90.0   // J4 upper limit
	OpeningMin    = 10.0   // thinnest desk
	OpeningMax    = 40.0   // thickest desk
	LinkLengthMin = 80.0   // shortest link
	LinkLengthMax = 150.0  // longest link
)

// Interface constants of the mount interface, in mm.
const (
	BossBaseRadius   = 6.0   // cone boss base radius
	BossHeight       = 2.0   // cone boss height
	RecessRadius     = 6.495 // 6 + 0.35 sqrt(2): the recess radius at the mating face
	RecessClearance  = 0.35  // normal clearance of the cone boss in the recess
	StudProtrusion   = 13.5  // stud tip over the interface face
	AdapterNutMin    = 4.7   // lowest height of an adapter nut's near face
	AdapterNutMax    = 8.3   // highest height of an adapter nut's near face
	AdapterStudClear = 14.5  // hole depth the adapter gives the stud
)

// Container names, in the order the model files its bodies.
const (
	FrameContainerName      = "Frame"
	PadContainerName        = "Pad"
	SwivelContainerName     = "Swivel"
	LinksContainerName      = "Links"
	HeadContainerName       = "Head"
	AdapterContainerName    = "Adapter"
	KnobsContainerName      = "Knobs"
	ScrewKnobsContainerName = "Screw Knobs"
	BoltsContainerName      = "Bolts"
	NutsContainerName       = "Nuts"
)

// Unexported dimensions, in mm.
const (
	swivelRise         = 14.26  // J1's swivel bottom face over the desk top
	jawTop             = -59.5  // the bottom jaw's top face
	finTop             = 28.26  // the upper fin's top face
	gapLow             = 14.0   // the J1 gap's lower face
	gapHigh            = 22.52  // the J1 gap's upper face
	gapEnd             = 46.0   // the upper fin's end
	jawEnd             = 50.0   // the top jaw's end
	swivelTongueLength = 30.0   // the swivel tongue's end x from the J1 axis (steps.md 7.5)
	padSocketRadius    = 2.85   // pad socket radius
	padSocketDepth     = 8.0    // pad socket depth
	padSocketRound     = 0.5    // pad socket floor round
	jawNutFloor        = -64.6  // the jaw nut pocket's floor
	jawNutDepth        = 5.1    // the jaw nut pocket's depth
	knobNutFloor       = 3.0    // the knob nut pocket's floor
	knobNutDepth       = 9.0    // the knob nut pocket's depth
	pressFloor         = 6.5    // the screw knob head pocket's floor
	markDiameter       = 1.5    // part-mark hole diameter
	markFirst          = 30.0   // first part-mark angle
	markStep           = 25.0   // part-mark angle step
	lobeRadius         = 4.0    // knob lobe radius
	lobeCentre         = 11.0   // knob lobe centre radius
	valleyRadius       = 3.0    // knob valley radius
	valleyCentre       = 13.857 // knob valley centre radius
	jointBolt          = 25.0   // M5 x 25 at J2 to J4
	panBolt            = 35.0   // M5 x 35 at J1
	studBolt           = 40.0   // M5 x 40 stud
	screwBolt          = 70.0   // M5 x 70 clamp screw
)

// Params are the clamp arm's inputs.
type Params struct {
	// LinkLength is the axis-to-axis length of each link, 80 to 150 mm.
	LinkLength float64
	// Opening is the desk thickness the pad is posed at, 10 to 40 mm.
	Opening float64
	// Pan is J1, the swivel's rotation about +Z, -J1Max to J1Max degrees.
	Pan float64
	// Shoulder is J2, link A from the swivel, ShoulderMin to JointMax degrees.
	Shoulder float64
	// Elbow is J3, link B from link A, -JointMax to JointMax degrees.
	Elbow float64
	// Wrist is J4, the head from link B, WristMin to WristMax degrees.
	Wrist float64
	// Placement maps the world frame. Nil selects the local frame.
	Placement *r3.Frame
}

// DefaultParams returns the default pose: the head level, 70 - 90 + 20 = 0.
func DefaultParams() Params {
	return Params{LinkLength: 120, Opening: 25, Pan: 0, Shoulder: 70, Elbow: -90, Wrist: 20}
}

// Dimensions are the derived poses, sites and frames, in the model's world frame before Placement. Deriving them
// calls no geometry.
type Dimensions struct {
	// Swivel, LinkA, LinkB, Head and Interface are the pose frames of the five moving parts.
	Swivel, LinkA, LinkB, Head, Interface r3.Frame
	// Joints are the joint axes' points: J1 on the swivel's bottom face, J2 to J4 in each part's mid-plane.
	Joints [4]r3.Vec
	// Axes are the unit joint axes: +Z for J1, the knob side for J2 to J4.
	Axes [4]r3.Vec
	// PanSite is the J1 bolt, JointSites the J2 to J4 bolts, StudSite the stud and ScrewSite the clamp screw.
	PanSite    m5fastener.Site
	JointSites [3]m5fastener.Site
	StudSite   m5fastener.Site
	ScrewSite  m5fastener.Site
	// KnobFrames are the J1 to J4 knobs' inner-face frames and ScrewKnobFrames the clamp screw's and the stud's. Each
	// U equals its site's Corner.
	KnobFrames      [4]r3.Frame
	ScrewKnobFrames [2]r3.Frame
	// KnobNutFrames are the nuts' lower-face frames in the J1 to J4 knobs.
	KnobNutFrames [4]r3.Frame
	// JawNutFrame and AdapterNutFrame are the nuts' lower-face frames in the bottom jaw and the adapter.
	JawNutFrame, AdapterNutFrame r3.Frame
	// PadFrame is the pad's local frame in the world.
	PadFrame r3.Frame
	// ScrewKnobGap is the screw knob's inner face to the bottom jaw's underside at this Opening.
	ScrewKnobGap float64
}

// Validate returns the error Derive returns.
func (p Params) Validate() error {
	_, err := p.Derive()
	return err
}

// Derive checks the parameters in order and returns the first failure, then derives the dimensions.
func (p Params) Derive() (Dimensions, error) {
	if err := p.check(); err != nil {
		return Dimensions{}, err
	}
	return p.derive()
}

func (p Params) check() error {
	floats := []struct {
		name  string
		value float64
	}{
		{"LinkLength", p.LinkLength}, {"Opening", p.Opening}, {"Pan", p.Pan},
		{"Shoulder", p.Shoulder}, {"Elbow", p.Elbow}, {"Wrist", p.Wrist},
	}
	for _, f := range floats {
		if math.IsNaN(f.value) || math.IsInf(f.value, 0) {
			return fmt.Errorf("clamparm: %s must be finite", f.name)
		}
	}
	if p.LinkLength < LinkLengthMin || p.LinkLength > LinkLengthMax {
		return fmt.Errorf("clamparm: LinkLength must be from %g to %g mm, got %g", LinkLengthMin, LinkLengthMax,
			p.LinkLength)
	}
	if p.Opening < OpeningMin || p.Opening > OpeningMax {
		return fmt.Errorf("clamparm: Opening must be from %g to %g mm, got %g", OpeningMin, OpeningMax, p.Opening)
	}
	angles := []struct {
		name         string
		value        float64
		minimum, max float64
	}{
		{"Pan", p.Pan, -J1Max, J1Max},
		{"Shoulder", p.Shoulder, ShoulderMin, JointMax},
		{"Elbow", p.Elbow, -JointMax, JointMax},
		{"Wrist", p.Wrist, WristMin, WristMax},
	}
	for _, a := range angles {
		if a.value < a.minimum || a.value > a.max {
			return fmt.Errorf("clamparm: %s must be from %g to %g degrees, got %g", a.name, a.minimum, a.max, a.value)
		}
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("clamparm: Placement is not a valid frame")
	}
	return nil
}

// builder records the first error of a run of transform and frame constructions.
type builder struct{ err error }

func (b *builder) fail(err error) {
	if b.err == nil {
		b.err = err
	}
}

func (b *builder) translate(x, y, z float64) r3.Transform {
	t, err := r3.Translation(r3.NewVec(x, y, z))
	b.fail(err)
	return t
}

// rotateZ is Rz(a), a rotation about +Z.
func (b *builder) rotateZ(a float64) r3.Transform {
	t, err := r3.Rotation(r3.NewVec(0, 0, 1), units.Degrees(a))
	b.fail(err)
	return t
}

// rotateJoint is R(a), a rotation about -Y that lifts local +X toward +Z.
func (b *builder) rotateJoint(a float64) r3.Transform {
	t, err := r3.Rotation(r3.NewVec(0, -1, 0), units.Degrees(a))
	b.fail(err)
	return t
}

// compose is the product ts[0] . ... . ts[n-1], which applies the last transform first.
func (b *builder) compose(ts ...r3.Transform) r3.Transform {
	acc := ts[len(ts)-1]
	for i := len(ts) - 2; i >= 0; i-- {
		next, err := acc.Then(ts[i])
		b.fail(err)
		acc = next
	}
	return acc
}

func (b *builder) frame(origin, u, v r3.Vec) r3.Frame {
	f, err := r3.NewFrame(origin, u, v)
	b.fail(err)
	return f
}

// frameOf is the frame t gives the local origin, X and Y axes.
func (b *builder) frameOf(t r3.Transform) r3.Frame {
	return b.frame(t.Apply(r3.Vec{}), t.ApplyDir(r3.NewVec(1, 0, 0)), t.ApplyDir(r3.NewVec(0, 1, 0)))
}

// across is site.Across(t).
func (b *builder) across(s m5fastener.Site, t float64) r3.Frame {
	f, err := s.Across(t)
	b.fail(err)
	return f
}

// moveFrame applies t to f.
func (b *builder) moveFrame(t r3.Transform, f r3.Frame) r3.Frame {
	return b.frame(t.Apply(f.Origin()), t.ApplyDir(f.U()), t.ApplyDir(f.V()))
}

// moveSite applies t to s.
func moveSite(t r3.Transform, s m5fastener.Site) m5fastener.Site {
	return m5fastener.Site{Head: t.Apply(s.Head), Dir: t.ApplyDir(s.Dir), Corner: t.ApplyDir(s.Corner),
		Length: s.Length}
}

// forkSite is the bolt of a fork at x = lf whose head pocket is in the -Y fin, at height z.
func forkSite(lf, z float64) m5fastener.Site {
	return m5fastener.Site{Head: r3.NewVec(lf, -10, z), Dir: r3.NewVec(0, 1, 0), Corner: r3.NewVec(0, 0, 1),
		Length: jointBolt}
}

// tongueSite is the bolt hole of a tongue at the origin, at height z.
func tongueSite(z float64) m5fastener.Site { return forkSite(0, z) }

// studLocal is the stud's site in the head's frame.
var studLocal = m5fastener.Site{Head: r3.NewVec(40, 0, -20), Dir: r3.NewVec(0, 0, 1), Corner: r3.NewVec(1, 0, 0),
	Length: studBolt}

// panLocal is the swivel's J1 hole in the swivel's own frame.
var panLocal = m5fastener.Site{Head: r3.NewVec(0, 0, -swivelRise), Dir: r3.NewVec(0, 0, 1),
	Corner: r3.NewVec(1, 0, 0), Length: panBolt}

// pocketFrame is the head pocket's floor frame for site: its origin is the floor's centre and its normal is -Dir,
// toward the opening.
func pocketFrame(site m5fastener.Site) (r3.Frame, error) {
	return r3.NewFrame(site.At(m5fastener.HeadPocketDepth), site.Corner, site.Dir.Scale(-1).Cross(site.Corner))
}

func (p Params) derive() (Dimensions, error) {
	b := &builder{}
	l, o := p.LinkLength, p.Opening
	s := b.compose(b.translate(PanAxisX, 0, swivelRise), b.rotateZ(p.Pan))
	a := b.compose(s, b.translate(SwivelLength, 0, 10), b.rotateJoint(p.Shoulder))
	bt := b.compose(a, b.translate(l, 0, 0), b.rotateJoint(p.Elbow))
	h := b.compose(bt, b.translate(l, 0, 0), b.rotateJoint(p.Wrist))
	i := b.compose(h, b.translate(StudOffset, 0, 10))
	if b.err != nil {
		return Dimensions{}, fmt.Errorf("clamparm: pose chain: %w", b.err)
	}

	var d Dimensions
	d.Swivel, d.LinkA, d.LinkB, d.Head, d.Interface = b.frameOf(s), b.frameOf(a), b.frameOf(bt), b.frameOf(h),
		b.frameOf(i)
	y := r3.NewVec(0, 1, 0)
	d.Joints = [4]r3.Vec{r3.NewVec(PanAxisX, 0, swivelRise), a.Apply(r3.Vec{}), bt.Apply(r3.Vec{}),
		h.Apply(r3.Vec{})}
	d.Axes = [4]r3.Vec{r3.NewVec(0, 0, 1), s.ApplyDir(y), a.ApplyDir(y), bt.ApplyDir(y)}

	d.PanSite = m5fastener.Site{Head: r3.NewVec(PanAxisX, 0, 0), Dir: r3.NewVec(0, 0, 1), Corner: y, Length: panBolt}
	d.JointSites = [3]m5fastener.Site{
		moveSite(s, forkSite(SwivelLength, 10)), moveSite(a, forkSite(l, 0)), moveSite(bt, forkSite(l, 0)),
	}
	d.StudSite = moveSite(h, studLocal)
	d.ScrewSite = m5fastener.Site{Head: r3.NewVec(ScrewX, 0, -o-77.5), Dir: r3.NewVec(0, 0, 1), Corner: y,
		Length: screwBolt}
	named := []struct {
		name string
		site m5fastener.Site
	}{
		{"PanSite", d.PanSite}, {"JointSites[0]", d.JointSites[0]}, {"JointSites[1]", d.JointSites[1]},
		{"JointSites[2]", d.JointSites[2]}, {"StudSite", d.StudSite}, {"ScrewSite", d.ScrewSite},
	}
	for _, n := range named {
		if err := n.site.Validate(); err != nil {
			return Dimensions{}, fmt.Errorf("clamparm: %s: %w", n.name, err)
		}
	}

	d.KnobFrames[0] = b.across(d.PanSite, 28.26)
	d.KnobNutFrames[0] = b.across(d.PanSite, 31.26)
	for k, site := range d.JointSites {
		d.KnobFrames[1+k] = b.across(site, 20)
		d.KnobNutFrames[1+k] = b.across(site, 23)
	}
	d.ScrewKnobFrames[0] = b.frame(r3.NewVec(ScrewX, 0, -o-67.5), y, r3.NewVec(1, 0, 0))
	d.ScrewKnobFrames[1] = b.moveFrame(h, b.frame(r3.NewVec(40, 0, -10), r3.NewVec(1, 0, 0), r3.NewVec(0, -1, 0)))
	d.JawNutFrame = b.frame(r3.NewVec(ScrewX, 0, jawNutFloor), y, r3.NewVec(-1, 0, 0))
	d.AdapterNutFrame = b.frame(d.Interface.Origin().Add(d.Interface.N().Scale(AdapterNutMin)), d.Interface.U(),
		d.Interface.V())
	d.PadFrame = b.frame(r3.NewVec(ScrewX, 0, -o), r3.NewVec(1, 0, 0), y)
	d.ScrewKnobGap = o - 6
	if b.err != nil {
		return Dimensions{}, fmt.Errorf("clamparm: frames: %w", b.err)
	}
	return d, nil
}
