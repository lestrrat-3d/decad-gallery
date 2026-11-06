package main

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"math"
	"slices"

	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// planetaryMotionFrames is the number of input angles the animation turns through, counting the rest pose. Each
// frame turns the input by half the sun's tooth pitch, so the sun's teeth never seem to run backward. The
// animation plays them forward and then back, so its last frame leads straight into its first and the motion
// has no jump. A seamless one-way loop is out of reach: the output shaft comes back to where it started only
// after the input has turned 84 times with one stage and 7,056 times with two.
const planetaryMotionFrames = 40

// planetaryMotionDelay is the time of one frame in hundredths of a second.
const planetaryMotionDelay = 5

// planetaryMotionWidth and planetaryMotionHeight are the size of the animation: three panels side by side with a
// caption strip above and below them.
const (
	planetaryMotionWidth  = 960
	planetaryMotionHeight = 320
)

// planetaryCaptionHeight is the height of each of the two caption strips.
const planetaryCaptionHeight = 18

// planetaryMover moves one body of the gearbox for an input angle. A nil move leaves the body where it is.
type planetaryMover struct {
	body planetaryBody
	mesh solidlens.TriangleSource
	move func(input float64) (r3.Transform, error)
}

// planetaryMovers pairs every body with how it moves when the input shaft turns by an angle. The ring and
// the covers are fixed. The input shaft and the first sun turn with the input. A carrier turns at
// Zs/(Zs+Zr) of its sun's angle, and the next stage's sun turns with it. A planet is carried around by its
// carrier and turns on its pin at -Zr/Zp of the carrier's angle, relative to the carrier, because it rolls
// inside the fixed ring.
func planetaryMovers(ctx context.Context, box planetarygearbox.Gearbox, p planetarygearbox.Params) ([]planetaryMover, error) {
	axis := r3.NewVec(0, 0, 1)
	about := func(angle float64) (r3.Transform, error) { return r3.Rotation(axis, units.Radians(angle)) }
	ratio := float64(p.SunTeeth) / float64(p.SunTeeth+p.RingTeeth)
	orbit := p.Module * float64(p.SunTeeth+p.PlanetTeeth) / 2

	var movers []planetaryMover
	add := func(body planetaryBody, move func(float64) (r3.Transform, error)) error {
		mesh, err := planetaryMesh(ctx, body)
		if err != nil {
			return err
		}
		movers = append(movers, planetaryMover{body: body, mesh: mesh, move: move})
		return nil
	}
	for _, body := range planetaryBodies(box) {
		var move func(float64) (r3.Transform, error)
		switch body.name {
		case planetarygearbox.InputShaftName:
			move = about
		default:
			for k, stage := range box.Stages {
				sunAngle := func(input float64) float64 { return input * math.Pow(ratio, float64(k)) }
				carrierAngle := func(input float64) float64 { return sunAngle(input) * ratio }
				switch body.body {
				case stage.Sun:
					move = func(input float64) (r3.Transform, error) { return about(sunAngle(input)) }
				case stage.Carrier, stage.Shaft:
					move = func(input float64) (r3.Transform, error) { return about(carrierAngle(input)) }
				}
				for i := range stage.Planets {
					if body.body != stage.Planets[i] && body.body != stage.Bearings[i] {
						continue
					}
					phi := planetarygearbox.PlanetOrbitAngle(p, i)
					center := r3.NewVec(orbit*math.Cos(phi), orbit*math.Sin(phi), 0)
					move = func(input float64) (r3.Transform, error) {
						carried := carrierAngle(input)
						spin, err := r3.RotationAround(center, axis, units.Radians(-float64(p.RingTeeth)/float64(p.PlanetTeeth)*carried))
						if err != nil {
							return r3.Transform{}, err
						}
						around, err := about(carried)
						if err != nil {
							return r3.Transform{}, err
						}
						return spin.Then(around)
					}
				}
			}
		}
		if err := add(body, move); err != nil {
			return nil, err
		}
	}
	return movers, nil
}

