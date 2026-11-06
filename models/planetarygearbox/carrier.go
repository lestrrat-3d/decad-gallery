package planetarygearbox

import (
	"context"
	"errors"
	"fmt"

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

// extrudeDisc sketches one circle on the plane at disc.z and extrudes it up disc.length.
func (r *run) extrudeDisc(ctx context.Context, c disc) (*decad.Body, error) {
	frame, err := r3.NewFrame(r3.NewVec(0, 0, c.z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	if err != nil {
		return nil, fmt.Errorf("%s: frame: %w", c.what, err)
	}
	plane, err := r.world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("%s: plane: %w", c.what, err)
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

// buildCarrier builds one carrier about the local origin: the plate from z=0 to the carrier
// thickness, one pin per planet hanging from z=-(gear thickness + gap), and a stub shaft on
// the axis that ends at stubTop. Pins and stub reach into the plate so the union overlaps.
func (r *run) buildCarrier(ctx context.Context, stubTop float64) (*decad.Body, error) {
	p, d := r.p, r.d
	if stubTop <= p.CarrierThickness/2 {
		return nil, errors.New("carrier: stub shaft does not clear the plate")
	}
	plate, err := r.extrudeDisc(ctx, disc{
		what: "carrier plate", length: p.CarrierThickness, diameter: 2 * d.carrier,
		expr: "2 * carrier_radius",
	})
	if err != nil {
		return nil, err
	}
	hang := p.GearThickness + p.StageGap
	pin, err := r.extrudeDisc(ctx, disc{
		what: "carrier pin", z: -hang, length: hang + p.CarrierThickness/2,
		x: d.orbit, diameter: p.PinDiameter, expr: "pin_diameter",
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
		diameter: p.ShaftDiameter, expr: "shaft_diameter",
	})
	if err != nil {
		return nil, err
	}
	carrier, err := decad.Union(ctx, plate, append([]*decad.Body{pin, stub}, pins...)...)
	if err != nil {
		return nil, fmt.Errorf("carrier: join plate, pins and stub shaft: %w", err)
	}
	if !carrier.IsSolid() {
		return nil, errors.New("carrier: result is not a solid")
	}
	return carrier, nil
}
