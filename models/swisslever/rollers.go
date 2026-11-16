package swisslever

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Safety roller and roller table heights and dimensions in mm. Both are built about the origin with the bore flat on
// +X and posed onto the balance staff.
const (
	safetyLow        = 3.30
	safetyLength     = 7.5
	safetyBoreReach  = 0.75
	safetyBoreLength = 9.0
	tableLow         = 10.85
	tableThickness   = 2.5
	tableRadius      = 11.0
	tableBoreReach   = 0.75
	tableBoreLength  = 4.0
	// pinHeight is the impulse pin's length from the table's rear face.
	pinHeight = 4.5
)

// safetySketch draws the safety roller's outline about the origin: the roller arc, long way round, and the crescent
// arc about its centre on the pin's ray, meeting at two lips that carry a round of CrescentLip.
func (b *builder) safetySketch() (*sketch.Sketch, error) {
	const what = "safety roller"
	s, err := b.flat(what, 0, 0, safetyLow)
	if err != nil {
		return nil, err
	}
	g := &b.g
	ref := xRef(s)
	o := s.Origin()
	cc := polar(s, ref, CrescentOffset, g.PinAngle)
	lipHi, lipLo, ok := circleCircle(SafetyRadius, Polar(CrescentOffset, g.PinAngle), CrescentRadius)
	if !ok {
		return nil, fmt.Errorf("%s: crescent does not meet the roller circle", what)
	}
	// lipHi is on the counter-clockwise side of the crescent's ray. The roller arc runs from it the long way round to
	// lipLo; the crescent arc runs counter-clockwise from lipHi to lipLo through its side nearest the origin.
	hi, lo := s.CreatePoint(lipHi.X, lipHi.Y), s.CreatePoint(lipLo.X, lipLo.Y)
	roller := s.CreateArc(o, hi, lo)
	crescent := s.CreateArc(cc, hi, lo)
	s.AddConstraint(sketch.NewRadius(roller, SafetyRadius), sketch.NewRadius(crescent, CrescentRadius))
	for i, c := range []struct {
		before, after sketch.Entity
		at            *sketch.Point
	}{{roller, crescent, lo}, {crescent, roller, hi}} {
		if err := sketchFillet(s, c.before, c.after, c.at, CrescentLip); err != nil {
			return nil, fmt.Errorf("%s: round lip %d: %w", what, i+1, err)
		}
	}
	return s, nil
}

// buildSafetyRoller builds the safety roller about the origin: the cylinder with its crescent cut through the length
// and the D press bore, flat on +X.
func (b *builder) buildSafetyRoller() (*decad.Body, error) {
	s, err := b.safetySketch()
	if err != nil {
		return nil, err
	}
	roller, err := b.extrude(s, 0, safetyLength, "safety roller")
	if err != nil {
		return nil, err
	}
	bore, err := b.dTool("safety roller bore", 0, 0, safetyLow-safetyBoreReach, BodyBoreRadius, BodyBoreFlat,
		safetyBoreLength)
	if err != nil {
		return nil, err
	}
	return b.cut(roller, bore, "safety roller bore")
}

// buildRollerTable builds the roller table about the origin: the disc, the impulse pin standing on its rear face with
// a root round, and the D press bore, flat on +X.
func (b *builder) buildRollerTable() (*decad.Body, error) {
	table, err := b.discAt("roller table", 0, 0, tableLow, tableRadius, tableThickness)
	if err != nil {
		return nil, err
	}
	pin := Polar(PinCentreRadius, b.g.PinAngle)
	face := decad.Faces(decad.FaceCreatedBy(decad.CapStart(table))).Exactly(1)
	if table, err = b.boss(table, face, r3.NewVec(pin.X, pin.Y, tableLow), 2*PinRadius, pinHeight,
		"impulse pin"); err != nil {
		return nil, err
	}
	bore, err := b.dTool("roller table bore", 0, 0, tableLow-tableBoreReach, BodyBoreRadius, BodyBoreFlat,
		tableBoreLength)
	if err != nil {
		return nil, err
	}
	return b.cut(table, bore, "roller table bore")
}
