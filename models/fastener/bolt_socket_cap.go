package fastener

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// socketCutterRise is how far the socket cutter extends above the head's top face.
const socketCutterRise = 0.5

// buildSocketCapBolt builds a HeadSocketCap bolt in the local frame and returns it with its body.
func buildSocketCapBolt(ctx context.Context, doc *decad.Document, size BoltSize, length float64) (*Bolt, *decad.Body, error) {
	b := &Bolt{}
	blank, err := revolveBlank(ctx, doc, b, size, length)
	if err != nil {
		return nil, nil, err
	}
	body, err := filletUnderhead(ctx, blank, size)
	if err != nil {
		return nil, nil, err
	}
	if body, err = cutSocket(ctx, doc, b, size, body); err != nil {
		return nil, nil, err
	}
	return b, body, nil
}

// revolveBlank revolves the head and shank section about the Z axis. The section is drawn on the XZ
// plane with U as radius and V as height, from the tip center up the axis to the top of the head.
func revolveBlank(ctx context.Context, doc *decad.Document, b *Bolt, size BoltSize, length float64) (*decad.Body, error) {
	sc := size.SocketCap
	world := sketch.NewWorld()
	s, err := world.CreateSketch(world.XZ())
	if err != nil {
		return nil, fmt.Errorf("fastener: create profile sketch: %w", err)
	}
	b.Profile = s
	origin := s.Origin()
	shankR := size.ShankDiameter / 2
	headR := sc.HeadDiameter / 2
	chamferHeight := sc.HeadChamferWidth * math.Tan(sc.HeadChamferAngle*math.Pi/180)

	tipCenter := s.CreatePoint(0, -length)
	tipEdge := s.CreatePoint(shankR-size.TipChamfer, -length)
	tipShoulder := s.CreatePoint(shankR, -length+size.TipChamfer)
	shankTop := s.CreatePoint(shankR, 0)
	headBottom := s.CreatePoint(headR, 0)
	headSide := s.CreatePoint(headR, size.HeadHeight-chamferHeight)
	headTopEdge := s.CreatePoint(headR-sc.HeadChamferWidth, size.HeadHeight)
	floor := size.HeadHeight - sc.SocketDepth
	pilotTop := s.CreatePoint(sc.DrillPointBaseRadius, size.HeadHeight)
	pilotFloor := s.CreatePoint(sc.DrillPointBaseRadius, floor-sc.PilotDepth)
	pilotApex := s.CreatePoint(0, floor-sc.PilotDepth-sc.DrillPointBaseRadius/math.Tan(sc.DrillPointAngle/2*math.Pi/180))

	axis := s.CreateLine(tipCenter, pilotApex)
	tip := s.CreateLine(tipCenter, tipEdge)
	s.CreateLine(tipEdge, tipShoulder)
	shank := s.CreateLine(tipShoulder, shankTop)
	underside := s.CreateLine(shankTop, headBottom)
	side := s.CreateLine(headBottom, headSide)
	s.CreateLine(headSide, headTopEdge)
	top := s.CreateLine(headTopEdge, pilotTop)
	pilotWall := s.CreateLine(pilotTop, pilotFloor)
	drillPoint := s.CreateLine(pilotFloor, pilotApex)

	s.AddConstraint(
		// The axis lies on the sketch's vertical through the origin, the underside plane is the origin's height.
		sketch.NewVertical(axis),
		sketch.NewPointOnLine(origin, axis),
		sketch.NewVerticalDistance(origin, tipCenter, -length),
		sketch.NewVerticalDistance(origin, shankTop, 0),
		// The tip face is flat; its chamfer is 45 degrees.
		sketch.NewHorizontal(tip),
		sketch.NewDistance(tipCenter, tipEdge, shankR-size.TipChamfer),
		sketch.NewHorizontalDistance(origin, tipShoulder, shankR),
		sketch.NewVerticalDistance(tipEdge, tipShoulder, size.TipChamfer),
		// The shank and head side are cylinders, the underside and top face are flat.
		sketch.NewVertical(shank),
		sketch.NewHorizontal(underside),
		sketch.NewHorizontalDistance(origin, headBottom, headR),
		sketch.NewVertical(side),
		sketch.NewHorizontal(top),
		// The head's top edge chamfer.
		sketch.NewVerticalDistance(headSide, headTopEdge, chamferHeight),
		sketch.NewHorizontalDistance(headTopEdge, headSide, sc.HeadChamferWidth),
		// A round pilot well of DrillPointBaseRadius ends in the drill-point cone. The hexagonal socket
		// later widens the well above the cone's base.
		sketch.NewHorizontalDistance(origin, pilotTop, sc.DrillPointBaseRadius),
		sketch.NewVerticalDistance(origin, pilotTop, size.HeadHeight),
		sketch.NewVertical(pilotWall),
		sketch.NewVerticalDistance(origin, pilotFloor, floor-sc.PilotDepth),
		// Measured from the upward axis to the cone line, which runs from the well wall down to the apex.
		sketch.NewAngle(axis, drillPoint, 180-sc.DrillPointAngle/2),
	)
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("fastener: solve profile sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("fastener: verify profile sketch: %w", err)
	}
	profile, err := onlyProfile(s)
	if err != nil {
		return nil, fmt.Errorf("fastener: select profile: %w", err)
	}
	axisLine := decad.SketchLine{Start: decad.Point2{U: 0, V: 0}, End: decad.Point2{U: 0, V: 1}}
	body, err := decad.Revolve(ctx, doc, s, profile, axisLine, decad.FullRevolution{})
	if err != nil {
		return nil, fmt.Errorf("fastener: revolve profile: %w", err)
	}
	return body, nil
}

