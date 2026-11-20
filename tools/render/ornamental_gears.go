package main

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

type wheelOrnament int

const (
	rosette wheelOrnament = iota
	compass
	vanes
	honeycomb
	perforated
	diamonds
)

// Dimensions of the ornament and its backing are fractions of the pitch radius.
type ornamentalWheel struct {
	teeth             int
	ornament          wheelOrnament
	center            r3.Vec
	base, plate, trim solidlens.Color
}

func ornamentalGears(ctx context.Context) (solidlens.Scene, error) {
	brass := solidlens.RGB(.54, .353, .142)
	bronze := solidlens.RGB(.151, .081, .038)
	burnished := solidlens.RGB(.76, .551, .273)
	wheels := []ornamentalWheel{
		{60, rosette, r3.NewVec(-96, 49, 0), bronze, brass, burnished},
		{36, compass, r3.NewVec(110, -67, 0), brass, burnished, bronze},
		{22, vanes, r3.NewVec(-80, -58, 0), brass, bronze, burnished},
		{17, honeycomb, r3.NewVec(98.5, 37.5, 0), bronze, brass, burnished},
	}
	scene := solidlens.Scene{
		Camera: solidlens.Camera{Position: r3.NewVec(7, -9, 255), Target: r3.NewVec(7, -9, 2),
			Up: r3.NewVec(0, 1, 0), FOV: 26},
		Background: solidlens.RGB(.95, .93, .87),
		DirectionalLights: []solidlens.DirectionalLight{
			{Direction: r3.NewVec(.15, .65, -.88), Color: solidlens.RGB(1, .92, .8), Intensity: .87},
			{Direction: r3.NewVec(-.9, -.2, -.5), Color: solidlens.RGB(.83, .87, .92), Intensity: .51},
		},
	}
	var templates [][]solidlens.Model
	for _, wheel := range wheels {
		models, err := wheel.models(ctx)
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("%d-tooth ornamental spur gear: %w", wheel.teeth, err)
		}
		scene.Models = append(scene.Models, models...)
		templates = append(templates, models)
	}
	extra := ornamentalWheel{17, perforated, r3.NewVec(0, 0, 0), bronze, brass, burnished}
	extraModels, err := extra.models(ctx)
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("perforated border gear: %w", err)
	}
	wheels = append(wheels, extra)
	templates = append(templates, extraModels)
	extra = ornamentalWheel{17, diamonds, r3.NewVec(0, 0, 0), brass, burnished, bronze}
	extraModels, err = extra.models(ctx)
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("diamond border gear: %w", err)
	}
	wheels = append(wheels, extra)
	templates = append(templates, extraModels)
	border, err := borderGears(wheels, templates)
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("border gears: %w", err)
	}
	scene.Models = append(scene.Models, border...)
	return scene, nil
}

