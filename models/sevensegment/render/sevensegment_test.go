package render

// CI: isolated-tests

import (
	"context"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestSevenSegmentPresets(t *testing.T) {
	m := Model()
	for _, name := range []string{"hero", "top", "oblique45", "oblique30", "front", "side", "cutaway", "detent", "digits",
		"cams", "exploded", "exploded-camshaft"} {
		p, ok := m.Presets[name]
		require.True(t, ok, name)
		require.Equal(t, 1600, p.Width, name)
		require.Equal(t, 900, p.Height, name)
	}
	seen := map[string]bool{}
	for _, img := range sevenPartImages {
		require.False(t, seen[img.file], img.file)
		seen[img.file] = true
	}
	// 10 plain printed parts, 7 cams, 3 slider types, 2 rotor images and 4 rotor bodies, bearing, 3 bolts, nut.
	require.Len(t, sevenPartImages, 31)
}

// TestSevenSegmentColors checks that every part has one fixed colour: digits 1 and 8 draw the same parts in the same
// colours although the rotors stand at different angles, and the yellow slab is the only yellow body.
func TestSevenSegmentColors(t *testing.T) {
	s, err := buildSevenSegmentSet(context.Background())
	require.NoError(t, err)
	dm, err := sevensegment.DefaultParams().Derive()
	require.NoError(t, err)
	one, err := sevenDigitModels(s, dm, 1, r3.Vec{})
	require.NoError(t, err)
	eight, err := sevenDigitModels(s, dm, 8, r3.Vec{})
	require.NoError(t, err)
	require.Len(t, one, len(eight))
	require.Len(t, one, 19) // the top plate, its four bolts and seven rotors of two bodies
	for i := range one {
		require.Equal(t, one[i].Material.Color, eight[i].Material.Color, "model %d", i)
	}
	for _, p := range s {
		want := p.color == sevenYellow
		require.Equal(t, want, len(p.name) > 5 && p.name[:5] == "Flag " && p.name[len(p.name)-6:] == "Yellow", p.name)
	}
}

// TestSevenSegmentMotionColors checks the motion clip's models: every part keeps one colour at every camshaft angle, in
// the angled view and in the cut, and only the rotors' yellow bodies are yellow.
func TestSevenSegmentMotionColors(t *testing.T) {
	s, err := buildSevenSegmentSet(context.Background())
	require.NoError(t, err)
	dm, err := sevensegment.DefaultParams().Derive()
	require.NoError(t, err)
	for _, top := range []bool{true, false} {
		base, err := sevenMotionModels(s, dm, 0, top)
		require.NoError(t, err)
		require.NotEmpty(t, base)
		for _, theta := range []float64{36, 150, 215, 330} {
			got, err := sevenMotionModels(s, dm, theta, top)
			require.NoError(t, err)
			require.Len(t, got, len(base), "theta %.0f top %v", theta, top)
			for i := range base {
				require.Equal(t, base[i].Material.Color, got[i].Material.Color, "model %d at theta %.0f top %v", i, theta, top)
			}
		}
		yellows := 0
		for _, m := range base {
			if m.Material.Color == sevenYellow {
				yellows++
			}
		}
		if top {
			require.Equal(t, 7, yellows, "the angled view draws the seven yellow bodies")
		}
	}
}
