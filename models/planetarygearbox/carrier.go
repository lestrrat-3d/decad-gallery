package planetarygearbox

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// disc describes a circular extrusion in the local frame.
type disc struct {
	what     string
	z        float64 // height of the sketch plane
	length   float64 // extrusion length along +Z
	x, y     float64 // center
	diameter float64
	expr     string // expression bound to the diameter
}

// planeAt returns the sketch plane parallel to XY at height z, with its origin on the axis.
func (r *run) planeAt(z float64) (*sketch.Plane, error) {
	frame, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	if err != nil {
		return nil, fmt.Errorf("plane frame: %w", err)
	}
	plane, err := r.world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("plane: %w", err)
	}
	return plane, nil
}

// extrudeDisc sketches one circle on the plane at disc.z and extrudes it up disc.length.
func (r *run) extrudeDisc(ctx context.Context, c disc) (*decad.Body, error) {
	plane, err := r.planeAt(c.z)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", c.what, err)
	}
	s, err := r.world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("%s: sketch: %w", c.what, err)
	}
	var center *sketch.Point
	if c.x == 0 && c.y == 0 {
		center = s.Origin()
	} else {
		center = sketchcheck.DimensionPoint(s, c.x, c.y)
	}
	circle := s.CreateCircle(center, c.diameter/2)
	dim := sketch.NewDiameter(circle, c.diameter)
	s.AddConstraint(dim)
	// A circle with no parameter of its own keeps just its dimension.
	if c.expr != "" {
		if err := s.Bind(dim, r.world.Params(), c.expr); err != nil {
			return nil, fmt.Errorf("%s: bind %q: %w", c.what, c.expr, err)
		}
	}
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("%s: solve: %w", c.what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("%s: verify: %w", c.what, err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("%s: expected one valid profile, got %d", c.what, len(profiles))
	}
	body, err := decad.Extrude(ctx, r.doc, s, profiles[0],
		decad.Distance{D: units.Millimeters(c.length), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("%s: extrude: %w", c.what, err)
	}
	return body, nil
}

// addShaftSection draws the round shaft cross-section on s about its origin.
func (r *run) addShaftSection(s *sketch.Sketch) error {
	return r.addCircle(s, r.d.shaft, "shaft_diameter")
}

// dimensionOnArc creates a point at (x, y) on an arc's circle and fixes one of its coordinates from the sketch
// origin. normal is the direction from the arc's center to the point. The point moves along the circle at right
// angles to that direction, so x changes fastest where the direction is vertical and y where it is horizontal;
// the coordinate that changes faster is the one to fix, which leaves no tangent, free direction.
func dimensionOnArc(s *sketch.Sketch, x, y, normal float64) *sketch.Point {
	point := s.CreatePoint(x, y)
	if math.Abs(math.Sin(normal)) >= math.Abs(math.Cos(normal)) {
		s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), point, x))
		return point
	}
	s.AddConstraint(sketch.NewVerticalDistance(s.Origin(), point, y))
	return point
}

