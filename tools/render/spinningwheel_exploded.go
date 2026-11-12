package main

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
)

// The exploded view of the spinning wheel moves each sub-assembly as one block. The model has 206 bodies and no
// parts.Set joint graph, so Decad's derived explode has nothing to read; the blocks follow the wheel's own
// containers instead (frame, wheel, treadle linkage, head, flyer, brake and drive band).

// wheelBlock is one sub-assembly of the exploded view.
type wheelBlock struct {
	name   string
	offset r3.Vec
}

var (
	wheelBlockFrame   = wheelBlock{"frame", r3.Vec{}}
	wheelBlockWheel   = wheelBlock{"wheel", r3.Vec{Y: 330}}
	wheelBlockTreadle = wheelBlock{"treadle", r3.Vec{Z: -250}}
	wheelBlockLinkage = wheelBlock{"linkage", r3.Vec{X: 330}}
	wheelBlockHead    = wheelBlock{"head", r3.Vec{Z: 230}}
	wheelBlockFlyer   = wheelBlock{"flyer", r3.Vec{Z: 560}}
	wheelBlockBrake   = wheelBlock{"brake", r3.Vec{X: -300, Y: -150, Z: 230}}
	wheelBlockDrive   = wheelBlock{"drive band", r3.Vec{X: 100, Y: -330, Z: 380}}
)

// wheelBlockOf returns the sub-assembly a part belongs to.
func wheelBlockOf(name, group, motion string) wheelBlock {
	switch {
	case strings.HasPrefix(name, "brake-"):
		return wheelBlockBrake
	case group == "drive-band":
		return wheelBlockDrive
	case strings.HasPrefix(name, "mother-"), strings.HasPrefix(name, "maiden-foot"),
		strings.HasSuffix(name, "-maiden"), name == "front-maiden-bush", strings.HasPrefix(name, "rear-flyer-bearing"):
		return wheelBlockHead
	case motion == "flyer", motion == "bobbin", strings.HasPrefix(name, "bobbin-"):
		return wheelBlockFlyer
	case motion == "wheel":
		return wheelBlockWheel
	case motion == "treadle", motion == "compliant":
		return wheelBlockTreadle
	case motion == "footman", motion == "gimbal":
		return wheelBlockLinkage
	}
	return wheelBlockFrame
}

func init() {
	presets["spinningwheel/exploded"] = preset{width: wheelExplodedWidth, height: wheelExplodedHeight,
		scene: func(ctx context.Context) (solidlens.Scene, error) {
			meshes, _, err := spinningWheelModels(ctx)
			if err != nil {
				return solidlens.Scene{}, err
			}
			return spinningWheelExplodedScene(meshes), nil
		}}
}

var wheelExplodedCamera = solidlens.Camera{
	Position: solidlens.Vec{X: 1000, Y: -1600, Z: 1050},
	Target:   solidlens.Vec{X: 30, Y: 20, Z: 500},
	Up:       solidlens.Vec{Z: 1},
	FOV:      32,
}

const (
	wheelExplodedWidth  = 1600
	wheelExplodedHeight = 1200
)

// spinningWheelExplodedScene draws the whole wheel with each sub-assembly moved to its own place.
func spinningWheelExplodedScene(meshes []wheelMesh) solidlens.Scene {
	models := make([]solidlens.Model, 0, len(meshes))
	for _, m := range meshes {
		model := m.model
		offset := wheelBlockOf(m.part.Name, m.part.Group, m.part.Motion).offset
		model.Mesh = placedMesh{TriangleSource: model.Mesh, scale: 1, offset: offset}
		models = append(models, model)
	}
	aspect := float64(wheelExplodedWidth) / wheelExplodedHeight
	return bevelScene(fitCamera(wheelExplodedCamera, models, aspect, .06), models)
}

// wheelKind is one distinct body of the wheel with every identical body that shares its row of the parts table.
type wheelKind struct {
	// slug names the part image and the table row.
	slug    string
	members []wheelMesh
}

var wheelNumber = regexp.MustCompile(`-\d+`)

// wheelKinds groups the wheel's bodies that have the same volume, surface area and face count, in model order.
// Spokes, collars, spacers and fasteners that repeat a body fall into one kind. A kind takes the name of its first
// body without the counting digits; a later kind that would share a name gets a letter.
func wheelKinds(meshes []wheelMesh) ([]wheelKind, error) {
	type key struct {
		volume, area int64
		faces        int
	}
	index := map[key]int{}
	var kinds []wheelKind
	for _, m := range meshes {
		volume, err := m.part.Body.Volume()
		if err != nil {
			return nil, fmt.Errorf("measure %s: %w", m.part.Name, err)
		}
		area, err := m.part.Body.Area()
		if err != nil {
			return nil, fmt.Errorf("measure %s: %w", m.part.Name, err)
		}
		k := key{int64(math.Round(volume.Value.Base() * 1e9 * 20)), int64(math.Round(area.Value.Base() * 1e6 * 20)), len(m.part.Body.Faces())}
		i, ok := index[k]
		if !ok {
			i = len(kinds)
			index[k] = i
			kinds = append(kinds, wheelKind{})
		}
		kinds[i].members = append(kinds[i].members, m)
	}
	used := map[string]int{}
	for i := range kinds {
		stem := wheelNumber.ReplaceAllString(kinds[i].members[0].part.Name, "")
		used[stem]++
		kinds[i].slug = stem
		if used[stem] > 1 {
			kinds[i].slug = fmt.Sprintf("%s-%c", stem, 'a'+used[stem]-1)
		}
	}
	return kinds, nil
}

// wheelPartCamera looks at one part from the front and above; fitCamera frames it.
var wheelPartCamera = solidlens.Camera{
	Position: solidlens.Vec{X: 70, Y: -150, Z: 110},
	Up:       solidlens.Vec{Z: 1},
	FOV:      35,
}

// spinningWheelPartScene shows the first body of a kind alone.
func spinningWheelPartScene(kind wheelKind) solidlens.Scene {
	model := kind.members[0].model
	camera := wheelPartCamera
	box, err := kind.members[0].part.Body.Bounds()
	if err == nil {
		c := box.Min.Add(box.Max).Scale(.5)
		camera.Target = solidlens.Vec(c)
		// Back the camera off in proportion to the part, so a 500 mm leg and a 3 mm set screw both stay in front of it.
		d := box.Max.Sub(box.Min).Len() * 3
		camera.Position = solidlens.Vec{X: c.X + d*.35, Y: c.Y - d*.75, Z: c.Z + d*.55}
	}
	models := []solidlens.Model{model}
	return bevelScene(fitCamera(camera, models, 16.0/9, .1), models)
}

// renderSpinningWheelExploded writes the exploded view and one image per distinct body into out and out/parts.
func renderSpinningWheelExploded(ctx context.Context, meshes []wheelMesh, out string) error {
	name := filepath.Join(out, "exploded.png")
	if err := renderPNG(ctx, name, spinningWheelExplodedScene(meshes), wheelExplodedWidth, wheelExplodedHeight); err != nil {
		return fmt.Errorf("render spinning wheel exploded view: %w", err)
	}
	kinds, err := wheelKinds(meshes)
	if err != nil {
		return err
	}
	for _, kind := range kinds {
		name := filepath.Join(out, "parts", kind.slug+".png")
		if err := renderPNG(ctx, name, spinningWheelPartScene(kind), 480, 270); err != nil {
			return fmt.Errorf("render spinning wheel part %s: %w", kind.slug, err)
		}
	}
	return nil
}
