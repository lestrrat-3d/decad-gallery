package planetarygearbox_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// turned returns the angle that name's transform turns a probe point about the gearbox axis by, in radians in
// (-pi, pi], and whether the body moves at all.
func turned(t *testing.T, m *planetarygearbox.Motion, name string, input float64) (float64, bool) {
	t.Helper()
	xf, moves, err := m.Transform(name, input)
	require.NoError(t, err, name)
	if !moves {
		return 0, false
	}
	probe := r3.NewVec(1, 0, 0)
	moved := xf.Apply(probe)
	return math.Atan2(moved.Y, moved.X), true
}
