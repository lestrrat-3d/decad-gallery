package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"math"
	"path/filepath"

	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
)

const (
	wheelOrbitFrames = 72
	wheelOrbitWidth  = 960
	wheelOrbitHeight = 540
	// wheelOrbitFOV is the vertical field of view in degrees. The camera distance follows from it.
	wheelOrbitFOV = 32.0
	// wheelOrbitDelay is the frame delay in hundredths of a second.
	wheelOrbitDelay = 6
)

// wheelOrbitFill is the share of the frame height or width that the model's largest projected extent fills.
const wheelOrbitFill = .88

// wheelOrbitRig holds what the turntable camera needs: a sample of the model's vertices, the centre of
// their bounding box, the hero elevation and azimuth, and one constant camera distance.
type wheelOrbitRig struct {
	points, all        []r3.Vec
	centre             r3.Vec
	elevation, azimuth float64
	distance, aspect   float64
}

// newWheelOrbitRig samples the mesh vertices and chooses the one camera distance at which the largest
// projected extent over the frames of a full turn fills wheelOrbitFill of the frame. The camera is
// re-centred on the projected extent at every angle, so the distance stays constant across frames.
func newWheelOrbitRig(meshes []wheelMesh, hero solidlens.Scene, aspect float64, frames int) *wheelOrbitRig {
	lo := r3.Vec{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}
	hi := r3.Vec{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}
	total := 0
	for _, mesh := range meshes {
		total += len(mesh.model.Mesh.Vertices())
	}
	stride := max(1, total/30000)
	rig := &wheelOrbitRig{aspect: aspect}
	n := 0
	for _, mesh := range meshes {
		for _, v := range mesh.model.Mesh.Vertices() {
			rig.all = append(rig.all, v)
			lo = r3.Vec{X: math.Min(lo.X, v.X), Y: math.Min(lo.Y, v.Y), Z: math.Min(lo.Z, v.Z)}
			hi = r3.Vec{X: math.Max(hi.X, v.X), Y: math.Max(hi.Y, v.Y), Z: math.Max(hi.Z, v.Z)}
			if n%stride == 0 {
				rig.points = append(rig.points, v)
			}
			n++
		}
	}
	rig.centre = lo.Add(hi).Scale(.5)
	offset := r3.Vec(hero.Camera.Position).Sub(r3.Vec(hero.Camera.Target))
	rig.azimuth = math.Atan2(offset.Y, offset.X)
	rig.elevation = math.Atan2(offset.Z, math.Hypot(offset.X, offset.Y))
	radius := hi.Sub(rig.centre).Len()
	want := math.Tan(wheelOrbitFOV*math.Pi/360) * wheelOrbitFill
	near, far := radius*1.05, radius*40
	for range 40 {
		rig.distance = (near + far) / 2
		var need float64
		for frame := range frames {
			_, extent := rig.camera(2 * math.Pi * float64(frame) / float64(frames))
			need = math.Max(need, extent)
		}
		if need > want {
			near = rig.distance
		} else {
			far = rig.distance
		}
	}
	rig.distance = far
	// The sample can miss thin tips, so rescale the distance until every vertex fits.
	rig.points = rig.all
	for range 3 {
		var need float64
		for frame := range frames {
			_, extent := rig.camera(2 * math.Pi * float64(frame) / float64(frames))
			need = math.Max(need, extent)
		}
		rig.distance *= need / want
	}
	return rig
}

// camera returns the camera turned by angle radians about the vertical axis through the box centre,
// shifted sideways and up until the projected extent is centred, and that extent's half size in image
// plane units for a unit-height half frame.
func (rig *wheelOrbitRig) camera(angle float64) (solidlens.Camera, float64) {
	azimuth := rig.azimuth + angle
	dir := r3.Vec{X: math.Cos(rig.elevation) * math.Cos(azimuth), Y: math.Cos(rig.elevation) * math.Sin(azimuth),
		Z: math.Sin(rig.elevation)}
	position, target := rig.centre.Add(dir.Scale(rig.distance)), rig.centre
	forward := dir.Scale(-1)
	right, _ := forward.Cross(r3.Vec{Z: 1}).Normalize()
	up := right.Cross(forward)
	var halfX, halfY float64
	for range 6 {
		minX, maxX := math.Inf(1), math.Inf(-1)
		minY, maxY := math.Inf(1), math.Inf(-1)
		var depthSum float64
		for _, v := range rig.points {
			d := v.Sub(position)
			depth := d.Dot(forward)
			x, y := d.Dot(right)/depth, d.Dot(up)/depth
			minX, maxX = math.Min(minX, x), math.Max(maxX, x)
			minY, maxY = math.Min(minY, y), math.Max(maxY, y)
			depthSum += depth
		}
		mean := depthSum / float64(len(rig.points))
		shift := right.Scale((minX + maxX) / 2 * mean).Add(up.Scale((minY + maxY) / 2 * mean))
		position, target = position.Add(shift), target.Add(shift)
		halfX, halfY = (maxX-minX)/2, (maxY-minY)/2
	}
	return solidlens.Camera{Position: solidlens.Vec(position), Target: solidlens.Vec(target),
		Up: solidlens.Vec{Z: 1}, FOV: wheelOrbitFOV}, math.Max(halfX/rig.aspect, halfY)
}

// scene returns the hero scene viewed from the orbit camera at angle radians. The lights turn with it.
func (rig *wheelOrbitRig) scene(hero solidlens.Scene, angle float64) solidlens.Scene {
	scene := hero
	scene.Camera, _ = rig.camera(angle)
	scene.DirectionalLights = make([]solidlens.DirectionalLight, len(hero.DirectionalLights))
	sin, cos := math.Sincos(angle)
	for i, light := range hero.DirectionalLights {
		light.Direction.X, light.Direction.Y = light.Direction.X*cos-light.Direction.Y*sin,
			light.Direction.X*sin+light.Direction.Y*cos
		scene.DirectionalLights[i] = light
	}
	return scene
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
	hero := spinningWheelScene(meshes, "hero")
	rig := newWheelOrbitRig(meshes, hero, float64(width)/float64(height), frames)
	animation := &gif.GIF{LoopCount: 0}
	for frame := range frames {
		if err := ctx.Err(); err != nil {
			return err
		}
		scene := rig.scene(hero, 2*math.Pi*float64(frame)/float64(frames))
		var buffer bytes.Buffer
		if err := solidlens.RenderPNG(ctx, &buffer, scene, solidlens.Settings{Width: width, Height: height}); err != nil {
			return fmt.Errorf("render orbit frame %d: %w", frame, err)
		}
		img, err := png.Decode(&buffer)
		if err != nil {
			return err
		}
		indexed := image.NewPaletted(img.Bounds(), palette.Plan9)
		draw.Draw(indexed, indexed.Bounds(), img, image.Point{}, draw.Src)
		animation.Image = append(animation.Image, indexed)
		animation.Delay = append(animation.Delay, wheelOrbitDelay)
		animation.Disposal = append(animation.Disposal, gif.DisposalNone)
	}
	animation.Config = image.Config{ColorModel: color.Palette(palette.Plan9), Width: width, Height: height}
	return writeWheelGIF(out, animation)
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
