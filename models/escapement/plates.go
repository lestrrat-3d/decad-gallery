package escapement

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Plate dimensions in mm.
const (
	plateLeft    = -52.0
	plateRight   = 70.0
	plateBottom  = -86.0
	backTop      = 98.0
	frontTop     = 82.0
	plateCorner  = 10.0
	backLow      = -6.0
	backHigh     = 0.0
	frontLow     = 28.0
	frontHigh    = 34.0
	pillarRadius = 5.0
	pillarLow    = -0.5
	pillarHigh   = 28.0
	seatRadius   = 7.05 // a 605ZZ's 7.0 radius and the 0.05 bearing-seat clearance
	seatDepth    = 6.0
	mountRadius  = 2.25
	boltLength   = 10.0
	nutCavityAt  = 8.0
	headTop      = 34.0
)

// pillars are the four pillar centres, in the order of the bolts and nuts.
var pillars = []Pt{{-44, 74}, {62, 74}, {-44, -78}, {62, -78}}

// mountHoles are the back plate's two mounting holes.
var mountHoles = []Pt{{-30, 90}, {48, 90}}

// sites returns the four bolt sites: head top at z 34, shank toward -Z, nut corners on +X.
func sites() []m2fastener.Site {
	var out []m2fastener.Site
	for _, p := range pillars {
		out = append(out, m2fastener.Site{Head: r3.Vec{X: p.X, Y: p.Y, Z: headTop}, Dir: r3.Vec{Z: -1},
			Corner: r3.Vec{X: 1}, Length: boltLength, Cavity: nutCavityAt})
	}
	return out
}

// axes returns the three bearing axes O, P and G.
func (b *builder) axes() []Pt { return []Pt{b.g.O, b.g.P, b.g.G} }

// roundedRect draws the plate outline: four straight sides and four corner arcs of radius plateCorner, each arc
// tangent to its two sides, each side placed by one distance from the sketch origin.
func roundedRect(s *sketch.Sketch, top float64) {
	o := s.Origin()
	x0, x1, y0, y1, rc := plateLeft, plateRight, plateBottom, top, plateCorner
	c1, c2 := s.CreatePoint(x0+rc, y0+rc), s.CreatePoint(x1-rc, y0+rc)
	c3, c4 := s.CreatePoint(x1-rc, y1-rc), s.CreatePoint(x0+rc, y1-rc)
	a1s, a1e := s.CreatePoint(x0, y0+rc), s.CreatePoint(x0+rc, y0)
	a2s, a2e := s.CreatePoint(x1-rc, y0), s.CreatePoint(x1, y0+rc)
	a3s, a3e := s.CreatePoint(x1, y1-rc), s.CreatePoint(x1-rc, y1)
	a4s, a4e := s.CreatePoint(x0+rc, y1), s.CreatePoint(x0, y1-rc)
	arc1, arc2 := s.CreateArc(c1, a1s, a1e), s.CreateArc(c2, a2s, a2e)
	arc3, arc4 := s.CreateArc(c3, a3s, a3e), s.CreateArc(c4, a4s, a4e)
	bottom, right := s.CreateLine(a1e, a2s), s.CreateLine(a2e, a3s)
	topLine, left := s.CreateLine(a3e, a4s), s.CreateLine(a4e, a1s)
	s.AddConstraint(
		sketch.NewHorizontal(bottom), sketch.NewVertical(right), sketch.NewHorizontal(topLine), sketch.NewVertical(left),
		sketch.NewRadius(arc1, rc), sketch.NewRadius(arc2, rc), sketch.NewRadius(arc3, rc), sketch.NewRadius(arc4, rc),
		sketch.NewTangent(left, arc1), sketch.NewTangent(bottom, arc1),
		sketch.NewTangent(bottom, arc2), sketch.NewTangent(right, arc2),
		sketch.NewTangent(right, arc3), sketch.NewTangent(topLine, arc3),
		sketch.NewTangent(topLine, arc4), sketch.NewTangent(left, arc4),
		sketch.NewVerticalDistance(o, a1e, y0), sketch.NewHorizontalDistance(o, a2e, x1),
		sketch.NewVerticalDistance(o, a3e, y1), sketch.NewHorizontalDistance(o, a4e, x0),
	)
}

// holeAt draws a circle of radius r centred at the polar point (c) about the sketch origin. A centre on the origin
// is the origin itself.
func holeAt(s *sketch.Sketch, ref *sketch.Line, c Pt, r float64) {
	if c.X == 0 && c.Y == 0 {
		circleAt(s, r)
		return
	}
	centre := polar(s, ref, c.Len(), c.Angle())
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(centre, r), r))
}