// planetaryPoseModels returns the models of movers for one input angle, with the housing cut open when cut is set. Fixed bodies keep their meshes; a
// moving body's vertices are moved rigidly, so every frame draws the real tessellated bodies.
func planetaryPoseModels(movers []planetaryMover, input float64, cut bool) ([]solidlens.Model, error) {
	models := make([]solidlens.Model, len(movers))
	for i, mover := range movers {
		mesh := mover.mesh
		if mover.move != nil {
			t, err := mover.move(input)
			if err != nil {
				return nil, fmt.Errorf("move %s: %w", mover.body.name, err)
			}
			vertices := mesh.Vertices()
			for j, v := range vertices {
				vertices[j] = t.Apply(v)
			}
			moved, err := solidlens.NewMesh(vertices, mesh.Triangles())
			if err != nil {
				return nil, fmt.Errorf("pose %s: %w", mover.body.name, err)
			}
			mesh = moved
		}
		models[i] = planetaryModel(mesh, mover.body, cut)
	}
	return models, nil
}

// planetaryMotionPanel is one camera of the animation and its caption.
type planetaryMotionPanel struct {
	caption string
	camera  solidlens.Camera
	cut     bool
}

// planetaryMotionPanels are the animation's three views: the hero's cutaway, the input end seen from below and the
// output end seen from above.
func planetaryMotionPanels() []planetaryMotionPanel {
	return []planetaryMotionPanel{
		{"Cutaway", planetaryHeroCamera, true},
		{"Input end", planetaryViews[1].camera, false},
		{"Output end", planetaryViews[0].camera, false},
	}
}

// renderPlanetaryMotion writes an animation of the two-stage gearbox as its input turns, from three cameras: the
// cutaway of the hero, the input end and the output end. Both shafts carry their flat, so the turning of the
// input and the much slower turning of the output show. The first frame is the gearbox at rest, and a caption
// strip below the panels reads the input and output angles. The frames play forward and back.
func renderPlanetaryMotion(ctx context.Context, output string, width, height, frames int) error {
	box, p, err := buildPlanetaryGearbox(ctx, 2)
	if err != nil {
		return err
	}
	movers, err := planetaryMovers(ctx, box, p)
	if err != nil {
		return err
	}
	panels := planetaryMotionPanels()
	panelWidth, panelHeight := width/len(panels), height-2*planetaryCaptionHeight
	cameras := make([]solidlens.Camera, len(panels))
	for i, panel := range panels {
		rest, err := planetaryPoseModels(movers, 0, panel.cut)
		if err != nil {
			return err
		}
		cameras[i] = fitCamera(panel.camera, rest, float64(panelWidth)/float64(panelHeight), bevelViewMargin)
	}
	step := math.Pi / float64(p.SunTeeth)
	// The frames play forward and then back, not repeating the rest pose or the farthest pose.
	var order []int
	for frame := range frames {
		order = append(order, frame)
	}
	for frame := frames - 2; frame > 0; frame-- {
		order = append(order, frame)
	}
	ratio := float64(p.SunTeeth) / float64(p.SunTeeth+p.RingTeeth)
	background := solidlens.RGB(.94, .96, .98).NRGBA()
	face := basicfont.Face7x13
	var rendered []image.Image
	for frame := range frames {
		if err := ctx.Err(); err != nil {
			return err
		}
		input := step * float64(frame)
		canvas := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(background), image.Point{}, draw.Src)
		for i, panel := range panels {
			models, err := planetaryPoseModels(movers, input, panel.cut)
			if err != nil {
				return err
			}
			var buffer bytes.Buffer
			scene := bevelScene(cameras[i], models)
			if err := solidlens.RenderPNG(ctx, &buffer, scene, solidlens.Settings{Width: panelWidth, Height: panelHeight}); err != nil {
				return fmt.Errorf("render frame %d panel %d: %w", frame, i, err)
			}
			img, err := png.Decode(&buffer)
			if err != nil {
				return fmt.Errorf("decode frame %d panel %d: %w", frame, i, err)
			}
			at := image.Rect(i*panelWidth, planetaryCaptionHeight, (i+1)*panelWidth, planetaryCaptionHeight+panelHeight)
			draw.Draw(canvas, at, img, image.Point{}, draw.Src)
			caption(canvas, face, panel.caption, i*panelWidth+6, planetaryCaptionHeight-5)
		}
		output := input * math.Pow(ratio, float64(len(box.Stages)))
		caption(canvas, face, fmt.Sprintf("Input shaft %5.0f deg    Output shaft %5.1f deg    (%.1f to 1)",
			math.Mod(input*180/math.Pi, 3600), output*180/math.Pi, p.Ratio()), 6, height-6)
		rendered = append(rendered, canvas)
	}
	// One palette serves every frame, so colors do not flicker between frames. It holds the colors the frames
	// use most, and each pixel takes its nearest entry without dithering, which keeps flat areas flat and the
	// file small. The last entry is transparent: after the first frame each frame stores only the box around the
	// pixels that changed, with unchanged pixels inside it transparent, and keeps the frame before it underneath.
	colors := adaptivePalette(rendered, 255)
	clear := uint8(len(colors))
	colors = append(colors, color.RGBA{})
	nearest := make(map[color.RGBA]uint8)
	var full []*image.Paletted
	for _, index := range order {
		img := rendered[index]
		indexed := image.NewPaletted(img.Bounds(), colors)
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
				index, ok := nearest[c]
				if !ok {
					index = uint8(colors.Index(c))
					nearest[c] = index
				}
				indexed.SetColorIndex(x, y, index)
			}
		}
		full = append(full, indexed)
	}
	animation := &gif.GIF{LoopCount: 0}
	for i, indexed := range full {
		frame := indexed
		if i > 0 {
			frame = changedBox(full[i-1], indexed, clear)
		}
		animation.Image = append(animation.Image, frame)
		animation.Delay = append(animation.Delay, planetaryMotionDelay)
		animation.Disposal = append(animation.Disposal, gif.DisposalNone)
	}
	animation.Config = image.Config{ColorModel: colors, Width: width, Height: height}
	return writeWheelGIF(output, animation)
}

