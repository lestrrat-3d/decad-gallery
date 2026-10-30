package main

import (
	"image/gif"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// All presets consume one real assembly. This includes actual raster and GIF encoding.
func TestSpinningWheelRendering(t *testing.T) {
	meshes, base, err := spinningWheelModels(t.Context())
	require.NoError(t, err)
	out := t.TempDir()
	for _, shot := range []string{"hero", "rear"} {
		_, exists := presets["spinningwheel/"+shot]
		require.True(t, exists)
		path := filepath.Join(out, shot+".png")
		require.NoError(t, renderPNG(t.Context(), path, spinningWheelScene(meshes, shot), 640, 360))
		requireWheelImage(t, path, 640, 360)
	}
	for _, shot := range []string{"front-back", "flyer", "brake", "whorl"} {
		path := filepath.Join(out, shot+".png")
		require.NoError(t, renderSpinningWheelSheet(t.Context(), meshes, shot, path, 1280, 720))
		requireWheelImage(t, path, 1280, 720)
	}
	path := filepath.Join(out, "motion.gif")
	require.NoError(t, renderSpinningWheelAnimation(t.Context(), meshes, base, path, 900, 600, 4))
	f, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	animation, err := gif.DecodeAll(f)
	require.NoError(t, err)
	require.Equal(t, 4, len(animation.Image))
	require.Equal(t, 900, animation.Config.Width)
	require.Equal(t, 600, animation.Config.Height)
	requireWheelAnimationJoints(t, meshes, base)
}

func requireWheelImage(t *testing.T, path string, width, height int) {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	img, err := png.Decode(f)
	require.NoError(t, err)
	require.Equal(t, width, img.Bounds().Dx())
	require.Equal(t, height, img.Bounds().Dy())
	background := img.At(0, 0)
	visible := 0
	for y := height / 3; y < 2*height/3; y++ {
		for x := range width {
			if img.At(x, y) != background {
				visible++
			}
		}
	}
	require.Greater(t, visible, width*height/25, "%s must contain actual model surfaces", path)
}

func requireWheelAnimationJoints(t *testing.T, meshes []wheelMesh, base spinningwheel.State) {
	t.Helper()
	for _, angle := range []float64{0, math.Pi / 2, math.Pi, 3 * math.Pi / 2} {
		p := spinningwheel.DefaultParams()
		p.WheelAngle = angle
		state, err := spinningwheel.Motion(p)
		require.NoError(t, err)
		posed, err := spinningWheelPose(meshes, base, state, angle-math.Pi/3)
		require.NoError(t, err)
		checked := 0
		for i, mesh := range posed {
			if mesh.part.Motion == "fixed" {
				require.True(t, meshes[i].model.Mesh == mesh.model.Mesh, "%s stays fixed", mesh.part.Name)
			}
			var want r3.Vec
			switch mesh.part.Name {
			case "crank-pin", "footman-bearing-2":
				want = state.CrankPin
			case "toe-pin", "footman-bearing-1":
				want = state.Toe
			default:
				continue
			}
			checked++
			// An odd-sided circle mesh can have an off-centre bounding box. Its uniform solid centroid stays centred.
			vertices := mesh.model.Mesh.Vertices()
			anchor := vertices[0]
			var moment r3.Vec
			volume6 := 0.0
			for _, triangle := range mesh.model.Mesh.Triangles() {
				a, b, c := vertices[triangle[0]].Sub(anchor), vertices[triangle[1]].Sub(anchor), vertices[triangle[2]].Sub(anchor)
				weight := a.Dot(b.Cross(c))
				volume6 += weight
				moment = moment.Add(a.Add(b).Add(c).Scale(weight))
			}
			require.Greater(t, math.Abs(volume6), 1e-6, "%s needs a closed occupied mesh", mesh.part.Name)
			centre := anchor.Add(moment.Scale(1 / (4 * volume6)))
			require.InDelta(t, want.X, centre.X, 1e-4, "%s X at %g", mesh.part.Name, angle)
			require.InDelta(t, want.Z, centre.Z, 1e-4, "%s Z at %g", mesh.part.Name, angle)
		}
		require.Equal(t, 4, checked, "both pins and both605 envelopes must be checked")
	}
}
