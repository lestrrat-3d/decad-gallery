package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// The display lies with the face up: +Z up, +Y toward segment a, the knob toward -Y. The set is built once at digit 2.
// Every printed part is drawn in one fixed colour, as a filament: it never changes with the digit, the angle or the view.
// A rotor is two bodies, a yellow slab and a black body; the sliders are black.
var (
	sevenWallColor    = solidlens.RGB(.58, .61, .66) // walls and detent plate
	sevenPlateColor   = solidlens.RGB(.84, .86, .88) // guide and top plates
	sevenCamColor     = solidlens.RGB(.20, .56, .60) // cams and ratchet wheel
	sevenYellow       = solidlens.RGB(1.0, .78, .05) // the yellow slab of every rotor
	sevenBlack        = solidlens.RGB(.08, .08, .09) // the black body of every rotor and every slider
	sevenShaftColor   = solidlens.RGB(.16, .17, .20) // shaft, collar ring and knob
	sevenHardware     = solidlens.RGB(.55, .57, .60) // bought bearings, bolts and nuts
	sevenDigitDefault = sevensegment.DefaultParams().Digit
)

// sevenCut is a clip plane whose cut face has the part's own colour.
func sevenCut(plane solidlens.ClipPlane, c solidlens.Color) *solidlens.ClipPlane {
	plane.Cap = &solidlens.Material{Color: c, Unlit: true}
	return &plane
}

// sevenModels returns the parts moved by pull, cut by clip when it is not nil; a part wholly on the removed side is
// left out. Each part keeps its one colour, and its cut face has that colour too.
func sevenModels(s clutchSet, pull map[string]r3.Vec, clip *solidlens.ClipPlane) ([]solidlens.Model, error) {
	var out []solidlens.Model
	for _, part := range s {
		if clip != nil && clipRemovesAll(part.mesh, *clip) {
			continue
		}
		move, err := r3.Translation(pull[part.name])
		if err != nil {
			return nil, err
		}
		m := clutchModel(rotatedMesh{TriangleSource: part.mesh, move: move}, part.color)
		if clip != nil {
			m.Clip = sevenCut(*clip, part.color)
		}
		out = append(out, m)
	}
	return out, nil
}

// sevenCenter is the middle of the model, where the cameras look.
var sevenCenter = solidlens.Vec{Y: -3, Z: 42}

// sevenExplode is each part's offset in the exploded view, from "Rendering" in the design: the plates up in Z, the
// walls, detent plate, bearings, collar ring and knob out along Y. Bolts follow the part their head sits in, 15 mm
// more along -Dir, and nuts follow the part their pocket is in, 15 mm more along +Dir.
var sevenExplode = map[string]r3.Vec{
	"Guide Plate": {Z: 10}, "Top Plate": {Z: 50}, "Cradle Plate": {Z: 15},
	// The sliders stand beside the stack, each pair stepped up in Z, so that none hides another or sits behind a plate: a, g
	// and d farthest out on +X, b and c on +X, e and f on -X, f higher (checked by eye on the rendered image).
	"Slider a": {X: 105}, "Slider g": {X: 105, Z: 45}, "Slider d": {X: 105, Z: 90}, "Slider b": {X: 60}, "Slider c": {X: 60, Z: 45},
	"Slider e": {X: -85, Z: 30}, "Slider f": {X: -85, Z: 100}, "Front Wall": {Y: -50}, "Detent Plate": {Y: -30},
	// The front bearing drops below the shaft, so the front wall does not hide it.
	"Front Bearing": {Y: -40, Z: -55}, "Collar Ring": {Y: -60}, "Knob": {Y: -75}, "Rear Wall": {Y: 45}, "Rear Bearing": {Y: 25},
}

// sevenSegmentCentre is the middle of segment s in plan.
func sevenSegmentCentre(s sevensegment.Segment) (x, y float64) {
	span := sevensegment.SegmentSpan
	switch s {
	case sevensegment.SegA:
		return 0, span
	case sevensegment.SegG:
		return 0, 0
	case sevensegment.SegD:
		return 0, -span
	case sevensegment.SegB:
		return span / 2, span / 2
	case sevensegment.SegC:
		return span / 2, -span / 2
	case sevensegment.SegE:
		return -span / 2, -span / 2
	}
	return -span / 2, span / 2
}

