package modelsketch

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/sketch"
)

// Verify requires s to be fully constrained: status FullyConstrained, zero degrees of freedom, no
// redundant constraints, and no conflicts. The constraints must already hold at the geometry the model
// placed, so Verify does not run the solver and cannot move that geometry. name identifies the sketch
// in the error. ctx and s must not be nil.
func Verify(ctx context.Context, name string, s *sketch.Sketch) error {
	r := s.Verify(ctx, sketch.WithoutConditioning())
	if !r.Solvable {
		return fmt.Errorf("%s does not satisfy its constraints: residual %g", name, r.Residual)
	}
	if r.Status == sketch.FullyConstrained && r.DOF == 0 && len(r.Redundant) == 0 && len(r.Conflicts) == 0 {
		return nil
	}
	return fmt.Errorf("%s is not fully constrained: status %s, DOF %d, %d redundant, %d conflicting",
		name, r.Status, r.DOF, len(r.Redundant), len(r.Conflicts))
}
