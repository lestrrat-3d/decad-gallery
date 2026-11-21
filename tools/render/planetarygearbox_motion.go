package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
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
	body       planetaryBody
	mesh       solidlens.TriangleSource
	move       func(input float64) (r3.Transform, error)
	preclipped bool
}

// planetaryMovers pairs every body with how it moves when the input shaft turns by an angle. The kinematics are
// planetarygearbox.Motion's, driven by the sun: the ring and the covers are fixed, the input shaft and the first sun
// turn with the input, and each later part turns as that type describes.
func planetaryMovers(ctx context.Context, box planetarygearbox.Gearbox, p planetarygearbox.Params) ([]planetaryMover, error) {
	motion, err := planetarygearbox.NewMotion(p, planetarygearbox.SunDriven)
	if err != nil {
		return nil, err
	}
	var movers []planetaryMover
	for _, body := range planetaryBodies(box) {
		mesh, err := planetaryMesh(ctx, body)
		if err != nil {
			return nil, err
		}
		name := body.name
		_, moves, err := motion.Transform(name, 0)
		if err != nil {
			return nil, err
		}
		var move func(float64) (r3.Transform, error)
		if moves {
			move = func(input float64) (r3.Transform, error) {
				t, _, err := motion.Transform(name, input)
				return t, err
			}
		}
		movers = append(movers, planetaryMover{body: body, mesh: mesh, move: move})
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
		if cut && mover.preclipped {
			models[i].BackMaterial = models[i].Clip.Cap
			models[i].Clip = nil
		}
	}
	return models, nil
}

// planetaryPanelMovers prepares reusable stationary housing meshes for each panel.
func planetaryPanelMovers(movers []planetaryMover, cut bool) ([]planetaryMover, error) {
	panel := append([]planetaryMover(nil), movers...)
	for i, mover := range panel {
		if !mover.body.housing || mover.move != nil {
			continue
		}
		var mesh *solidlens.Mesh
		var err error
		if cut {
			model := planetaryModel(mover.mesh, mover.body, true)
			mesh, err = solidlens.ClipMesh(mover.mesh, *model.Clip)
			panel[i].preclipped = true
		} else {
			mesh, err = solidlens.MeshFrom(mover.mesh)
		}
		if err != nil {
			return nil, fmt.Errorf("prepare %s for motion: %w", mover.body.name, err)
		}
		panel[i].mesh = mesh
	}
	return panel, nil
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

// renderPlanetaryPanels runs at most two independent panel renders at a time.
func renderPlanetaryPanels(ctx context.Context, count int, render func(context.Context, int) (image.Image, error)) ([]image.Image, error) {
	images := make([]image.Image, count)
	for start := 0; start < count; start += 2 {
		batchCtx, cancel := context.WithCancel(ctx)
		type result struct {
			image image.Image
			err   error
		}
		done := make(chan result, 1)
		go func(index int) {
			img, err := render(batchCtx, index)
			if err != nil {
				cancel()
			}
			done <- result{img, err}
		}(start)
		var second result
		if start+1 < count {
			second.image, second.err = render(batchCtx, start+1)
			if second.err != nil {
				cancel()
			}
		}
		first := <-done
		cancel()
		if err := errors.Join(first.err, second.err); err != nil {
			return nil, err
		}
		images[start] = first.image
		if start+1 < count {
			images[start+1] = second.image
		}
	}
	return images, nil
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
	return renderPlanetaryMotionFrom(ctx, output, width, height, frames, box, p)
}

func renderPlanetaryMotionFrom(ctx context.Context, output string, width, height, frames int,
	box planetarygearbox.Gearbox, p planetarygearbox.Params) error {
	movers, err := planetaryMovers(ctx, box, p)
	if err != nil {
		return err
	}
	panels := planetaryMotionPanels()
	panelWidth, panelHeight := width/len(panels), height-2*planetaryCaptionHeight
	cameras := make([]solidlens.Camera, len(panels))
	panelMovers := make([][]planetaryMover, len(panels))
	for i, panel := range panels {
		panelMovers[i], err = planetaryPanelMovers(movers, panel.cut)
		if err != nil {
			return err
		}
		rest, err := planetaryPoseModels(movers, 0, panel.cut)
		if err != nil {
			return err
		}
		cameras[i] = renderkit.FitCamera(panel.camera, rest, float64(panelWidth)/float64(panelHeight), bevelViewMargin)
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
	motion, err := planetarygearbox.NewMotion(p, planetarygearbox.SunDriven)
	if err != nil {
		return err
	}
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
		images, err := renderPlanetaryPanels(ctx, len(panels), func(ctx context.Context, i int) (image.Image, error) {
			panel := panels[i]
			models, err := planetaryPoseModels(panelMovers[i], input, panel.cut)
			if err != nil {
				return nil, err
			}
			scene := renderkit.Scene(cameras[i], models)
			img, err := solidlens.Render(ctx, scene, solidlens.Settings{Width: panelWidth, Height: panelHeight})
			if err != nil {
				return nil, fmt.Errorf("render frame %d panel %d: %w", frame, i, err)
			}
			return img, nil
		})
		if err != nil {
			return err
		}
		for i, panel := range panels {
			at := image.Rect(i*panelWidth, planetaryCaptionHeight, (i+1)*panelWidth, planetaryCaptionHeight+panelHeight)
			draw.Draw(canvas, at, images[i], image.Point{}, draw.Src)
			caption(canvas, face, panel.caption, i*panelWidth+6, planetaryCaptionHeight-5)
		}
		output := motion.OutputAngle(input)
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
	return renderkit.ChangedBox(before, next, clear)
}

// caption draws text with its baseline at (x, y).
func caption(canvas draw.Image, face font.Face, text string, x, y int) {
	renderkit.Caption(canvas, face, text, x, y)
}

// adaptivePalette returns the size colors that the images use most, found by grouping colors into buckets of
// their top five bits per channel and averaging each popular bucket.
func adaptivePalette(images []image.Image, size int) color.Palette {
	return renderkit.AdaptivePalette(images, size)
}