// borderGears reuses the verified ornament meshes at varied sizes and positions.
func borderGears(wheels []ornamentalWheel, templates [][]solidlens.Model) ([]solidlens.Model, error) {
	type borderGear struct {
		x, y, radius float64
		style        int
	}
	gears := []borderGear{
		{-25, 48, 5.0, 4}, {-12, 40, 3.4, 2}, {0, 49, 7.0, 5}, {14, 42, 3.0, 3},
		{25, 50, 5.2, 0}, {37, 43, 4.0, 4}, {50, 50, 6.4, 2}, {64, 40, 5.0, 1},
		{76, 49, 3.2, 3}, {-19, 35, 3.0, 1}, {8, 33, 3.0, 4}, {33, 33, 3.2, 5},
		{56, 33, 2.8, 0}, {-47, -66, 6.0, 1}, {-33, -59, 3.4, 3}, {-20, -68, 6.8, 4},
		{-5, -60, 3.0, 5}, {7, -67, 5.8, 0}, {20, -60, 4.2, 1}, {34, -68, 7.1, 4},
		{49, -61, 3.1, 3}, {61, -67, 5.2, 2}, {-40, -52, 2.6, 5}, {-12, -52, 2.8, 2},
		{14, -51, 2.7, 4}, {42, -51, 2.8, 0}, {109, 10, 6.4, 5}, {102, -3, 3.0, 4},
		{111, -15, 5.4, 1}, {101, -22, 3.0, 3}, {-95, -23, 6.2, 2},
	}
	var models []solidlens.Model
	for i, gear := range gears {
		turn, err := r3.Rotation(r3.NewVec(0, 0, 1), units.Radians(float64(i)*.71))
		if err != nil {
			return nil, fmt.Errorf("rotate border gear %d: %w", i, err)
		}
		pos := r3.NewVec(gear.x, gear.y, 0)
		scale := gear.radius / float64(wheels[gear.style].teeth)
		for _, part := range templates[gear.style] {
			mesh, ok := part.Mesh.(placedMesh)
			if !ok {
				return nil, fmt.Errorf("border template is not a placed mesh")
			}
			part.Mesh = borderMesh{TriangleSource: mesh.TriangleSource, scale: scale, turn: turn,
				origin: mesh.offset.Sub(wheels[gear.style].center), offset: pos}
			models = append(models, part)
		}
	}
	return models, nil
}

// borderMesh rotates and places a reused ornamental mesh.
type borderMesh struct {
	solidlens.TriangleSource
	scale          float64
	turn           r3.Transform
	origin, offset r3.Vec
}

func (m borderMesh) Vertices() []r3.Vec {
	source := m.TriangleSource.Vertices()
	vertices := make([]r3.Vec, len(source))
	for i, v := range source {
		vertices[i] = m.turn.Apply(v.Add(m.origin)).Scale(m.scale).Add(m.offset)
	}
	return vertices
}

func (w ornamentalWheel) models(ctx context.Context) ([]solidlens.Model, error) {
	doc := decad.New()
	world := sketch.NewWorld()
	p := spurgear.DefaultParams()
	p.Module, p.Teeth = 2, w.teeth
	r := p.Module * float64(p.Teeth) / 2
	p.BoreDiameter, p.Thickness = .18*r, .135*r
	if w.ornament != rosette {
		// The pierced plate spans the opening and joins the toothed rim.
		p.BoreDiameter = 1.424 * r
	}
	gear, err := spurgear.Build(ctx, doc, p)
	if err != nil {
		return nil, err
	}
	if w.ornament == rosette {
		gear, err = drillRosette(ctx, doc, world, gear, r, p.Thickness)
		if err != nil {
			return nil, err
		}
	}

	faceThickness := .028 * r
	plate, err := w.faceplate(ctx, doc, world, r, p.Thickness, faceThickness)
	if err != nil {
		return nil, fmt.Errorf("pierced faceplate: %w", err)
	}
	hub, err := turnedOrnament(ctx, doc, world, r, p.Thickness+faceThickness, [][2]float64{
		{.09, 0}, {.222, 0}, {.222, .011}, {.211, .022}, {.165, .022},
		{.148, .044}, {.105, .044}, {.09, .029},
	})
	if err != nil {
		return nil, fmt.Errorf("stepped hub: %w", err)
	}
	rim, err := turnedOrnament(ctx, doc, world, r, p.Thickness, [][2]float64{
		{.817, 0}, {.861, 0}, {.852, .013}, {.842, .013},
		{.837, .006}, {.831, .006}, {.826, .013},
	})
	if err != nil {
		return nil, fmt.Errorf("turned rim: %w", err)
	}
	var models []solidlens.Model
	parts := []struct {
		name  string
		body  *decad.Body
		color solidlens.Color
	}{{"gear", gear, w.base}, {"faceplate", plate, w.plate}, {"rim", rim, w.trim}, {"hub", hub, w.trim}}
	for _, part := range parts {
		mesh, err := part.body.Tessellate(ctx, units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return nil, fmt.Errorf("tessellate %s: %w", part.name, err)
		}
		models = append(models, solidlens.Model{Mesh: placedMesh{TriangleSource: mesh, scale: 1, offset: w.center},
			Material: solidlens.Matte(part.color),
			Edges:    solidlens.Edges{Enabled: true, Color: solidlens.RGB(.16, .09, .045), Width: 1.4}})
	}
	if w.ornament == rosette {
		eyelet, err := turnedOrnament(ctx, doc, world, r, p.Thickness, [][2]float64{
			{.046, 0}, {.063, 0}, {.062, .008}, {.058, .012}, {.051, .012}, {.046, .007},
		})
		if err != nil {
			return nil, fmt.Errorf("eyelet: %w", err)
		}
		mesh, err := eyelet.Tessellate(ctx, units.Millimeters(.04), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return nil, fmt.Errorf("tessellate eyelet: %w", err)
		}
		for i := range rosettePetals {
			x, y := polarPoint(.737*r, (float64(i)+.5)*2*math.Pi/rosettePetals)
			models = append(models, solidlens.Model{Mesh: placedMesh{TriangleSource: mesh, scale: 1, offset: w.center.Add(r3.NewVec(x, y, 0))},
				Material: solidlens.Matte(w.trim),
				Edges:    solidlens.Edges{Enabled: true, Color: solidlens.RGB(.16, .09, .045), Width: 1.4}})
		}
	}
	return models, nil
}

