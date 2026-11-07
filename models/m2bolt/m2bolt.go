// Package m2bolt constructs an M2 socket head cap screw with a configurable length.
package m2bolt

import (
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Nominal M2 socket head cap screw dimensions in millimeters, following ISO 4762 / JIS B 1176.
const (
	// ShankDiameter is the nominal diameter ds. The thread is not modeled.
	ShankDiameter = 2.0
	// HeadDiameter is the head diameter dk.
	HeadDiameter = 3.8
	// HeadHeight is the head height k.
	HeadHeight = 2.0
	// SocketAcrossFlats is the hex socket size s.
	SocketAcrossFlats = 1.5
	// SocketDepth is the hex socket depth t, measured from the top face to the start of the drill point.
	SocketDepth = 1.0
	// HeadChamferWidth is the radial width v of the chamfer on the head's top edge.
	HeadChamferWidth = 0.2
	// HeadChamferAngle is the chamfer's angle from the top face, in degrees.
	HeadChamferAngle = 30.0
	// UnderheadRadius is the fillet radius r where the shank meets the head.
	UnderheadRadius = 0.1
	// TipChamfer is the radial and axial size of the 45 degree chamfer at the tip.
	TipChamfer = 0.25
	// DrillPointAngle is the included angle of the cone that closes the socket bottom, in degrees.
	DrillPointAngle = 118.0
	// DrillPointBaseRadius is the radius of the pilot well and of the cone's base, inside the socket's flats.
	DrillPointBaseRadius = 0.7
	// PilotDepth is how far the round pilot well runs below the socket floor before the drill point starts.
	PilotDepth = 0.1

	// socketCutterRise is how far the socket cutter extends above the head's top face.
	socketCutterRise = 0.5
)

// StandardLengths returns the nominal lengths in millimeters that Build accepts, in ascending order.
func StandardLengths() []float64 {
	return []float64{6, 10, 16, 18, 20, 25}
}

// Params specifies the bolt. Length is the nominal length l in millimeters, one of StandardLengths, measured from the
// underside of the head to the tip. A nil Placement selects the local frame, whose origin is the
// center of the head's underside and whose +Z points from the underside to the head. The shank
// extends along -Z.
type Params struct {
	Length    float64
	Placement *r3.Frame
}

// DefaultParams returns an M2x10 bolt.
func DefaultParams() Params {
	return Params{Length: 10}
}

// Validate rejects a nonfinite Length, one that is not a standard length, and an invalid Placement.
func (p Params) Validate() error {
	if math.IsNaN(p.Length) || math.IsInf(p.Length, 0) {
		return fmt.Errorf("m2bolt: Length must be finite")
	}
	if !slices.Contains(StandardLengths(), p.Length) {
		return fmt.Errorf("m2bolt: Length must be one of %v mm", StandardLengths())
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("m2bolt: Placement must be a valid frame")
	}
	return nil
}

// Model exposes the finished bolt and the sketches that built it, all in the local frame.
type Model struct {
	Body *decad.Body
	// Profile is the axial section revolved into the head, shank and the drill point below the socket.
	Profile *sketch.Sketch
	// Socket is the hexagon cut into the head.
	Socket *sketch.Sketch
}

// Build adds one bolt to doc. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error) {
	if ctx == nil {
		return nil, fmt.Errorf("m2bolt: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("m2bolt: document must not be nil")
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("m2bolt: before creating profile: %w", err)
	}

	m := &Model{}
	blank, err := revolveBlank(ctx, doc, m, p.Length)
	if err != nil {
		return nil, err
	}
	body, err := filletUnderhead(ctx, blank)
	if err != nil {
		return nil, err
	}
	if body, err = cutSocket(ctx, doc, m, body); err != nil {
		return nil, err
	}
	if p.Placement != nil {
		transform, err := r3.FromFrame(*p.Placement)
		if err != nil {
			return nil, fmt.Errorf("m2bolt: create placement transform: %w", err)
		}
		if body, err = decad.Transform(ctx, body, transform); err != nil {
			return nil, fmt.Errorf("m2bolt: place bolt body: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("m2bolt: verify bolt body: body is not solid")
	}
	m.Body = body
	return m, nil
}

// revolveBlank revolves the head and shank section about the Z axis. The section is drawn on the XZ
// plane with U as radius and V as height, from the tip center up the axis to the top of the head.
func revolveBlank(ctx context.Context, doc *decad.Document, m *Model, length float64) (*decad.Body, error) {
	world := sketch.NewWorld()
	s, err := world.CreateSketch(world.XZ())
	if err != nil {
		return nil, fmt.Errorf("m2bolt: create profile sketch: %w", err)
	}
	m.Profile = s
	origin := s.Origin()
	shankR := ShankDiameter / 2
	headR := HeadDiameter / 2
	chamferHeight := HeadChamferWidth * math.Tan(HeadChamferAngle*math.Pi/180)

	tipCenter := s.CreatePoint(0, -length)
	tipEdge := s.CreatePoint(shankR-TipChamfer, -length)
	tipShoulder := s.CreatePoint(shankR, -length+TipChamfer)
	shankTop := s.CreatePoint(shankR, 0)
	headBottom := s.CreatePoint(headR, 0)
	headSide := s.CreatePoint(headR, HeadHeight-chamferHeight)
	headTopEdge := s.CreatePoint(headR-HeadChamferWidth, HeadHeight)
	floor := HeadHeight - SocketDepth
	pilotTop := s.CreatePoint(DrillPointBaseRadius, HeadHeight)
	pilotFloor := s.CreatePoint(DrillPointBaseRadius, floor-PilotDepth)
	pilotApex := s.CreatePoint(0, floor-PilotDepth-DrillPointBaseRadius/math.Tan(DrillPointAngle/2*math.Pi/180))

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
		sketch.NewDistance(tipCenter, tipEdge, shankR-TipChamfer),
		sketch.NewHorizontalDistance(origin, tipShoulder, shankR),
		sketch.NewVerticalDistance(tipEdge, tipShoulder, TipChamfer),
		// The shank and head side are cylinders, the underside and top face are flat.
		sketch.NewVertical(shank),
		sketch.NewHorizontal(underside),
		sketch.NewHorizontalDistance(origin, headBottom, headR),
		sketch.NewVertical(side),
		sketch.NewHorizontal(top),
		// The head's top edge chamfer.
		sketch.NewVerticalDistance(headSide, headTopEdge, chamferHeight),
		sketch.NewHorizontalDistance(headTopEdge, headSide, HeadChamferWidth),
		// A round pilot well of DrillPointBaseRadius ends in the drill-point cone. The hexagonal socket
		// later widens the well above the cone's base.
		sketch.NewHorizontalDistance(origin, pilotTop, DrillPointBaseRadius),
		sketch.NewVerticalDistance(origin, pilotTop, HeadHeight),
		sketch.NewVertical(pilotWall),
		sketch.NewVerticalDistance(origin, pilotFloor, floor-PilotDepth),
		// Measured from the upward axis to the cone line, which runs from the well wall down to the apex.
		sketch.NewAngle(axis, drillPoint, 180-DrillPointAngle/2),
	)
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("m2bolt: solve profile sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("m2bolt: verify profile sketch: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("m2bolt: select profile: got %d profiles, want one valid region", len(profiles))
	}
	axisLine := decad.SketchLine{Start: decad.Point2{U: 0, V: 0}, End: decad.Point2{U: 0, V: 1}}
	body, err := decad.Revolve(ctx, doc, s, profiles[0], axisLine, decad.FullRevolution{})
	if err != nil {
		return nil, fmt.Errorf("m2bolt: revolve profile: %w", err)
	}
	return body, nil
}

// cutSocket removes the hex socket and its conical bottom from the head of body.
func cutSocket(ctx context.Context, doc *decad.Document, m *Model, body *decad.Body) (*decad.Body, error) {
	floor := HeadHeight - SocketDepth

	world := sketch.NewWorld()
	hexSketch, err := world.CreateSketch(world.XY())
	if err != nil {
		return nil, fmt.Errorf("m2bolt: create socket sketch: %w", err)
	}
	m.Socket = hexSketch
	hex, err := hexSketch.CreatePolygon(0, 0, 6, SocketAcrossFlats/math.Sqrt(3))
	if err != nil {
		return nil, fmt.Errorf("m2bolt: create socket hexagon: %w", err)
	}
	hexSketch.AddConstraint(
		sketch.NewCoincident(hex.Center, hexSketch.Origin()),
		// With the first corner on +X, a flat is parallel to X.
		sketch.NewHorizontal(hex.Sides[1]),
		// The center sits half the across-flats size from a flat; regularity makes the other flats follow.
		sketch.NewDistancePointLine(hex.Center, hex.Sides[1], SocketAcrossFlats/2),
	)
	if _, err := hexSketch.Solve(ctx); err != nil {
		return nil, fmt.Errorf("m2bolt: solve socket sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, hexSketch); err != nil {
		return nil, fmt.Errorf("m2bolt: verify socket sketch: %w", err)
	}
	hexProfile, err := onlyProfile(hexSketch)
	if err != nil {
		return nil, fmt.Errorf("m2bolt: select socket profile: %w", err)
	}
	prism, err := decad.Extrude(ctx, doc, hexSketch, hexProfile,
		decad.Distance{D: units.Millimeters(SocketDepth + socketCutterRise), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("m2bolt: extrude socket cutter: %w", err)
	}
	lift, err := r3.Translation(r3.Vec{Z: floor})
	if err != nil {
		return nil, fmt.Errorf("m2bolt: create socket lift: %w", err)
	}
	if prism, err = decad.Transform(ctx, prism, lift); err != nil {
		return nil, fmt.Errorf("m2bolt: place socket cutter: %w", err)
	}

	if body, err = decad.Cut(ctx, body, prism); err != nil {
		return nil, fmt.Errorf("m2bolt: cut socket: %w", err)
	}
	return body, nil
}

// filletUnderhead rounds the concave circle where the shank meets the head. The pilot well's floor edge is
// also concave, so the edge is picked by its radius.
func filletUnderhead(ctx context.Context, blank *decad.Body) (*decad.Body, error) {
	edges, err := decad.Edges(decad.Concave()).SelectEdges(blank)
	if err != nil {
		return nil, fmt.Errorf("m2bolt: select concave edges: %w", err)
	}
	var corner []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		if math.Abs(math.Hypot(start.X, start.Y)-ShankDiameter/2) < 1e-6 {
			corner = append(corner, start)
		}
	}
	if len(corner) != 1 {
		return nil, fmt.Errorf("m2bolt: select underhead edge: matched %d of %d concave edges", len(corner), len(edges))
	}
	selection := decad.Edges(decad.Concave(), decad.EndpointAt(corner[0])).Exactly(1)
	body, err := decad.Fillet(ctx, blank, selection, units.Millimeters(UnderheadRadius))
	if err != nil {
		return nil, fmt.Errorf("m2bolt: fillet underhead edge: %w", err)
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
