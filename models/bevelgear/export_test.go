package bevelgear

import (
	"context"

	"github.com/lestrrat-3d/sketch"
)

// BuildSpiralMemberSerialForTest builds a spiral member and returns the overshoot, in modules, it used.
var BuildSpiralMemberSerialForTest = buildSpiralMemberSerialTrace

// BuildFilletedToothProfileForTest draws the straight member's back-cone tooth, which carries the root fillet where
// the tooth has one.
func BuildFilletedToothProfileForTest(ctx context.Context, world *sketch.World, p Params) (*sketch.Sketch,
	*sketch.Profile, error) {
	p.rootFillet = true
	return buildBackConeToothProfile(ctx, world, nil, localShaft, p)
}