func polarPoint(radius, angle float64) (float64, float64) {
	sin, cos := math.Sincos(angle)
	return radius * cos, radius * sin
}

// Given both dimensioned ends, Sketch's arc already constrains the center
// to their perpendicular bisector. Dimension one coordinate along that line.
func ornamentArc(s *sketch.Sketch, start, end *sketch.Point, cx, cy, chordX, chordY float64) {
	center := s.CreatePoint(cx, cy)
	if math.Abs(chordX) >= math.Abs(chordY) {
		s.AddConstraint(sketch.NewVerticalDistance(s.Origin(), center, cy))
	} else {
		s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), center, cx))
	}
	s.CreateArc(center, start, end)
}

func (w ornamentalWheel) faceplate(ctx context.Context, doc *decad.Document, world *sketch.World,
	r, z, thickness float64) (*decad.Body, error) {
	plane, err := world.CreateOffsetPlane(world.XY(), z)
	if err != nil {
		return nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, err
	}
	var openings int
	switch w.ornament {
	case rosette:
		openings = drawRosette(s, r)
	case compass:
		openings = drawCompass(s, r)
	case vanes:
		openings = drawVanes(s, r)
	case honeycomb:
		openings = drawHoneycomb(s, r)
	case perforated:
		openings = drawPerforated(s, r)
	case diamonds:
		openings = drawDiamonds(s, r)
	default:
		return nil, fmt.Errorf("unknown wheel ornament %d", w.ornament)
	}
	bore := s.CreateCircle(s.Origin(), .09*r)
	s.AddConstraint(sketch.NewRadius(bore, .09*r))
	profiles, err := ornamentalProfiles(ctx, s)
	if err != nil {
		return nil, err
	}
	for _, profile := range profiles {
		if len(profile.Holes) == openings+1 {
			return decad.Extrude(ctx, doc, s, profile, decad.Distance{D: units.Millimeters(thickness), Dir: decad.Along})
		}
	}
	return nil, fmt.Errorf("no profile contains the bore and %d windows", openings)
}

const rosettePetals = 7

