package render

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"path/filepath"
	"strings"

	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
	"golang.org/x/image/font/basicfont"
)

// An engagement view is a close-up of one segment where its slider drives its rotor: the slider's drive cone in the
// rotor's cone pocket, the groove pin in the slider's stem, and the plates around them, all cut by one plane so that the
// plates in front of the plane do not hide the engagement. The parts come from the same default build as every other
// image, posed by the shaft angle theta like the motion clip: the cam lifts the slider and the rotor turns.
type sevenEngage struct {
	file  string
	seg   sevensegment.Segment
	theta float64             // the shaft angle that poses the segment
	plane solidlens.ClipPlane // the cut; the side the normal points to is removed
	cam   solidlens.Camera
	cams  bool // draw the segment's cam, to show the pin in the groove
}

// The slider and the groove pin are black in the model, like the rotor's body. In the engagement views they take their own
// colors, so that the cone in the notch and the pin in the stem read against the black rotor and each other.
var (
	sevenSliderTint = solidlens.RGB(.25, .42, .80)
	sevenPinTint    = solidlens.RGB(.85, .25, .30)
)

// sevenEngageModels returns the posed, cut parts of one engagement.
func sevenEngageModels(s partSet, dm sevensegment.Dimensions, e sevenEngage) ([]solidlens.Model, error) {
	seg := e.seg
	turn, err := r3.RotationAround(r3.Vec{Z: sevensegment.ShaftZ}, r3.Vec{Y: 1}, units.Degrees(e.theta-dm.Theta))
	if err != nil {
		return nil, err
	}
	letter := seg.String()
	var out []solidlens.Model
	for _, p := range s {
		var src solidlens.TriangleSource = p.mesh
		switch {
		case p.name == "Cradle Plate" || p.name == "Top Plate" || p.name == "Guide Plate":
		case p.name == "Flag "+letter+" Yellow" || p.name == "Flag "+letter+" Black":
			t, err := sevenFlagMove(seg, dm.FlagAngle(seg, e.theta)-dm.FlagAngle(seg, dm.Theta))
			if err != nil {
				return nil, err
			}
			src = nest(src, t)
			if strings.HasSuffix(p.name, "Yellow") {
				// The yellow body shares its faces with the black one; scaling it down by 0.5 % keeps the cut faces apart.
				low, high := renderkit.MeshBounds(p.mesh)
				src = nest(sevenShrunk{TriangleSource: p.mesh, centre: low.Add(high).Scale(.5), factor: .995}, t)
			}
		case p.name == "Slider "+letter || p.name == "Groove Pin "+letter:
			built, _ := sevensegment.Lit(dm.Digit, seg)
			dz := dm.Lift(seg, e.theta)
			if built {
				dz -= sevensegment.Travel
			}
			t, err := r3.Translation(r3.Vec{Z: dz})
			if err != nil {
				return nil, err
			}
			src = nest(src, t)
		case e.cams && p.name == "Cam "+letter:
			src = nest(src, turn)
		default:
			continue
		}
		if renderkit.ClipRemovesAll(p.mesh, e.plane) && !strings.HasPrefix(p.name, "Flag ") {
			continue
		}
		clipped, err := solidlens.ClipMesh(src, e.plane)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.name, err)
		}
		color := p.color
		switch {
		case strings.HasPrefix(p.name, "Slider "):
			color = sevenSliderTint
		case strings.HasPrefix(p.name, "Groove Pin "):
			color = sevenPinTint
		}
		m := renderkit.PartModel(clipped, color)
		m.BackMaterial = &solidlens.Material{Color: color, Unlit: true}
		out = append(out, m)
	}
	return out, nil
}

