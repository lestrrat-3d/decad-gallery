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
	if err := s.Bind(dim, r.world.Params(), c.expr); err != nil {
		return nil, fmt.Errorf("%s: bind %q: %w", c.what, c.expr, err)
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
// thickness with the shaft section cut through its middle, and for each planet one pin hanging from the plate
// down to z=-(gear thickness + gap). Each pin has a collar of the shaft collar diameter under the plate, as tall
// as the gap, which stands in for a washer: it rests on the bearing's inner race and keeps the planet's
// bearing clear of the plate. Everything stands on one face of the plate, so the carrier prints plate-down.
// Pins and collars reach into the plate so the union overlaps. The shafts are separate bodies.
func (r *run) buildCarrier(ctx context.Context) (*decad.Body, error) {
	p, d := r.p, r.d
	plate, err := r.extrudePlate(ctx)
	if err != nil {
		return nil, err
	}
	hang := p.GearThickness + p.StageGap
	pin, err := r.extrudeDisc(ctx, disc{
		what: "carrier pin", z: -hang, length: hang + p.CarrierThickness/2,
		x: d.orbit, diameter: d.pin, expr: "pin_diameter",
	})
	if err != nil {
		return nil, err
	}
	// One pin with its collar is joined once and patterned, so the carrier needs two unions however many
	// planets it has, not one per pin and collar.
	if p.StageGap >= planetBearingLift+pinCollarClearance+pinCollarThickness {
		collar, err := r.extrudeDisc(ctx, disc{
			what: "carrier pin collar", z: -p.StageGap + planetBearingLift + pinCollarClearance,
			length: p.StageGap - planetBearingLift - pinCollarClearance + p.CarrierThickness/2,
			x:      d.orbit, diameter: d.collar, expr: "shaft_collar_diameter",
		})
		if err != nil {
			return nil, err
		}
		if pin, err = decad.Union(ctx, pin, collar); err != nil {
			return nil, fmt.Errorf("carrier: join pin and collar: %w", err)
		}
	}
	pins, err := decad.PatternCopies(ctx, pin, decad.CircularPattern{
		Center: r3.NewVec(0, 0, 0), Axis: r3.NewVec(0, 0, 1), Count: p.PlanetCount,
	})
	if err != nil {
		return nil, fmt.Errorf("carrier pins: pattern: %w", err)
	}
	parts := append([]*decad.Body{pin}, pins...)
	carrier, err := decad.Union(ctx, plate, parts...)
	if err != nil {
		return nil, fmt.Errorf("carrier: join plate, pins and collars: %w", err)
	}
	// The shaft hole is cut last, so the plate keeps a tooth the full plate thickness where the shaft has its
	// groove.
	tool, err := r.extrudeKeyTool(ctx, "carrier shaft hole", -1, p.CarrierThickness+2)
	if err != nil {
		return nil, err
	}
	if carrier, err = decad.Cut(ctx, carrier, tool); err != nil {
		return nil, fmt.Errorf("carrier: cut shaft hole: %w", err)
	}
	if !carrier.IsSolid() {
		return nil, errors.New("carrier: result is not a solid")
	}
	// The plate's outline loses its edge on both flat faces.
	for _, z := range []float64{0, p.CarrierThickness} {
		if carrier, err = r.finishFaceEdges(ctx, carrier, "carrier", z, outerLoop); err != nil {
			return nil, err
		}
	}
	return carrier, nil
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

// extrudeKeyBox extrudes the keyway groove's box from z=bottom to z=top: KeyWidth wide about the X axis, from
// KeyDepth inside the shaft's +X edge to a millimeter beyond it.
func (r *run) extrudeKeyBox(ctx context.Context, what string, bottom, top float64) (*decad.Body, error) {
	plane, err := r.planeAt(bottom)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	s, err := r.world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("%s: sketch: %w", what, err)
	}
	x0 := r.d.shaft/2 - r.p.KeyDepth
	x1 := r.d.shaft/2 + 1
	half := r.p.KeyWidth / 2
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

// keyCut cuts the three keyway grooves over each of the spans out of body, which is the shaft or, for a bore, a
// tool shaped like the shaft. With KeyDepth zero it returns body unchanged. The grooves of every span join into
// one tool, so the body takes one cut however many spans there are.
func (r *run) keyCut(ctx context.Context, what string, body *decad.Body, spans ...span) (*decad.Body, error) {
	if r.p.KeyDepth == 0 {
		return body, nil
	}
	var tools []*decad.Body
	for _, sp := range spans {
		box, err := r.extrudeKeyBox(ctx, what+" keyway", sp.lo, sp.hi)
		if err != nil {
			return nil, err
		}
		// The other grooves are copies of the first, spread evenly round the axis.
		copies, err := decad.PatternCopies(ctx, box, decad.CircularPattern{
			Center: r3.NewVec(0, 0, 0), Axis: r3.NewVec(0, 0, 1), Count: keyCount,
		})
		if err != nil {
			return nil, fmt.Errorf("%s: pattern keyways: %w", what, err)
		}
		tools = append(tools, box)
		tools = append(tools, copies...)
	}
	tool, err := decad.Union(ctx, tools[0], tools[1:]...)
	if err != nil {
		return nil, fmt.Errorf("%s: join keyways: %w", what, err)
	}
	cut, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("%s: cut keyway: %w", what, err)
	}
	if !cut.IsSolid() {
		return nil, fmt.Errorf("%s: result is not a solid", what)
	}
	return cut, nil
}

// extrudeKeyTool extrudes a bore tool with fitClearance around the shaft, from z up length. Its round arc is
// fitClearance outside the shaft and its key teeth have the same clearance on their sides and floors. The outline
// is a long arc about the origin, from the groove's upper wall round to its lower wall, and three lines that
// run in along the lower wall, across the groove floor and out along the upper wall. With KeyDepth zero the
// tool is the round shaft.
func (r *run) extrudeKeyTool(ctx context.Context, what string, z, length float64) (*decad.Body, error) {
	if r.p.KeyDepth == 0 {
		return r.extrudeDisc(ctx, disc{
			what: what, z: z, length: length, diameter: r.d.shaftBore, expr: "shaft_bore_diameter",
		})
	}
	plane, err := r.planeAt(z)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	s, err := r.world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("%s: sketch: %w", what, err)
	}
	radius := r.d.shaft/2 + fitClearance
	half := (r.p.KeyWidth - 2*fitClearance) / 2
	floor := radius - r.p.KeyDepth
	wall := math.Sqrt(radius*radius - half*half)
	// Groove k lies at angle k*2pi/keyCount. Going counterclockwise the outline meets its lower wall first: a
	// line in from the rim to the floor, across the floor, and out along the upper wall, then an arc of the
	// circle to the next groove's lower wall. Each arc's start (the upper rim point) is fully dimensioned and its
	// end (the next lower rim point) by the one coordinate that changes along the arc there.
	rotate := func(angle, x, y float64) (float64, float64) {
		sin, cos := math.Sincos(angle)
		return x*cos - y*sin, x*sin + y*cos
	}
	spread := math.Asin(half / radius)
	upper := make([]*sketch.Point, keyCount)
	lower := make([]*sketch.Point, keyCount)
	floorUpper := make([]*sketch.Point, keyCount)
	floorLower := make([]*sketch.Point, keyCount)
	for k := range keyCount {
		angle := 2 * math.Pi * float64(k) / keyCount
		x, y := rotate(angle, wall, half)
		upper[k] = sketchcheck.DimensionPoint(s, x, y)
		x, y = rotate(angle, floor, half)
		floorUpper[k] = sketchcheck.DimensionPoint(s, x, y)
		x, y = rotate(angle, floor, -half)
		floorLower[k] = sketchcheck.DimensionPoint(s, x, y)
	}
	for k := range keyCount {
		angle := 2 * math.Pi * float64(k) / keyCount
		x, y := rotate(angle, wall, -half)
		lower[k] = dimensionOnArc(s, x, y, angle-spread)
	}
	for k := range keyCount {
		next := (k + 1) % keyCount
		s.CreateArc(s.Origin(), upper[k], lower[next])
		s.CreateLine(lower[k], floorLower[k])
		s.CreateLine(floorLower[k], floorUpper[k])
		s.CreateLine(floorUpper[k], upper[k])
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

// buildShaft builds a separate shaft about the local origin: a full round cylinder from z=bottom to z=top with,
// when collarTop is above collarBottom, a round collar of the shaft collar diameter between those heights, and a
// keyway grooves over each of the grooves' spans. A collar sits between a bearing's inner race and the part next to
// it and takes the place of a washer. The shaft is round where a bearing sits, so the groove lies only where a
// sun or carrier sits, and stops short of the collar. The collar lies strictly inside the rod's length, so the
// union crosses the rod's faces instead of sharing them.
func (r *run) buildShaft(ctx context.Context, what string, bottom, top, collarBottom, collarTop float64, grooves ...span) (*decad.Body, error) {
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
	return r.keyCut(ctx, what, shaft, grooves...)
}