func drawRosette(s *sketch.Sketch, r float64) int {
	const step = 2 * math.Pi / rosettePetals
	valley, crest := .53*r, .775*r
	points := make([]*sketch.Point, rosettePetals)
	for i := range points {
		x, y := polarPoint(valley, (float64(i)-.5)*step)
		points[i] = sketchcheck.DimensionPoint(s, x, y)
	}
	// One circular lobe joins each pair of valleys and reaches crest on the bisector.
	lobeCenter := (crest*crest - valley*valley) / (2 * (crest - valley*math.Cos(step/2)))
	for i, from := range points {
		a := float64(i) * step
		cx, cy := polarPoint(lobeCenter, a)
		dx, dy := polarPoint(1, a+math.Pi/2)
		ornamentArc(s, from, points[(i+1)%len(points)], cx, cy, dx, dy)

		x, y := polarPoint(.53*r, a)
		dx, dy = polarPoint(.199*r, a+.48)
		almondWindow(s, x, y, dx, dy, .72)

		x, y = polarPoint(.354*r, a+step/2)
		dx, dy = polarPoint(.105*r, a+step/2-.48)
		almondWindow(s, x, y, dx, dy, 1.32)
	}
	return 2 * rosettePetals
}

func drillRosette(ctx context.Context, doc *decad.Document, world *sketch.World, gear *decad.Body, r, height float64) (*decad.Body, error) {
	base, err := world.CreateSketch(world.XY())
	if err != nil {
		return nil, err
	}
	const step = 2 * math.Pi / rosettePetals
	for i := range rosettePetals {
		a := float64(i) * step
		for _, drill := range []struct{ radial, phi, radius float64 }{
			{.53 * r, a, .207 * r}, {.737 * r, a + step/2, .046 * r},
		} {
			x, y := polarPoint(drill.radial, drill.phi)
			center := sketchcheck.DimensionPoint(base, x, y)
			base.AddConstraint(sketch.NewRadius(base.CreateCircle(center, drill.radius), drill.radius))
		}
	}
	profiles, err := ornamentalProfiles(ctx, base)
	if err != nil {
		return nil, fmt.Errorf("drill sketch: %w", err)
	}
	if len(profiles) != 2*rosettePetals {
		return nil, fmt.Errorf("drill sketch has %d profiles, expected %d", len(profiles), 2*rosettePetals)
	}
	for i, profile := range profiles {
		tool, err := decad.Extrude(ctx, doc, base, profile, decad.Distance{D: units.Millimeters(height), Dir: decad.Along})
		if err != nil {
			return nil, fmt.Errorf("drill %d: %w", i, err)
		}
		gear, err = decad.Cut(ctx, gear, tool)
		if err != nil {
			return nil, fmt.Errorf("cut opening %d: %w", i, err)
		}
	}
	return gear, nil
}

func almondWindow(s *sketch.Sketch, x, y, dx, dy, offset float64) {
	first := sketchcheck.DimensionPoint(s, x-dx, y-dy)
	last := sketchcheck.DimensionPoint(s, x+dx, y+dy)
	ornamentArc(s, first, last, x-offset*dy, y+offset*dx, dx, dy)
	ornamentArc(s, last, first, x+offset*dy, y-offset*dx, dx, dy)
}

func turnedOrnament(ctx context.Context, doc *decad.Document, world *sketch.World,
	r, z float64, meridian [][2]float64) (*decad.Body, error) {
	s, err := world.CreateSketch(world.XZ())
	if err != nil {
		return nil, err
	}
	points := make([]*sketch.Point, len(meridian))
	for i, p := range meridian {
		points[i] = sketchcheck.DimensionPoint(s, p[0]*r, z+p[1]*r)
	}
	for i, p := range points {
		s.CreateLine(p, points[(i+1)%len(points)])
	}
	profiles, err := ornamentalProfiles(ctx, s)
	if err != nil {
		return nil, err
	}
	if len(profiles) != 1 {
		return nil, fmt.Errorf("meridian sketch has %d profiles", len(profiles))
	}
	axis := decad.SketchLine{Start: decad.Point2{}, End: decad.Point2{V: 1}}
	return decad.Revolve(ctx, doc, s, profiles[0], axis, decad.FullRevolution{})
}

func ornamentalProfiles(ctx context.Context, s *sketch.Sketch) ([]*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, err
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, err
	}
	profiles := s.Profiles()
	for i, p := range profiles {
		if !p.Valid {
			return nil, fmt.Errorf("profile %d is invalid", i)
		}
	}
	return profiles, nil
}
