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
	defer func() { _ = file.Close() }()
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
	defer func() { _ = file.Close() }()
	image, err := png.Decode(file)
	require.NoError(t, err)
	require.Equal(t, 120, image.Bounds().Dx())
	require.Equal(t, 68, image.Bounds().Dy())
	require.NotEqual(t, image.At(0, 0), image.At(60, 34), "the real gear should be visible against the background")
}

func TestBevelGearStraightHigh(t *testing.T) {
	output := filepath.Join(t.TempDir(), "straight-high.png")
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "bevelgear", "-shot", "straight-high",
		"-out", output, "-width", "120", "-height", "68")
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, string(result))

	file, err := os.Open(output)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	image, err := png.Decode(file)
	require.NoError(t, err)
	require.Equal(t, 120, image.Bounds().Dx())
	require.Equal(t, 68, image.Bounds().Dy())
	require.NotEqual(t, image.At(0, 0), image.At(60, 34), "the real gear should be visible against the background")
}

func TestHelicalGearHero(t *testing.T) {
	output := filepath.Join(t.TempDir(), "hero.png")
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "helicalgear", "-shot", "hero",
		"-out", output, "-width", "120", "-height", "68")
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, string(result))

	file, err := os.Open(output)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	image, err := png.Decode(file)
	require.NoError(t, err)
	require.Equal(t, 120, image.Bounds().Dx())
	require.Equal(t, 68, image.Bounds().Dy())
	require.NotEqual(t, image.At(0, 0), image.At(60, 34), "the real gear should be visible against the background")
}

func TestHerringboneGearHero(t *testing.T) {
	output := filepath.Join(t.TempDir(), "hero.png")
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "herringbonegear", "-shot", "hero",
		"-out", output, "-width", "120", "-height", "68")
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, string(result))

	file, err := os.Open(output)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	image, err := png.Decode(file)
	require.NoError(t, err)
	require.Equal(t, 120, image.Bounds().Dx())
	require.Equal(t, 68, image.Bounds().Dy())
	require.NotEqual(t, image.At(0, 0), image.At(60, 34), "the real gear should be visible against the background")
}

func TestCycloidalHero(t *testing.T) {
	output := filepath.Join(t.TempDir(), "hero.png")
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "cycloidal", "-shot", "hero",
		"-out", output, "-width", "160", "-height", "90")
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, string(result))

	file, err := os.Open(output)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	image, err := png.Decode(file)
	require.NoError(t, err)
	require.Equal(t, 160, image.Bounds().Dx())
	require.Equal(t, 90, image.Bounds().Dy())
	background := image.At(0, 0)
	visible := 0
	for y := range image.Bounds().Dy() {
		for x := range image.Bounds().Dx() {
			if image.At(x, y) != background {
				visible++
			}
		}
	}
	require.Greater(t, visible, 300, "the real drive should be visible against the background")
}

func TestGalleryHero(t *testing.T) {
	output := filepath.Join(t.TempDir(), "gallery.png")
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "gallery", "-shot", "hero",
		"-out", output, "-width", "320", "-height", "180")
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, string(result))

	file, err := os.Open(output)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	image, err := png.Decode(file)
	require.NoError(t, err)
	require.Equal(t, 320, image.Bounds().Dx())
	require.Equal(t, 180, image.Bounds().Dy())
	background := image.At(0, 0)
	for _, quadrant := range []struct {
		name string
		x, y int
	}{
		{"spur gear", 64, 50}, {"helical gear", 160, 50}, {"herringbone gear", 256, 50},
		{"cycloidal drive", 100, 150}, {"bevel gears", 260, 130},
	} {
		visible := 0
		for dy := -20; dy <= 20; dy++ {
			for dx := -20; dx <= 20; dx++ {
				if image.At(quadrant.x+dx, quadrant.y+dy) != background {
					visible++
				}
			}
		}
		require.Positive(t, visible, "the %s should be visible against the background", quadrant.name)
	}
}
