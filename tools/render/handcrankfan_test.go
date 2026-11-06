package main_test

import (
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
		require.Greater(t, foreground, shot.width*shot.height/20, "the real fan must be visible")
	}
}
