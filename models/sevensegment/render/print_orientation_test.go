package render

// CI: isolated-tests

import (
	"context"
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentPrintOrientation checks that each slider lies on the bed in its documented print orientation: its
// lowest face is on z = 0 and spans the slider's length, and the part stands far lower than the 61.7 mm it is as exported.
func TestSevenSegmentPrintOrientation(t *testing.T) {
	s, err := buildSevenSegmentSet(context.Background())
	require.NoError(t, err)
	for i, v := range sevenPrintViews {
		for _, p := range s {
			if p.name != v.part {
				continue
			}
			placed, size, err := sevenPrintPose(p.mesh, v.axis)
			require.NoError(t, err, v.file)
			require.Less(t, size.Z, 25.0, v.file)
			require.Greater(t, size.Len(), 60.0, v.file)
			var onBed []r3.Vec
			for _, q := range placed.Vertices() {
				require.GreaterOrEqual(t, q.Z, -1e-6, v.file)
				if q.Z < 1e-6 {
					onBed = append(onBed, q)
				}
			}
			require.NotEmpty(t, onBed, v.file)
			lo, hi := onBed[0], onBed[0]
			for _, q := range onBed {
				lo = r3.Vec{X: min(lo.X, q.X), Y: min(lo.Y, q.Y)}
				hi = r3.Vec{X: max(hi.X, q.X), Y: max(hi.Y, q.Y)}
			}
			require.Greater(t, hi.Sub(lo).Len(), 55.0, "the bed face of %s spans the slider", v.file)
			_, err = sevenPrintScene(s, i)
			require.NoError(t, err, v.file)
		}
	}
}
