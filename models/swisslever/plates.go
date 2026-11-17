package swisslever

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Plate dimensions of instructions.md "Frame plates and fasteners", in mm.
const (
	plateHalfWidth   = 56.0
	plateBottomEdge  = -50.0
	plateTopEdge     = 146.0
	plateCornerRound = 10.0
	plateThickness   = 6.0
	backPlateLow     = -6.0
	backPlateHigh    = 0.0
	frontPlateLow    = FrontPlateInner
	frontPlateHigh   = FrontPlateInner + plateThickness
	pillarDiameter   = 10.0
	pillarHeight     = 27.0
	bankingDiameter  = 2 * BankingPostRadius
	studDiameter     = 3.0
	studHeight       = 8.05
	// seatRadius is a 605ZZ's 7.0 radius and the 0.05 bearing-seat clearance; lipRadius the bore through the lip.
	seatRadius  = 7.05
	seatDepth   = 6.0
	lipRadius   = 5.0
	barHalf     = 4.0
	hubRing     = 10.0
	winSide     = 45.0
	winSpine    = 11.0
	winCorner   = 6.0
	winRound    = 1.0
	winW1Bottom = -33.5
	winW4Top    = 129.0
	// backBearingZ and frontBearingZ are the lower cap heights of the back and front bearings.
	backBearingZ  = -5.0
	frontBearingZ = 27.0
)

// sites returns the four bolt sites: head top at z 33, shank toward -Z, nut corners on +X.
func (b *builder) sites() []m2fastener.Site { return b.g.Sites }

// axes returns the three bearing axes O, P and B.
func (b *builder) axes() []Pt { return []Pt{b.g.O, b.g.P, b.g.B} }

// plateOutline draws the rounded rectangle of a plate: four straight sides each placed by one distance from the sketch
// origin and four corner arcs, each tangent to its two sides.
func plateOutline(s *sketch.Sketch) {
	o := s.Origin()
	x0, x1, y0, y1, rc := -plateHalfWidth, plateHalfWidth, plateBottomEdge, plateTopEdge, plateCornerRound
	c1, c2 := s.CreatePoint(x0+rc, y0+rc), s.CreatePoint(x1-rc, y0+rc)
	c3, c4 := s.CreatePoint(x1-rc, y1-rc), s.CreatePoint(x0+rc, y1-rc)
	a1s, a1e := s.CreatePoint(x0, y0+rc), s.CreatePoint(x0+rc, y0)
	a2s, a2e := s.CreatePoint(x1-rc, y0), s.CreatePoint(x1, y0+rc)
	a3s, a3e := s.CreatePoint(x1, y1-rc), s.CreatePoint(x1-rc, y1)
	a4s, a4e := s.CreatePoint(x0+rc, y1), s.CreatePoint(x0, y1-rc)
	arc1, arc2 := s.CreateArc(c1, a1s, a1e), s.CreateArc(c2, a2s, a2e)
	arc3, arc4 := s.CreateArc(c3, a3s, a3e), s.CreateArc(c4, a4s, a4e)
	bottom, right := s.CreateLine(a1e, a2s), s.CreateLine(a2e, a3s)
	top, left := s.CreateLine(a3e, a4s), s.CreateLine(a4e, a1s)
	s.AddConstraint(
		sketch.NewHorizontal(bottom), sketch.NewVertical(right), sketch.NewHorizontal(top), sketch.NewVertical(left),
		sketch.NewRadius(arc1, rc), sketch.NewRadius(arc2, rc), sketch.NewRadius(arc3, rc), sketch.NewRadius(arc4, rc),
		sketch.NewTangent(left, arc1), sketch.NewTangent(bottom, arc1),
		sketch.NewTangent(bottom, arc2), sketch.NewTangent(right, arc2),
		sketch.NewTangent(right, arc3), sketch.NewTangent(top, arc3),
		sketch.NewTangent(top, arc4), sketch.NewTangent(left, arc4),
		sketch.NewVerticalDistance(o, a1e, y0), sketch.NewHorizontalDistance(o, a2e, x1),
		sketch.NewVerticalDistance(o, a3e, y1), sketch.NewHorizontalDistance(o, a4e, x0),
	)
}