// changedBox returns the part of next inside the smallest box around the pixels that differ from before, with the
// pixels in the box that do not differ set to the transparent palette index clear. A frame that did not change
// at all becomes one transparent pixel.
func changedBox(before, next *image.Paletted, clear uint8) *image.Paletted {
	bounds := next.Bounds()
	box := image.Rectangle{Min: bounds.Max, Max: bounds.Min}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if before.ColorIndexAt(x, y) == next.ColorIndexAt(x, y) {
				continue
			}
			box.Min.X, box.Min.Y = min(box.Min.X, x), min(box.Min.Y, y)
			box.Max.X, box.Max.Y = max(box.Max.X, x+1), max(box.Max.Y, y+1)
		}
	}
	if box.Empty() {
		box = image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Min.X+1, bounds.Min.Y+1)
	}
	out := image.NewPaletted(box, next.Palette)
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			if before.ColorIndexAt(x, y) == next.ColorIndexAt(x, y) {
				out.SetColorIndex(x, y, clear)
				continue
			}
			out.SetColorIndex(x, y, next.ColorIndexAt(x, y))
		}
	}
	return out
}

// caption draws text with its baseline at (x, y).
func caption(canvas draw.Image, face font.Face, text string, x, y int) {
	drawer := font.Drawer{Dst: canvas, Src: image.NewUniform(color.RGBA{R: 26, G: 38, B: 46, A: 255}), Face: face,
		Dot: fixed.P(x, y)}
	drawer.DrawString(text)
}

// adaptivePalette returns the size colors that the images use most, found by grouping colors into buckets of
// their top five bits per channel and averaging each popular bucket.
func adaptivePalette(images []image.Image, size int) color.Palette {
	type bucket struct {
		count   int
		r, g, b int
	}
	buckets := make(map[uint16]*bucket)
	for _, img := range images {
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
				key := uint16(c.R>>3)<<10 | uint16(c.G>>3)<<5 | uint16(c.B>>3)
				b := buckets[key]
				if b == nil {
					b = &bucket{}
					buckets[key] = b
				}
				b.count++
				b.r += int(c.R)
				b.g += int(c.G)
				b.b += int(c.B)
			}
		}
	}
	popular := make([]*bucket, 0, len(buckets))
	for _, b := range buckets {
		popular = append(popular, b)
	}
	slices.SortFunc(popular, func(a, b *bucket) int { return cmp.Compare(b.count, a.count) })
	if len(popular) > size {
		popular = popular[:size]
	}
	colors := make(color.Palette, 0, len(popular))
	for _, b := range popular {
		colors = append(colors, color.RGBA{R: uint8(b.r / b.count), G: uint8(b.g / b.count), B: uint8(b.b / b.count), A: 255})
	}
	return colors
}
