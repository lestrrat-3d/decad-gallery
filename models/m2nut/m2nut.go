// Package m2nut constructs an M2 hex nut.
package m2nut

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/hexnut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Nominal M2 hex nut dimensions in millimeters, following ISO 4032 and JIS B 1181.
const (
	// BoreDiameter is the nominal thread diameter d. The thread is not modeled.
	BoreDiameter = 2.0
	// AcrossFlats is the wrench size s.
	AcrossFlats = 4.0
	// Thickness is the nut height m.
	Thickness = 1.6
	// BearingDiameter is the diameter dw of the flat bearing circle on each face.
	BearingDiameter = 3.48
	// ChamferAngle is the angle of the corner chamfer from the face, in degrees.
	ChamferAngle = hexnut.ChamferAngle
)

// AcrossCorners returns the distance e between opposite corners of the hexagon.
func AcrossCorners() float64 {
	return AcrossFlats / math.Cos(math.Pi/6)
}

// Params specifies the nut. A nil Placement selects the local frame: the origin is the center of the
// lower face and +Z runs from the lower face to the upper one.
type Params struct {
	Placement *r3.Frame
}

// DefaultParams returns the M2 nut in its local frame.
func DefaultParams() Params {
	return Params{}
}

// Validate rejects an invalid Placement.
func (p Params) Validate() error {
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("m2nut: Placement must be a valid frame")
	}
	return nil
}

// Model exposes the finished nut and the sketches that built it, all in the local frame.
type Model struct {
	Body *decad.Body
	// Hexagon is the hexagonal outline and bore that are extruded along the axis.
	Hexagon *sketch.Sketch
	// LowerChamfer and UpperChamfer are the axial sections of the rings cut from the corners of each face.
	LowerChamfer, UpperChamfer *sketch.Sketch
}

// Build adds one nut to doc. ctx and doc must not be nil. The construction is internal/hexnut's.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error) {
	if ctx == nil {
		return nil, fmt.Errorf("m2nut: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("m2nut: document must not be nil")
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	size := hexnut.Size{
		Thread:          BoreDiameter,
		AcrossFlats:     AcrossFlats,
		Thickness:       Thickness,
		BearingDiameter: BearingDiameter,
	}
	r, err := hexnut.Build(ctx, doc, size, hexnut.Options{Bore: true, ChamferLower: true, ChamferUpper: true}, p.Placement)
	if err != nil {
		return nil, fmt.Errorf("m2nut: %w", err)
	}
	return &Model{Body: r.Body, Hexagon: r.Hexagon, LowerChamfer: r.LowerChamfer, UpperChamfer: r.UpperChamfer}, nil
}