// sevenFlagName is the part name of segment seg's yellow or black rotor body.
func sevenFlagName(seg sevensegment.Segment, kind string) string {
	return "Flag " + seg.String() + " " + kind
}

// sevenFlagSegment returns the segment a "Flag x Kind" part belongs to.
func sevenFlagSegment(name string) sevensegment.Segment { return sevensegment.Segments()[name[5]-'a'] }

// sevenFlagMove turns a rotor by dphi degrees about its axis. The axis runs along X for the horizontal segments and
// along Y for the vertical ones.
func sevenFlagMove(seg sevensegment.Segment, dphi float64) (r3.Transform, error) {
	var point, axis r3.Vec
	span := sevensegment.SegmentSpan
	switch seg {
	case sevensegment.SegA:
		point, axis = r3.Vec{Y: span}, r3.Vec{X: 1}
	case sevensegment.SegG:
		point, axis = r3.Vec{}, r3.Vec{X: 1}
	case sevensegment.SegD:
		point, axis = r3.Vec{Y: -span}, r3.Vec{X: 1}
	case sevensegment.SegB:
		point, axis = r3.Vec{X: span / 2, Y: span / 2}, r3.Vec{Y: 1}
	case sevensegment.SegC:
		point, axis = r3.Vec{X: span / 2, Y: -span / 2}, r3.Vec{Y: 1}
	case sevensegment.SegE:
		point, axis = r3.Vec{X: -span / 2, Y: -span / 2}, r3.Vec{Y: 1}
	default:
		point, axis = r3.Vec{X: -span / 2, Y: span / 2}, r3.Vec{Y: 1}
	}
	point.Z = sevensegment.RotorAxisZ
	return r3.RotationAround(point, axis, units.Degrees(dphi))
}

// sevenCamshaftOrder lists the parts that slide off the shaft, rear first.
var sevenCamshaftOrder = []string{"Cam a", "Cam f", "Cam b", "Cam g", "Cam c", "Cam e", "Cam d", "Ratchet Wheel",
	"Collar Ring", "Knob"}

// sevenSet builds the default display once and tessellates its 60 bodies.
func buildSevenSegmentSet(ctx context.Context) (clutchSet, error) {
	doc := decad.New()
	m, err := sevensegment.Build(ctx, doc, sevensegment.DefaultParams())
	if err != nil {
		return nil, fmt.Errorf("build seven-segment display: %w", err)
	}
	type named struct {
		name  string
		body  *decad.Body
		color solidlens.Color
	}
	parts := []named{
		{"Front Wall", m.FrontWall, sevenWallColor}, {"Rear Wall", m.RearWall, sevenWallColor},
		{"Detent Plate", m.DetentPlate, sevenWallColor}, {"Guide Plate", m.GuidePlate, sevenPlateColor},
		{"Top Plate", m.TopPlate, sevenPlateColor}, {"Cradle Plate", m.CradlePlate, sevenWallColor}, {"Shaft", m.Shaft, sevenShaftColor},
		{"Ratchet Wheel", m.RatchetWheel, sevenCamColor}, {"Collar Ring", m.CollarRing, sevenShaftColor},
		{"Knob", m.Knob, sevenShaftColor}, {"Front Bearing", m.Bearings[0].Body, sevenHardware},
		{"Rear Bearing", m.Bearings[1].Body, sevenHardware},
	}
	for _, s := range sevensegment.Segments() {
		parts = append(parts, named{"Cam " + s.String(), m.Cams[s], sevenCamColor},
			named{"Slider " + s.String(), m.Sliders[s], sevenBlack},
			named{sevenFlagName(s, "Yellow"), m.Flags[s].Yellow, sevenYellow},
			named{sevenFlagName(s, "Black"), m.Flags[s].Black, sevenBlack})
		// The rotors rise 30 and spread outward from the middle of the face, so that none hides another.
		cx, cy := sevenSegmentCentre(s)
		sevenExplode[sevenFlagName(s, "Yellow")] = r3.Vec{X: .9 * cx, Y: .9 * cy, Z: 30}
		sevenExplode[sevenFlagName(s, "Black")] = r3.Vec{X: .9 * cx, Y: .9 * cy, Z: 30}
	}
	for i := range m.Bolts {
		parts = append(parts, named{fmt.Sprintf("Bolt %d", i+1), m.Bolts[i].Body, sevenHardware},
			named{fmt.Sprintf("Nut %d", i+1), m.Nuts[i].Body, sevenHardware})
	}
	set := make(clutchSet, 0, len(parts))
	for _, p := range parts {
		tol := .04
		if strings.HasPrefix(p.name, "Bolt") || strings.HasPrefix(p.name, "Nut") {
			tol = .01
		}
		mesh, err := p.body.Tessellate(ctx, units.Millimeters(tol), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return nil, fmt.Errorf("mesh %s: %w", p.name, err)
		}
		set = append(set, clutchPart{p.name, mesh, p.color})
	}
	// The exploded offsets of the fasteners follow the part they sit in.
	dm := m.Dimensions
	for i, site := range dm.Sites {
		headIn, nutIn := "Top Plate", "Front Wall"
		switch {
		case i < 4:
			if i >= 2 {
				nutIn = "Rear Wall"
			}
		case i < 8:
			headIn, nutIn = "Front Wall", "Guide Plate"
			if i >= 6 {
				headIn = "Rear Wall"
			}
		default:
			headIn, nutIn = "Detent Plate", "Front Wall"
		}
		dir := r3.Vec(site.Dir)
		sevenExplode[fmt.Sprintf("Bolt %d", i+1)] = sevenExplode[headIn].Sub(dir.Scale(15))
		sevenExplode[fmt.Sprintf("Nut %d", i+1)] = sevenExplode[nutIn].Add(dir.Scale(15))
	}
	return set, nil
}

