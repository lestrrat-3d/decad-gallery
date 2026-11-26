package render

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
	sevenShaftColor   = solidlens.RGB(.16, .17, .20) // shaft and hub
	sevenHousingColor = solidlens.RGB(.28, .55, .35) // click-indexer housing
	sevenRotorColor   = solidlens.RGB(.55, .38, .72) // click-indexer rotor
	sevenButtonColor  = solidlens.RGB(.86, .38, .12) // button
	sevenHardware     = solidlens.RGB(.55, .57, .60) // bought bearings, bolts and nuts
	sevenPivotColor   = solidlens.RGB(.88, .48, .20) // the 14 pivot blocks
	sevenDigitDefault = sevensegment.DefaultParams().Digit
)

// sevenCut is a clip plane whose cut face has the part's own colour.
func sevenCut(plane solidlens.ClipPlane, c solidlens.Color) *solidlens.ClipPlane {
	plane.Cap = &solidlens.Material{Color: c, Unlit: true}
	return &plane
}

// sevenModels returns the parts moved by pull, cut by clip when it is not nil; a part wholly on the removed side is
// left out. Each part keeps its one colour, and its cut face has that colour too.
func sevenModels(s partSet, pull map[string]r3.Vec, clip *solidlens.ClipPlane) ([]solidlens.Model, error) {
	var out []solidlens.Model
	for _, part := range s {
		if clip != nil && renderkit.ClipRemovesAll(part.mesh, *clip) {
			continue
		}
		move, err := r3.Translation(pull[part.name])
		if err != nil {
			return nil, err
		}
		m := renderkit.PartModel(renderkit.TransformedMesh{TriangleSource: part.mesh, Transform: move}, part.color)
		if clip != nil {
			m.Clip = sevenCut(*clip, part.color)
		}
		out = append(out, m)
	}
	return out, nil
}

// sevenClickerY is the middle of the click indexer along Y. It keeps its size, but stands where the front wall's outer face
// puts it: -94.75, which is -66 less the 28.75 that the face has moved.
const sevenClickerY = -94.75

// sevenCenter is the middle of the model, where the cameras look.
var sevenCenter = solidlens.Vec{Y: -4.5, Z: 65.25}

