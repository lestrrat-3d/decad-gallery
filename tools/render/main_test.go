package main_test

import (
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpurGearHero(t *testing.T) {
	output := filepath.Join(t.TempDir(), "hero.png")
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "spurgear", "-shot", "hero",
		"-out", output, "-width", "120", "-height", "68")
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, string(result))

	file, err := os.Open(output)
	require.NoError(t, err)
	defer file.Close()
	image, err := png.Decode(file)
	require.NoError(t, err)
	require.Equal(t, 120, image.Bounds().Dx())
	require.Equal(t, 68, image.Bounds().Dy())
	require.NotEqual(t, image.At(0, 0), image.At(60, 34), "the real gear should be visible against the background")
}

func TestBevelGearHero(t *testing.T) {
	output := filepath.Join(t.TempDir(), "hero.png")
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "bevelgear", "-shot", "hero",
		"-out", output, "-width", "120", "-height", "68")
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, string(result))

	file, err := os.Open(output)
	require.NoError(t, err)
	defer file.Close()
	image, err := png.Decode(file)
	require.NoError(t, err)
	require.Equal(t, 120, image.Bounds().Dx())
	require.Equal(t, 68, image.Bounds().Dy())
	require.NotEqual(t, image.At(0, 0), image.At(60, 34), "the real gear should be visible against the background")
}
