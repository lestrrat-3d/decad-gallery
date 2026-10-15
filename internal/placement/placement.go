// Package placement moves a finished gear body to its reference point and plane.
package placement

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

// Place moves body from the local frame to f. The local origin lands on f's origin (the reference
// point), the local X and Y axes land on f's U and V (the reference plane), and the local Z axis
// lands on f's normal U×V. A nil f leaves body where it is. ctx and body must not be nil.
//
// Gear models whose construction joins many bodies build in the local frame and call Place last.
// Booleans such as the tooth Union fail on bodies built at an offset (a z offset of 7 mm is
// enough), so the offset is applied only after the last boolean.
func Place(ctx context.Context, body *decad.Body, f *r3.Frame) (*decad.Body, error) {
	if f == nil {
		return body, nil
	}
	move, err := r3.FromFrame(*f)
	if err != nil {
		return nil, fmt.Errorf("placement frame: %w", err)
	}
	placed, err := decad.Transform(ctx, body, move)
	if err != nil {
		return nil, fmt.Errorf("place gear on the reference plane: %w", err)
	}
	return placed, nil
}
