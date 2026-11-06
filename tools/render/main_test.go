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

// TestPlanetaryGearboxHero builds the real two-stage gearbox once for its hero preset.
func TestPlanetaryGearboxHero(t *testing.T) {
	output := filepath.Join(t.TempDir(), "hero.png")
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "planetarygearbox", "-shot", "hero",
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
	require.Greater(t, visible, 300, "the real gearbox should be visible against the background")
}

// TestPlanetaryGearboxPartPreset builds the real two-stage gearbox once and renders its output carrier alone.
func TestPlanetaryGearboxPartPreset(t *testing.T) {
	output := filepath.Join(t.TempDir(), "part.png")
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "planetarygearbox", "-shot", "part-output-carrier",
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
	require.Greater(t, visible, 300, "the real carrier should be visible against the background")
}

func TestPlanetaryGearboxAssembledPreset(t *testing.T) {
	output := filepath.Join(t.TempDir(), "assembled.png")
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "planetarygearbox", "-shot", "assembled-input",
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
	require.Greater(t, visible, 300, "the real assembled gearbox should be visible against the background")
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

// TestScrewGearReadme shares one real build across the index, hero, top, low and cutaway images, so the top, low and
// cutaway presets are checked without more 40 s builds (the CI render job stops the package at five minutes).
func TestScrewGearReadme(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "screwgear", "-shot", "readme", "-out", dir)
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, string(result))
	requireMechanismImage(t, filepath.Join(dir, "index.png"), 480, 270)
	for _, name := range []string{"hero.png", "top.png", "low.png"} {
		requireMechanismImage(t, filepath.Join(dir, name), 1600, 900)
	}
	requireMechanismImage(t, filepath.Join(dir, "cutaway.png"), 1600, 900)
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
		{"spinning wheel", 70, 120}, {"spur gear", 168, 54}, {"helical gear", 224, 54},
		{"herringbone gear", 280, 56}, {"cycloidal drive", 172, 120}, {"bevel gears", 272, 122},
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
