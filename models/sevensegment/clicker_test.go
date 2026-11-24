package sevensegment_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/stretchr/testify/require"
)

// TestClickerBodies requires every click-indexer body to be one solid lump, and each tooth ring to have ten teeth.
func TestClickerBodies(t *testing.T) {
	t.Parallel()
	_, m := builtDefault(t)
	for name, b := range map[string]*decad.Body{"hub": m.Hub, "rotor": m.Rotor, "housing": m.Housing, "button": m.Button} {
		require.True(t, b.IsSolid(), name)
	}
	for name, lumps := range map[string]int{"hub": len(m.Hub.Lumps()), "rotor": len(m.Rotor.Lumps()),
		"housing": len(m.Housing.Lumps()), "button": len(m.Button.Lumps())} {
		require.Equal(t, 1, lumps, name)
	}
	require.Equal(t, sevensegment.ClickerSteps, teethAround(t, m.Rotor, -70.86-2, 7), "rotor inner teeth")
	require.Equal(t, sevensegment.ClickerSteps, teethAround(t, m.Rotor, -70.86-0.8, 13.4), "rotor outer teeth")
	require.Equal(t, sevensegment.ClickerSteps, teethAround(t, m.Housing, -72.5+0.8, 13.7), "crown teeth")
	require.Equal(t, sevensegment.ClickerSteps, teethAround(t, m.Button, -78.9+2, 7), "button teeth")
}

// TestClickerHeights reads the button's heights from the built bodies: the head stands 23.9 mm above the front wall at rest
// and ends flush with the housing's bezel at the bottom of its stroke.
func TestClickerHeights(t *testing.T) {
	t.Parallel()
	_, m := builtDefault(t)
	wall, err := m.FrontWall.Bounds()
	require.NoError(t, err)
	button, err := m.Button.Bounds()
	require.NoError(t, err)
	housing, err := m.Housing.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 23.9, wall.Min.Y-button.Min.Y, 1e-6, "the head's height above the front wall at rest")
	require.InDelta(t, 17.9, wall.Min.Y-housing.Min.Y, 1e-6, "the bezel's height above the front wall")
	require.InDelta(t, housing.Min.Y, button.Min.Y+sevensegment.ButtonStroke, 1e-6, "the head is flush with the bezel at the bottom")
}

// TestClickerPoseClearances moves the parts through a press and a release and requires the parts that are not mating teeth to
// keep their fits: the rotor to the hub's keys and walls, the button's lugs to the housing's slots.
func TestClickerPoseClearances(t *testing.T) {
	t.Parallel()
	_, m := builtDefault(t)
	type pose struct{ s, drop, turn float64 }
	var poses []pose
	for _, s := range []float64{0, 2, 4, 6} {
		d, tn := sevensegment.PressPose(s)
		poses = append(poses, pose{s, d, tn})
	}
	for _, s := range []float64{4, 2, 0.5} {
		d, tn := sevensegment.ReleasePose(s)
		poses = append(poses, pose{s, d, tn})
	}
	for _, p := range poses {
		hub := movedMesh(t, m.Hub, 0.01, p.turn, 0)
		rotor := movedMesh(t, m.Rotor, 0.1, p.turn, p.drop)
		housing := movedMesh(t, m.Housing, 0.1, 0, 0)
		button := movedMesh(t, m.Button, 0.1, 0, p.s)
		sep, err := hub.ExceedsClearance(rotor, 0)
		require.NoError(t, err)
		require.True(t, sep, "hub and rotor at button drop %.1f", p.s)
		sep, err = button.ExceedsClearance(housing, 0)
		require.NoError(t, err)
		require.True(t, sep, "button and housing at button drop %.1f", p.s)
	}
}
