package main_test

import (
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The fan's view presets render the whole assembled fan from other angles. This renders one of them, which builds the
// fan once, and checks the image size and that the fan stays clear of every edge. The other views share the code path
// and differ only in their camera.
func TestHandCrankFanView(t *testing.T) {
	path := filepath.Join(t.TempDir(), "view.png")
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "handcrankfan", "-shot", "view-rear",
		"-out", path)
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", result)
	file, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	img, err := png.Decode(file)
	require.NoError(t, err)
	bounds := img.Bounds()
	require.Equal(t, 1200, bounds.Dx())
	require.Equal(t, 675, bounds.Dy())
	background := img.At(bounds.Min.X, bounds.Min.Y)
	differs := func(x, y int) bool { return img.At(x, y) != background }
	count := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if differs(x, y) {
				count++
			}
		}
	}
	require.Greater(t, count, 20000, "the fan fills part of the image")
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		require.False(t, differs(x, bounds.Min.Y) || differs(x, bounds.Max.Y-1), "the fan touches the top or bottom edge")
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		require.False(t, differs(bounds.Min.X, y) || differs(bounds.Max.X-1, y), "the fan touches the left or right edge")
	}
}
