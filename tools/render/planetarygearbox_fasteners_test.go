package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-3d/solidlens"
	"github.com/stretchr/testify/require"
)

// TestPlanetaryGearboxFastenerPresets builds the real one-stage gearbox once and renders both fastener cutaways.
func TestPlanetaryGearboxFastenerPresets(t *testing.T) {
	section, exploded, err := planetaryFastenerScenes(t.Context())
	require.NoError(t, err)
	for name, scene := range map[string]solidlens.Scene{"fasteners-section": section, "fasteners-exploded": exploded} {
		output := filepath.Join(t.TempDir(), name+".png")
		require.NoError(t, renderPNG(t.Context(), output, scene, 160, 90), name)

		file, err := os.Open(output)
		require.NoError(t, err)
		image, err := png.Decode(file)
		require.NoError(t, file.Close())
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
		require.Greater(t, visible, 300, "%s: the real housing should be visible against the background", name)
	}
}