// windowSpec is one window of a plate: a rounded rectangle whose bottom and top edges may carry a hub ring. A hub ring
// is an arc of radius hubRing about the hub centre (on the bar's axis, barHalf beyond the edge) that bulges into the
// window between the two points where it meets the edge.
type windowSpec struct {
	x0, x1, bottom, top float64
	bottomHub, topHub   *sketch.Point
}

// barWindow draws one window as a closed loop of lines and arcs. Each straight edge is horizontal or vertical and placed
// by its distance from the sketch origin; each corner arc has radius winCorner and is tangent to both its lines; each hub
// arc shares its centre with the lip circle. A round of winRound then softens each junction of a bar edge and a hub arc.
func barWindow(s *sketch.Sketch, w windowSpec) error {
	o := s.Origin()
	rc, h := winCorner, HubChordHalf
	// Corner arcs, counter-clockwise: bottom left, bottom right, top right, top left.
	blS, blE := s.CreatePoint(w.x0, w.bottom+rc), s.CreatePoint(w.x0+rc, w.bottom)
	brS, brE := s.CreatePoint(w.x1-rc, w.bottom), s.CreatePoint(w.x1, w.bottom+rc)
	trS, trE := s.CreatePoint(w.x1, w.top-rc), s.CreatePoint(w.x1-rc, w.top)
	tlS, tlE := s.CreatePoint(w.x0+rc, w.top), s.CreatePoint(w.x0, w.top-rc)
	bl := s.CreateArc(s.CreatePoint(w.x0+rc, w.bottom+rc), blS, blE)
	br := s.CreateArc(s.CreatePoint(w.x1-rc, w.bottom+rc), brS, brE)
	tr := s.CreateArc(s.CreatePoint(w.x1-rc, w.top-rc), trS, trE)
	tl := s.CreateArc(s.CreatePoint(w.x0+rc, w.top-rc), tlS, tlE)
	s.AddConstraint(sketch.NewRadius(bl, rc), sketch.NewRadius(br, rc), sketch.NewRadius(tr, rc), sketch.NewRadius(tl, rc))

	left, right := s.CreateLine(tlE, blS), s.CreateLine(brE, trS)
	s.AddConstraint(
		sketch.NewVertical(left), sketch.NewDistancePointLine(o, left, math.Abs(w.x0)),
		sketch.NewVertical(right), sketch.NewDistancePointLine(o, right, math.Abs(w.x1)),
		sketch.NewTangent(left, bl), sketch.NewTangent(left, tl), sketch.NewTangent(right, br), sketch.NewTangent(right, tr),
	)
	// edge draws a horizontal edge between two points at height y.
	edge := func(from, to *sketch.Point, y float64) *sketch.Line {
		l := s.CreateLine(from, to)
		s.AddConstraint(sketch.NewHorizontal(l), sketch.NewDistancePointLine(o, l, math.Abs(y)))
		return l
	}
	type junction struct {
		before, after sketch.Entity
		at            *sketch.Point
	}
	var junctions []junction
	// Bottom edge, left to right.
	if w.bottomHub == nil {
		l := edge(blE, brS, w.bottom)
		s.AddConstraint(sketch.NewTangent(l, bl), sketch.NewTangent(l, br))
	} else {
		a, c := s.CreatePoint(-h, w.bottom), s.CreatePoint(h, w.bottom)
		l1, l2 := edge(blE, a, w.bottom), edge(c, brS, w.bottom)
		arc := s.CreateArc(w.bottomHub, c, a)
		s.AddConstraint(sketch.NewRadius(arc, hubRing), sketch.NewTangent(l1, bl), sketch.NewTangent(l2, br))
		junctions = append(junctions, junction{l1, arc, a}, junction{arc, l2, c})
	}
	// Top edge, right to left.
	if w.topHub == nil {
		l := edge(tlS, trE, w.top)
		s.AddConstraint(sketch.NewTangent(l, tl), sketch.NewTangent(l, tr))
	} else {
		a, c := s.CreatePoint(h, w.top), s.CreatePoint(-h, w.top)
		l1, l2 := edge(a, trE, w.top), edge(tlS, c, w.top)
		arc := s.CreateArc(w.topHub, c, a)
		s.AddConstraint(sketch.NewRadius(arc, hubRing), sketch.NewTangent(l1, tr), sketch.NewTangent(l2, tl))
		junctions = append(junctions, junction{l1, arc, a}, junction{arc, l2, c})
	}
	for i, j := range junctions {
		if err := sketchFillet(s, j.before, j.after, j.at, winRound); err != nil {
			return fmt.Errorf("round window junction %d: %w", i+1, err)
		}
	}
	return nil
}