// sevenEngageViews lists the engagement images. Segment a stands for a, g and d, segment b for b, c, e and f. Digit 1 shows
// a unlit and digit 2 lights it (theta is 36 degrees a digit); digit 5 shows b unlit and digit 2 lights it. The mid-travel
// pose is half way up the cam ramp, which is done by 29.55 degrees.
func sevenEngageViews() []sevenEngage {
	cutY := func(y float64) solidlens.ClipPlane {
		return solidlens.ClipPlane{Origin: solidlens.Vec{Y: y}, Normal: solidlens.Vec{Y: -1}}
	}
	cutX := func(x float64) solidlens.ClipPlane {
		return solidlens.ClipPlane{Origin: solidlens.Vec{X: x}, Normal: solidlens.Vec{X: 1}}
	}
	// The drive cone stands 2.1 mm off the rotor axis (y 51.65 for a, x 22.65 for b), so the cut passes through the cone's axis.
	// a, g and d are seen from the front, b, c, e and f from the side.
	camA := solidlens.Camera{Position: solidlens.Vec{X: 2, Y: -200, Z: 123}, Target: solidlens.Vec{X: 2, Y: 51.65, Z: 123},
		Up: solidlens.Vec{Z: 1}, FOV: 7.5}
	camB := solidlens.Camera{Position: solidlens.Vec{X: 300, Y: 24.75, Z: 123}, Target: solidlens.Vec{X: 22.65, Y: 24.75, Z: 123},
		Up: solidlens.Vec{Z: 1}, FOV: 7.5}
	camPin := solidlens.Camera{Position: solidlens.Vec{X: 300, Y: 50.5, Z: 75}, Target: solidlens.Vec{Y: 50.5, Z: 75},
		Up: solidlens.Vec{Z: 1}, FOV: 6}
	return []sevenEngage{
		{file: "engage-a-rest.png", seg: sevensegment.SegA, theta: 36, plane: cutY(51.65), cam: camA},
		{file: "engage-a-mid.png", seg: sevensegment.SegA, theta: 36 + .5*sevensegment.SegmentsDoneBy, plane: cutY(51.65), cam: camA},
		{file: "engage-a-lit.png", seg: sevensegment.SegA, theta: 72, plane: cutY(51.65), cam: camA},
		{file: "engage-a-pin.png", seg: sevensegment.SegA, theta: 72, plane: cutX(0), cam: camPin, cams: true},
		{file: "engage-b-rest.png", seg: sevensegment.SegB, theta: 180, plane: cutX(22.65), cam: camB},
		{file: "engage-b-lit.png", seg: sevensegment.SegB, theta: 72, plane: cutX(22.65), cam: camB},
	}
}

// renderSevenSegmentEngage writes every engagement image to outputDir.
func renderSevenSegmentEngage(ctx context.Context, s partSet, outputDir string) error {
	dm, err := sevensegment.DefaultParams().Derive()
	if err != nil {
		return err
	}
	for _, e := range sevenEngageViews() {
		models, err := sevenEngageModels(s, dm, e)
		if err != nil {
			return err
		}
		if err := renderkit.RenderPNG(ctx, filepath.Join(outputDir, e.file), renderkit.Scene(e.cam, models), 1600, 900); err != nil {
			return err
		}
	}
	return nil
}

// renderSevenSegmentEngageMotion writes the animation of segment a's drive cone in its rotor's notch as the shaft turns one
// digit: the cam lifts the slider, the cone carries the rotor round, and the first and last poses hold for a moment.
func renderSevenSegmentEngageMotion(ctx context.Context, s partSet, output string, width, height int) error {
	dm, err := sevensegment.DefaultParams().Derive()
	if err != nil {
		return err
	}
	var rendered []image.Image
	const steps = 14
	for n := range steps + 1 {
		if err := ctx.Err(); err != nil {
			return err
		}
		e := sevenEngageViews()[0]
		e.theta = 36 + sevensegment.SegmentsDoneBy*float64(n)/steps
		models, err := sevenEngageModels(s, dm, e)
		if err != nil {
			return err
		}
		img, err := solidlens.Render(ctx, renderkit.Scene(e.cam, models), solidlens.Settings{Width: width, Height: height})
		if err != nil {
			return fmt.Errorf("render frame %d: %w", n, err)
		}
		canvas := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(canvas, canvas.Bounds(), img, image.Point{}, draw.Src)
		renderkit.Caption(canvas, basicfont.Face7x13, "Segment a: the drive cone carries the rotor as the cam lifts the slider", 8, 16)
		renderkit.Caption(canvas, basicfont.Face7x13, fmt.Sprintf("shaft %.1f deg  slider up %.2f mm", e.theta-36,
			dm.Lift(e.seg, e.theta)), 8, height-8)
		hold := 1
		if n == 0 || n == steps {
			hold = 5
		}
		for range hold {
			rendered = append(rendered, canvas)
		}
	}
	return renderkit.WriteGIF(output, renderkit.EncodeMotionGIF(rendered, width, height, 12))
}
