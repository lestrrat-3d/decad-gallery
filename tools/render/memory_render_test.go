package main

import (
	"bytes"
	"image/color"
	"image/png"
	"testing"

	"github.com/lestrrat-3d/solidlens"
	"github.com/stretchr/testify/require"
)

func TestDirectRenderMatchesPNGForClippedBearing(t *testing.T) {
	scene, err := bearing608Hero(t.Context())
	require.NoError(t, err)
	require.Len(t, scene.Models, 1)
	require.True(t, scene.Models[0].Edges.Enabled)
	scene.Models[0].Clip = &solidlens.ClipPlane{Normal: solidlens.Vec{X: 1}}
	settings := solidlens.Settings{Width: 240, Height: 135}

	direct, err := solidlens.Render(t.Context(), scene, settings)
	require.NoError(t, err)
	var encoded bytes.Buffer
	require.NoError(t, solidlens.RenderPNG(t.Context(), &encoded, scene, settings))
	decoded, err := png.Decode(&encoded)
	require.NoError(t, err)
	require.Equal(t, direct.Bounds(), decoded.Bounds())
	background := color.RGBAModel.Convert(direct.At(0, 0))
	foreground := 0
	for y := direct.Bounds().Min.Y; y < direct.Bounds().Max.Y; y++ {
		for x := direct.Bounds().Min.X; x < direct.Bounds().Max.X; x++ {
			got := color.RGBAModel.Convert(direct.At(x, y))
			want := color.RGBAModel.Convert(decoded.At(x, y))
			if got != want {
				t.Fatalf("pixel (%d, %d): direct %v, PNG %v", x, y, got, want)
			}
			if got != background {
				foreground++
			}
		}
	}
	require.Positive(t, foreground)
}
