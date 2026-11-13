package planetarygearbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

func externalGearWithRootClearance(module, rootMargin float64) spurgear.Variant {
	return spurgear.Variant{Define: func(t *param.Table) error {
		if err := t.SetValue("gear_root_clearance", units.Millimeters(gearRootClearance(module))); err != nil {
			return fmt.Errorf("define gear root clearance: %w", err)
		}
		if err := t.SetValue("gear_root_margin", units.Millimeters(rootMargin)); err != nil {
			return fmt.Errorf("define gear root margin: %w", err)
		}
		if err := t.SetValue("gear_tip_measurement_margin", units.Millimeters(gearRootMeasurementMargin)); err != nil {
			return fmt.Errorf("define gear tip measurement margin: %w", err)
		}
		if err := t.SetExpr("root_radius", "pitch_radius - module - gear_root_clearance - gear_root_margin", units.Millimeter); err != nil {
			return fmt.Errorf("define gear root radius: %w", err)
		}
		if err := t.SetExpr("tip_radius", "pitch_radius + module - gear_tip_measurement_margin", units.Millimeter); err != nil {
			return fmt.Errorf("define gear tip radius: %w", err)
		}
		return nil
	}}
}

// buildSun builds the sun gear about the local origin, from z=0 to the gear thickness.
func (r *run) buildSun(ctx context.Context) (*decad.Body, error) {
	p := r.p
	// A D-shaped bore is cut after the gear is built; a round one is the spur gear's own bore.
	bore := r.d.shaftBore
	if p.FlatDepth > 0 {
		bore = 0
	}
	sun, err := externalGearWithRootClearance(p.Module, 0).Build(ctx, r.doc, spurgear.Params{
		Module: p.Module, Teeth: p.SunTeeth, PressureAngle: p.PressureAngle,
		Thickness: p.GearThickness, Samples: p.Samples, BoreDiameter: bore, Backlash: p.Backlash,
	})
	if err != nil {
		return nil, fmt.Errorf("sun gear: %w", err)
	}
	if p.FlatDepth == 0 {
		return sun, nil
	}
	// The bore keeps a tooth for the shaft's flat.
	tool, err := r.extrudeFlatTool(ctx, "sun bore tool", -1, p.GearThickness+2)
	if err != nil {
		return nil, err
	}
	bored, err := decad.Cut(ctx, sun, tool)
	if err != nil {
		return nil, fmt.Errorf("sun gear: cut D-shaped bore: %w", err)
	}
	return bored, nil
}

// buildPlanet builds a planet gear about the local origin, from z=0 to the gear thickness. It has a
// pocket for its bearing open on the upper face, deep as the bearing is wide, and a lip below the pocket
// whose bore is smaller than the bearing so the bearing cannot fall out.
func (r *run) buildPlanet(ctx context.Context) (*decad.Body, error) {
	p, d := r.p, r.d
	planet, err := externalGearWithRootClearance(p.Module, gearRootMeasurementMargin).Build(ctx, r.doc, spurgear.Params{
		Module: p.Module, Teeth: p.PlanetTeeth, PressureAngle: p.PressureAngle,
		Thickness: p.GearThickness, Samples: p.Samples, BoreDiameter: d.planetLip, Backlash: p.Backlash,
	})
	if err != nil {
		return nil, fmt.Errorf("planet gear: %w", err)
	}
	// The pocket tool starts at the top of the lip and reaches one millimeter past the upper face.
	tool, err := r.extrudeDisc(ctx, disc{
		what: "planet bearing pocket", z: p.GearThickness - planetBearing(p).Width,
		length: planetBearing(p).Width + 1, diameter: d.planetPocket, expr: "planet_pocket_diameter",
	})
	if err != nil {
		return nil, err
	}
	pocketed, err := decad.Cut(ctx, planet, tool)
	if err != nil {
		return nil, fmt.Errorf("planet gear: cut bearing pocket: %w", err)
	}
	if !pocketed.IsSolid() {
		return nil, errors.New("planet gear: result is not a solid")
	}
	return r.chamferSeat(ctx, pocketed, "planet bearing pocket", p.GearThickness, d.planetPocket)
}

// buildBearing builds the catalog bearing b about the local origin, from z=0 to its width. what names the bearing in
// errors.
func (r *run) buildBearing(ctx context.Context, what string, b ballbearing.Spec) (*decad.Body, error) {
	model, err := ballbearing.Build(ctx, r.doc, b.Params())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	return model.Body, nil
}

// ringTool is the spur pipeline with the ring's radii: the external gear whose teeth are the
// ring's tooth spaces. Its root circle is the ring's tip circle and its tip circle is the ring's
// root circle. It has no root fillet.
func ringTool(module float64) spurgear.Variant {
	return spurgear.Variant{Define: func(t *param.Table) error {
		if err := t.SetValue("gear_root_clearance", units.Millimeters(gearRootClearance(module))); err != nil {
			return fmt.Errorf("define gear root clearance: %w", err)
		}
		for _, e := range []struct{ name, expr string }{
			{"root_radius", "pitch_radius - module"},
			{"tip_radius", "pitch_radius + module + gear_root_clearance"},
			{"fillet_radius", "0"},
		} {
			if err := t.SetExpr(e.name, e.expr, units.Millimeter); err != nil {
				return fmt.Errorf("define parameter %s: %w", e.name, err)
			}
		}
		return nil
	}}
}

// ringMargin is how far the tooth-space tool extends past each end of the ring blank.
const ringMargin = 1

// buildRing builds the housing: an outer cylinder with the tooth spaces cut through it. It is
// built about the local origin from z=-StageGap to the ring length, with its tooth-space tool
// turned to the planets' phase.
func (r *run) buildRing(ctx context.Context) (*decad.Body, error) {
	p, d := r.p, r.d
	tool, err := ringTool(p.Module).Build(ctx, r.doc, spurgear.Params{
		Module: p.Module, Teeth: p.RingTeeth, PressureAngle: p.PressureAngle,
		Thickness: d.ringLength + 2*ringMargin, Samples: p.Samples,
		// A negative backlash thickens the tool's teeth, which widens the ring's tooth spaces.
		Backlash: -p.Backlash,
	})
	if err != nil {
		return nil, fmt.Errorf("ring tooth spaces: %w", err)
	}
	if angle := RingToothAngle(p); angle != 0 {
		turn, err := r3.Rotation(r3.NewVec(0, 0, 1), units.Radians(angle))
		if err != nil {
			return nil, fmt.Errorf("ring tooth phase: %w", err)
		}
		if tool, err = decad.Transform(ctx, tool, turn); err != nil {
			return nil, fmt.Errorf("turn ring tooth spaces: %w", err)
		}
	}
	blank, err := r.extrudeRingBlank(ctx)
	if err != nil {
		return nil, err
	}
	if blank, err = r.cutNutPockets(ctx, blank); err != nil {
		return nil, err
	}
	ring, err := decad.Cut(ctx, blank, tool)
	if err != nil {
		return nil, fmt.Errorf("cut ring tooth spaces: %w", err)
	}
	if !ring.IsSolid() {
		return nil, fmt.Errorf("ring is not a solid")
	}
	if ring, err = r.finishFacesEdges(ctx, ring, "ring", []float64{ringMargin, ringMargin + d.ringLength},
		outerLoop); err != nil {
		return nil, err
	}
	return ring, nil
}

// ringShift is the move from the ring's construction height to its place: the blank is built
// ringMargin above the origin and belongs StageGap below it.
func (r *run) ringShift() float64 { return -ringMargin - r.p.StageGap }
