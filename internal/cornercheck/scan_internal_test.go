package cornercheck

import (
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestScanIgnoresMicroZigzag feeds scanLoop the outline a mesh Boolean can leave: vertices micrometres apart whose
// segments double back. Their turns cancel, so the straight edge they sit on is not a corner.
func TestScanIgnoresMicroZigzag(t *testing.T) {
	loop := []r3.Vec{
		{X: 0, Y: 0}, {X: 10, Y: 0},
		{X: 10, Y: 5}, {X: 10, Y: 5.0004}, {X: 10, Y: 5.0001}, {X: 10, Y: 5.0006}, {X: 10, Y: 5.0002}, {X: 10, Y: 5.001},
		{X: 10, Y: 10}, {X: 0, Y: 10},
	}
	fs := scanLoop(loop, r3.Vec{Z: 1}, 0)
	require.Empty(t, fs, Report(fs, 5))
}