// cutSocket removes the hex socket and its conical bottom from the head of body.
func cutSocket(ctx context.Context, doc *decad.Document, b *Bolt, size BoltSize, body *decad.Body) (*decad.Body, error) {
	sc := size.SocketCap
	floor := size.HeadHeight - sc.SocketDepth

	world := sketch.NewWorld()
	hexSketch, err := world.CreateSketch(world.XY())
	if err != nil {
		return nil, fmt.Errorf("fastener: create socket sketch: %w", err)
	}
	b.Socket = hexSketch
	hex, err := hexSketch.CreatePolygon(0, 0, 6, sc.SocketAcrossFlats/math.Sqrt(3))
	if err != nil {
		return nil, fmt.Errorf("fastener: create socket hexagon: %w", err)
	}
	hexSketch.AddConstraint(
		sketch.NewCoincident(hex.Center, hexSketch.Origin()),
		// With the first corner on +X, a flat is parallel to X.
		sketch.NewHorizontal(hex.Sides[1]),
		// The center sits half the across-flats size from a flat; regularity makes the other flats follow.
		sketch.NewDistancePointLine(hex.Center, hex.Sides[1], sc.SocketAcrossFlats/2),
	)
	if _, err := hexSketch.Solve(ctx); err != nil {
		return nil, fmt.Errorf("fastener: solve socket sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, hexSketch); err != nil {
		return nil, fmt.Errorf("fastener: verify socket sketch: %w", err)
	}
	hexProfile, err := onlyProfile(hexSketch)
	if err != nil {
		return nil, fmt.Errorf("fastener: select socket profile: %w", err)
	}
	prism, err := decad.Extrude(ctx, doc, hexSketch, hexProfile,
		decad.Distance{D: units.Millimeters(sc.SocketDepth + socketCutterRise), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("fastener: extrude socket cutter: %w", err)
	}
	lift, err := r3.Translation(r3.Vec{Z: floor})
	if err != nil {
		return nil, fmt.Errorf("fastener: create socket lift: %w", err)
	}
	if prism, err = decad.Transform(ctx, prism, lift); err != nil {
		return nil, fmt.Errorf("fastener: place socket cutter: %w", err)
	}

	if body, err = decad.Cut(ctx, body, prism); err != nil {
		return nil, fmt.Errorf("fastener: cut socket: %w", err)
	}
	return body, nil
}

// filletUnderhead rounds the concave circle where the shank meets the head. The pilot well's floor edge is
// also concave, so the edge is picked by its radius.
func filletUnderhead(ctx context.Context, blank *decad.Body, size BoltSize) (*decad.Body, error) {
	edges, err := decad.Edges(decad.Concave()).SelectEdges(blank)
	if err != nil {
		return nil, fmt.Errorf("fastener: select concave edges: %w", err)
	}
	var corner []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		if math.Abs(math.Hypot(start.X, start.Y)-size.ShankDiameter/2) < 1e-6 {
			corner = append(corner, start)
		}
	}
	if len(corner) != 1 {
		return nil, fmt.Errorf("fastener: select underhead edge: matched %d of %d concave edges", len(corner), len(edges))
	}
	selection := decad.Edges(decad.Concave(), decad.EndpointAt(corner[0])).Exactly(1)
	body, err := decad.Fillet(ctx, blank, selection, units.Millimeters(size.SocketCap.UnderheadRadius))
	if err != nil {
		return nil, fmt.Errorf("fastener: fillet underhead edge: %w", err)
	}
	return body, nil
}

// onlyProfile returns the one valid profile of s.
func onlyProfile(s *sketch.Sketch) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("got %d profiles, want one valid region", len(profiles))
	}
	return profiles[0], nil
}
