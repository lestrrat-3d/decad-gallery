package main_test

import (
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