// plateSection extrudes one plate section from a single sketch at z low: the outline, the windows and the three lip
// circles. back selects the back plate's two-half W3.
func (b *builder) plateSection(what string, low float64, back bool) (*decad.Body, error) {
	s, err := b.flat(what, 0, 0, low)
	if err != nil {
		return nil, err
	}
	ref := xRef(s)
	plateOutline(s)
	g := &b.g
	// The hub centres: O is the origin; P and B are polar points shared by the hub rings and the lip circles.
	oc := s.Origin()
	pc := polar(s, ref, g.P.Len(), 90*deg)
	bc := polar(s, ref, g.B.Len(), 90*deg)
	barTop := func(c Pt) float64 { return c.Y - barHalf }
	barBottom := func(c Pt) float64 { return c.Y + barHalf }
	windows := []windowSpec{
		{-winSide, winSide, winW1Bottom, barTop(g.O), nil, oc},
		{-winSide, winSide, barBottom(g.O), barTop(g.P), oc, pc},
		{-winSide, winSide, barBottom(g.B), winW4Top, bc, nil},
	}
	if back {
		windows = append(windows,
			windowSpec{-winSide, -winSpine, barBottom(g.P), barTop(g.B), nil, nil},
			windowSpec{winSpine, winSide, barBottom(g.P), barTop(g.B), nil, nil})
	} else {
		windows = append(windows, windowSpec{-winSide, winSide, barBottom(g.P), barTop(g.B), pc, bc})
	}
	for i, w := range windows {
		if err := barWindow(s, w); err != nil {
			return nil, fmt.Errorf("%s: window %d: %w", what, i+1, err)
		}
	}
	for _, c := range []*sketch.Point{oc, pc, bc} {
		s.AddConstraint(sketch.NewRadius(s.CreateCircle(c, lipRadius), lipRadius))
	}
	return b.extrude(s, len(windows)+3, plateThickness, what)
}

// m2fastenerTool is an m2fastener tool builder: HoleTool, PocketTool or CavityTool.
type m2fastenerTool func(context.Context, *decad.Document, m2fastener.Site) (*decad.Body, *sketch.Sketch, error)

// fastenerTools builds the tool of each site made by f and records its sketch.
func (b *builder) fastenerTools(f m2fastenerTool) ([]*decad.Body, error) {
	var out []*decad.Body
	for _, s := range b.sites() {
		body, sk, err := f(b.ctx, b.doc, s)
		if err != nil {
			return nil, err
		}
		b.sketches = append(b.sketches, sk)
		out = append(out, body)
	}
	return out, nil
}