// sevenExplode is each part's offset in the exploded view, from "Rendering" in the design: the plates up in Z, the
// walls, detent plate, bearings, collar ring and knob out along Y. Bolts follow the part their head sits in, 15 mm
// more along -Dir, and nuts follow the part their pocket is in, 15 mm more along +Dir.
var sevenExplode = map[string]r3.Vec{
	"Guide Plate": {Z: 15.0}, "Top Plate": {Z: 75.0}, "Cradle Plate": {Z: 22.5},
	// The sliders stand beside the stack, each pair stepped up in Z, so that none hides another or sits behind a plate: a, g
	// and d farthest out on +X, b and c on +X, e and f on -X, f higher (checked by eye on the rendered image).
	"Slider a": {X: 157.5}, "Slider g": {X: 157.5, Z: 67.5}, "Slider d": {X: 157.5, Z: 135.0},
	"Slider b": {X: 90.0}, "Slider c": {X: 90.0, Z: 67.5},
	"Slider e": {X: -127.5, Z: 45.0}, "Slider f": {X: -127.5, Z: 150.0}, "Front Wall": {Y: -75.0},
	// Each groove pin comes out of its slider's keyed hole along +Y, 30 mm past the stem's face.
	"Groove Pin a": {X: 157.5, Y: 30.0}, "Groove Pin g": {X: 157.5, Y: 30.0, Z: 67.5}, "Groove Pin d": {X: 157.5, Y: 30.0, Z: 135.0},
	// The click indexer comes off the front end in order, each part past the one before: hub, housing, spring, rotor, button.
	"Hub": {Y: -93.0}, "Housing": {Y: -120.0}, "Spring": {Y: -150.0}, "Rotor": {Y: -183.0},
	"Button": {Y: -240.0},
	// The front bearing drops below the shaft, so the front wall does not hide it.
	"Front Bearing": {Y: -60.0, Z: -82.5}, "Rear Wall": {Y: 67.5}, "Rear Bearing": {Y: 37.5},
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

// sevenBlockName is the part name of segment seg's pivot block at its -u (end 0) or +u (end 1) end.
func sevenBlockName(seg sevensegment.Segment, end int) string {
	return "Pivot Block " + seg.String() + [2]string{" -u", " +u"}[end]
}

// sevenSegmentAxis reports whether segment s's rotor axis runs along X (a, g, d).
func sevenSegmentAxis(s sevensegment.Segment) (x, y float64, horizontal bool) {
	switch s {
	case sevensegment.SegA, sevensegment.SegG, sevensegment.SegD:
		return 1, 0, true
	}
	return 0, 1, false
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
var sevenCamshaftOrder = []string{"Cam a", "Cam f", "Cam b", "Cam g", "Cam c", "Cam e", "Cam d", "Hub", "Spring", "Rotor"}

// sevenSet builds the default display once and tessellates its 78 bodies.
func buildSevenSegmentSet(ctx context.Context) (partSet, error) {
	return buildSevenSegmentSetOpt(ctx, false)
}

// buildSevenSegmentSetOpt builds the set; preview leaves the click indexer's teeth as separate wedge parts.
func buildSevenSegmentSetOpt(ctx context.Context, _ bool) (partSet, error) {
	doc := sevensegment.NewDocument()
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
		{"Housing", m.Housing, sevenHousingColor}, {"Guide Plate", m.GuidePlate, sevenPlateColor},
		{"Top Plate", m.TopPlate, sevenPlateColor}, {"Cradle Plate", m.CradlePlate, sevenWallColor}, {"Shaft", m.Shaft, sevenShaftColor},
		{"Hub", m.Hub, sevenShaftColor}, {"Rotor", m.Rotor, sevenRotorColor}, {"Button", m.Button, sevenButtonColor},
		{"Spring", m.Spring, sevenHardware}, {"Front Bearing", m.Bearings[0].Body, sevenHardware},
		{"Rear Bearing", m.Bearings[1].Body, sevenHardware},
	}
	for _, s := range sevensegment.Segments() {
		if m.GroovePins[s] != nil {
			parts = append(parts, named{"Groove Pin " + s.String(), m.GroovePins[s], sevenBlack})
		}
		parts = append(parts, named{"Cam " + s.String(), m.Cams[s], sevenCamColor},
			named{"Slider " + s.String(), m.Sliders[s], sevenBlack},
			named{sevenFlagName(s, "Yellow"), m.Flags[s].Yellow, sevenYellow},
			named{sevenFlagName(s, "Black"), m.Flags[s].Black, sevenBlack})
		// The rotors rise 30 and spread outward from the middle of the face, so that none hides another.
		cx, cy := sevenSegmentCentre(s)
		_, _, horizontal := sevenSegmentAxis(s)
		for end, sgn := range []float64{-1, 1} {
			name := sevenBlockName(s, end)
			parts = append(parts, named{name, m.PivotBlocks[s][end], sevenPivotColor})
			// Each block stays with its rotor and slides 14 along the axis, out past the rotor's end.
			out := r3.Vec{X: .9 * cx, Y: .9 * cy, Z: 45.0}
			if horizontal {
				out.X += 21.0 * sgn
			} else {
				out.Y += 21.0 * sgn
			}
			sevenExplode[name] = out
		}
		sevenExplode[sevenFlagName(s, "Yellow")] = r3.Vec{X: .9 * cx, Y: .9 * cy, Z: 45.0}
		sevenExplode[sevenFlagName(s, "Black")] = r3.Vec{X: .9 * cx, Y: .9 * cy, Z: 45.0}
	}
	for i := range m.Bolts {
		parts = append(parts, named{fmt.Sprintf("Bolt %d", i+1), m.Bolts[i].Body, sevenHardware},
			named{fmt.Sprintf("Nut %d", i+1), m.Nuts[i].Body, sevenHardware})
	}
	set := make(partSet, 0, len(parts))
	for _, p := range parts {
		tol := .05 // decad holds the sliders no finer than 0.0401
		if p.name == "Rotor" || p.name == "Button" || p.name == "Housing" {
			tol = .1 // the helicoid teeth mesh no finer
		}
		if strings.HasPrefix(p.name, "Bolt") || strings.HasPrefix(p.name, "Nut") {
			tol = .01
		}
		if strings.HasPrefix(p.name, "Groove Pin") {
			tol = .09 // the pin's faceted wings hold a mesh bound of 0.086
		}
		mesh, err := p.body.Tessellate(ctx, units.Millimeters(tol), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return nil, fmt.Errorf("mesh %s: %w", p.name, err)
		}
		set = append(set, part{p.name, mesh, p.color})
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
			headIn, nutIn = "Housing", "Front Wall"
		}
		dir := r3.Vec(site.Dir)
		sevenExplode[fmt.Sprintf("Bolt %d", i+1)] = sevenExplode[headIn].Sub(dir.Scale(15))
		sevenExplode[fmt.Sprintf("Nut %d", i+1)] = sevenExplode[nutIn].Add(dir.Scale(15))
	}
	return set, nil
}

