package main

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"math"
	"slices"

	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// planetaryMotionFrames is the frame count of the motion animation. The default one-stage gearbox looks the
// same again after its carrier turns a third of a turn, because the three planets then swap places and every
// tooth count divides evenly, so the animation covers exactly that and loops without a jump.
const planetaryMotionFrames = 48

// planetaryMotionDelay is the time of one frame in hundredths of a second.
const planetaryMotionDelay = 5

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
		switch {
		case body.name == planetarygearbox.InputShaftName:
			move = about
		default:
			for k, stage := range box.Stages {
				sunAngle := func(input float64) float64 { return input * math.Pow(ratio, float64(k)) }
				carrierAngle := func(input float64) float64 { return sunAngle(input) * ratio }
				switch body.body {
				case stage.Sun:
					move = func(input float64) (r3.Transform, error) { return about(sunAngle(input)) }
				case stage.Carrier:
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

// planetaryPoseModels returns the models of movers for one input angle. Fixed bodies keep their meshes; a
// moving body's vertices are moved rigidly, so every frame draws the real tessellated bodies.
func planetaryPoseModels(movers []planetaryMover, input float64) ([]solidlens.Model, error) {
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
		models[i] = planetaryModel(mesh, mover.body, true)
	}
	return models, nil
}

// renderPlanetaryMotion writes an animation of the default one-stage gearbox, ring and covers cut open, as its
// input turns. The first frame is the assembled gearbox at rest.
func renderPlanetaryMotion(ctx context.Context, output string, width, height, frames int) error {
	box, p, err := buildPlanetaryGearbox(ctx, 1)
	if err != nil {
		return err
	}
	movers, err := planetaryMovers(ctx, box, p)
	if err != nil {
		return err
	}
	rest, err := planetaryPoseModels(movers, 0)
	if err != nil {
		return err
	}
	camera := fitCamera(planetaryHeroCamera, rest, float64(width)/float64(height), bevelViewMargin)
	// A third of a carrier turn is (Zs+Zr)/Zs sun turns per carrier turn, three times over.
	input := 2 * math.Pi / float64(p.PlanetCount) * float64(p.SunTeeth+p.RingTeeth) / float64(p.SunTeeth)
	var rendered []image.Image
	for frame := range frames {
		if err := ctx.Err(); err != nil {
			return err
		}
		models, err := planetaryPoseModels(movers, input*float64(frame)/float64(frames))
		if err != nil {
			return err
		}
		var buffer bytes.Buffer
		scene := bevelScene(camera, models)
		if err := solidlens.RenderPNG(ctx, &buffer, scene, solidlens.Settings{Width: width, Height: height}); err != nil {
			return fmt.Errorf("render frame %d: %w", frame, err)
		}
		img, err := png.Decode(&buffer)
		if err != nil {
			return fmt.Errorf("decode frame %d: %w", frame, err)
		}
		rendered = append(rendered, img)
	}
	// One palette serves every frame, so colors do not flicker between frames. It holds the colors the frames
	// use most, and each pixel takes its nearest entry without dithering, which keeps flat areas flat and the
	// file small.
	colors := adaptivePalette(rendered, 255)
	nearest := make(map[color.RGBA]uint8)
	animation := &gif.GIF{LoopCount: 0}
	for _, img := range rendered {
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
		animation.Image = append(animation.Image, indexed)
		animation.Delay = append(animation.Delay, planetaryMotionDelay)
		animation.Disposal = append(animation.Disposal, gif.DisposalNone)
	}
	return writeWheelGIF(output, animation)
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
