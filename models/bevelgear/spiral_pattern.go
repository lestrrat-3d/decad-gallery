package bevelgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// patternSpiralTooth makes the source's full circular pattern before any join.
// The source-default member's shaft is the local X axis through the apex.
func patternSpiralTooth(ctx context.Context, trimmed *decad.Body, teeth int) ([]*decad.Body, error) {
	pattern := make([]*decad.Body, teeth)
	pattern[0] = trimmed
	for i := 1; i < teeth; i++ {
		angle := units.Radians(2 * math.Pi * float64(i) / float64(teeth))
		turn, err := r3.Rotation(r3.NewVec(1, 0, 0), angle)
		if err != nil {
			return nil, fmt.Errorf("rotate spiral tooth %d: %w", i+1, err)
		}
		pattern[i], err = decad.Copy(ctx, trimmed, decad.WithCopyPlacement(turn))
		if err != nil {
			return nil, fmt.Errorf("place spiral tooth %d: %w", i+1, err)
		}
	}
	return pattern, nil
}