// extrudePlate sketches the carrier plate and extrudes it from z=0 up the carrier thickness. The outline is the
// convex hull of the n pin bosses, a rounded polygon: for each pair of neighbouring pins a straight line
// tangent to both boss circles on the outside, and around each pin an arc of its boss circle between the two
// lines that meet there. The shaft section is a hole through the middle. The plate is solid between the pins, so
// it takes the sideways load the planets put on their pins.
func (r *run) extrudePlate(ctx context.Context) (*decad.Body, error) {
	p, d := r.p, r.d
	n := p.PlanetCount
	radius := d.boss / 2
	s, err := r.world.CreateSketch(r.world.XY())
	if err != nil {
		return nil, fmt.Errorf("carrier plate: sketch: %w", err)
	}
	// pin returns the center of pin k.
	pin := func(k int) (float64, float64) {
		phi := PlanetOrbitAngle(p, k)
		return d.orbit * math.Cos(phi), d.orbit * math.Sin(phi)
	}
	// Edge k runs from pin k to pin k+1; its outward normal points at angle phi_k + pi/n.
	start := make([]*sketch.Point, n)
	end := make([]*sketch.Point, n)
	centers := make([]*sketch.Point, n)
	for k := range n {
		x, y := pin(k)
		centers[k] = sketchcheck.DimensionPoint(s, x, y)
	}
	for k := range n {
		normal := PlanetOrbitAngle(p, k) + math.Pi/float64(n)
		dx, dy := radius*math.Cos(normal), radius*math.Sin(normal)
		ax, ay := pin(k)
		bx, by := pin((k + 1) % n)
		// Each arc has one fully dimensioned end, the end of an edge, and one end its radius relation fixes. The
		// one coordinate given to the second end must change as the point moves around the arc; at an arc's
		// leftmost or rightmost point the x coordinate does not, so that end is dimensioned in y instead.
		end[k] = sketchcheck.DimensionPoint(s, bx+dx, by+dy)
		start[k] = dimensionOnArc(s, ax+dx, ay+dy, normal)
	}
	for k := range n {
		next := (k + 1) % n
		s.CreateLine(start[k], end[k])
		s.CreateArc(centers[next], end[k], start[next])
	}
	profile, err := profileWithHoles(ctx, s, 0, "carrier plate")
	if err != nil {
		return nil, err
	}
	plate, err := decad.Extrude(ctx, r.doc, s, profile,
		decad.Distance{D: units.Millimeters(p.CarrierThickness), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("carrier plate: extrude: %w", err)
	}
	return plate, nil
}

// buildCarrier builds one carrier about the local origin: the triangular plate from z=0 to the carrier
// thickness with, for each planet, a D-shaped socket through the plate that takes the thick part of a separate pin
// as a press fit, and a D-shaped centre hole that takes the thick part of a link or output shaft as a press fit.
// The pins and the shafts are separate bodies, see buildPin and buildSteppedShaft. The carrier prints plate-down.
func (r *run) buildCarrier(ctx context.Context) (*decad.Body, error) {
	p, d := r.p, r.d
	plate, err := r.extrudePlate(ctx)
	if err != nil {
		return nil, err
	}
	// One socket is cut by a pattern of copies of one tool, so the plate takes two booleans however many
	// planets it has. The tool starts a millimeter below the plate's face and ends a millimeter above the other,
	// so the cut crosses both.
	socket, err := r.extrudeDSection(ctx, dSection{
		what: "carrier pin socket", z: -1, length: p.CarrierThickness + 2,
		x: d.orbit, radius: d.pinCollar/2 + pressFitClearance, depth: p.FlatDepth,
	})
	if err != nil {
		return nil, err
	}
	sockets, err := decad.PatternCopies(ctx, socket, decad.CircularPattern{
		Center: r3.NewVec(0, 0, 0), Axis: r3.NewVec(0, 0, 1), Count: p.PlanetCount,
	})
	if err != nil {
		return nil, fmt.Errorf("carrier sockets: pattern: %w", err)
	}
	tool, err := decad.Union(ctx, socket, sockets...)
	if err != nil {
		return nil, fmt.Errorf("carrier: join sockets: %w", err)
	}
	carrier, err := decad.Cut(ctx, plate, tool)
	if err != nil {
		return nil, fmt.Errorf("carrier: cut sockets: %w", err)
	}
	// The centre hole is cut last, so the plate keeps a tooth the full plate thickness where the shaft has its
	// flat.
	hole, err := r.extrudeDSection(ctx, dSection{
		what: "carrier shaft hole", z: -1, length: p.CarrierThickness + 2,
		radius: d.collar/2 + pressFitClearance, depth: p.FlatDepth,
	})
	if err != nil {
		return nil, err
	}
	if carrier, err = decad.Cut(ctx, carrier, hole); err != nil {
		return nil, fmt.Errorf("carrier: cut shaft hole: %w", err)
	}
	if !carrier.IsSolid() {
		return nil, errors.New("carrier: result is not a solid")
	}
	// The plate's outline loses its edge on both flat faces.
	if carrier, err = r.finishFacesEdges(ctx, carrier, "carrier", []float64{0, p.CarrierThickness}, outerLoop); err != nil {
		return nil, err
	}
	return carrier, nil
}

// buildPin builds one carrier pin about its own axis, in the carrier's frame: from the bottom, a thin rod down to
// z=-(gear thickness + gap) that the planet's bearing sits on, and a D-shaped thick part of the pin collar diameter
// that starts pinShoulderClearance above the bearing's top face, passes through the plate's socket and ends level
// with the plate's upper face. The thick part fits the socket with pressFitClearance all round. The rod overlaps the
// thick part, so the union crosses faces instead of sharing them. The pin prints standing on its thick end.
func (r *run) buildPin(ctx context.Context) (*decad.Body, error) {
	p, d := r.p, r.d
	hang := p.GearThickness + p.StageGap
	thickBottom := -p.StageGap + planetBearingLift + pinShoulderClearance
	rod, err := r.extrudeDisc(ctx, disc{
		what: "carrier pin rod", z: -hang, length: hang + thickBottom + stepOverlap,
		diameter: d.pin, expr: "pin_diameter",
	})
	if err != nil {
		return nil, err
	}
	thick, err := r.extrudeDSection(ctx, dSection{
		what: "carrier pin thick part", z: thickBottom, length: p.CarrierThickness - thickBottom,
		radius: d.pinCollar / 2, depth: p.FlatDepth, expr: "pin_collar_diameter",
	})
	if err != nil {
		return nil, err
	}
	pin, err := decad.Union(ctx, rod, thick)
	if err != nil {
		return nil, fmt.Errorf("carrier pin: join rod and thick part: %w", err)
	}
	if !pin.IsSolid() {
		return nil, errors.New("carrier pin: result is not a solid")
	}
	return pin, nil
}

// extrudeShaft extrudes the round shaft cross-section from z up length.
func (r *run) extrudeShaft(ctx context.Context, what string, z, length float64) (*decad.Body, error) {
	plane, err := r.planeAt(z)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	s, err := r.world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("%s: sketch: %w", what, err)
	}
	if err := r.addShaftSection(s); err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	profile, err := profileWithHoles(ctx, s, 0, what)
	if err != nil {
		return nil, err
	}
	body, err := decad.Extrude(ctx, r.doc, s, profile,
		decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("%s: extrude: %w", what, err)
	}
	return body, nil
}

