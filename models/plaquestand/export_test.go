package plaquestand

import "context"

// ProfileConditioning returns the conditioning sketch's Verify reports for a bracket's profile sketch at p, which
// TestPlaqueStandProfileConditioning keeps above sketch's near-singular gate.
func ProfileConditioning(ctx context.Context, p Params) (float64, error) {
	dm, err := p.Derive()
	if err != nil {
		return 0, err
	}
	s, _, err := profileSketch(ctx, dm, 0)
	if err != nil {
		return 0, err
	}
	return s.Verify(ctx).Conditioning, nil
}
