package main_test

import (
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestM2BoltRendering(t *testing.T) {
	for _, shot := range []string{"hero", "head", "side", "underside", "lengths"} {
		t.Run(shot, func(t *testing.T) { requireM2BoltShot(t, shot) })
	}
}

func requireM2BoltShot(t *testing.T, shot string) {
	t.Helper()
	output := filepath.Join(t.TempDir(), shot+".png")
	cmd := exec.CommandContext(t.Context(), "go", "run", ".", "-model", "m2bolt", "-shot", shot,
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
	foreground := 0
	for y := range 90 {
		for x := range 160 {
			if img.At(x, y) != background {
				foreground++
			}
		}
	}
	require.Greater(t, foreground, 900, "the real M2 bolt must be visible")
}
