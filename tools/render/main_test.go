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

// TestScrewGearHero builds the real screw gear once (about 40 s) for the single hero preset.
func TestScrewGearHero(t *testing.T) {
	output := filepath.Join(t.TempDir(), "hero.png")
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "screwgear", "-shot", "hero",
		"-out", output, "-width", "160", "-height", "90")
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, string(result))
	requireMechanismImage(t, output, 160, 90)
}

// TestScrewGearReadme shares one real build across the index, hero, top and low images, so the top and low
// presets are checked without two more 40 s builds.
func TestScrewGearReadme(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "screwgear", "-shot", "readme", "-out", dir)
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, string(result))
	requireMechanismImage(t, filepath.Join(dir, "index.png"), 480, 270)
	for _, name := range []string{"hero.png", "top.png", "low.png"} {
		requireMechanismImage(t, filepath.Join(dir, name), 1600, 900)
	}
}

// TestScrewGearSections renders the three plane sections of the real ribbon pair and checks each shows both
// gears (blue and orange fills) and no red overlap, since the default pair does not interpenetrate.
func TestScrewGearSections(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "screwgear", "-shot", "section", "-out", dir)
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, string(result))
	for _, name := range []string{"section-below.png", "section-mid.png", "section-above.png"} {
		file, err := os.Open(filepath.Join(dir, name))
		require.NoError(t, err)
		img, err := png.Decode(file)
		require.NoError(t, file.Close())
		require.NoError(t, err)
		require.Equal(t, 1200, img.Bounds().Dx())
		require.Equal(t, 900, img.Bounds().Dy())
		var blue, orange, red int
		for y := range 900 {
			for x := range 1200 {
				r, g, b, _ := img.At(x, y).RGBA()
				r, g, b = r>>8, g>>8, b>>8
				switch {
				case r > 200 && g < 90 && b < 90:
					// The legend's overlap swatch, in the top left corner, is red too.
					if x >= 130 || y >= 130 {
						red++
					}
				case b > r+40 && b > g:
					blue++
				case r > b+40 && r > g:
					orange++
				}
			}
		}
		require.Greater(t, blue, 2000, "%s: gear A's section should be filled", name)
		require.Greater(t, orange, 2000, "%s: gear B's section should be filled", name)
		require.Zero(t, red, "%s: the ribbons should not overlap", name)
	}
}

// requireMechanismImage checks the PNG size and that its centre column holds the sleeve and ribbons: the
// pixels across the middle third of the image must include several colours besides the background.
func requireMechanismImage(t *testing.T, path string, width, height int) {
	t.Helper()
	file, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	image, err := png.Decode(file)
	require.NoError(t, err)
	require.Equal(t, width, image.Bounds().Dx())
	require.Equal(t, height, image.Bounds().Dy())
	background := image.At(0, 0)
	visible := 0
	for y := height / 3; y < 2*height/3; y++ {
		for x := range width {
			if image.At(x, y) != background {
				visible++
			}
		}
	}
	require.Greater(t, visible, width*height/20, "%s: the real mechanism should be visible against the background", path)
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
