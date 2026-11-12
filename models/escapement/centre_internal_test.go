package escapement

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCentreLine checks that the parts that set the time stand on one plumb line, x = 0, that the default pose hangs
// the pendulum at rest, and that the frame is mirror-symmetric about the line.
func TestCentreLine(t *testing.T) {
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	for name, p := range map[string]Pt{"O": g.O, "P": g.P, "S": g.S, "G": g.G} {
		require.InDelta(t, 0, p.X, 1e-9, name)
	}

	s, err := Motion(DefaultParams())
	require.NoError(t, err)
	require.InDelta(t, 0, s.AnchorAngle, 1e-12)
	require.InDelta(t, 0, s.PendulumAngle, 1e-12)

	for _, set := range [][]Pt{pillars, mountHoles} {
		for i := 0; i < len(set); i += 2 {
			require.InDelta(t, set[i].X, -set[i+1].X, 1e-9)
			require.InDelta(t, set[i].Y, set[i+1].Y, 1e-9)
		}
	}
	_, m, err := sharedDefault(t)
	require.NoError(t, err)
	for _, nb := range m.Named() {
		if nb.Name != BackPlateName && nb.Name != FrontPlateName {
			continue
		}
		b, err := nb.Body.Bounds()
		require.NoError(t, err)
		require.InDelta(t, -b.Min.X, b.Max.X, 2e-3, "%s is symmetric about the line", nb.Name)
	}
}
