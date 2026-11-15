package escapement

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Arbor dimensions in mm: the shaft is the 605ZZ bore, the collar is 7.4 across, the rear peg's flat is 0.5 deep on +X.
const (
	ArborRadius  = 2.5
	CollarRadius = 3.7
	FlatX        = 2.0
	flatReach    = 3.5
)

// The body between the bearing seats is larger than the seat, so a part bored with a D chord at bodyPressFlat
// slides over a seat on its way to the body. Every body section has its flat at bodyFlatX, and the body's step down to
// the seat is the collar on the side its parts are fitted from.
const (
	bodyRadius       = 3.3  // arbor body between the seats: escape, pallet and great wheel front section, and the carrier section
	journalRadius    = 3.05 // great wheel arbor under the great wheel, where it turns
	bodyFlatX        = 2.55 // flat plane of the body sections; above 2.5, so the cutting box never touches a seat
	carrierFlatX     = 2.80 // flat plane of the carrier and spring hub section
	bodyPressRadius  = 3.35 // D bore arc on bodyRadius, 0.05 press
	bodyPressFlat    = 2.60 // D bore chord on the body flat, 0.05 press; above 2.5, so it passes a seat
	carrierPressFlat = 2.85 // D bore chord of the pawl carrier on the carrier flat, 0.05 press
	// The arbor ends outside the front plate step down from the seat to r 2.0, 4.0 across, with a flat at x 1.5. The
	// crutch, knob, keyed rings and cap bore onto them with 0.05 press: arc r 2.05, chord x 1.55.
	endRadius      = 2.0
	endFlatX       = 1.5
	endPressRadius = 2.05
	endPressFlat   = 1.55
	endStepZ       = 33.25 // where a pallet or great wheel arbor steps from the seat down to the end, 0.25 past the bearing
	journalEndZ    = 5.20  // where the great wheel journal steps up to the body, 0.25 from the wheel's bore edge
)

// Every step of an arbor that does not face a purchased bearing is a cone of 30 degrees half angle, so a printed half
// lying on its cut face shows no square inside corner (docs/model-pipeline.md Split shafts). A flat that stops
// partway ends in a plane at the same angle to the axis.
const (
	coneHalfAngle = math.Pi / 6
	// rampReach is how far past the axis the ramp tool of a flat reaches: beyond the collar radius, so a ramp leaves
	// the surface by itself.
	rampReach = 4.0
)

// coneRun is the length along the axis of a cone of 30 degrees half angle over a radius change dr.
func coneRun(dr float64) float64 { return dr / math.Tan(coneHalfAngle) }

// coneSeat is the axial shift that puts a point offset radially by dr from the small end of a cone at the normal
// distance gap from the cone's surface. A negative shift moves the point toward the cone's body.
func coneSeat(dr, gap float64) float64 { return (dr*math.Cos(coneHalfAngle) - gap) / math.Sin(coneHalfAngle) }

// meridianPoints is a (radius, z) polygon with a point on the axis at each end.
type meridianPoints [][2]float64

// polygon draws the closed polygon pts on s. The first point is fixed by both distances from the origin. Each later
// point shares one coordinate with its predecessor along a horizontal or vertical line and takes the other by a
// distance; a slanted line fixes both. The closing line follows from the two end points.
func polygon(s *sketch.Sketch, pts meridianPoints) {
	ps := make([]*sketch.Point, len(pts))
	for i, p := range pts {
		ps[i] = s.CreatePoint(p[0], p[1])
	}
	s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), ps[0], pts[0][0]),
		sketch.NewVerticalDistance(s.Origin(), ps[0], pts[0][1]))
	for i := 1; i < len(pts); i++ {
		line := s.CreateLine(ps[i-1], ps[i])
		switch {
		case pts[i-1][1] == pts[i][1]:
			s.AddConstraint(sketch.NewHorizontal(line), sketch.NewHorizontalDistance(s.Origin(), ps[i], pts[i][0]))
		case pts[i-1][0] == pts[i][0]:
			s.AddConstraint(sketch.NewVertical(line), sketch.NewVerticalDistance(s.Origin(), ps[i], pts[i][1]))
		default:
			s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), ps[i], pts[i][0]),
				sketch.NewVerticalDistance(s.Origin(), ps[i], pts[i][1]))
		}
	}
	s.CreateLine(ps[len(ps)-1], ps[0])
}

