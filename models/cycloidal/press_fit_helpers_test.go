package cycloidal_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// loopRadii returns the smallest and largest distance from the drive axis of each loop of body's section at z.
func loopRadii(t *testing.T, body *decad.Body, z float64) (least, most []float64) {
	t.Helper()
	frame, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	outline, err := body.SectionOutline(t.Context(), frame, units.Millimeters(0.01))
	require.NoError(t, err)
	for _, loop := range outline.Loops {
		lo, hi := math.Inf(1), 0.0
		for _, p := range loop {
			r := math.Hypot(p.X, p.Y)
			lo, hi = math.Min(lo, r), math.Max(hi, r)
		}
		least, most = append(least, lo), append(most, hi)
	}
	return least, most
}
