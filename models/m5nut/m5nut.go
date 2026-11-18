// Package m5nut constructs the M5 hex nut that pairs with the M5 bolt. It follows ISO 4032 (style 1) and
// JIS B 1181. The thread is not modeled; the bore is the nominal 5 mm.
package m5nut

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/hexnut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

const (
	// BoreDiameter is the bore diameter d, the nominal thread diameter, in millimeters.
	BoreDiameter = 5.0
	// AcrossFlats is the width across flats s, in millimeters.
	AcrossFlats = 8.0
	// Thickness is the nut thickness m, in millimeters.
	Thickness = 4.7
	// BearingDiameter is the diameter dw of the flat bearing circle on each face, in millimeters.
	BearingDiameter = 6.9
	// ChamferAngle is the angle of the corner chamfer from the face, in degrees.
	ChamferAngle = hexnut.ChamferAngle
)

// AcrossCorners returns the width across corners e, in millimeters.
func AcrossCorners() float64 {
	return AcrossFlats / math.Cos(math.Pi/6)
}

// Params configures Build.
type Params struct {
	// Placement maps the local frame onto it. Nil selects the local frame: the origin is the center of the
	// lower face, +Z runs to the upper face, and the corners lie on the X axis.
	Placement *r3.Frame
}

// DefaultParams returns the parameters of the nut in its local frame.
func DefaultParams() Params {
	return Params{}
}

// Validate rejects parameters that cannot build a nut.
func (p Params) Validate() error {
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("m5nut: Placement must be a valid frame")
	}
	return nil
}

// Model holds the finished nut and the sketches that built it.
type Model struct {
	// Body is the nut.
	Body *decad.Body
	// Hexagon is the hexagon-and-bore outline that is extruded.
	Hexagon *sketch.Sketch
	// LowerChamfer and UpperChamfer are the sections of the rings cut from the corners of each face.
	LowerChamfer, UpperChamfer *sketch.Sketch
}

// Build adds one M5 nut body to doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error) {
	if ctx == nil {
		return nil, fmt.Errorf("m5nut: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("m5nut: document must not be nil")
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	r, err := hexnut.Build(ctx, doc, hexnut.Size{
		Thread:          BoreDiameter,
		AcrossFlats:     AcrossFlats,
		Thickness:       Thickness,
		BearingDiameter: BearingDiameter,
	}, hexnut.Options{Bore: true, ChamferLower: true, ChamferUpper: true}, p.Placement)
	if err != nil {
		return nil, fmt.Errorf("m5nut: %w", err)
	}
	return &Model{Body: r.Body, Hexagon: r.Hexagon, LowerChamfer: r.LowerChamfer, UpperChamfer: r.UpperChamfer}, nil
}
