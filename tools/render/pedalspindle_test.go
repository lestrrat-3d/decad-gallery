package main_test

import (
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPedalSpindlePresets(t *testing.T) {
	for _, shot := range []string{"hero", "guarded", "drive"} {
		t.Run(shot, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), shot+".png")
			cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "pedalspindle", "-shot", shot,
				"-out", output, "-width", "160", "-height", "90")
			result, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", result)
			file, err := os.Open(output)
			require.NoError(t, err)
			defer func() { _ = file.Close() }()
			img, err := png.Decode(file)
			require.NoError(t, err)
			require.Equal(t, 160, img.Bounds().Dx())
			require.Equal(t, 90, img.Bounds().Dy())
			background := img.At(0, 0)
			var visible, orange, blue int
			for y := range 90 {
				for x := range 160 {
					if img.At(x, y) != background {
						visible++
					}
					r, g, b, _ := img.At(x, y).RGBA()
					if r > b+8000 && r > g+4000 {
						orange++
					}
					if b > r+4000 {
						blue++
					}
				}
			}
			require.Greater(t, visible, 700, "the actual assembly must occupy the image")
			if shot == "drive" {
				require.Greater(t, orange, 100, "the imported gears must be visible")
				require.Greater(t, blue, 100, "the crank and flywheel must be visible")
			}
		})
	}
}

func TestPedalSpindleSheets(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "pedalspindle", "-shot", "readme", "-out", dir)
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", result)
	for _, shot := range []struct {
		name                         string
		width, height, columns, rows int
	}{
		{"hero-pair", 2000, 1100, 2, 1},
		{"drive-pair", 2000, 1100, 2, 1},
		{"mesh-pairs", 2000, 1400, 2, 2},
		{"cycle", 2000, 1000, 4, 1},
		{"heel-support", 2000, 1100, 2, 1},
		{"coupling", 2000, 1100, 2, 1},
		{"frame-support", 2000, 1100, 2, 1},
	} {
		t.Run(shot.name, func(t *testing.T) {
			file, err := os.Open(filepath.Join(dir, shot.name+".png"))
			require.NoError(t, err)
			defer func() { _ = file.Close() }()
			img, err := png.Decode(file)
			require.NoError(t, err)
			require.Equal(t, shot.width, img.Bounds().Dx())
			require.Equal(t, shot.height, img.Bounds().Dy())
			for row := range shot.rows {
				for column := range shot.columns {
					pw, ph := shot.width/shot.columns, shot.height/shot.rows
					// Exclude captions so text alone cannot make an empty view pass.
					area := image.Rect(column*pw, row*ph+50, (column+1)*pw, (row+1)*ph-60)
					background := img.At(0, 0)
					var foreground, blue, warm int
					for y := area.Min.Y; y < area.Max.Y; y++ {
						for x := area.Min.X; x < area.Max.X; x++ {
							if img.At(x, y) != background {
								foreground++
							}
							r, g, b, _ := img.At(x, y).RGBA()
							if b > r+4000 {
								blue++
							}
							if r > b+8000 && r > g+4000 {
								warm++
							}
						}
					}
					require.Greater(t, foreground, 2000, "row %d column %d must show real geometry", row, column)
					if shot.name == "mesh-pairs" || shot.name == "cycle" {
						require.Greater(t, warm, 200, "row %d column %d must show gears or footman", row, column)
					}
					if shot.name == "cycle" || (shot.name == "mesh-pairs" && row == 0) {
						require.Greater(t, blue, 200, "row %d column %d must show blue input bodies", row, column)
					}
				}
			}
		})
	}
}
