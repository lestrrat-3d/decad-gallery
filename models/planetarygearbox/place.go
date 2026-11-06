package planetarygearbox

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// frameTransform returns the motion from the local frame (origin, +X, +Y, +Z) onto the
// placement frame, or the identity for a nil placement.
func frameTransform(p Params) (r3.Transform, error) {
	if p.Placement == nil {
		return r3.Identity(), nil
	}
	t, err := r3.FromFrame(*p.Placement)
	if err != nil {
		return r3.Transform{}, fmt.Errorf("placement frame: %w", err)
	}
	return t, nil
}

// pose turns a body by angle about the local axis, moves it to (x, y, z) and then onto the
// placement frame.
func pose(frame r3.Transform, angle, x, y, z float64) (r3.Transform, error) {
	turn, err := r3.Rotation(r3.NewVec(0, 0, 1), units.Radians(angle))
	if err != nil {
		return r3.Transform{}, fmt.Errorf("rotation: %w", err)
	}
	move, err := r3.Translation(r3.NewVec(x, y, z))
	if err != nil {
		return r3.Transform{}, fmt.Errorf("translation: %w", err)
	}
	local, err := turn.Then(move)
	if err != nil {
		return r3.Transform{}, fmt.Errorf("compose pose: %w", err)
	}
	placed, err := local.Then(frame)
	if err != nil {
		return r3.Transform{}, fmt.Errorf("compose placement: %w", err)
	}
	return placed, nil
}

// poseFlipped turns a body over about the local X axis, so its +Z face looks down, then moves it to height z
// and onto the placement frame.
func poseFlipped(frame r3.Transform, z float64) (r3.Transform, error) {
	turn, err := r3.Rotation(r3.NewVec(1, 0, 0), units.Radians(math.Pi))
	if err != nil {
		return r3.Transform{}, fmt.Errorf("rotation: %w", err)
	}
	move, err := r3.Translation(r3.NewVec(0, 0, z))
	if err != nil {
		return r3.Transform{}, fmt.Errorf("translation: %w", err)
	}
	local, err := turn.Then(move)
	if err != nil {
		return r3.Transform{}, fmt.Errorf("compose pose: %w", err)
	}
	placed, err := local.Then(frame)
	if err != nil {
		return r3.Transform{}, fmt.Errorf("compose placement: %w", err)
	}
	return placed, nil
}

// instances returns one body per transform. All but the last are copies of master;
// the last is master itself moved, so master leaves the document.
func instances(ctx context.Context, master *decad.Body, moves []r3.Transform) ([]*decad.Body, error) {
	out := make([]*decad.Body, 0, len(moves))
	for _, move := range moves[:len(moves)-1] {
		copied, err := decad.Copy(ctx, master, decad.WithCopyPlacement(move))
		if err != nil {
			return nil, fmt.Errorf("copy: %w", err)
		}
		out = append(out, copied)
	}
	last, err := decad.Transform(ctx, master, moves[len(moves)-1])
	if err != nil {
		return nil, fmt.Errorf("move: %w", err)
	}
	return append(out, last), nil
}
