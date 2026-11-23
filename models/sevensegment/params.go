package sevensegment

import (
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/r3"
)

// Scale is the size of the display against its base design, which Scale 1.0 is. Every shape (layout, rotors, sliders, cams,
// frame, pivot) is written as Scale times its base size in mm. Fits, clearances, print limits, layer-bound thicknesses and
// purchased hardware keep their size in mm and are added outside the product (see instructions.md "Scale").
const Scale = 1.5

// Fixed dimensions in mm and degrees. See instructions.md for what each one is.
const (
	SegmentSpan   = Scale * 33.0
	Travel        = Scale * 3.0
	FaceZ         = flagFaceZ
	ShaftZ        = Scale * 29.5
	StationPitch  = Scale * 11.0
	CamRadius     = Scale * 23.5
	CamThickness  = Scale * 3.8
	HubRadius     = Scale * 5.0
	OffRadius     = Scale * 16.5
	OnRadius      = Scale * 19.5
	GrooveWidth   = 2*sliderPinR + 0.8 // the groove pin's diameter plus a 0.8 fit
	GrooveDepth   = Scale * 1.9
	RampStart     = 8.0 // degrees
	RampEnd       = 29.5
	RampRound     = Scale * 2.5
	EntryAngle    = 18.0
	PinRadius     = Scale * 1.5
	PinLength     = Scale*1.9 + 0.15 // the groove depth plus a 0.15 end gap
	StemThickness = Scale * 3.6
	StemRound     = Scale * 0.3

	WallThickness = Scale * 8.0
	WallHalfWidth = Scale * 23.4 // plates x +-23.4; the b, c, e, f bores (x +-22.046) keep 0.85 inside the 0.5 chamfer

	Seam            = 0.25
	PressFit        = 0.05
	TightFit        = 0.25
	RegularFit      = 0.40
	LooseFit        = 0.55
	TimingAllowance = 1.5

	// The flag mechanism: each segment is a two-colour rotor on two printed pivot blocks, turned by its slider.
	RotorAxisZ      = cradleFrom + cradleThick // modeled height of every rotor axis: the seam plane
	RotorHalfWidth  = Scale * 3.75             // distance of a rotor face from its axis
	RotorHalfLength = Scale * 12.25
	JournalRadius   = pivotPinRadius + 0.2 // the rotor's end bore, its turning surface; the pivot pin turns in it with 0.2 to spare
	PinOffset       = Scale * 1.5          // distance of the slider pin line from the rotor axis, along the rotor's x

	RearCollarRadius = Scale * 3.8
)

// The shaft bearings are two 608ZZ (ballbearing.Lookup("608ZZ"): bore 8, outside 22, width 7, inner ring face 12). The
// constants are pinned to the catalog by a test; a catalog has no bore chamfer, and the 0.3 that the shaft's steps clear is an
// assumption (instructions.md "Bearings").
const (
	bearingBoreR       = 4.0   // half the bore: the shaft's journals
	bearingWidth       = 7.0   //
	bearingSeatR       = 11.05 // half the outside diameter plus the 0.05 seat fit
	bearingRaceR       = 6.0   // half the inner ring's face: the least a collar facing the bearing may reach
	bearingBoreChamfer = 0.3   // assumed, not in the catalog
)

// frameBoltX is |x| of the T and G bolts: head pocket HeadPocketRadius 2.2, 1.1 of material and the plates' 0.5 chamfer
// inside WallHalfWidth. The hardware does not scale, so the offset does not.
const frameBoltX = WallHalfWidth - 3.8 // 19.6

// Segment names one of the seven segments.
type Segment int

// The segments in the usual order.
const (
	SegA Segment = iota
	SegB
	SegC
	SegD
	SegE
	SegF
	SegG
)

// String returns "a" to "g".
func (s Segment) String() string {
	if s < SegA || s > SegG {
		return fmt.Sprintf("Segment(%d)", int(s))
	}
	return string(rune('a' + int(s)))
}

// Segments returns the segments a to g.
func Segments() [7]Segment { return [7]Segment{SegA, SegB, SegC, SegD, SegE, SegF, SegG} }

// truth lists the lit segments a..g of each digit. The box stands on its back wall with the button end up, so the model's
// segment a, at +Y, is the bottom of the digit as the viewer reads it: the roles of a and d, b and e, and c and f are
// swapped against the usual order, and each digit reads upright.
var truth = [10][7]bool{
	{true, true, true, true, true, true, false},
	{false, false, false, false, true, true, false},
	{true, true, false, true, true, false, true},
	{true, false, false, true, true, true, true},
	{false, false, true, false, true, true, true},
	{true, false, true, true, false, true, true},
	{true, true, true, true, false, true, true},
	{false, false, false, true, true, true, false},
	{true, true, true, true, true, true, true},
	{true, false, true, true, true, true, true},
}

