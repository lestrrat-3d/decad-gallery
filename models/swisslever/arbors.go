package swisslever

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Arbor dimensions in mm (instructions.md "Arbors, bearings and axial layout"): the seat is the 605ZZ bore, the
// shoulder is 7.4 across, the body between the seats is 6.6 across with a flat at bodyFlatX.
const (
	// ArborRadius is the bearing seat radius.
	ArborRadius = 2.5
	// CollarRadius is the shoulder radius.
	CollarRadius = 3.7
	bodyRadius   = 3.3
	bodyFlatX    = 2.55
	endRadius    = 2.0
	endFlatX     = 1.5
	flatReach    = 3.5
)

// Every step that does not face a purchased bearing is a cone of 30 degrees half angle. A flat that stops partway
// ends in a plane at the same angle to the axis, which rises from the flat plane outward until it leaves the shaft.
const (
	coneHalfAngle = math.Pi / 6
	// rampReach is how far past the axis the ramp tool of a flat reaches.
	rampReach = 4.0
	// pressGap is the 0.05 mm press between a pressed part's bore corner and the cone it faces.
	pressGap = 0.05
)

// Axial places of the arbors in mm.
const (
	// collarBackZ and collarFrontZ are the arbors' shoulder and body step heights, 0.25 from the bearings.
	collarBackZ  = 0.25
	collarFrontZ = 26.75
	// The escape arbor's end outside the front plate steps down from the seat 0.25 past the bearing.
	escapeEndStepZ = 32.25
	escapeEndZ     = 37.25
	// The pallet arbor and balance staff end at the front bearing's outer face.
	palletEndZ = 32.0
)

// coneRun is the length along the axis of a cone of 30 degrees half angle over a radius change dr.
func coneRun(dr float64) float64 { return dr / math.Tan(coneHalfAngle) }

// coneSeat is the axial shift that puts a point offset radially by dr from the small end of a cone at the normal
// distance gap from the cone's surface. A negative shift moves the point toward the cone's body.
func coneSeat(dr, gap float64) float64 {
	return (dr*math.Cos(coneHalfAngle) - gap) / math.Sin(coneHalfAngle)
}

// rampRun is the length along the axis of a ramp from the flat plane at x to the ramp tool's reach.
func rampRun(x float64) float64 { return coneRun(rampReach - x) }

// Cone places. The escape wheel's front bore corner (r 3.35, z 13.35) stands 0.05 from the shoulder's cone; the
// lever's and safety roller's rear bore corner (r 3.35, z 3.30) stands 0.05 from the pallet shoulder's cone.
var (
	escapeConeLow  = wheelHigh - coneSeat(bodyPressRadius-bodyRadius, pressGap)
	escapeConeHigh = escapeConeLow + coneRun(CollarRadius-bodyRadius)
	palletConeHigh = palletPartLow + coneSeat(bodyPressRadius-bodyRadius, pressGap)
	palletConeLow  = palletConeHigh - coneRun(CollarRadius-bodyRadius)
	endConeHigh    = escapeEndStepZ + coneRun(ArborRadius-endRadius)
)

const (
	// bodyPressRadius is the D bore arc of the body family.
	bodyPressRadius = BodyBoreRadius
	// palletPartLow is the rear face of the lever and the safety roller on the pallet arbor and the staff.
	palletPartLow = 3.30
	// palletFlatEnd is where the pallet arbor's flat ends.
	palletFlatEnd = 26.80
)

// meridianPoints is a (radius, z) polygon with a point on the axis at each end.
type meridianPoints [][2]float64

// flatSpan is one flat cut along an arbor: the flat plane x = X over z in [Z0, Z1]. A flat that stops partway ends in
// a ramp where RampStart or RampEnd is set.
type flatSpan struct {
	X, Z0, Z1          float64
	RampStart, RampEnd bool
}

var (
	escapeArborPoints = meridianPoints{{0, -5}, {2.5, -5}, {2.5, collarBackZ}, {bodyRadius, collarBackZ},
		{bodyRadius, escapeConeLow}, {CollarRadius, escapeConeHigh}, {CollarRadius, collarFrontZ},
		{2.5, collarFrontZ}, {2.5, escapeEndStepZ}, {endRadius, endConeHigh}, {endRadius, escapeEndZ}, {0, escapeEndZ}}
	palletArborPoints = meridianPoints{{0, -5}, {2.5, -5}, {2.5, collarBackZ}, {CollarRadius, collarBackZ},
		{CollarRadius, palletConeLow}, {bodyRadius, palletConeHigh}, {bodyRadius, collarFrontZ}, {2.5, collarFrontZ},
		{2.5, palletEndZ}, {0, palletEndZ}}

	escapeArborFlats = []flatSpan{{bodyFlatX, 0.20, wheelHigh, false, true}, {endFlatX, 33.25, 38.25, false, false}}
	palletArborFlats = []flatSpan{{bodyFlatX, palletPartLow, palletFlatEnd, true, false}}
)

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

// stepped revolves the polygon pts about the vertical line through (x, y).
func (b *builder) stepped(what string, x, y float64, pts meridianPoints) (*decad.Body, error) {
	s, err := b.meridian(what, x, y)
	if err != nil {
		return nil, err
	}
	polygon(s, pts)
	return b.revolve(s, what)
}

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
// way across the axis.
func (b *builder) flatRamp(what string, x, y float64, sp flatSpan) (*decad.Body, error) {
	s, err := b.meridian(what, x, y)
	if err != nil {
		return nil, err
	}
	rampEnd, rampStart := sp.Z1, sp.Z0
	if sp.RampEnd {
		rampEnd = sp.Z1 + rampRun(sp.X)
	}
	if sp.RampStart {
		rampStart = sp.Z0 - rampRun(sp.X)
	}
	polygon(s, meridianPoints{{sp.X, sp.Z0}, {sp.X, sp.Z1}, {rampReach, rampEnd}, {rampReach, rampStart}})
	p, err := b.profile(s, 0, what)
	if err != nil {
		return nil, err
	}
	body, err := decad.Extrude(b.doc, s, p, decad.Symmetric{D: units.Millimeters(rampReach)}).Do(b.ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: extrude: %w", what, err)
	}
	return body, nil
}

// arbor builds a stepped shaft and cuts its flats over the spans in one cut.
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

// buildEscapeArbor builds the escape arbor about the origin O, unposed.
func (b *builder) buildEscapeArbor() (*decad.Body, error) {
	return b.arbor("escape arbor", b.g.O.X, b.g.O.Y, escapeArborPoints, escapeArborFlats)
}

// buildPalletArbor builds the pallet arbor about P, unposed. The balance staff is the same part about B: see
// buildBalanceStaff.
func (b *builder) buildPalletArbor() (*decad.Body, error) {
	return b.arbor("pallet arbor", b.g.P.X, b.g.P.Y, palletArborPoints, palletArborFlats)
}

// buildBalanceStaff builds the balance staff, the pallet arbor's part about B.
func (b *builder) buildBalanceStaff() (*decad.Body, error) {
	return b.arbor("balance staff", b.g.B.X, b.g.B.Y, palletArborPoints, palletArborFlats)
}
