package fastener

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/hexnut"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// buildHexHeadBolt builds a HeadHex bolt in the local frame and returns it with its body. The caller sets Body.
func buildHexHeadBolt(ctx context.Context, doc *decad.Document, size BoltSize, length float64) (*Bolt, *decad.Body, error) {
	r, err := hexnut.Build(ctx, doc, hexnut.Size{
		Thread:          size.ShankDiameter,
		AcrossFlats:     size.Hex.AcrossFlats,
		Thickness:       size.HeadHeight,
		BearingDiameter: size.Hex.BearingDiameter,
	}, hexnut.Options{Bore: false, ChamferLower: true, ChamferUpper: true}, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("fastener: head: %w", err)
	}
	b := &Bolt{Head: r.Hexagon}

	shank, err := buildShank(ctx, doc, b, size, length)
	if err != nil {
		return nil, nil, err
	}
	body, err := decad.Union(r.Body, shank).Do(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("fastener: join head and shank: %w", err)
	}
	return b, body, nil
}

// buildShank sketches the shank's axial section on the XZ plane (U radius, V height) and revolves it about Z.
func buildShank(ctx context.Context, doc *decad.Document, b *Bolt, size BoltSize, length float64) (*decad.Body, error) {
	world := sketch.NewWorld()
	s, err := world.CreateSketch(world.XZ())
	if err != nil {
		return nil, fmt.Errorf("fastener: create shank sketch: %w", err)
	}
	b.Profile = s
	origin := s.Origin()
	r, c := size.ShankDiameter/2, size.TipChamfer

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
		return nil, fmt.Errorf("fastener: solve shank sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("fastener: verify shank sketch: %w", err)
	}
	profile, err := onlyProfile(s)
	if err != nil {
		return nil, fmt.Errorf("fastener: select shank profile: %w", err)
	}
	shank, err := decad.Revolve(doc, s, profile,
		decad.SketchLine{Start: decad.Point2{U: 0, V: 0}, End: decad.Point2{U: 0, V: 1}}, decad.FullRevolution{}).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("fastener: revolve shank: %w", err)
	}
	return shank, nil
}
