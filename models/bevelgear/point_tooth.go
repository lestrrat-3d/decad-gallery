package bevelgear

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// BuildPointTooth lofts one fitted back-cone tooth from the exact common apex.
// ctx and doc must not be nil. The tooth is untrimmed and has no bore.
func BuildPointTooth(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	world := sketch.NewWorld()
	s, profile, err := BuildBackConeToothProfile(ctx, world, p)
	if err != nil {
		return nil, fmt.Errorf("build back-cone tooth profile: %w", err)
	}
	tooth, err := decad.LoftFromPoint(ctx, doc, r3.NewVec(0, 0, 0), s, profile)
	if err != nil {
		return nil, fmt.Errorf("loft tooth from common apex: %w", err)
	}
	return tooth, nil
}