// plateOutline extrudes a plate outline at height low with holes through it.
func (b *builder) plateOutline(what string, low, high, top float64, holes []Pt, radii []float64) (*decad.Body, error) {
	s, err := b.flat(what, 0, 0, low)
	if err != nil {
		return nil, err
	}
	ref := xRef(s)
	roundedRect(s, top)
	for i, h := range holes {
		holeAt(s, ref, h, radii[i])
	}
	return b.extrude(s, len(holes), high-low, what)
}

// fastenerTools builds the tool of each site made by f and records its sketch.
func (b *builder) fastenerTools(f toolFunc) ([]*decad.Body, error) {
	var out []*decad.Body
	for _, s := range sites() {
		body, sk, err := f(b.ctx, b.doc, s)
		if err != nil {
			return nil, err
		}
		b.sketches = append(b.sketches, sk)
		out = append(out, body)
	}
	return out, nil
}

// toolFunc is an m2fastener tool builder: HoleTool, PocketTool or CavityTool.
type toolFunc func(context.Context, *decad.Document, m2fastener.Site) (*decad.Body, *sketch.Sketch, error)

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

// buildBackPlate builds the back plate with its pillars, bearing seats, bolt holes and nut cavities.
func (b *builder) buildBackPlate() (*decad.Body, error) {
	holes := append(b.axes(), mountHoles...)
	radii := []float64{lipRadius, lipRadius, lipRadius, mountRadius, mountRadius}
	plate, err := b.plateOutline("back plate outline", backLow, backHigh, backTop, holes, radii)
	if err != nil {
		return nil, err
	}
	var posts []*decad.Body
	for i, p := range pillars {
		post, err := b.discAt(fmt.Sprintf("pillar %d", i+1), p.X, p.Y, pillarLow, pillarRadius, pillarHigh-pillarLow)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	if plate, err = b.union("back plate pillars", plate, posts...); err != nil {
		return nil, err
	}
	tools, err := b.seatTools(backLow + 1)
	if err != nil {
		return nil, err
	}
	holeTools, err := b.fastenerTools(m2fastener.HoleTool)
	if err != nil {
		return nil, err
	}
	cavities, err := b.fastenerTools(m2fastener.CavityTool)
	if err != nil {
		return nil, err
	}
	tools = append(append(tools, holeTools...), cavities...)
	tool, err := b.union("back plate tools", tools[0], tools[1:]...)
	if err != nil {
		return nil, err
	}
	if plate, err = b.cut(plate, tool, "back plate"); err != nil {
		return nil, err
	}
	plate, _, err = edgefinish.Chamfer(b.ctx, plate, zAxis, []float64{backLow, backHigh},
		rimKeepAll(b.axes(), seatRadius, lipRadius), edgefinish.MinChamfer)
	if err != nil {
		return nil, fmt.Errorf("back plate edges: %w", err)
	}
	return plate, nil
}

// buildFrontPlate builds the front plate with its bearing seats, bolt holes, head pockets and suspension socket.
func (b *builder) buildFrontPlate() (*decad.Body, error) {
	plate, err := b.plateOutline("front plate outline", frontLow, frontHigh, frontTop, b.axes(),
		[]float64{lipRadius, lipRadius, lipRadius})
	if err != nil {
		return nil, err
	}
	tools, err := b.seatTools(frontLow - 1)
	if err != nil {
		return nil, err
	}
	holeTools, err := b.fastenerTools(m2fastener.HoleTool)
	if err != nil {
		return nil, err
	}
	pockets, err := b.fastenerTools(m2fastener.PocketTool)
	if err != nil {
		return nil, err
	}
	socket, err := b.dTool("suspension socket", b.g.S.X, b.g.S.Y, 31, pressRadius, pressFlat, 4)
	if err != nil {
		return nil, err
	}
	tools = append(append(append(tools, holeTools...), pockets...), socket)
	tool, err := b.union("front plate tools", tools[0], tools[1:]...)
	if err != nil {
		return nil, err
	}
	if plate, err = b.cut(plate, tool, "front plate"); err != nil {
		return nil, err
	}
	plate, _, err = edgefinish.Chamfer(b.ctx, plate, zAxis, []float64{frontLow, frontHigh},
		rimKeepAll(b.axes(), seatRadius, lipRadius), edgefinish.MinChamfer)
	if err != nil {
		return nil, fmt.Errorf("front plate edges: %w", err)
	}
	return plate, nil
}
