package fastener

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/hexnut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// NutParams specifies a nut. Preset is a catalog name. A nil Placement selects the local frame: origin at the center
// of the lower face, +Z to the upper face, corners on the X axis.
type NutParams struct {
	Preset    string
	Placement *r3.Frame
}

// DefaultNutParams returns the M2 nut in the local frame.
func DefaultNutParams() NutParams {
	return NutParams{Preset: "M2"}
}

// Validate rejects an unknown Preset, an invalid preset and an invalid Placement.
func (p NutParams) Validate() error {
	preset, ok := Lookup(p.Preset)
	if !ok {
		return fmt.Errorf("fastener: unknown preset %q", p.Preset)
	}
	if err := preset.Validate(); err != nil {
		return err
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("fastener: Placement must be a valid frame")
	}
	return nil
}

// Nut is the finished nut and the sketches that built it, all in the local frame.
type Nut struct {
	Body *decad.Body
	// Hexagon is the hexagonal outline and bore extruded along the axis.
	Hexagon *sketch.Sketch
	// LowerChamfer and UpperChamfer are the axial sections of the rings cut from the corners of each face.
	LowerChamfer, UpperChamfer *sketch.Sketch
}

// Sketches returns Hexagon, LowerChamfer, UpperChamfer.
func (n *Nut) Sketches() []*sketch.Sketch {
	return []*sketch.Sketch{n.Hexagon, n.LowerChamfer, n.UpperChamfer}
}

// BuildNut adds one nut body to doc. ctx and doc must not be nil. The construction is internal/hexnut's.
func BuildNut(ctx context.Context, doc *decad.Document, p NutParams) (*Nut, error) {
	if ctx == nil {
		return nil, fmt.Errorf("fastener: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("fastener: document must not be nil")
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	preset, _ := Lookup(p.Preset)
	n := preset.Nut
	size := hexnut.Size{
		Thread:          n.BoreDiameter,
		AcrossFlats:     n.AcrossFlats,
		Thickness:       n.Thickness,
		BearingDiameter: n.BearingDiameter,
	}
	r, err := hexnut.Build(ctx, doc, size, hexnut.Options{Bore: true, ChamferLower: true, ChamferUpper: true}, p.Placement)
	if err != nil {
		return nil, fmt.Errorf("fastener: %w", err)
	}
	return &Nut{Body: r.Body, Hexagon: r.Hexagon, LowerChamfer: r.LowerChamfer, UpperChamfer: r.UpperChamfer}, nil
}
