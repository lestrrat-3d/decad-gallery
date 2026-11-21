package renderkit_test

// CI: isolated-tests

import (
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
	"github.com/stretchr/testify/require"
)

func TestMotionGIFRoundTrip(t *testing.T) {
	first := image.NewRGBA(image.Rect(0, 0, 2, 1))
	second := image.NewRGBA(first.Bounds())
	first.Set(0, 0, color.RGBA{R: 255, A: 255})
	first.Set(1, 0, color.RGBA{B: 255, A: 255})
	second.Set(0, 0, color.RGBA{R: 255, A: 255})
	second.Set(1, 0, color.RGBA{G: 255, A: 255})
	output := filepath.Join(t.TempDir(), "nested", "motion.gif")
	require.NoError(t, renderkit.WriteGIF(output, renderkit.EncodeMotionGIF([]image.Image{first, second}, 2, 1, 4)))
	file, err := os.Open(output)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	animation, err := gif.DecodeAll(file)
	require.NoError(t, err)
	require.Len(t, animation.Image, 2)
	require.Equal(t, []int{4, 4}, animation.Delay)
	require.Equal(t, image.Rect(1, 0, 2, 1), animation.Image[1].Bounds())
	require.Equal(t, color.RGBAModel.Convert(second.At(1, 0)), color.RGBAModel.Convert(animation.Image[1].At(1, 0)))
}