// stepped revolves the polygon pts about the vertical line through (x, y). The last point lies on the axis and the
// closing line down the axis follows from the two zero radii.
func (b *builder) stepped(what string, x, y float64, pts meridianPoints) (*decad.Body, error) {
	s, err := b.meridian(what, x, y)
	if err != nil {
		return nil, err
	}
	polygon(s, pts)
	return b.revolve(s, what)
}

// flatSpan is one flat cut along an arbor: the flat plane x = X over z in [Z0, Z1]. A flat that stops partway ends in
// a ramp where RampStart or RampEnd is set: a plane at 30 degrees to the axis that rises from the flat plane outward,
// behind Z0 or past Z1, until it leaves the shaft.
type flatSpan struct {
	X, Z0, Z1          float64
	RampStart, RampEnd bool
}

// rampRun is the length along the axis of a ramp from the flat plane at x to the ramp tool's reach.
func rampRun(x float64) float64 { return coneRun(rampReach - x) }

// flatBox extrudes the box x in [flatX, flatReach], y in [-3.5, 3.5] over z in [z0, z1] about the axis (x, y).
func (b *builder) flatBox(what string, x, y, flatX, z0, z1 float64) (*decad.Body, error) {
	s, err := b.flat(what, x, y, z0)
	if err != nil {
		return nil, err
	}
	rect := s.CreateRectangle(flatX, -flatReach, flatReach, flatReach)
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), rect.A, flatX),
		sketch.NewVerticalDistance(s.Origin(), rect.A, -flatReach),
		sketch.NewHorizontalDistance(rect.A, rect.B, flatReach-flatX),
		sketch.NewVerticalDistance(rect.A, rect.D, 2*flatReach),
	)
	return b.extrude(s, 0, z1-z0, what)
}

// flatRamp extrudes the cutter of a flat with a ramp at one or both ends about the axis (x, y): the polygon (u, z) in
// the meridian plane through the axis, its flat side at u = X and its far side at u = rampReach, swept rampReach each
// way across the axis. Each ramp is a slanted edge of the polygon, fixed by its two end points.
func (b *builder) flatRamp(what string, x, y float64, sp flatSpan) (*decad.Body, error) {
	s, err := b.meridian(what, x, y)
	if err != nil {
		return nil, err
	}
	z0, z1 := sp.Z0, sp.Z1
	rampEnd, rampStart := z1, z0
	if sp.RampEnd {
		rampEnd = z1 + rampRun(sp.X)
	}
	if sp.RampStart {
		rampStart = z0 - rampRun(sp.X)
	}
	polygon(s, meridianPoints{{sp.X, z0}, {sp.X, z1}, {rampReach, rampEnd}, {rampReach, rampStart}})
	p, err := b.profile(s, 0, what)
	if err != nil {
		return nil, err
	}
	body, err := decad.Extrude(b.ctx, b.doc, s, p, decad.Symmetric{D: units.Millimeters(rampReach)})
	if err != nil {
		return nil, fmt.Errorf("%s: extrude: %w", what, err)
	}
	return body, nil
}

// arbor builds a stepped shaft and cuts its flats over the spans.
func (b *builder) arbor(what string, x, y float64, pts meridianPoints, spans []flatSpan) (*decad.Body, error) {
	body, err := b.stepped(what, x, y, pts)
	if err != nil {
		return nil, err
	}
	var boxes []*decad.Body
	for i, sp := range spans {
		name := fmt.Sprintf("%s flat %d", what, i+1)
		var box *decad.Body
		if sp.RampStart || sp.RampEnd {
			box, err = b.flatRamp(name, x, y, sp)
		} else {
			box, err = b.flatBox(name, x, y, sp.X, sp.Z0, sp.Z1)
		}
		if err != nil {
			return nil, err
		}
		boxes = append(boxes, box)
	}
	tool := boxes[0]
	if len(boxes) > 1 {
		if tool, err = b.union(what+" flats", boxes[0], boxes[1:]...); err != nil {
			return nil, err
		}
	}
	return b.cut(body, tool, what+" flats")
}