// Lit reports whether segment s is raised for digit.
func Lit(digit int, s Segment) (bool, error) {
	if digit < 0 || digit > 9 {
		return false, fmt.Errorf("sevensegment: digit %d out of range 0..9", digit)
	}
	if s < SegA || s > SegG {
		return false, fmt.Errorf("sevensegment: segment %d out of range", int(s))
	}
	return truth[digit][s], nil
}

// stationOrder lists the segments by station index i: d, e, c, g, b, f, a.
var stationOrder = [7]Segment{SegD, SegE, SegC, SegG, SegB, SegF, SegA}

// Params are the model's inputs.
type Params struct {
	// Digit is the digit shown, 0..9. It sets the camshaft angle 36*Digit and every slider's height.
	Digit int
	// Placement places every body; nil keeps the world frame of instructions.md.
	Placement *r3.Frame
}

// DefaultParams shows digit 2 in the world frame.
func DefaultParams() Params { return Params{Digit: 2} }

// Validate checks the inputs, first failure wins.
func (p Params) Validate() error {
	if p.Digit < 0 || p.Digit > 9 {
		return fmt.Errorf("sevensegment: digit %d out of range 0..9", p.Digit)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return errors.New("sevensegment: invalid placement frame")
	}
	return nil
}

// Ramp is one cam ramp: the step From to (From+1) mod 10 changes the segment.
type Ramp struct {
	From   int
	Rising bool
	Start  float64 // cam-local degrees, psi_From + RampStart, not wrapped
	End    float64
}

// RampGeometry is a ramp's centreline: two arcs of RampRound about C1 and C2 and the line T1 to T2, cam-local, Z = 0.
type RampGeometry struct {
	C1, C2, T1, T2 r3.Vec
	LineLength     float64
}

func polar(radius, deg float64) r3.Vec {
	s, c := math.Sincos(deg * math.Pi / 180)
	return r3.Vec{X: radius * c, Y: radius * s}
}

func wrap360(a float64) float64 { return a - 360*math.Floor(a/360) }

// risingGeometry solves the rising ramp from polar angle a to b (degrees): arc 1 about OffRadius+RampRound at a, arc 2
// about OnRadius-RampRound at b, joined by their internal common tangent.
func risingGeometry(a, b float64) (RampGeometry, float64, float64) {
	c1 := polar(OffRadius+RampRound, a)
	c2 := polar(OnRadius-RampRound, b)
	dv := c2.Sub(c1)
	base := math.Atan2(dv.Y, dv.X) * 180 / math.Pi
	off := math.Acos(2*RampRound/math.Hypot(dv.X, dv.Y)) * 180 / math.Pi
	for _, phi1 := range []float64{base + off, base - off} {
		n := polar(1, phi1)
		d := r3.Vec{X: n.Y, Y: -n.X}
		if dv.Dot(d) <= 0 {
			continue
		}
		sweep1 := wrap360((a + 180) - phi1)
		sweep2 := wrap360(b - (phi1 + 180))
		if sweep1 <= 0 || sweep1 > 180 || sweep2 <= 0 || sweep2 > 180 {
			continue
		}
		t1 := c1.Add(n.Scale(RampRound))
		t2 := c2.Sub(n.Scale(RampRound))
		return RampGeometry{C1: c1, C2: c2, T1: t1, T2: t2, LineLength: t2.Sub(t1).Len()}, phi1, sweep1
	}
	return RampGeometry{}, 0, 0
}

// reflect mirrors v about the line through the origin at polar angle m (degrees).
func reflect(v r3.Vec, m float64) r3.Vec {
	s, c := math.Sincos(2 * m * math.Pi / 180)
	return r3.Vec{X: v.X*c + v.Y*s, Y: v.X*s - v.Y*c}
}

// Geometry returns the ramp's centreline arcs and line, cam-local.
func (r Ramp) Geometry() RampGeometry {
	g, _, _ := risingGeometry(r.Start, r.End)
	if r.Rising {
		return g
	}
	m := (r.Start + r.End) / 2
	return RampGeometry{C1: reflect(g.C2, m), C2: reflect(g.C1, m), T1: reflect(g.T2, m), T2: reflect(g.T1, m),
		LineLength: g.LineLength}
}

// risingPath is a rising ramp's centreline as a function of a parameter in [0, 3): arc 1, the line, arc 2.
type risingPath struct {
	g              RampGeometry
	a0, phi1       float64 // arc 1 starts at angle a0 about C1 and runs clockwise to phi1
	sweep1, sweep2 float64
	start, end     float64
}