var sevenCameras = map[string]solidlens.Camera{
	"hero": {Position: solidlens.Vec{X: 165.0, Y: -187.5, Z: 352.5}, Target: sevenCenter, Up: solidlens.Vec{Y: -1}, FOV: 35},
	"top":  {Position: solidlens.Vec{Y: -4.5, Z: 480.0}, Target: solidlens.Vec{Y: -4.5, Z: 126.0}, Up: solidlens.Vec{Y: -1}, FOV: 30},
	"oblique45": {Position: solidlens.Vec{Y: -244.5, Z: 366.0}, Target: solidlens.Vec{Y: -4.5, Z: 126.0},
		Up: solidlens.Vec{Y: -1}, FOV: 30},
	"oblique30": {Position: solidlens.Vec{Y: -289.5, Z: 291.0}, Target: solidlens.Vec{Y: -4.5, Z: 126.0},
		Up: solidlens.Vec{Y: -1}, FOV: 30},
	"front": {Position: solidlens.Vec{Y: -450.0, Z: 90.0}, Target: sevenCenter, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"side":  {Position: solidlens.Vec{X: 450.0, Y: -30.0, Z: 90.0}, Target: sevenCenter, Up: solidlens.Vec{Y: -1}, FOV: 35},
	"cutaway": {Position: solidlens.Vec{X: 450.0, Y: -30.0, Z: 90.0}, Target: sevenCenter, Up: solidlens.Vec{Y: -1},
		FOV: 35},
	"indexer": {Position: solidlens.Vec{X: 260, Y: sevenClickerY, Z: sevensegment.ShaftZ},
		Target: solidlens.Vec{Y: sevenClickerY, Z: sevensegment.ShaftZ}, Up: solidlens.Vec{Y: -1}, FOV: 30},
	"exploded": {Position: solidlens.Vec{X: 495.0, Y: -495.0, Z: 345.0}, Target: solidlens.Vec{Y: -4.5, Z: 93.0},
		Up: solidlens.Vec{Z: 1}, FOV: 35},
	"exploded-camshaft": {Position: solidlens.Vec{X: 495.0, Y: -495.0, Z: 255.0},
		Target: solidlens.Vec{Y: -105.0, Z: sevensegment.ShaftZ}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	// Segment a's rotor axis runs along X at y = SegmentSpan. The exploded view looks at it from the front, a little to the
	// right and level with the axis, so the plate halves pulled above and below show their seam faces; the section
	// looks into the cut through the axis from -Y, at the +X end.
	"exploded-pivot": {Position: solidlens.Vec{X: 165.0, Y: -225.0, Z: sevensegment.RotorAxisZ + 6.0},
		Target: solidlens.Vec{Y: sevensegment.SegmentSpan, Z: sevensegment.RotorAxisZ}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"pivot-section": {Position: solidlens.Vec{X: 18.9, Y: -150.0, Z: sevensegment.RotorAxisZ + 1.05},
		Target: solidlens.Vec{X: 18.9, Y: sevensegment.SegmentSpan, Z: sevensegment.RotorAxisZ + 1.05}, Up: solidlens.Vec{Z: 1}, FOV: 6.0},
}

var (
	sevenAxisCut = solidlens.ClipPlane{Normal: solidlens.Vec{X: 1}}
)

// sevenCamshaftPull spreads the parts on the shaft along -Y, 12 mm per part from the rear.
func sevenCamshaftPull() map[string]r3.Vec {
	pull := map[string]r3.Vec{}
	for i, name := range sevenCamshaftOrder {
		pull[name] = r3.Vec{Y: -18.0 * float64(i+1)}
	}
	pull["Shaft"] = r3.Vec{Y: 150.0} // the shaft slides out of the rear, so every cam's bore shows
	return pull
}

// view draws one named view of the display.
func sevenSegmentView(s partSet, name string) (solidlens.Scene, error) {
	if name == "exploded-pivot" || name == "pivot-section" {
		return sevenPivotView(s, name)
	}
	camera, ok := sevenCameras[name]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown seven-segment view %q", name)
	}
	var clip *solidlens.ClipPlane
	var pull map[string]r3.Vec
	switch name {
	case "cutaway":
		clip = &sevenAxisCut
	case "indexer":
		clip = &sevenAxisCut
	case "exploded":
		pull = sevenExplode
	case "exploded-camshaft":
		pull = sevenCamshaftPull()
	}
	set := s
	if name == "exploded-camshaft" { // the frame, bearings and fasteners are left out so the shaft and every cam show
		set = nil
		for _, p := range s {
			if p.name == "Shaft" || p.name == "Hub" || p.name == "Rotor" || p.name == "Spring" || strings.HasPrefix(p.name, "Cam ") {
				set = append(set, p)
			}
		}
	}
	if name == "indexer" { // the indexer and the front wall
		set = nil
		for _, p := range s {
			switch p.name {
			case "Hub", "Rotor", "Housing", "Button", "Spring", "Front Wall":
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

// sevenPivotExplode is each part's offset in the pivot exploded view of segment a: the cradle plate down, the top plate
// up, and each block 3 mm out along the rotor axis, which frees its pin from the bore and leaves the block level with its own
// pocket.
var sevenPivotExplode = map[string]r3.Vec{
	"Cradle Plate": {Z: -12.0}, "Top Plate": {Z: 21.0},
	"Pivot Block a -u": {X: -4.5}, "Pivot Block a +u": {X: 4.5},
}

// sevenPivotParts lists the parts of the pivot views, from segment a: its rotor, blocks and the plate halves.
var sevenPivotParts = []string{"Flag a Yellow", "Flag a Black", "Pivot Block a -u", "Pivot Block a +u",
	"Cradle Plate", "Top Plate"}

// sevenShrunk draws a mesh scaled by factor about the centre of its bounds.
type sevenShrunk struct {
	solidlens.TriangleSource
	centre r3.Vec
	factor float64
}

func (m sevenShrunk) Vertices() []r3.Vec {
	source := m.TriangleSource.Vertices()
	out := make([]r3.Vec, len(source))
	for i, v := range source {
		out[i] = m.centre.Add(v.Sub(m.centre).Scale(m.factor))
	}
	return out
}

// sevenPivotView draws segment a's rotor end and its pivot blocks with the two plate halves that hold them.
//
// The plates are cut to a block 42 mm long, 18 mm deep and about 10 mm high round the rotor so the pockets show large. Exploded pulls the parts
// apart; pivot-section instead cuts every part by the plane through the rotor axis, the -Y half removed, and looks into
// the cut. Every cut is made on the mesh (solidlens.ClipMesh) and the inside is drawn in the part's own color.
//
// The yellow body of a rotor shares its faces with the black body. Where the camera looks through the cut, the yellow
// body's back face and the black body's front face lie in the same place and fight for the pixel (speckle). The section
// scales the yellow body down by 0.5 % about its centre, which leaves the shared face 0.03 mm clear on the yellow side,
// so the cut faces of both bodies win cleanly.
func sevenPivotView(s partSet, name string) (solidlens.Scene, error) {
	axisY := sevensegment.SegmentSpan
	exploded := name == "exploded-pivot"
	var models []solidlens.Model
	for _, p := range s {
		found := false
		for _, want := range sevenPivotParts {
			found = found || p.name == want
		}
		if !found {
			continue
		}
		var src solidlens.TriangleSource = p.mesh
		var planes []solidlens.ClipPlane
		if exploded {
			move, err := r3.Translation(sevenPivotExplode[p.name])
			if err != nil {
				return solidlens.Scene{}, err
			}
			src = renderkit.TransformedMesh{TriangleSource: src, Transform: move}
			if p.name == "Cradle Plate" || p.name == "Top Plate" {
				// Each plate keeps 4.5 mm of its thickness beyond the pocket's floor or roof.
				z, normal := sevensegment.RotorAxisZ+sevenPivotExplode[p.name].Z-6.75, -1.0
				if p.name == "Top Plate" {
					z, normal = sevensegment.RotorAxisZ+sevenPivotExplode[p.name].Z+8.25, 1
				}
				planes = []solidlens.ClipPlane{
					{Origin: solidlens.Vec{Y: axisY - 13.5}, Normal: solidlens.Vec{Y: -1}},
					{Origin: solidlens.Vec{Y: axisY + 13.5}, Normal: solidlens.Vec{Y: 1}},
					{Origin: solidlens.Vec{X: -31.5}, Normal: solidlens.Vec{X: -1}},
					{Origin: solidlens.Vec{X: 31.5}, Normal: solidlens.Vec{X: 1}},
					{Origin: solidlens.Vec{Z: z}, Normal: solidlens.Vec{Z: normal}},
				}
			}
		} else {
			if p.name == "Flag a Yellow" {
				low, high := renderkit.MeshBounds(p.mesh)
				src = sevenShrunk{TriangleSource: src, centre: low.Add(high).Scale(.5), factor: .995}
			}
			planes = []solidlens.ClipPlane{{Origin: solidlens.Vec{Y: axisY}, Normal: solidlens.Vec{Y: -1}}}
		}
		for _, plane := range planes {
			clipped, err := solidlens.ClipMesh(src, plane)
			if err != nil {
				return solidlens.Scene{}, fmt.Errorf("%s: %w", p.name, err)
			}
			src = clipped
		}
		m := renderkit.PartModel(src, p.color)
		m.BackMaterial = &solidlens.Material{Color: p.color, Unlit: true}
		models = append(models, m)
	}
	camera := sevenCameras[name]
	if !exploded {
		return renderkit.Scene(camera, models), nil
	}
	return renderkit.Scene(renderkit.FitCamera(camera, models, 16.0/9, .03), models), nil
}

// sevenFaceBolt reports whether name is one of the four top-plate bolts, whose heads lie flush in the face.
func sevenFaceBolt(name string) bool {
	return name == "Bolt 1" || name == "Bolt 2" || name == "Bolt 3" || name == "Bolt 4"
}

// sevenDigitModels returns the face of the display showing digit k, placed at at: the top plate, its four bolts and the rotors, each
// rotor turned from its built angle to the one the digit asks of it. Every part keeps its one colour.
func sevenDigitModels(s partSet, dm sevensegment.Dimensions, k int, at r3.Vec) ([]solidlens.Model, error) {
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
			src = renderkit.TransformedMesh{TriangleSource: src, Transform: turn}
		}
		models = append(models, renderkit.PartModel(renderkit.TransformedMesh{TriangleSource: src, Transform: place}, p.color))
	}
	return models, nil
}

// sevenSegmentDigits draws digits 0 to 9 in a 5 by 2 grid, seen from the front at an angle so the turned rotors show
// their yellow faces.
func sevenSegmentDigits(s partSet) (solidlens.Scene, error) {
	dm, err := sevensegment.DefaultParams().Derive()
	if err != nil {
		return solidlens.Scene{}, err
	}
	var models []solidlens.Model
	for k := range 10 {
		m, err := sevenDigitModels(s, dm, k, r3.Vec{X: float64(k%5) * 99.0, Y: float64(k/5) * 177.0})
		if err != nil {
			return solidlens.Scene{}, err
		}
		models = append(models, m...)
	}
	camera := solidlens.Camera{Position: solidlens.Vec{X: 198.0, Y: -690.0, Z: 495.0},
		Target: solidlens.Vec{X: 198.0, Y: 84.0, Z: 126.0},
		Up:     solidlens.Vec{Y: -1}, FOV: 30}
	return renderkit.Scene(renderkit.FitCamera(camera, models, 16.0/9, .04), models), nil
}

// sevenUnposed returns the model of a camshaft part turned back from the digit's pose, so its letters stand upright.
func sevenUnposed(p part, dx, dz float64) (solidlens.Model, error) {
	turn, err := r3.RotationAround(r3.Vec{Z: sevensegment.ShaftZ}, r3.Vec{Y: 1},
		units.Degrees(-36*float64(sevenDigitDefault)))
	if err != nil {
		return solidlens.Model{}, err
	}
	shift, err := r3.Translation(r3.Vec{X: dx, Z: dz})
	if err != nil {
		return solidlens.Model{}, err
	}
	return renderkit.PartModel(renderkit.TransformedMesh{TriangleSource: renderkit.TransformedMesh{TriangleSource: p.mesh, Transform: turn}, Transform: shift},
		sevenCamColor), nil
}

// sevenSegmentCams draws the seven cams side by side, a to d on the top row and e to g below, seen from behind so each cam's engraved
// letter shows on its back face.
func sevenSegmentCams(s partSet) (solidlens.Scene, error) {
	var models []solidlens.Model
	for k, seg := range sevensegment.Segments() {
		for _, p := range s {
			if p.name != "Cam "+seg.String() {
				continue
			}
			m, err := sevenUnposed(p, -(float64(k%4)-1.5)*93.0, -float64(k/4)*93.0)
			if err != nil {
				return solidlens.Scene{}, err
			}
			models = append(models, m)
		}
	}
	camera := solidlens.Camera{Position: solidlens.Vec{Y: 750.0, Z: -2.25}, Target: solidlens.Vec{Y: 30.0, Z: -2.25},
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

// sevenPartImages lists the distinct bodies and their images: the printed parts (sliders a, g and d share one image, and so do their groove pins,
// b and c another, e and f a third; rotors a, g and d are type A, b, c, e and f type B), a bearing, a short and a long bolt
// and a nut. A rotor image holds both bodies, the yellow slab and the black body.
var sevenPartImages = []struct {
	file  string
	parts []string
}{
	{"front-wall", []string{"Front Wall"}}, {"rear-wall", []string{"Rear Wall"}},
	{"guide-plate", []string{"Guide Plate"}},
	{"cradle-plate", []string{"Cradle Plate"}}, {"top-plate", []string{"Top Plate"}}, {"shaft", []string{"Shaft"}},
	{"hub", []string{"Hub"}}, {"rotor", []string{"Rotor"}}, {"housing", []string{"Housing"}}, {"button", []string{"Button"}},
	{"spring", []string{"Spring"}},
	{"cam-a", []string{"Cam a"}}, {"cam-b", []string{"Cam b"}}, {"cam-c", []string{"Cam c"}}, {"cam-d", []string{"Cam d"}},
	{"cam-e", []string{"Cam e"}}, {"cam-f", []string{"Cam f"}}, {"cam-g", []string{"Cam g"}},
	{"slider-a-g-d", []string{"Slider a"}}, {"groove-pin", []string{"Groove Pin g"}}, {"slider-b-c", []string{"Slider b"}}, {"slider-e-f", []string{"Slider e"}},
	{"rotor-a", []string{"Flag a Yellow", "Flag a Black"}}, {"rotor-a-yellow", []string{"Flag a Yellow"}},
	{"rotor-a-black", []string{"Flag a Black"}}, {"rotor-b", []string{"Flag b Yellow", "Flag b Black"}},
	{"rotor-b-yellow", []string{"Flag b Yellow"}}, {"rotor-b-black", []string{"Flag b Black"}},
	{"bearing", []string{"Front Bearing"}}, {"bolt-m2x16", []string{"Bolt 1"}},
	{"bolt-m2x10", []string{"Bolt 5"}},
	{"nut", []string{"Nut 1"}},
	{"pivot-block", []string{"Pivot Block g -u"}},
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
		{"indexer", "indexer.png", 1600, 900}, {"exploded", "exploded.png", 1600, 900},
		{"exploded-camshaft", "exploded-camshaft.png", 1600, 900},
		{"exploded-pivot", "exploded-pivot.png", 1600, 900}, {"pivot-section", "pivot-section.png", 1600, 900},
	} {
		scene, err := sevenSegmentView(s, shot.view)
		if err != nil {
			return err
		}
		if err := renderkit.RenderPNG(ctx, filepath.Join(outputDir, shot.file), scene, shot.width, shot.height); err != nil {
			return err
		}
	}
	if err := renderSevenSegmentEngage(ctx, s, outputDir); err != nil {
		return err
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
		var found []part
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
		low, high := renderkit.MeshBounds(found[0].mesh)
		for _, part := range found[1:] {
			l, h := renderkit.MeshBounds(part.mesh)
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
			models = append(models, renderkit.PartModel(part.mesh, part.color))
		}
		scene := renderkit.Scene(renderkit.FitCamera(camera, models, 16.0/9, .1), models)
		if err := renderkit.RenderPNG(ctx, filepath.Join(outputDir, "parts", img.file+".png"), scene, 480, 270); err != nil {
			return err
		}
	}
	return nil
}
