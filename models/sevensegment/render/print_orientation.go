package render

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// A print-orientation image shows one slider on a bed plate as the README says to print it: the broad flat face down and
// the drive cone pointing up. Each slider's tongue grows from its print face along one model axis and ends in the cone,
// so turning that axis to +Z puts the print face on the bed and the cone on top.
var sevenPrintViews = []struct {
	file   string // the image in the model's images directory
	part   string // the part that stands for the slider type
	axis   r3.Vec // the model axis the drive cone points along; the print face is on the opposite side
	toward r3.Vec // the camera's side, as a direction on the bed
}{
	{"print-slider-a-g-d.png", "Slider a", r3.Vec{X: 1}, r3.Vec{X: -1, Y: -1}},
	{"print-slider-b-c.png", "Slider b", r3.Vec{Y: 1}, r3.Vec{X: 1, Y: -1}},
	{"print-slider-e-f.png", "Slider e", r3.Vec{Y: 1}, r3.Vec{X: 1, Y: -1}},
}

// sevenPrintPose returns the slider's mesh turned so that axis points to +Z and moved so that its lowest point is on the
// bed plane z = 0 and its middle is over the origin. It also returns the size of the turned part.
func sevenPrintPose(mesh *decad.Mesh, axis r3.Vec) (solidlens.TriangleSource, r3.Vec, error) {
	turn, err := r3.Rotation(r3.Vec{X: axis.Y, Y: -axis.X}, units.Degrees(90))
	if err != nil {
		return nil, r3.Vec{}, err
	}
	turned := renderkit.TransformedMesh{TriangleSource: mesh, Transform: turn}
	var low, high r3.Vec
	for i, v := range turned.Vertices() {
		if i == 0 {
			low, high = v, v
			continue
		}
		low = r3.Vec{X: min(low.X, v.X), Y: min(low.Y, v.Y), Z: min(low.Z, v.Z)}
		high = r3.Vec{X: max(high.X, v.X), Y: max(high.Y, v.Y), Z: max(high.Z, v.Z)}
	}
	// Centre the part over the origin so the bed and the camera need no other reference.
	shift, err := r3.Translation(r3.Vec{X: -(low.X + high.X) / 2, Y: -(low.Y + high.Y) / 2, Z: -low.Z})
	if err != nil {
		return nil, r3.Vec{}, err
	}
	placed := renderkit.TransformedMesh{TriangleSource: turned, Transform: shift}
	return placed, high.Sub(low), nil
}

// sevenBedPlate returns a thin plate whose top face is the plane z = 0 and which reaches margin past the box
// [low, high] on every side.
func sevenBedPlate(low, high r3.Vec, margin, thickness float64) (*solidlens.Mesh, error) {
	x0, x1, y0, y1 := low.X-margin, high.X+margin, low.Y-margin, high.Y+margin
	vertices := []r3.Vec{
		{X: x0, Y: y0, Z: -thickness}, {X: x1, Y: y0, Z: -thickness}, {X: x1, Y: y1, Z: -thickness}, {X: x0, Y: y1, Z: -thickness},
		{X: x0, Y: y0}, {X: x1, Y: y0}, {X: x1, Y: y1}, {X: x0, Y: y1},
	}
	triangles := [][3]int{
		{0, 2, 1}, {0, 3, 2}, {4, 5, 6}, {4, 6, 7}, {0, 1, 5}, {0, 5, 4},
		{1, 2, 6}, {1, 6, 5}, {2, 3, 7}, {2, 7, 6}, {3, 0, 4}, {3, 4, 7},
	}
	return solidlens.NewMesh(vertices, triangles)
}

// sevenPrintScene draws one slider on its bed in its print orientation.
func sevenPrintScene(s partSet, view int) (solidlens.Scene, error) {
	v := sevenPrintViews[view]
	for _, p := range s {
		if p.name != v.part {
			continue
		}
		placed, size, err := sevenPrintPose(p.mesh, v.axis)
		if err != nil {
			return solidlens.Scene{}, err
		}
		bed, err := sevenBedPlate(size.Scale(-.5), size.Scale(.5), 6, 3)
		if err != nil {
			return solidlens.Scene{}, err
		}
		models := []solidlens.Model{
			renderkit.PartModel(placed, sevenSliderTint),
			renderkit.PartModel(bed, solidlens.RGB(.62, .66, .70)),
		}
		centre := r3.Vec{Z: size.Z / 2}
		d, _ := v.toward.Add(r3.Vec{Z: .8}).Normalize()
		camera := solidlens.Camera{Position: solidlens.Vec(centre.Add(d.Scale(4 * size.Len()))),
			Target: solidlens.Vec(centre), Up: solidlens.Vec{Z: 1}, FOV: 35}
		return renderkit.Scene(renderkit.FitCamera(camera, models, 16.0/9, .06), models), nil
	}
	return solidlens.Scene{}, fmt.Errorf("seven-segment print image %s: part %q not in the set", v.file, v.part)
}

// renderSevenSegmentPrint writes the print-orientation images to outputDir.
func renderSevenSegmentPrint(ctx context.Context, s partSet, outputDir string) error {
	for i, v := range sevenPrintViews {
		scene, err := sevenPrintScene(s, i)
		if err != nil {
			return err
		}
		if err := renderkit.RenderPNG(ctx, filepath.Join(outputDir, v.file), scene, 1600, 900); err != nil {
			return err
		}
	}
	return nil
}
