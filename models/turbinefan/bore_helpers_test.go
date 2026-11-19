package turbinefan_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// boreLoop returns the loop of o whose vertices all lie within limit of the shaft axis.
func boreLoop(t *testing.T, loops [][]r3.Vec, limit float64) []r3.Vec {
	t.Helper()
	var found [][]r3.Vec
	for _, l := range loops {
		inside := true
		for _, v := range l {
			inside = inside && math.Hypot(v.X, v.Y) <= limit
		}
		if inside {
			found = append(found, l)
		}
	}
	require.Len(t, found, 1, "one loop is the bore")
	return found[0]
}
