package main

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"path/filepath"
	"time"

	"github.com/lestrrat-3d/solidlens"
)

const (
	wheelOrbitFrames = 72
	wheelOrbitWidth  = 960
	wheelOrbitHeight = 540
	// wheelOrbitFOV is the vertical field of view in degrees.
	wheelOrbitFOV   = 32.0
	wheelOrbitDelay = 60 * time.Millisecond
)

// wheelOrbitFill is the share of the frame height or width that the model's largest projected extent fills.
const wheelOrbitFill = .88

// newWheelOrbit returns the frames of a turntable that turns once around the vertical axis through the
// centre of the wheel's bounding box. It starts from the hero view's yaw and pitch and uses the hero scene's
// models, background and materials. The scene has no lights, so solidlens lights each frame relative to its
// camera. The second result is the largest extent over the frames, as a share of the half frame.
//
// FitOrbit frames the bounding sphere, which leaves a flat wheel small, and a camera that always looks at the
// box centre leaves the wheel off-centre at some angles. Each frame is therefore shifted sideways and up until
// its projected extent is centred, and the one constant distance is rescaled until the largest extent over
// all frames is wheelOrbitFill of the frame.
func newWheelOrbit(meshes []wheelMesh, hero solidlens.Scene, aspect float64, frames int) (solidlens.Scenes, float64, error) {
	sources := make([]solidlens.TriangleSource, len(meshes))
	for i, mesh := range meshes {
		sources[i] = mesh.model.Mesh
	}
	bounds, err := solidlens.BoundsOf(sources...)
	if err != nil {
		return nil, 0, err
	}
	offset := hero.Camera.Position.Sub(hero.Camera.Target)
	orbit := solidlens.FitOrbit(bounds, wheelOrbitFOV)
	orbit.Yaw = math.Atan2(offset.Y, offset.X) * 180 / math.Pi
	orbit.Pitch = math.Atan2(offset.Z, math.Hypot(offset.X, offset.Y)) * 180 / math.Pi
	base := hero
	base.DirectionalLights = nil
	var scenes solidlens.Scenes
	var extent float64
	// The projected size is not exactly inverse to the distance, so a few rescales converge on the fill.
	for range 5 {
		turntable := solidlens.Turntable{Base: base, Orbit: orbit, Frames: frames}
		scenes, extent = scenes[:0], 0
		for frame := range frames {
			scene, frameExtent := recentreWheelScene(turntable.Scene(frame), sources, aspect)
			scenes = append(scenes, scene)
			extent = math.Max(extent, frameExtent)
		}
		orbit.Distance *= extent / wheelOrbitFill
	}
	return scenes, extent, nil
}

// recentreWheelScene shifts the scene's camera sideways and up, without turning it, until the projected
// extent of the vertices is centred. It returns the scene and the extent's half size as a share of the half
// frame height. The lights depend only on the camera's direction, so the shift leaves them alone.
func recentreWheelScene(scene solidlens.Scene, sources []solidlens.TriangleSource, aspect float64) (solidlens.Scene, float64) {
	camera := scene.Camera
	forward, _ := camera.Target.Sub(camera.Position).Normalize()
	right, _ := forward.Cross(camera.Up).Normalize()
	up := right.Cross(forward)
	var halfX, halfY float64
	for range 4 {
		minX, maxX := math.Inf(1), math.Inf(-1)
		minY, maxY := math.Inf(1), math.Inf(-1)
		var depthSum float64
		var count int
		for _, source := range sources {
			for _, v := range source.Vertices() {
				d := v.Sub(camera.Position)
				depth := d.Dot(forward)
				x, y := d.Dot(right)/depth, d.Dot(up)/depth
				minX, maxX = math.Min(minX, x), math.Max(maxX, x)
				minY, maxY = math.Min(minY, y), math.Max(maxY, y)
				depthSum += depth
				count++
			}
		}
		mean := depthSum / float64(count)
		shift := right.Scale((minX + maxX) / 2 * mean).Add(up.Scale((minY + maxY) / 2 * mean))
		camera.Position, camera.Target = camera.Position.Add(shift), camera.Target.Add(shift)
		halfX, halfY = (maxX-minX)/2, (maxY-minY)/2
	}
	scene.Camera = camera
	return scene, math.Max(halfX/aspect, halfY) / math.Tan(camera.FOV*math.Pi/360)
}

// renderSpinningWheelOrbit writes a looping turntable GIF of the static wheel. Frame i is turned
// 360*i/frames degrees from the hero view, so the last frame is one step short of the first.
func renderSpinningWheelOrbit(ctx context.Context, meshes []wheelMesh, out string, width, height, frames int) error {
	if filepath.Ext(out) != ".gif" {
		return fmt.Errorf("orbit requires a .gif output path")
	}
	if frames < 2 {
		return fmt.Errorf("orbit needs at least two frames")
	}
	if width < 160 || height < 90 || width > 1920 || height > 1080 {
		return fmt.Errorf("orbit dimensions must be within 160x90 and 1920x1080")
	}
	scenes, _, err := newWheelOrbit(meshes, spinningWheelScene(meshes, "hero"), float64(width)/float64(height), frames)
	if err != nil {
		return err
	}
	var buffer bytes.Buffer
	if err := solidlens.RenderGIF(ctx, &buffer, scenes,
		solidlens.GIFOptions{Width: width, Height: height, Delay: wheelOrbitDelay}); err != nil {
		return fmt.Errorf("render orbit: %w", err)
	}
	return writeWheelFile(out, buffer.Bytes())
}

func runSpinningWheelOrbit(ctx context.Context, out string, width, height int) error {
	if out == "" {
		return fmt.Errorf("-out is required")
	}
	if width == 0 {
		width = wheelOrbitWidth
	}
	if height == 0 {
		height = wheelOrbitHeight
	}
	meshes, _, err := spinningWheelModels(ctx)
	if err != nil {
		return err
	}
	return renderSpinningWheelOrbit(ctx, meshes, out, width, height, wheelOrbitFrames)
}