// extrudeFlatBox extrudes the box that cuts the shaft's flat from z=bottom to z=top: from FlatDepth inside the
// shaft's +X edge to a millimeter beyond it, and a millimeter wider than the shaft on each side.
func (r *run) extrudeFlatBox(ctx context.Context, what string, bottom, top float64) (*decad.Body, error) {
	plane, err := r.planeAt(bottom)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	s, err := r.world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("%s: sketch: %w", what, err)
	}
	x0 := r.d.shaft/2 - r.p.FlatDepth
	x1 := r.d.shaft/2 + 1
	half := r.d.shaft/2 + 1
	box := s.CreateRectangle(x0, -half, x1, half)
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), box.A, x0),
		sketch.NewVerticalDistance(s.Origin(), box.A, -half),
		sketch.NewHorizontalDistance(box.A, box.B, x1-x0),
		sketch.NewVerticalDistance(box.B, box.C, 2*half),
	)
	profile, err := profileWithHoles(ctx, s, 0, what)
	if err != nil {
		return nil, err
	}
	body, err := decad.Extrude(ctx, r.doc, s, profile,
		decad.Distance{D: units.Millimeters(top - bottom), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("%s: extrude: %w", what, err)
	}
	return body, nil
}

// span is a range of heights, from lo up to hi.
type span struct{ lo, hi float64 }

// flatCut cuts the flat over each of the spans out of body, which is the shaft or, for a bore, a tool shaped like
// the shaft. With FlatDepth zero or no spans it returns body unchanged. The flats of every span join into one tool, so the
// body takes one cut however many spans there are.
func (r *run) flatCut(ctx context.Context, what string, body *decad.Body, spans ...span) (*decad.Body, error) {
	if r.p.FlatDepth == 0 || len(spans) == 0 {
		return body, nil
	}
	var tools []*decad.Body
	for _, sp := range spans {
		box, err := r.extrudeFlatBox(ctx, what+" flat", sp.lo, sp.hi)
		if err != nil {
			return nil, err
		}
		tools = append(tools, box)
	}
	tool := tools[0]
	if len(tools) > 1 {
		var err error
		if tool, err = decad.Union(ctx, tools[0], tools[1:]...); err != nil {
			return nil, fmt.Errorf("%s: join flats: %w", what, err)
		}
	}
	cut, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("%s: cut flat: %w", what, err)
	}
	if !cut.IsSolid() {
		return nil, fmt.Errorf("%s: result is not a solid", what)
	}
	return cut, nil
}

// dSection describes a D-shaped extrusion in the local frame: a circle of the given radius about (x, 0) with a flat
// depth inside its rim on the +X side.
type dSection struct {
	what          string
	z             float64 // height of the sketch plane
	length        float64 // extrusion length along +Z
	x             float64 // center's X
	radius, depth float64
	expr          string // table expression bound to the diameter of the round section when depth is zero
}