func newRisingPath(a, b float64) risingPath {
	g, phi1, sweep1 := risingGeometry(a, b)
	return risingPath{g: g, a0: a + 180, phi1: phi1, sweep1: sweep1, sweep2: wrap360(b - (phi1 + 180)), start: a, end: b}
}

// at returns the centreline point and its unit tangent at parameter t in [0, 3].
func (p risingPath) at(t float64) (r3.Vec, r3.Vec) {
	switch {
	case t < 1:
		al := (p.a0 - p.sweep1*t) * math.Pi / 180
		s, c := math.Sincos(al)
		return p.g.C1.Add(r3.Vec{X: RampRound * c, Y: RampRound * s}), r3.Vec{X: s, Y: -c}
	case t < 2:
		d := p.g.T2.Sub(p.g.T1)
		return p.g.T1.Add(d.Scale(t - 1)), d.Scale(1 / d.Len())
	default:
		al := (p.phi1 + 180 + p.sweep2*(t-2)) * math.Pi / 180
		s, c := math.Sincos(al)
		return p.g.C2.Add(r3.Vec{X: RampRound * c, Y: RampRound * s}), r3.Vec{X: -s, Y: c}
	}
}

// find returns the parameter whose point lies at polar angle psi (degrees from p.start, in [0, end-start]).
func (p risingPath) find(psi float64) float64 {
	lo, hi := 0.0, 3.0
	for range 60 {
		mid := (lo + hi) / 2
		pt, _ := p.at(mid)
		if wrap360(math.Atan2(pt.Y, pt.X)*180/math.Pi-p.start+180)-180 < psi {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2
}

// radius returns the centreline radius and pressure angle (degrees) at polar angle psi, a rising ramp.
func (p risingPath) radius(psi float64) (float64, float64) {
	t := p.find(psi - p.start)
	pt, tan := p.at(t)
	r := pt.Len()
	er := r3.Vec{X: pt.X / r, Y: pt.Y / r}
	radial := math.Abs(tan.Dot(er))
	circ := math.Abs(tan.X*(-er.Y) + tan.Y*er.X)
	return r, math.Atan2(radial, circ) * 180 / math.Pi
}

// Dimensions are the values Derive computes.
type Dimensions struct {
	Digit int
	Theta float64 // degrees
	// Ramps lists each segment's ramps in increasing Start.
	Ramps [7][]Ramp
	// Sites are T1..T4, G1..G4, H1, H2 in the world frame, before Placement.
	Sites [10]m2fastener.Site
}

// psiOf returns the cam-local angle of digit k: 90 + 36k.
func psiOf(k int) float64 { return 90 + 36*float64(k) }

// Derive computes the ramps, the pawl seat and the fastener sites. It validates p first.
func (p Params) Derive() (Dimensions, error) {
	if err := p.Validate(); err != nil {
		return Dimensions{}, err
	}
	d := Dimensions{Digit: p.Digit, Theta: 36 * float64(p.Digit)}
	for _, s := range Segments() {
		for k := range 10 {
			a, _ := Lit(k, s)
			b, _ := Lit((k+1)%10, s)
			if a == b {
				continue
			}
			start := psiOf(k) + RampStart
			d.Ramps[s] = append(d.Ramps[s], Ramp{From: k, Rising: b, Start: start, End: start + RampEnd - RampStart})
		}
	}
	var err2 error
	if d.Sites, err2 = sites(); err2 != nil {
		return Dimensions{}, err2
	}
	return d, nil
}

// sites returns T1..T4, G1..G4, H1, H2.
func sites() ([10]m2fastener.Site, error) {
	var out [10]m2fastener.Site
	// G3/G4 at y 32 stand between the discs of cams f and a, so no cam lies under their heads.
	gy := [4]float64{Scale * -34.2, Scale * -34.2, Scale * 32.0, Scale * 32.0}
	xs := [4]float64{-frameBoltX, frameBoltX, -frameBoltX, frameBoltX}
	ts := flagSites()
	copy(out[:4], ts[:])
	for i := range 4 {
		out[4+i] = m2fastener.Site{Head: r3.Vec{X: xs[i], Y: gy[i], Z: ledgeFrom}, Dir: r3.Vec{Z: 1}, Corner: r3.Vec{Y: 1},
			Length: 10, Cavity: guideTop - ledgeFrom - m2fastener.NutCavityHeight}
	}
	// H1, H2 hold the click indexer's housing on the front wall: the head sinks in the housing's base, the nut sits in a hex
	// pocket open to the wall's inner face.
	for i, x := range [2]float64{-21, 21} { // the click indexer keeps its size, so its bolts do
		out[8+i] = m2fastener.Site{Head: r3.Vec{X: x, Y: housingBaseY - housingBaseThick, Z: ShaftZ}, Dir: r3.Vec{Y: 1},
			Corner: r3.Vec{Z: 1}, Length: 16, Cavity: frontInnerY - (housingBaseY - housingBaseThick) - m2fastener.NutCavityHeight}
	}
	for i, s := range out {
		if err := s.Validate(); err != nil {
			return out, fmt.Errorf("sevensegment: site %d: %w", i+1, err)
		}
	}
	return out, nil
}

// Station returns the Y of segment s's slider centre plane.
func (d Dimensions) Station(s Segment) float64 {
	for i, o := range stationOrder {
		if o == s {
			return -SegmentSpan + StationPitch*float64(i)
		}
	}
	return math.NaN()
}

// GrooveRadius returns the centreline radius of segment s's groove at cam-local angle psi (degrees).
func (d Dimensions) GrooveRadius(s Segment, psi float64) float64 {
	r, _ := d.groove(s, psi)
	return r
}

// PressureAngle returns the angle in degrees between the centreline's tangent and the circumferential direction.
func (d Dimensions) PressureAngle(s Segment, psi float64) float64 {
	_, a := d.groove(s, psi)
	return a
}

// groove returns the radius and pressure angle of segment s at psi.
func (d Dimensions) groove(s Segment, psi float64) (float64, float64) {
	u := wrap360(psi - 90)
	shifted := u + 6.5
	k := int(math.Floor(shifted/36)) % 10
	rel := shifted - 36*math.Floor(shifted/36)
	on, _ := Lit(k, s)
	level := OffRadius
	if on {
		level = OnRadius
	}
	if rel < RampStart+6.5 {
		return level, 0
	}
	next, _ := Lit((k+1)%10, s)
	if next == on {
		return level, 0
	}
	// psi inside the window of step k: psi_k + RampStart .. psi_k + RampEnd.
	a := 90 + 36*float64(k) + RampStart
	b := 90 + 36*float64(k) + RampEnd
	local := a + rel - (RampStart + 6.5)
	path := newRisingPath(a, b)
	if next {
		return path.radius(local)
	}
	return path.radius(a + b - local)
}

// Lift returns segment s's lift at camshaft angle theta (degrees): GrooveRadius(s, 90+theta) - OffRadius.
func (d Dimensions) Lift(s Segment, theta float64) float64 {
	return d.GrooveRadius(s, 90+theta) - OffRadius
}

// journalSettle is the side play of the rotor on its pivot pins, settled: the clearance (bore less pin) times
// sin(atan 0.5), the slope at which the settled rotor rests. journalExtra is the extra when it sits fully across.
const (
	journalSettle = (JournalRadius - pivotPinRadius) * 0.4472135955
	journalExtra  = (JournalRadius - pivotPinRadius) - journalSettle
)

// PinLinePlay is the play of a rotor's pin line along the rotor's x (+-), the sum of its sources: the blade across its hole
// (bladeAcross), the top plate to the cradle (pegs 0.05), the yaw of the stem in its slot (0.08) and the rotor journal
// settled on its pins (journalSettle). The tests take the rest bands, the drive audit and the tongue sweep from it.
const PinLinePlay = bladeAcross + 0.05 + 0.08 + journalSettle

// JournalAcross is the extra side play when the journal sits fully across its hole instead of settled.
const JournalAcross = journalExtra

// RaiseBelowDroop is how far a b-f slider's built-in pin height sits below the droop it settles to (the unlit pose is lowered
// by this much, which moves the unlit band end of b to f away from lit).
const RaiseBelowDroop = 0.05

// raise returns the unlit droop that is built into segment s's pin height: 0 for a, g, d, 0.20 for b, c and 0.25 for e, f.
// A b-f slider droops 0.254 (b, c) and 0.305 (e, f) at its pin about its groove pin (the blade keeps 0.25 across its hole and
// the stem 0.25 in its slot: tilt 0.5 / (ridge - 54)); the model's pin sits 0.05 below that droop.
func raise(s Segment) float64 {
	switch s {
	case SegB, SegC:
		return 0.20
	case SegE, SegF:
		return 0.25
	}
	return 0
}

// FlagAngle returns segment s's modeled rotor angle in degrees at camshaft angle theta: atan2(Lift(s, theta) - 1.75 +
// raise, PinOffset). The pin line is 1.5 from the rotor axis and the slider's lift is 0 to 3.
func (d Dimensions) FlagAngle(s Segment, theta float64) float64 {
	return math.Atan2(d.Lift(s, theta)-Scale*1.75+raise(s), PinOffset) * 180 / math.Pi
}
