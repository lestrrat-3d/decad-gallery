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

// extrudeSpider sketches the carrier's hub and arms as one outline and extrudes it up the carrier
// thickness. Each of the n arms runs from the hub to a round boss at a pin position, with two straight
// edges a half arm width either side of its axis. Walking counterclockwise, each arm contributes a
// line out along its lower edge, a long arc around its boss, and a line back along its upper edge;
// a short arc of the hub circle joins it to the next arm. The outline has 4n curves.
func (r *run) extrudeSpider(ctx context.Context) (*decad.Body, error) {
	p, d := r.p, r.d
	n := p.PlanetCount
	half := d.arm / 2
	if half >= d.hub/2 || half >= d.boss/2 {
		return nil, fmt.Errorf("carrier: arm width %v is not narrower than the hub %v and the boss %v", d.arm, d.hub, d.boss)
	}
	hubX := math.Sqrt(d.hub*d.hub/4 - half*half)
	bossX := d.orbit - math.Sqrt(d.boss*d.boss/4-half*half)
	if bossX <= hubX {
		return nil, errors.New("carrier: hub and boss circles meet")
	}
	if 2*math.Asin(half/(d.hub/2)) >= 2*math.Pi/float64(n) {
		return nil, fmt.Errorf("carrier: %d arms of width %v overlap on a hub of diameter %v", n, d.arm, d.hub)
	}
	s, err := r.world.CreateSketch(r.world.XY())
	if err != nil {
		return nil, fmt.Errorf("carrier: spider sketch: %w", err)
	}
	// at returns the point along an arm's axis at distance x and sideways offset y, for arm k.
	at := func(k int, x, y float64) (float64, float64) {
		sin, cos := math.Sincos(2 * math.Pi * float64(k) / float64(n))
		return x*cos - y*sin, x*sin + y*cos
	}
	type armPoints struct{ lower, bossLower, bossUpper, upper *sketch.Point }
	arms := make([]armPoints, n)
	centers := make([]*sketch.Point, n)
	for k := range n {
		x, y := at(k, hubX, -half)
		lower := sketchcheck.DimensionPoint(s, x, y)
		if k > 0 {
			lower = sketchcheck.DimensionArcEnd(s, x, y)
		}
		x, y = at(k, bossX, -half)
		bossLower := sketchcheck.DimensionPoint(s, x, y)
		x, y = at(k, bossX, half)
		bossUpper := sketchcheck.DimensionArcEnd(s, x, y)
		x, y = at(k, hubX, half)
		// Each hub arc has one fully dimensioned end and one end that its radius relation fixes: the next
		// arm's lower point is the second kind, except that the first arm's closes the last arc.
		upper := sketchcheck.DimensionPoint(s, x, y)
		if k == n-1 {
			upper = sketchcheck.DimensionArcEnd(s, x, y)
		}
		x, y = at(k, d.orbit, 0)
		centers[k] = sketchcheck.DimensionPoint(s, x, y)
		arms[k] = armPoints{lower, bossLower, bossUpper, upper}
	}
	for k := range n {
		a := arms[k]
		s.CreateLine(a.lower, a.bossLower)
		s.CreateArc(centers[k], a.bossLower, a.bossUpper)
		s.CreateLine(a.bossUpper, a.upper)
		next := arms[(k+1)%n].lower
		s.CreateArc(s.Origin(), a.upper, next)
	}
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("carrier: solve spider: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("carrier: verify spider: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid || len(profiles[0].Outer) != 4*n {
		return nil, fmt.Errorf("carrier: expected one valid %d-curve spider profile, got %d profiles", 4*n, len(profiles))
	}
	spider, err := decad.Extrude(ctx, r.doc, s, profiles[0],
		decad.Distance{D: units.Millimeters(p.CarrierThickness), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("carrier: extrude spider: %w", err)
	}
	return spider, nil
}

// buildCarrier builds one carrier about the local origin: one thin arm from a hub on the axis to each
// planet pin, one pin per planet hanging from the arm's end down to z=-(gear thickness + gap), and a stub
// shaft on the axis that ends at stubTop. The arms span z=0 to the carrier thickness. Pins and stub
// reach into the arms so the union overlaps.
func (r *run) buildCarrier(ctx context.Context, stubTop float64) (*decad.Body, error) {
	p, d := r.p, r.d
	if stubTop <= p.CarrierThickness/2 {
		return nil, errors.New("carrier: stub shaft does not clear the arms")
	}
	spider, err := r.extrudeSpider(ctx)
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
	pins, err := decad.PatternCopies(ctx, pin, decad.CircularPattern{
		Center: r3.NewVec(0, 0, 0), Axis: r3.NewVec(0, 0, 1), Count: p.PlanetCount,
	})
	if err != nil {
		return nil, fmt.Errorf("carrier pins: pattern: %w", err)
	}
	stub, err := r.extrudeDisc(ctx, disc{
		what: "carrier stub shaft", z: p.CarrierThickness / 2, length: stubTop - p.CarrierThickness/2,
		diameter: d.shaft, expr: "shaft_diameter",
	})
	if err != nil {
		return nil, err
	}
	parts := append([]*decad.Body{pin, stub}, pins...)
	carrier, err := decad.Union(ctx, spider, parts...)
	if err != nil {
		return nil, fmt.Errorf("carrier: join spider, pins and stub shaft: %w", err)
	}
	if !carrier.IsSolid() {
		return nil, errors.New("carrier: result is not a solid")
	}
	return carrier, nil
}
