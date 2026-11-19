package spinningwheel_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// The swept-volume test must accept a ring that is radially apart and refuse one that touches or is buried.
func TestSpinningWheelSweptMeshClearance(t *testing.T) {
	ring := func(bore, outer, width, y float64) surfaceTriangles {
		f, err := r3.NewFrame(r3.Vec{Y: y}, r3.Vec{X: 1}, r3.Vec{Z: -1})
		require.NoError(t, err)
		m, err := ballbearing.Build(t.Context(), decad.New(), ballbearing.Params{BoreDiameter: bore, OuterDiameter: outer, Width: width, Placement: &f})
		require.NoError(t, err)
		return tessellateForSweep(t, m.Body, .05)
	}
	member := ring(8, 22, 7, 0)
	for _, c := range []struct {
		name                  string
		bore, outer, width, y float64
		clear                 bool
	}{
		{"radially_apart", 40, 50, 5, 1, true},
		{"axially_apart", 8, 22, 1, 20, true},
		{"crossing_wall", 16, 30, 5, 1, false},
		{"buried_in_wall", 14, 16, 2, 2, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, why := sweptMeshClearance(t, member, ring(c.bore, c.outer, c.width, c.y), 0, 0)
			require.Equal(t, c.clear, why == "", why)
		})
	}
}
