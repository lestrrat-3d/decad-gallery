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

// addShaftSection draws the shaft cross-section on s about its origin: a circle of the shaft diameter, or with a
// nonzero ShaftFlat a D, a circle cut by a line ShaftFlat inside its +X edge, so the flat faces the sun's tooth
// zero. A shaft, its sun's bore and its carrier's hole all use it, so each fits the others. The D has two
// curves, a long arc about the origin and the line that closes it. The line's upper end is dimensioned from the
// origin and its lower end by one coordinate, so the arc's radius relation fixes the rest.
func (r *run) addShaftSection(s *sketch.Sketch) error {
	if r.p.ShaftFlat == 0 {
		return r.addCircle(s, r.d.shaft, "shaft_diameter")
	}
	radius := r.d.shaft / 2
	flat := radius - r.p.ShaftFlat
	half := math.Sqrt(radius*radius - flat*flat)
	upper := sketchcheck.DimensionPoint(s, flat, half)
	lower := sketchcheck.DimensionArcEnd(s, flat, -half)
	s.CreateArc(s.Origin(), upper, lower)
	s.CreateLine(lower, upper)
	return nil
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
	if err := r.addShaftSection(s); err != nil {
		return nil, fmt.Errorf("carrier plate: %w", err)
	}
	profile, err := profileWithHoles(ctx, s, 1, "carrier plate")
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
	if p.StageGap > 0 {
		collar, err := r.extrudeDisc(ctx, disc{
			what: "carrier pin collar", z: -p.StageGap, length: p.StageGap + p.CarrierThickness/2,
			x: d.orbit, diameter: d.collar, expr: "shaft_collar_diameter",
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
	if !carrier.IsSolid() {
		return nil, errors.New("carrier: result is not a solid")
	}
	return carrier, nil
}

// extrudeShaft extrudes the shaft cross-section from z up length.
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
	want := 2
	if r.p.ShaftFlat == 0 {
		want = 1
	}
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("%s: solve: %w", what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("%s: verify: %w", what, err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid || len(profiles[0].Outer) != want {
		return nil, fmt.Errorf("%s: expected one valid %d-curve profile, got %d profiles", what, want, len(profiles))
	}
	body, err := decad.Extrude(ctx, r.doc, s, profiles[0],
		decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("%s: extrude: %w", what, err)
	}
	return body, nil
}

// buildShaft builds a separate shaft about the local origin: the shaft section from z=bottom to z=top and,
// when collarTop is above collarBottom, a round collar of the shaft collar diameter between those heights. A
// collar sits between a bearing's inner race and the part next to it and takes the place of a washer. It lies
// strictly inside the rod's length, so the union crosses the rod's faces instead of sharing them.
func (r *run) buildShaft(ctx context.Context, what string, bottom, top, collarBottom, collarTop float64) (*decad.Body, error) {
	rod, err := r.extrudeShaft(ctx, what, bottom, top-bottom)
	if err != nil {
		return nil, err
	}
	if collarTop <= collarBottom {
		return rod, nil
	}
	collar, err := r.extrudeDisc(ctx, disc{
		what: what + " collar", z: collarBottom, length: collarTop - collarBottom,
		diameter: r.d.collar, expr: "shaft_collar_diameter",
	})
	if err != nil {
		return nil, err
	}
	shaft, err := decad.Union(ctx, rod, collar)
	if err != nil {
		return nil, fmt.Errorf("%s: join rod and collar: %w", what, err)
	}
	if !shaft.IsSolid() {
		return nil, fmt.Errorf("%s: result is not a solid", what)
	}
	return shaft, nil
}
