package render

// CI: isolated-tests

import (
	"context"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestSevenSegmentPresets(t *testing.T) {
	m := Model()
	for _, name := range []string{"hero", "top", "oblique45", "oblique30", "front", "side", "cutaway", "indexer", "digits",
		"cams", "exploded", "exploded-camshaft", "exploded-pivot", "pivot-section"} {
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
	// 11 plain printed or bought parts, 7 cams, 3 slider types, the groove pin, 2 rotor images and 4 rotor bodies, bearing, 2 bolts, nut, pivot block.
	require.Len(t, sevenPartImages, 33)
}

// TestSevenSegmentColors checks that every part has one fixed colour: digits 1 and 8 draw the same parts in the same
// colours although the rotors stand at different angles, and the yellow slab is the only yellow body.
func TestSevenSegmentColors(t *testing.T) {
	s := sevenSegmentTestSet(t)
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

// TestSevenSegmentMotionColors checks the motion clip's models: every part keeps one colour in every pose of the press cycle,
// in the face view and in the cut, and only the rotors' yellow bodies are yellow.
func TestSevenSegmentMotionColors(t *testing.T) {
	s := sevenSegmentTestSet(t)
	dm, err := sevensegment.DefaultParams().Derive()
	require.NoError(t, err)
	springs := map[float64]*decad.Mesh{}
	for _, l := range []float64{6, 7, 8} {
		body, err := sevensegment.BuildSpring(context.Background(), decad.New(), l)
		require.NoError(t, err)
		springs[l], err = body.Tessellate(context.Background(), units.Millimeters(.05))
		require.NoError(t, err)
	}
	states := motionStates()
	for _, face := range []bool{true, false} {
		base, err := motionModels(s, springs, dm, states[0], face)
		require.NoError(t, err)
		require.NotEmpty(t, base)
		for _, st := range []clickerState{states[3], states[6], states[40], states[len(states)-1]} {
			got, err := motionModels(s, springs, dm, st, face)
			require.NoError(t, err)
			require.Len(t, got, len(base), "state %+v face %v", st, face)
			for i := range base {
				require.Equal(t, base[i].Material.Color, got[i].Material.Color, "model %d at %+v face %v", i, st, face)
			}
		}
		yellows := 0
		for _, m := range base {
			if m.Material.Color == sevenYellow {
				yellows++
			}
		}
		if face {
			require.Equal(t, 7, yellows, "the face view draws the seven yellow bodies")
		}
	}
}
