package main_test

import (
	"image"
	"image/gif"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPedalSpindleAnimation(t *testing.T) {
	output := filepath.Join(t.TempDir(), "linkage.gif")
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "pedalspindle", "-shot", "animation",
		"-out", output, "-width", "960", "-height", "680")
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", result)
	file, err := os.Open(output)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	animation, err := gif.DecodeAll(file)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(animation.Image), 24, "a full turn must have enough intermediate poses")
	require.Equal(t, 0, animation.LoopCount, "the animation must loop")
	require.Equal(t, 960, animation.Config.Width)
	require.Equal(t, 680, animation.Config.Height)
	first := animation.Image[0]
	fixed := fixedPivotPixels(first)
	fixedCount := 0
	for _, v := range fixed {
		if v {
			fixedCount++
		}
	}
	require.Greater(t, fixedCount, 100, "both views must mark their fixed pivots")
	for i, frame := range animation.Image {
		require.Equal(t, 960, frame.Bounds().Dx(), "frame %d", i)
		require.Equal(t, 680, frame.Bounds().Dy(), "frame %d", i)
		current := fixedPivotPixels(frame)
		changed := 0
		for j, v := range fixed {
			if current[j] != v {
				changed++
			}
		}
		require.Zero(t, changed, "fixed axes must not drift or disappear in frame %d", i)
		require.Greater(t, animation.Delay[i], 0)
	}
	// Check actual body motion rather than changes to angle text in the footer.
	quarter := animation.Image[len(animation.Image)/4]
	for _, area := range []image.Rectangle{image.Rect(0, 90, 480, 560), image.Rect(480, 90, 960, 560)} {
		changed := 0
		for y := area.Min.Y; y < area.Max.Y; y++ {
			for x := area.Min.X; x < area.Max.X; x++ {
				if first.ColorIndexAt(x, y) != quarter.ColorIndexAt(x, y) {
					changed++
				}
			}
		}
		require.Greater(t, changed, 1000, "the linkage must move in each view")
	}
	// Locate B from its fixed crosshair, then check the real heel bracket below it stays still.
	minX, maxY := 960, 0
	for y := 340; y < 560; y++ {
		for x := 0; x < 480; x++ {
			if fixed[y*960+x] {
				minX = min(minX, x)
				maxY = max(maxY, y)
			}
		}
	}
	require.Less(t, minX, 480, "the heel pivot must be marked in the assembled view")
	area := image.Rect(minX+8, maxY+10, minX+14, maxY+16)
	for i, frame := range animation.Image {
		changed := 0
		for y := area.Min.Y; y < area.Max.Y; y++ {
			for x := area.Min.X; x < area.Max.X; x++ {
				if frame.ColorIndexAt(x, y) != first.ColorIndexAt(x, y) {
					changed++
				}
			}
		}
		require.Zero(t, changed, "the fixed heel bracket must not move in frame %d", i)
	}
}

func fixedPivotPixels(frame *image.Paletted) []bool {
	index := -1
	for i, c := range frame.Palette {
		r, g, b, _ := c.RGBA()
		if r == 190*257 && g == 35*257 && b == 180*257 {
			index = i
			break
		}
	}
	mask := make([]bool, frame.Bounds().Dx()*frame.Bounds().Dy())
	if index < 0 {
		return mask
	}
	for y := range frame.Bounds().Dy() {
		for x := range frame.Bounds().Dx() {
			mask[y*frame.Bounds().Dx()+x] = int(frame.ColorIndexAt(x, y)) == index
		}
	}
	return mask
}
