package main

// CI: isolated-tests

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
	"github.com/stretchr/testify/require"
)

// The siren's view presets render the whole assembled siren from other angles. This renders one of them, which builds the
// siren once, and checks the image size and that the siren stays clear of every edge. The other views share the code path
// and differ only in their camera.
func TestHandCrankSirenView(t *testing.T) {
	path := filepath.Join(t.TempDir(), "view.png")
	err := renderkit.Run(t.Context(), handcranksirenModel(), "view-side",
		renderkit.Options{Width: 320, Height: 180, Output: path})
	require.NoError(t, err)
	file, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	img, err := png.Decode(file)
	require.NoError(t, err)
	bounds := img.Bounds()
	require.Equal(t, 320, bounds.Dx())
	require.Equal(t, 180, bounds.Dy())
	background := img.At(bounds.Min.X, bounds.Min.Y)
	differs := func(x, y int) bool { return img.At(x, y) != background }
	count := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if differs(x, y) {
				count++
			}
		}
	}
	require.Greater(t, count, 1000, "the siren fills part of the image")
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		require.False(t, differs(x, bounds.Min.Y) || differs(x, bounds.Max.Y-1), "the siren touches the top or bottom edge")
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		require.False(t, differs(bounds.Min.X, y) || differs(bounds.Max.X-1, y), "the siren touches the left or right edge")
	}
}
