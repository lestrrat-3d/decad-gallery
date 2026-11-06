package main_test

import (
	"image/gif"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandCrankFanRendering(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "handcrankfan", "-shot", "readme", "-out", dir)
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", result)
	for _, shot := range []struct {
		name          string
		width, height int
	}{
		{"hero.png", 1600, 900}, {"index.png", 480, 270}, {"exploded.png", 1600, 900},
		{"parts/gearbox.png", 480, 270}, {"parts/grip.png", 480, 270}, {"parts/grip-screw.png", 480, 270},
		{"parts/crank-gear.png", 480, 270}, {"parts/drive-gear.png", 480, 270}, {"parts/driving-gear.png", 480, 270},
		{"parts/pinion.png", 480, 270}, {"parts/crank-shaft.png", 480, 270}, {"parts/drive-shaft.png", 480, 270},
		{"parts/fan-shaft.png", 480, 270}, {"parts/bearing.png", 480, 270}, {"parts/crank-arm.png", 480, 270},
		{"parts/crank-pin.png", 480, 270}, {"parts/crank-pin-screw.png", 480, 270}, {"parts/crank-handle.png", 480, 270},
		{"parts/fan-hub.png", 480, 270}, {"parts/fan-blade.png", 480, 270}, {"parts/fan-bolt.png", 480, 270},
	} {
		file, err := os.Open(filepath.Join(dir, shot.name))
		require.NoError(t, err)
		img, err := png.Decode(file)
		require.NoError(t, err)
		require.NoError(t, file.Close())
		require.Equal(t, shot.width, img.Bounds().Dx())
		require.Equal(t, shot.height, img.Bounds().Dy())
		background := img.At(0, 0)
		foreground := 0
		for y := range shot.height {
			for x := range shot.width {
				if img.At(x, y) != background {
					foreground++
				}
			}
		}
		minimum := shot.width * shot.height / 20
		if shot.height == 270 && filepath.Dir(shot.name) == "parts" {
			// A screw or a bolt is a thin part: it only needs to show against the background.
			minimum = 300
		}
		require.Greater(t, foreground, minimum, "%s: the real part must be visible", shot.name)
	}
}

// The animation turns the crank and the fan between frames: two frames a quarter turn apart differ, and the frames
// cover one full turn.
func TestHandCrankFanAnimation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "motion.gif")
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "handcrankfan", "-shot", "animation",
		"-frames", "4", "-out", path)
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", result)
	file, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	animation, err := gif.DecodeAll(file)
	require.NoError(t, err)
	require.Len(t, animation.Image, 4)
	require.Equal(t, 1000, animation.Config.Width)
	require.Equal(t, 560, animation.Config.Height)
	changed := func(a, b int) int {
		count := 0
		first, second := animation.Image[a], animation.Image[b]
		for y := first.Bounds().Min.Y; y < first.Bounds().Max.Y; y++ {
			for x := first.Bounds().Min.X; x < first.Bounds().Max.X; x++ {
				if first.At(x, y) != second.At(x, y) {
					count++
				}
			}
		}
		return count
	}
	require.Greater(t, changed(0, 1), 1000, "the crank and the fan move between frames")
	require.Greater(t, changed(0, 2), 1000, "the half turn differs from the start")
}
