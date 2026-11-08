package main_test

import (
	"image/gif"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The animation turns the crank and the fan between frames: two frames a quarter turn apart differ, and the frames
// cover one full turn.
func TestHandCrankFanMotion(t *testing.T) {
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
