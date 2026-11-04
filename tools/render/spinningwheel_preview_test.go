package main

import (
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpinningWheelPreviewRendering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preview.png")
	require.NoError(t, run(t.Context(), "spinningwheel-preview", "hero", path, 320, 180))
	file, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	img, err := png.Decode(file)
	require.NoError(t, err)
	require.Equal(t, 320, img.Bounds().Dx())
	require.Equal(t, 180, img.Bounds().Dy())
	br, bg, bb, _ := img.At(0, 40).RGBA()
	visible := 0
	for y := 50; y < 160; y++ {
		for x := 0; x < 320; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r != br || g != bg || b != bb {
				visible++
			}
		}
	}
	require.Greater(t, visible, 1000, "the native preview must contain visible model surfaces below its banner")
	data, err := os.ReadFile(path + ".json")
	require.NoError(t, err)
	var report wheelPreviewReport
	require.NoError(t, json.Unmarshal(data, &report))
	require.False(t, report.JointCutsPending)
	require.Greater(t, report.RenderedParts, 0)
	require.LessOrEqual(t, report.RenderedParts, report.BuiltParts)
}