// extrudeDSection extrudes the D section from z up length. The outline is a long arc about the center, from the
// flat's upper end round to its lower end, and the line that closes it across the flat.
func (r *run) extrudeDSection(ctx context.Context, c dSection) (*decad.Body, error) {
	// With no flat the section is the round one.
	if c.depth == 0 {
		return r.extrudeDisc(ctx, disc{what: c.what, z: c.z, length: c.length, x: c.x, diameter: 2 * c.radius, expr: c.expr})
	}
	plane, err := r.planeAt(c.z)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", c.what, err)
	}
	s, err := r.world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("%s: sketch: %w", c.what, err)
	}
	flat := c.radius - c.depth
	half := math.Sqrt(c.radius*c.radius - flat*flat)
	center := s.Origin()
	if c.x != 0 {
		center = sketchcheck.DimensionPoint(s, c.x, 0)
	}
	// The arc runs counterclockwise from the flat's upper end, fully dimensioned, to its lower end, dimensioned by
	// the one coordinate that changes along the arc there.
	upper := sketchcheck.DimensionPoint(s, c.x+flat, half)
	lower := dimensionOnArc(s, c.x+flat, -half, -math.Atan2(half, flat))
	s.CreateArc(center, upper, lower)
	s.CreateLine(lower, upper)
	profile, err := profileWithHoles(ctx, s, 0, c.what)
	if err != nil {
		return nil, err
	}
	body, err := decad.Extrude(ctx, r.doc, s, profile,
		decad.Distance{D: units.Millimeters(c.length), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("%s: extrude: %w", c.what, err)
	}
	return body, nil
}

// extrudeFlatTool extrudes a bore tool with pressFitClearance around the shaft, from z up length. Its round arc is
// pressFitClearance outside the shaft and its flat lies pressFitClearance outside the shaft's flat. With FlatDepth zero the
// tool is the round shaft.
func (r *run) extrudeFlatTool(ctx context.Context, what string, z, length float64) (*decad.Body, error) {
	if r.p.FlatDepth == 0 {
		return r.extrudeDisc(ctx, disc{
			what: what, z: z, length: length, diameter: r.d.shaftBore, expr: "shaft_bore_diameter",
		})
	}
	return r.extrudeDSection(ctx, dSection{
		what: what, z: z, length: length, radius: r.d.shaft/2 + pressFitClearance, depth: r.p.FlatDepth,
	})
}

// buildSteppedShaft builds a carrier shaft about the local origin, its bottom on the carrier's lower face: a D thick
// part of the shaft collar diameter, with the flat FlatDepth deep on +X, from z=0 up to thickTop, which is pressed
// through the carrier's hole, and a round rod of the shaft diameter from stepOverlap below thickTop up to top, joined
// in one union, then a flat cut over each of the flats' spans.
func (r *run) buildSteppedShaft(ctx context.Context, what string, thickTop, top float64, flats ...span) (*decad.Body, error) {
	thick, err := r.extrudeDSection(ctx, dSection{
		what: what + " thick part", z: 0, length: thickTop,
		radius: r.d.collar / 2, depth: r.p.FlatDepth, expr: "shaft_collar_diameter",
	})
	if err != nil {
		return nil, err
	}
	rod, err := r.extrudeShaft(ctx, what, thickTop-stepOverlap, top-thickTop+stepOverlap)
	if err != nil {
		return nil, err
	}
	shaft, err := decad.Union(ctx, thick, rod)
	if err != nil {
		return nil, fmt.Errorf("%s: join thick part and rod: %w", what, err)
	}
	if !shaft.IsSolid() {
		return nil, fmt.Errorf("%s: result is not a solid", what)
	}
	return r.flatCut(ctx, what, shaft, flats...)
}

// buildShaft builds the input shaft about the local origin: a full round cylinder from z=bottom to z=top with,
// when collarTop is above collarBottom, a round collar of the shaft collar diameter between those heights, and a
// flat over each of the flats' spans. A collar sits between a bearing's inner race and the part next to it and
// takes the place of a washer. The shaft is round where a bearing sits, so the flat lies only where a sun
// sits, and stops short of the collar. The collar lies strictly inside the rod's length, so the union
// crosses the rod's faces instead of sharing them.
func (r *run) buildShaft(ctx context.Context, what string, bottom, top, collarBottom, collarTop float64, flats ...span) (*decad.Body, error) {
	shaft, err := r.extrudeShaft(ctx, what, bottom, top-bottom)
	if err != nil {
		return nil, err
	}
	if collarTop > collarBottom {
		collar, err := r.extrudeDisc(ctx, disc{
			what: what + " collar", z: collarBottom, length: collarTop - collarBottom,
			diameter: r.d.collar, expr: "shaft_collar_diameter",
		})
		if err != nil {
			return nil, err
		}
		if shaft, err = decad.Union(ctx, shaft, collar); err != nil {
			return nil, fmt.Errorf("%s: join rod and collar: %w", what, err)
		}
		if !shaft.IsSolid() {
			return nil, fmt.Errorf("%s: result is not a solid", what)
		}
	}
	return r.flatCut(ctx, what, shaft, flats...)
}
