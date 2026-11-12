package escapement

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/gearroot"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
)

// gearVariant returns the spur gear variant of the movement: the root is deepened by the gear-train clearance rule
// and both the root and tip carry the 0.02 mm measurement margin. The pinion and the great wheel share it.
func gearVariant() spurgear.Variant {
	return spurgear.Variant{Define: gearroot.Define(gearroot.Clearance(Module), GearRootMargin)}
}

// buildGear builds a spur gear of teeth teeth and thickness at height z with tooth 0 toward the math angle phase.
func (b *builder) buildGear(what string, teeth int, thickness, z, phase, bore float64) (*decad.Body, error) {
	f, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(math.Cos(phase), math.Sin(phase), 0),
		r3.NewVec(-math.Sin(phase), math.Cos(phase), 0))
	if err != nil {
		return nil, fmt.Errorf("%s: placement: %w", what, err)
	}
	gear, err := gearVariant().Build(b.ctx, b.doc, spurgear.Params{
		Module: Module, Teeth: teeth, PressureAngle: PressureAngle, Thickness: thickness, Samples: 15,
		Backlash: b.g.Params.Backlash, BoreDiameter: bore, Placement: &f,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	return gear, nil
}