var sevenCameras = map[string]solidlens.Camera{
	"hero": {Position: solidlens.Vec{X: 110, Y: -125, Z: 235}, Target: sevenCenter, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"top":  {Position: solidlens.Vec{Y: -3, Z: 320}, Target: solidlens.Vec{Y: -3, Z: 84}, Up: solidlens.Vec{Y: 1}, FOV: 30},
	"oblique45": {Position: solidlens.Vec{Y: -3 - 160, Z: 84 + 160}, Target: solidlens.Vec{Y: -3, Z: 84},
		Up: solidlens.Vec{Z: 1}, FOV: 30},
	"oblique30": {Position: solidlens.Vec{Y: -3 - 190, Z: 84 + 110}, Target: solidlens.Vec{Y: -3, Z: 84},
		Up: solidlens.Vec{Z: 1}, FOV: 30},
	"front": {Position: solidlens.Vec{Y: -300, Z: 60}, Target: sevenCenter, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"side":  {Position: solidlens.Vec{X: 300, Y: -20, Z: 60}, Target: sevenCenter, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"cutaway": {Position: solidlens.Vec{X: 300, Y: -20, Z: 60}, Target: sevenCenter, Up: solidlens.Vec{Z: 1},
		FOV: 35},
	"detent": {Position: solidlens.Vec{Y: -160, Z: 28}, Target: solidlens.Vec{Y: -43.25, Z: 28}, Up: solidlens.Vec{Z: 1},
		FOV: 30},
	"exploded": {Position: solidlens.Vec{X: 330, Y: -330, Z: 230}, Target: solidlens.Vec{Y: -3, Z: 62},
		Up: solidlens.Vec{Z: 1}, FOV: 35},
	"exploded-camshaft": {Position: solidlens.Vec{X: 330, Y: -330, Z: 170}, Target: solidlens.Vec{Y: -70, Z: 28},
		Up: solidlens.Vec{Z: 1}, FOV: 35},
}

var (
	sevenAxisCut   = solidlens.ClipPlane{Normal: solidlens.Vec{X: 1}}
	sevenDetentCut = solidlens.ClipPlane{Origin: solidlens.Vec{Y: -43.25}, Normal: solidlens.Vec{Y: -1}}
)

// sevenCamshaftPull spreads the parts on the shaft along -Y, 12 mm per part from the rear.
func sevenCamshaftPull() map[string]r3.Vec {
	pull := map[string]r3.Vec{}
	for i, name := range sevenCamshaftOrder {
		pull[name] = r3.Vec{Y: -12 * float64(i+1)}
	}
	pull["Shaft"] = r3.Vec{Y: 100} // the shaft slides out of the rear, so every cam's bore shows
	return pull
}

// view draws one named view of the display.
func sevenSegmentView(s clutchSet, name string) (solidlens.Scene, error) {
	camera, ok := sevenCameras[name]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown seven-segment view %q", name)
	}
	var clip *solidlens.ClipPlane
	var pull map[string]r3.Vec
	switch name {
	case "cutaway":
		clip = &sevenAxisCut
	case "detent":
		clip = &sevenDetentCut
	case "exploded":
		pull = sevenExplode
	case "exploded-camshaft":
		pull = sevenCamshaftPull()
	}
	set := s
	if name == "exploded-camshaft" { // the frame, bearings and fasteners are left out so the shaft and every cam show
		set = nil
		for _, p := range s {
			if p.name == "Shaft" || p.name == "Ratchet Wheel" || p.name == "Collar Ring" || p.name == "Knob" ||
				strings.HasPrefix(p.name, "Cam ") {
				set = append(set, p)
			}
		}
	}
	if name == "top" || name == "oblique45" || name == "oblique30" { // the face: the top plate and the rotors in its windows
		set = nil
		for _, p := range s {
			// The face: the top plate, the rotors in its windows and the four bolts sunk flush in its corners.
			if p.name == "Top Plate" || strings.HasPrefix(p.name, "Flag ") || sevenFaceBolt(p.name) {
				set = append(set, p)
			}
		}
	}
	models, err := sevenModels(set, pull, clip)
	if err != nil {
		return solidlens.Scene{}, err
	}
	aspect := 16.0 / 9
	margin := .08
	if strings.HasPrefix(name, "exploded") {
		margin = .03 // the exploded view fills at least 80% of the frame
	}
	return renderkit.Scene(renderkit.FitCamera(camera, models, aspect, margin), models), nil
}

// sevenFaceBolt reports whether name is one of the four top-plate bolts, whose heads lie flush in the face.
func sevenFaceBolt(name string) bool {
	return name == "Bolt 1" || name == "Bolt 2" || name == "Bolt 3" || name == "Bolt 4"
}

// sevenDigitModels returns the face of the display showing digit k, placed at at: the top plate, its four bolts and the rotors, each
// rotor turned from its built angle to the one the digit asks of it. Every part keeps its one colour.
func sevenDigitModels(s clutchSet, dm sevensegment.Dimensions, k int, at r3.Vec) ([]solidlens.Model, error) {
	var models []solidlens.Model
	place, err := r3.Translation(at)
	if err != nil {
		return nil, err
	}
	for _, p := range s {
		isFlag := strings.HasPrefix(p.name, "Flag ")
		if !isFlag && p.name != "Top Plate" && !sevenFaceBolt(p.name) {
			continue
		}
		var src solidlens.TriangleSource = p.mesh
		if isFlag {
			seg := sevenFlagSegment(p.name)
			turn, err := sevenFlagMove(seg, dm.FlagAngle(seg, 36*float64(k))-dm.FlagAngle(seg, dm.Theta))
			if err != nil {
				return nil, err
			}
			src = rotatedMesh{TriangleSource: src, move: turn}
		}
		models = append(models, clutchModel(rotatedMesh{TriangleSource: src, move: place}, p.color))
	}
	return models, nil
}

// sevenSegmentDigits draws digits 0 to 9 in a 5 by 2 grid, seen from the front at an angle so the turned rotors show
// their yellow faces.
func sevenSegmentDigits(s clutchSet) (solidlens.Scene, error) {
	dm, err := sevensegment.DefaultParams().Derive()
	if err != nil {
		return solidlens.Scene{}, err
	}
	var models []solidlens.Model
	for k := range 10 {
		m, err := sevenDigitModels(s, dm, k, r3.Vec{X: float64(k%5) * 66, Y: -float64(k/5) * 118})
		if err != nil {
			return solidlens.Scene{}, err
		}
		models = append(models, m...)
	}
	camera := solidlens.Camera{Position: solidlens.Vec{X: 132, Y: -460, Z: 330}, Target: solidlens.Vec{X: 132, Y: -62, Z: 84},
		Up: solidlens.Vec{Z: 1}, FOV: 30}
	return renderkit.Scene(renderkit.FitCamera(camera, models, 16.0/9, .04), models), nil
}

// sevenUnposed returns the model of a camshaft part turned back from the digit's pose, so its letters stand upright.
func sevenUnposed(p clutchPart, dx, dz float64) (solidlens.Model, error) {
	turn, err := r3.RotationAround(r3.Vec{Z: sevensegment.ShaftZ}, r3.Vec{Y: 1},
		units.Degrees(-36*float64(sevenDigitDefault)))
	if err != nil {
		return solidlens.Model{}, err
	}
	shift, err := r3.Translation(r3.Vec{X: dx, Z: dz})
	if err != nil {
		return solidlens.Model{}, err
	}
	return clutchModel(rotatedMesh{TriangleSource: rotatedMesh{TriangleSource: p.mesh, move: turn}, move: shift},
		sevenCamColor), nil
}

// sevenSegmentCams draws the seven cams side by side, a to d on the top row and e to g below, seen from behind so each cam's engraved
// letter shows on its back face.
func sevenSegmentCams(s clutchSet) (solidlens.Scene, error) {
	var models []solidlens.Model
	for k, seg := range sevensegment.Segments() {
		for _, p := range s {
			if p.name != "Cam "+seg.String() {
				continue
			}
			m, err := sevenUnposed(p, -(float64(k%4)-1.5)*62, -float64(k/4)*62)
			if err != nil {
				return solidlens.Scene{}, err
			}
			models = append(models, m)
		}
	}
	camera := solidlens.Camera{Position: solidlens.Vec{Y: 500, Z: -3}, Target: solidlens.Vec{Y: 20, Z: -3},
		Up: solidlens.Vec{Z: 1}, FOV: 30}
	return renderkit.Scene(renderkit.FitCamera(camera, models, 16.0/9, .05), models), nil
}

// sevenSegmentPreset returns the preset for a named view of the default display.
func sevenSegmentPreset(name string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		s, err := buildSevenSegmentSet(ctx)
		if err != nil {
			return solidlens.Scene{}, err
		}
		if name == "digits" {
			return sevenSegmentDigits(s)
		}
		if name == "cams" {
			return sevenSegmentCams(s)
		}
		return sevenSegmentView(s, name)
	}
}

// sevenPartImages lists the distinct bodies and their images: the printed parts (sliders a, g and d share one image,
// b and c another, e and f a third; rotors a, g and d are type A, b, c, e and f type B), a bearing, a short and a long bolt
// and a nut. A rotor image holds both bodies, the yellow slab and the black body.
var sevenPartImages = []struct {
	file  string
	parts []string
}{
	{"front-wall", []string{"Front Wall"}}, {"rear-wall", []string{"Rear Wall"}},
	{"detent-plate", []string{"Detent Plate"}}, {"guide-plate", []string{"Guide Plate"}},
	{"cradle-plate", []string{"Cradle Plate"}}, {"top-plate", []string{"Top Plate"}}, {"shaft", []string{"Shaft"}},
	{"ratchet-wheel", []string{"Ratchet Wheel"}}, {"collar-ring", []string{"Collar Ring"}}, {"knob", []string{"Knob"}},
	{"cam-a", []string{"Cam a"}}, {"cam-b", []string{"Cam b"}}, {"cam-c", []string{"Cam c"}}, {"cam-d", []string{"Cam d"}},
	{"cam-e", []string{"Cam e"}}, {"cam-f", []string{"Cam f"}}, {"cam-g", []string{"Cam g"}},
	{"slider-a-g-d", []string{"Slider a"}}, {"slider-b-c", []string{"Slider b"}}, {"slider-e-f", []string{"Slider e"}},
	{"rotor-a", []string{"Flag a Yellow", "Flag a Black"}}, {"rotor-a-yellow", []string{"Flag a Yellow"}},
	{"rotor-a-black", []string{"Flag a Black"}}, {"rotor-b", []string{"Flag b Yellow", "Flag b Black"}},
	{"rotor-b-yellow", []string{"Flag b Yellow"}}, {"rotor-b-black", []string{"Flag b Black"}},
	{"bearing", []string{"Front Bearing"}}, {"bolt-m2x10", []string{"Bolt 1"}},
	{"bolt-m2x6", []string{"Bolt 5"}}, {"bolt-m2x18", []string{"Bolt 9"}},
	{"nut", []string{"Nut 1"}},
}

// renderSevenSegmentReadme builds the display once and writes every README image.
func renderSevenSegmentReadme(ctx context.Context, outputDir string) error {
	s, err := buildSevenSegmentSet(ctx)
	if err != nil {
		return err
	}
	for _, shot := range []struct {
		view, file    string
		width, height int
	}{
		{"hero", "hero.png", 1600, 900}, {"hero", "index.png", 480, 270}, {"top", "top.png", 1600, 900},
		{"oblique45", "oblique45.png", 1600, 900}, {"oblique30", "oblique30.png", 1600, 900},
		{"front", "front.png", 1600, 900}, {"side", "side.png", 1600, 900}, {"cutaway", "cutaway.png", 1600, 900},
		{"detent", "detent.png", 1600, 900}, {"exploded", "exploded.png", 1600, 900},
		{"exploded-camshaft", "exploded-camshaft.png", 1600, 900},
	} {
		scene, err := sevenSegmentView(s, shot.view)
		if err != nil {
			return err
		}
		if err := renderkit.RenderPNG(ctx, filepath.Join(outputDir, shot.file), scene, shot.width, shot.height); err != nil {
			return err
		}
	}
	digits, err := sevenSegmentDigits(s)
	if err != nil {
		return err
	}
	if err := renderkit.RenderPNG(ctx, filepath.Join(outputDir, "digits.png"), digits, 1600, 900); err != nil {
		return err
	}
	cams, err := sevenSegmentCams(s)
	if err != nil {
		return err
	}
	if err := renderkit.RenderPNG(ctx, filepath.Join(outputDir, "cams.png"), cams, 1600, 900); err != nil {
		return err
	}
	for _, img := range sevenPartImages {
		var found []clutchPart
		for _, name := range img.parts {
			for _, part := range s {
				if part.name == name {
					found = append(found, part)
				}
			}
		}
		if len(found) != len(img.parts) {
			return fmt.Errorf("seven-segment part image %s: parts %v not all in the set", img.file, img.parts)
		}
		low, high := meshBounds(found[0].mesh)
		for _, part := range found[1:] {
			l, h := meshBounds(part.mesh)
			low = r3.Vec{X: min(low.X, l.X), Y: min(low.Y, l.Y), Z: min(low.Z, l.Z)}
			high = r3.Vec{X: max(high.X, h.X), Y: max(high.Y, h.Y), Z: max(high.Z, h.Z)}
		}
		center := low.Add(high).Scale(.5)
		span := high.Sub(low).Len()
		camera := solidlens.Camera{
			Position: solidlens.Vec(center.Add(r3.Vec{X: .9 * span, Y: -.9 * span, Z: .8 * span})),
			Target:   solidlens.Vec(center), Up: solidlens.Vec{Z: 1}, FOV: 35}
		var models []solidlens.Model
		for _, part := range found {
			if strings.HasPrefix(part.name, "Cam ") { // from behind, the engraved back face upright
				camera.Position = solidlens.Vec(center.Add(r3.Vec{X: .15 * span, Y: 1.1 * span, Z: .45 * span}))
				m, err := sevenUnposed(part, 0, 0)
				if err != nil {
					return err
				}
				models = append(models, m)
				continue
			}
			models = append(models, clutchModel(part.mesh, part.color))
		}
		scene := renderkit.Scene(renderkit.FitCamera(camera, models, 16.0/9, .1), models)
		if err := renderkit.RenderPNG(ctx, filepath.Join(outputDir, "parts", img.file+".png"), scene, 480, 270); err != nil {
			return err
		}
	}
	return nil
}