// seatTools returns one bearing-seat pocket tool per axis, from height z over seatDepth.
func (b *builder) seatTools(z float64) ([]*decad.Body, error) {
	var out []*decad.Body
	for i, a := range b.axes() {
		t, err := b.discAt(fmt.Sprintf("bearing seat %d", i+1), a.X, a.Y, z, seatRadius, seatDepth)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// plateKeep chooses the loops of a plate that are chamfered: the outer loop, each window (more than four edges) and the
// single-edge circles of radius 7.05 (the pocket rims) and 5 (the lip rims) centred on the three axes. The pillar,
// post and stud root circles have other radii and stay out.
func (b *builder) plateKeep() edgefinish.LoopFilter {
	axes := b.axes()
	return func(l *decad.Loop, e []*decad.Edge) bool {
		if l.IsOuter() || len(e) > 4 {
			return true
		}
		if len(e) != 1 {
			return false
		}
		c, ok := e[0].Curve().(decad.Circle3)
		if !ok {
			return false
		}
		r := c.Radius.Base()
		if math.Abs(r-seatRadius) > 0.01 && math.Abs(r-lipRadius) > 0.01 {
			return false
		}
		for _, a := range axes {
			if math.Hypot(c.Center.X-a.X, c.Center.Y-a.Y) < 0.01 {
				return true
			}
		}
		return false
	}
}

// raise raises each post of centres at (x, y, face z) on the face the selector names.
func (b *builder) raise(body *decad.Body, face decad.FaceSelector, z float64, centres []Pt, diameter, height float64,
	what string) (*decad.Body, error) {
	var err error
	for i, c := range centres {
		if body, err = b.boss(body, face, r3.NewVec(c.X, c.Y, z), diameter, height, fmt.Sprintf("%s %d", what, i+1)); err != nil {
			return nil, err
		}
	}
	return body, nil
}

// buildBackPlate builds the back plate with its four pillars, two banking posts, bearing pockets, bolt holes and nut
// cavities.
func (b *builder) buildBackPlate() (*decad.Body, error) {
	g := &b.g
	base, err := b.plateSection("back plate", backPlateLow, true)
	if err != nil {
		return nil, err
	}
	face := decad.Faces(decad.FaceCreatedBy(decad.CapEnd(base))).Exactly(1)
	plate, err := b.raise(base, face, backPlateHigh, g.Pillars[:], pillarDiameter, pillarHeight, "pillar")
	if err != nil {
		return nil, err
	}
	plate, err = b.raise(plate, face, backPlateHigh, []Pt{g.BankingLeft, g.BankingRight}, bankingDiameter,
		BankingPostHeight, "banking post")
	if err != nil {
		return nil, err
	}
	tools, err := b.seatTools(backPlateLow + 1)
	if err != nil {
		return nil, err
	}
	holes, err := b.fastenerTools(m2fastener.HoleTool)
	if err != nil {
		return nil, err
	}
	cavities, err := b.fastenerTools(m2fastener.CavityTool)
	if err != nil {
		return nil, err
	}
	tools = append(append(tools, holes...), cavities...)
	tool, err := b.union("back plate tools", tools[0], tools[1:]...)
	if err != nil {
		return nil, err
	}
	if plate, err = b.cut(plate, tool, "back plate"); err != nil {
		return nil, err
	}
	if plate, err = b.finish(plate, []float64{backPlateLow, backPlateHigh}, b.plateKeep()); err != nil {
		return nil, fmt.Errorf("back plate edges: %w", err)
	}
	return plate, nil
}

// buildFrontPlate builds the front plate with the hairspring stud, bearing pockets, bolt holes and head pockets.
func (b *builder) buildFrontPlate() (*decad.Body, error) {
	base, err := b.plateSection("front plate", frontPlateLow, false)
	if err != nil {
		return nil, err
	}
	face := decad.Faces(decad.FaceCreatedBy(decad.CapStart(base))).Exactly(1)
	plate, err := b.raise(base, face, frontPlateLow, []Pt{b.g.Stud}, studDiameter, studHeight, "hairspring stud")
	if err != nil {
		return nil, err
	}
	tools, err := b.seatTools(frontPlateLow - 1)
	if err != nil {
		return nil, err
	}
	holes, err := b.fastenerTools(m2fastener.HoleTool)
	if err != nil {
		return nil, err
	}
	pockets, err := b.fastenerTools(m2fastener.PocketTool)
	if err != nil {
		return nil, err
	}
	tools = append(append(tools, holes...), pockets...)
	tool, err := b.union("front plate tools", tools[0], tools[1:]...)
	if err != nil {
		return nil, err
	}
	if plate, err = b.cut(plate, tool, "front plate"); err != nil {
		return nil, err
	}
	if plate, err = b.finish(plate, []float64{frontPlateLow, frontPlateHigh}, b.plateKeep()); err != nil {
		return nil, fmt.Errorf("front plate edges: %w", err)
	}
	return plate, nil
}
