// Package m5bolt constructs an M5 hex head bolt envelope of a chosen standard length. The shape follows the
// dimension drawing of ISO 4017 / JIS B 1180: a hexagonal head chamfered on both faces, a plain shank and a
// 45 degree chamfer at the tip. The thread is not modeled.
package m5bolt

import (
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/hexnut"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

const (
	// ShankDiameter is the nominal shank diameter ds in millimeters.
	ShankDiameter = 5.0
	// AcrossFlats is the head width across flats s in millimeters.
	AcrossFlats = 8.0
	// HeadHeight is the head height k in millimeters.
	HeadHeight = 3.5
	// BearingDiameter is the diameter dw of the flat bearing circle on each head face in millimeters.
	BearingDiameter = 6.88
	// ChamferAngle is the angle of the head's corner chamfer from the face, in degrees.
	ChamferAngle = hexnut.ChamferAngle
	// TipChamfer is the radial and axial size u of the tip chamfer in millimeters.
	TipChamfer = 0.5

	// shankOverlap is how far the shank reaches into the head.
	shankOverlap = 1.0
)

// AcrossCorners returns the head width across corners e in millimeters.
func AcrossCorners() float64 {
	return AcrossFlats / math.Cos(math.Pi/6)
}

// StandardLengths returns the standard bolt lengths in millimeters. Each call returns a fresh slice.
func StandardLengths() []float64 {
	return []float64{16, 20, 25, 30, 35, 40, 45, 50, 60, 70, 80}
}

// Params selects the bolt length and placement.
type Params struct {
	// Length is the distance from the underside of the head to the tip, in millimeters.
	Length float64
	// Placement maps the local frame onto the world. Nil selects the local frame: the origin is the center of
	// the head's underside, the head lies along +Z, the shank along -Z, and the head's corners lie on the X axis.
	Placement *r3.Frame
}

// DefaultParams returns a 25 mm bolt in the local frame.
func DefaultParams() Params {
	return Params{Length: 25}
}

// Validate rejects parameters that cannot build a bolt.
func (p Params) Validate() error {
	if math.IsNaN(p.Length) || math.IsInf(p.Length, 0) {
		return fmt.Errorf("m5bolt: Length must be finite")
	}
	if !slices.Contains(StandardLengths(), p.Length) {
		return fmt.Errorf("m5bolt: Length must be one of %v mm", StandardLengths())
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("m5bolt: Placement must be a valid frame")
	}
	return nil
}

// Model is the finished bolt and the sketches that built it.
type Model struct {
	Body *decad.Body
	// Head is the hexagon of the head.
	Head *sketch.Sketch
	// Shank is the axial section of the shank.
	Shank *sketch.Sketch
}

// Build adds one bolt body to doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error) {
	if ctx == nil {
		return nil, fmt.Errorf("m5bolt: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("m5bolt: document must not be nil")
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("m5bolt: before creating head: %w", err)
	}

	before := len(doc.Bodies())
	r, err := hexnut.Build(ctx, doc, hexnut.Size{
		Thread:          ShankDiameter,
		AcrossFlats:     AcrossFlats,
		Thickness:       HeadHeight,
		BearingDiameter: BearingDiameter,
	}, hexnut.Options{Bore: false, ChamferLower: true, ChamferUpper: true}, nil)
	if err != nil {
		return nil, fmt.Errorf("m5bolt: head: %w", err)
	}
	m := &Model{Head: r.Hexagon}

	shank, err := buildShank(ctx, doc, m, p.Length)
	if err != nil {
		return nil, err
	}
	body, err := decad.Union(ctx, r.Body, shank)
	if err != nil {
		return nil, fmt.Errorf("m5bolt: join head and shank: %w", err)
	}
	if n := len(doc.Bodies()) - before; n != 1 {
		return nil, fmt.Errorf("m5bolt: join head and shank: bolt left %d bodies in the document, want 1", n)
	}
	if p.Placement != nil {
		transform, err := r3.FromFrame(*p.Placement)
		if err != nil {
			return nil, fmt.Errorf("m5bolt: create placement transform: %w", err)
		}
		if body, err = decad.Transform(ctx, body, transform); err != nil {
			return nil, fmt.Errorf("m5bolt: place bolt body: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("m5bolt: verify bolt body: body is not solid")
	}
	m.Body = body
	return m, nil
}

// buildShank sketches the shank's axial section on the XZ plane (U radius, V height) and revolves it about Z.
func buildShank(ctx context.Context, doc *decad.Document, m *Model, length float64) (*decad.Body, error) {
	world := sketch.NewWorld()
	s, err := world.CreateSketch(world.XZ())
	if err != nil {
		return nil, fmt.Errorf("m5bolt: create shank sketch: %w", err)
	}
	m.Shank = s
	origin := s.Origin()
	r, c := ShankDiameter/2, TipChamfer

	tipCenter := s.CreatePoint(0, -length)
	tipEdge := s.CreatePoint(r-c, -length)
	tipShoulder := s.CreatePoint(r, -length+c)
	shankTop := s.CreatePoint(r, shankOverlap)
	topCenter := s.CreatePoint(0, shankOverlap)

	axis := s.CreateLine(tipCenter, topCenter)
	tip := s.CreateLine(tipCenter, tipEdge)
	s.CreateLine(tipEdge, tipShoulder)
	wall := s.CreateLine(tipShoulder, shankTop)
	top := s.CreateLine(shankTop, topCenter)

	s.AddConstraint(
		sketch.NewVertical(axis),
		sketch.NewPointOnLine(origin, axis),
		sketch.NewVerticalDistance(origin, tipCenter, -length),
		sketch.NewVerticalDistance(origin, topCenter, shankOverlap),
		sketch.NewHorizontal(tip),
		sketch.NewDistance(tipCenter, tipEdge, r-c),
		sketch.NewHorizontalDistance(origin, tipShoulder, r),
		sketch.NewVerticalDistance(tipEdge, tipShoulder, c),
		sketch.NewVertical(wall),
		sketch.NewHorizontal(top),
	)
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("m5bolt: solve shank sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("m5bolt: verify shank sketch: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("m5bolt: select shank profile: got %d profiles, want one valid region", len(profiles))
	}
	shank, err := decad.Revolve(ctx, doc, s, profiles[0],
		decad.SketchLine{Start: decad.Point2{U: 0, V: 0}, End: decad.Point2{U: 0, V: 1}}, decad.FullRevolution{})
	if err != nil {
		return nil, fmt.Errorf("m5bolt: revolve shank: %w", err)
	}
	return shank, nil
}
