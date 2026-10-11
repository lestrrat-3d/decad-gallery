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
	return buildEndCone(ctx, doc, d, apexStation, "toe cone")
}

// BuildHeelCone revolves the exact cone inside the blank's heel-cone surface.
// ctx and doc must not be nil. The cone is a cutting tool, not a finished tooth.
func BuildHeelCone(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	d, err := derive(p)
	if err != nil {
		return nil, err
	}
	if _, err := blankSection(p, d); err != nil {
		return nil, err
	}
	apexStation := d.coneDist / math.Cos(d.coneAngle)
	return buildEndCone(ctx, doc, d, apexStation, "heel cone")
}

func buildEndCone(ctx context.Context, doc *decad.Document, d dimensions,
	apexStation float64, label string) (*decad.Body, error) {
	farStation := -0.1 * d.coneDist
	farRadius := (apexStation - farStation) / math.Tan(d.coneAngle)
	if !finite(apexStation, farStation, farRadius) || apexStation <= farStation || farRadius <= 0 {
		return nil, fmt.Errorf("%s has no positive finite axial section", label)
	}
	section := []axialPoint{
		{station: farStation, radius: 0},
		{station: farStation, radius: farRadius},
		{station: apexStation, radius: 0},
	}
	return revolveAxialSection(ctx, doc, section, label)
}
