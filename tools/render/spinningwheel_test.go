package main

import (
	"bytes"
	"image/gif"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
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
	for _, shot := range []string{"front-back", "layout", "flyer", "brake", "whorl"} {
		path := filepath.Join(out, shot+".png")
		require.NoError(t, renderSpinningWheelSheet(t.Context(), meshes, shot, path, 1280, 720))
		requireWheelImage(t, path, 1280, 720)
	}
	require.NoError(t, renderSpinningWheelReadmeStills(t.Context(), meshes, out, 640, 360))
	for _, name := range []string{"hero", "index", "rear", "front-back", "layout", "flyer", "brake", "whorl"} {
		_, err := os.Stat(filepath.Join(out, name+".png"))
		require.NoError(t, err, name)
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
	requireWheelOrbit(t, meshes, filepath.Join(out, "orbit.gif"))
}

func requireWheelOrbit(t *testing.T, meshes []wheelMesh, path string) {
	t.Helper()
	require.NoError(t, renderSpinningWheelOrbit(t.Context(), meshes, path, 320, 180, 4))
	f, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	orbit, err := gif.DecodeAll(f)
	require.NoError(t, err)
	require.Equal(t, 4, len(orbit.Image))
	require.Equal(t, 0, orbit.LoopCount, "the turntable loops forever")
	require.Equal(t, 320, orbit.Config.Width)
	require.Equal(t, 180, orbit.Config.Height)
	require.False(t, bytes.Equal(orbit.Image[0].Pix, orbit.Image[2].Pix), "the camera must move between frames")
	require.False(t, bytes.Equal(orbit.Image[0].Pix, orbit.Image[3].Pix), "the camera must move between frames")

	// A turn of 360 degrees returns the first camera. The distance to the target and the elevation stay fixed.
	hero := spinningWheelScene(meshes, "hero")
	scenes, fill, err := newWheelOrbit(meshes, hero, 320.0/180, 72)
	require.NoError(t, err)
	require.Len(t, scenes, 72)
	require.InDelta(t, wheelOrbitFill, fill, .01, "the wheel fills the frame at its widest angle without clipping")
	first, quarter := scenes[0], scenes[18]
	require.Equal(t, hero.Background, first.Background)
	require.Len(t, first.DirectionalLights, 2, "solidlens lights each frame relative to its camera")
	require.False(t, first.DirectionalLights[0].Direction == quarter.DirectionalLights[0].Direction, "the lights turn")
	look := func(c solidlens.Camera) r3.Vec { return c.Position.Sub(c.Target) }
	require.InDelta(t, look(first.Camera).Len(), look(quarter.Camera).Len(), 1e-6, "the distance stays constant")
	require.InDelta(t, look(first.Camera).Z, look(quarter.Camera).Z, 1e-6, "the elevation stays fixed")
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
		posed, err := spinningWheelPose(t.Context(), meshes, base, p, angle-math.Pi/3)
		require.NoError(t, err)
		checked := 0
		for i, mesh := range posed {
			if mesh.part.Motion == "fixed" {
				require.True(t, meshes[i].model.Mesh == mesh.model.Mesh, "%s stays fixed", mesh.part.Name)
				continue
			}
			if mesh.part.Motion == "compliant" {
				require.True(t, mesh.model.Mesh != meshes[i].model.Mesh, "the connector must be rebuilt at each pose")
				for _, end := range []struct{ centre, direction r3.Vec }{
					{state.Toe.Sub(state.PedalFrame.N().Scale(12 * p.Scale)), state.PedalFrame.N()},
					{state.Toe.Add(state.RodFrame.U().Scale(26 * p.Scale)), state.RodFrame.U()},
				} {
					hole, err := mesh.part.Body.LocatePoint(t.Context(), end.centre, units.Millimeters(.025))
					require.NoError(t, err)
					require.True(t, hole == decad.PointOutside, "connector clamp passage at %g must remain open", angle)
					// Decad answers Undecided for points within about 0.6 mm of the passage wall, so probe the solid 3.6 mm out.
					rubber, err := mesh.part.Body.LocatePoint(t.Context(), end.centre.Add(end.direction.Scale(3.6)), units.Millimeters(.025))
					require.NoError(t, err)
					require.True(t, rubber == decad.PointInside, "connector needs material around both clamp passages at %g", angle)
				}
				checked++
				continue
			}
			var want r3.Vec
			switch mesh.part.Name {
			case "gimbal-crank-605":
				want = state.CrankPin
			case "gimbal-cross-pin":
				want = state.RodHead
			case "footman-cross-605-1":
				// Bearing 1 sits the 0.01 mm designed crossGap off its retainer (see models/spinningwheel drive.go).
				want = state.RodHead.Add(state.RodFrame.V().Scale(-12*p.Scale + 2.5 + 0.01))
			case "footman-cross-605-2":
				// Bearing 2 ends the same 0.01 mm designed crossGap short of its retainer.
				want = state.RodHead.Add(state.RodFrame.V().Scale(12*p.Scale - 2.5 - 0.01))
			case "treadle":
				want = state.PedalFrame.ToWorld(base.PedalFrame.ToLocal(wheelMeshCentre(t, meshes[i])))
			case "wheel-spoke-1":
				want = state.Wheel.Add(r3.Vec{X: 142.5 * p.Scale * math.Cos(angle), Z: 142.5 * p.Scale * math.Sin(angle)})
			case "flyer-arm-1", "bobbin-witness-bar":
				phase := state.FlyerAngle - base.FlyerAngle
				if mesh.part.Motion == "bobbin" {
					phase = state.BobbinAngle - base.BobbinAngle
				}
				d := wheelMeshCentre(t, meshes[i]).Sub(base.Flyer)
				want = base.Flyer.Add(r3.Vec{X: d.X*math.Cos(phase) - d.Z*math.Sin(phase), Y: d.Y,
					Z: d.X*math.Sin(phase) + d.Z*math.Cos(phase)})
			case "treadle-lower-clamp-pin", "footman-upper-clamp-pin":
				face, axis := 12*p.Scale, state.PedalFrame.U()
				centre := state.Toe.Sub(state.PedalFrame.N().Scale(12 * p.Scale))
				if mesh.part.Motion == "footman" {
					face, axis = 10*p.Scale, state.RodFrame.V()
					centre = state.Toe.Add(state.RodFrame.U().Scale(26 * p.Scale))
				}
				// Integrate the one-piece bolt's two cylindrical sections independently.
				head, shank := 3.5*3.5*2, 2.0*2*(2*face+2.2)
				offset := (head*(-face-1) + shank*1.1) / (head + shank)
				want = centre.Add(axis.Scale(offset))
			default:
				continue
			}
			checked++
			centre := wheelMeshCentre(t, mesh)
			require.InDelta(t, want.X, centre.X, 1e-4, "%s X at %g", mesh.part.Name, angle)
			require.InDelta(t, want.Y, centre.Y, 1e-4, "%s Y at %g", mesh.part.Name, angle)
			require.InDelta(t, want.Z, centre.Z, 1e-4, "%s Z at %g", mesh.part.Name, angle)
		}
		require.Equal(t, 11, checked, "all motion groups, both clamps and the flexible body must follow their actual joints")
	}
}

func wheelMeshCentre(t *testing.T, mesh wheelMesh) r3.Vec {
	t.Helper()
	vertices := mesh.model.Mesh.Vertices()
	require.Greater(t, len(vertices), 0)
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
	return anchor.Add(moment.Scale(1 / (4 * volume6)))
}
