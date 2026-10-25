package bevelgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// BuildToeCone revolves the exact cone inside the blank's toe-dish surface.
// ctx and doc must not be nil. The cone is a cutting tool, not a finished tooth.
func BuildToeCone(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	return buildToeConeIn(ctx, doc, nil, localShaft, p)
}

// buildToeConeIn is BuildToeCone about the shaft of f, with its sketch placed in c; a nil c leaves it in the root
// container.
func buildToeConeIn(ctx context.Context, doc *decad.Document, c *decad.Container, f shaftFrame,
	p Params) (*decad.Body, error) {
	d, err := derive(p)
	if err != nil {
		return nil, err
	}
	blank, err := blankSection(p, d)
	if err != nil {
		return nil, err
	}
	toeRoot := blank[4]
	apexStation := toeRoot.station + toeRoot.radius*math.Tan(d.coneAngle)
	return buildEndCone(ctx, doc, c, f, d, apexStation, "toe cone")
}

// BuildHeelCone revolves the exact cone inside the blank's heel-cone surface.
// ctx and doc must not be nil. The cone is a cutting tool, not a finished tooth.
func BuildHeelCone(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	return buildHeelConeIn(ctx, doc, nil, localShaft, p)
}

// buildHeelConeIn is BuildHeelCone about the shaft of f, with its sketch placed in c; a nil c leaves it in the root
// container.
func buildHeelConeIn(ctx context.Context, doc *decad.Document, c *decad.Container, f shaftFrame,
	p Params) (*decad.Body, error) {
	d, err := derive(p)
	if err != nil {
		return nil, err
	}
	if _, err := blankSection(p, d); err != nil {
		return nil, err
	}
	apexStation := d.coneDist / math.Cos(d.coneAngle)
	return buildEndCone(ctx, doc, c, f, d, apexStation, "heel cone")
}

// endConeSection is the axial section of an end cone with its apex at apexStation: a triangle from the apex to a
// far plane behind the blank.
func endConeSection(d dimensions, apexStation float64, label string) ([]axialPoint, error) {
	farStation := -0.1 * d.coneDist
	farRadius := (apexStation - farStation) / math.Tan(d.coneAngle)
	if !finite(apexStation, farStation, farRadius) || apexStation <= farStation || farRadius <= 0 {
		return nil, fmt.Errorf("%s has no positive finite axial section", label)
	}
	return []axialPoint{
		{station: farStation, radius: 0},
		{station: farStation, radius: farRadius},
		{station: apexStation, radius: 0},
	}, nil
}

func buildEndCone(ctx context.Context, doc *decad.Document, c *decad.Container, f shaftFrame, d dimensions,
	apexStation float64, label string) (*decad.Body, error) {
	section, err := endConeSection(d, apexStation, label)
	if err != nil {
		return nil, err
	}
	return revolveAxialSectionIn(ctx, doc, c, f, section, label)
}

// buildShiftedEndCone revolves the toe or heel cutting cone moved by shift along the shaft. A positive
// shift moves the apex away from the toe and heel, so the heel cone grows and the toe cone grows too.
func buildShiftedEndCone(ctx context.Context, doc *decad.Document, p Params, heel bool,
	shift float64) (*decad.Body, error) {
	d, err := derive(p)
	if err != nil {
		return nil, err
	}
	blank, err := blankSection(p, d)
	if err != nil {
		return nil, err
	}
	if heel {
		return buildEndCone(ctx, doc, nil, localShaft, d, d.coneDist/math.Cos(d.coneAngle)+shift, "heel cone")
	}
	toeRoot := blank[4]
	return buildEndCone(ctx, doc, nil, localShaft, d, toeRoot.station+toeRoot.radius*math.Tan(d.coneAngle)+shift, "toe cone")
}