// Cone places in mm. A cone sits where the part it faces keeps its old fit, measured normal to the cone. Each is
// listed in instructions.md, Arbors, bearings and axial layout.
var (
	// The escape wheel's front counterbore floor (z 23.95) meets the D bore (r 3.35) in a corner that stands 0.05 from
	// the collar's cone, as it stood 0.05 from the old collar face.
	escapeConeLow  = wheelCounterFloor - coneSeat(bodyPressRadius-bodyRadius, bodyPressRadius-bodyRadius)
	escapeConeHigh = escapeConeLow + coneRun(CollarRadius-bodyRadius)
	// The anchor's rear face (z 21.0) meets its D bore (r 3.35) in a corner that stands 0.05 from the shoulder's cone.
	palletConeHigh = anchorLow + coneSeat(bodyPressRadius-bodyRadius, bodyPressRadius-bodyRadius)
	palletConeLow  = palletConeHigh - coneRun(CollarRadius-bodyRadius)
	// The great wheel arbor's collar ends in a cone down to the journal; the collar is 3.0 long to the cone's end.
	greatCollarEnd     = 3.25
	greatCollarConeLow = greatCollarEnd - coneRun(CollarRadius-journalRadius)
	// The journal steps up to the body in a cone.
	greatJournalConeHigh = journalEndZ + coneRun(bodyRadius-journalRadius)
	// The end steps (r 2.5 to r 2.0) run into the end section as cones.
	endConeHigh = endStepZ + coneRun(ArborRadius-endRadius)
	// The suspension pin's rear peg widens into the shoulder in a cone from the front plate's outer face.
	pinConeHigh = frontHigh + coneRun(CollarRadius-ArborRadius)
	// The front peg's step to the end section is a cone over the peg, which the cap's countersink then meets.
	pinEndStepZ    = 45.25
	pinEndConeHigh = pinEndStepZ + coneRun(ArborRadius-endRadius)
)

var (
	escapeArborPoints = meridianPoints{{0, -5}, {2.5, -5}, {2.5, 0.25}, {bodyRadius, 0.25}, {bodyRadius, escapeConeLow},
		{CollarRadius, escapeConeHigh}, {CollarRadius, 27.75}, {2.5, 27.75}, {2.5, 33}, {0, 33}}
	palletArborPoints = meridianPoints{{0, -5}, {2.5, -5}, {2.5, 0.25}, {CollarRadius, 0.25},
		{CollarRadius, palletConeLow}, {bodyRadius, palletConeHigh}, {bodyRadius, 27.75}, {2.5, 27.75}, {2.5, endStepZ},
		{endRadius, endConeHigh}, {endRadius, 38.25}, {0, 38.25}}
	greatArborPoints = meridianPoints{{0, -5}, {2.5, -5}, {2.5, 0.25}, {CollarRadius, 0.25},
		{CollarRadius, greatCollarConeLow}, {journalRadius, greatCollarEnd}, {journalRadius, journalEndZ},
		{bodyRadius, greatJournalConeHigh}, {bodyRadius, 27.75}, {2.5, 27.75}, {2.5, endStepZ}, {endRadius, endConeHigh},
		{endRadius, 38.25}, {0, 38.25}}
	pinPoints = meridianPoints{{0, 31.05}, {2.5, 31.05}, {2.5, frontHigh}, {CollarRadius, pinConeHigh},
		{CollarRadius, 39.80}, {2.5, 39.80}, {2.5, pinEndStepZ}, {endRadius, pinEndConeHigh}, {endRadius, 46.75}, {0, 46.75}}

	escapeArborFlats = []flatSpan{{bodyFlatX, 0.20, 23.95, false, true}}
	palletArborFlats = []flatSpan{{bodyFlatX, anchorLow, 27.80, true, false}, {endFlatX, 34.25, 39.25, false, false}}
	greatArborFlats  = []flatSpan{{carrierFlatX, layerLow, 17.55, true, false}, {bodyFlatX, 17.45, 27.80, false, false},
		{endFlatX, 34.25, 39.25, false, false}}
	pinFlats = []flatSpan{{FlatX, 30.05, frontHigh, false, true}, {endFlatX, 45.30, 47.75, false, false}}
)

func (b *builder) buildEscapeArbor() (*decad.Body, error) {
	return b.arbor("escape arbor", 0, 0, escapeArborPoints, escapeArborFlats)
}

func (b *builder) buildPalletArbor() (*decad.Body, error) {
	return b.arbor("pallet arbor", 0, b.g.P.Y, palletArborPoints, palletArborFlats)
}

func (b *builder) buildGreatWheelArbor() (*decad.Body, error) {
	return b.arbor("great wheel arbor", b.g.G.X, b.g.G.Y, greatArborPoints, greatArborFlats)
}

func (b *builder) buildSuspensionPin() (*decad.Body, error) {
	return b.arbor("suspension pin", 0, b.g.S.Y, pinPoints, pinFlats)
}
