package fastener

import (
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// BoltParams specifies a bolt. Preset is a catalog name. Length is one of the preset's StandardLengths. A nil Placement
// selects the local frame: origin at the center of the head's underside, +Z toward the head, shank along -Z. A hex
// head's corners lie on the X axis.
type BoltParams struct {
	Preset    string
	Length    float64
	Placement *r3.Frame
}

// DefaultBoltParams returns an M2x10 bolt in the local frame.
func DefaultBoltParams() BoltParams {
	return BoltParams{Preset: "M2", Length: 10}
}

// Validate rejects an unknown Preset, an invalid preset, a nonfinite Length or one not in StandardLengths, and an
// invalid Placement.
func (p BoltParams) Validate() error {
	preset, ok := Lookup(p.Preset)
	if !ok {
		return fmt.Errorf("fastener: unknown preset %q", p.Preset)
	}
	if err := preset.Validate(); err != nil {
		return err
	}
	if math.IsNaN(p.Length) || math.IsInf(p.Length, 0) {
		return fmt.Errorf("fastener: Length must be finite")
	}
	if !slices.Contains(preset.Bolt.StandardLengths, p.Length) {
		return fmt.Errorf("fastener: %s Length must be one of %v mm", preset.Name, preset.Bolt.StandardLengths)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("fastener: Placement must be a valid frame")
	}
	return nil
}

// Bolt is the finished bolt and the sketches that built it, all in the local frame.
type Bolt struct {
	Body *decad.Body
	// Profile is the axial section revolved about Z: head, shank and drill point for HeadSocketCap; the shank for
	// HeadHex.
	Profile *sketch.Sketch
	// Socket is the hexagon cut into a HeadSocketCap head. Nil for HeadHex.
	Socket *sketch.Sketch
	// Head is the hexagon of a HeadHex head. Nil for HeadSocketCap.
	Head *sketch.Sketch
}

// Sketches returns the non-nil sketches in build order: Profile, Socket for HeadSocketCap; Head, Profile for HeadHex.
func (b *Bolt) Sketches() []*sketch.Sketch {
	var out []*sketch.Sketch
	for _, s := range []*sketch.Sketch{b.Head, b.Profile, b.Socket} {
		if s != nil {
			out = append(out, s)
		}
	}
	return out
}

// BuildBolt adds one bolt body to doc. ctx and doc must not be nil.
func BuildBolt(ctx context.Context, doc *decad.Document, p BoltParams) (*Bolt, error) {
	if ctx == nil {
		return nil, fmt.Errorf("fastener: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("fastener: document must not be nil")
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("fastener: before creating bolt: %w", err)
	}
	preset, _ := Lookup(p.Preset)

	var (
		b    *Bolt
		body *decad.Body
		err  error
	)
	switch preset.Bolt.Head {
	case HeadSocketCap:
		b, body, err = buildSocketCapBolt(ctx, doc, preset.Bolt, p.Length)
	case HeadHex:
		b, body, err = buildHexHeadBolt(ctx, doc, preset.Bolt, p.Length)
	default:
		err = fmt.Errorf("fastener: unknown head style %v", preset.Bolt.Head)
	}
	if err != nil {
		return nil, err
	}
	if p.Placement != nil {
		transform, err := r3.FromFrame(*p.Placement)
		if err != nil {
			return nil, fmt.Errorf("fastener: create placement transform: %w", err)
		}
		if body, err = decad.Transform(body, transform).Do(ctx); err != nil {
			return nil, fmt.Errorf("fastener: place bolt body: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("fastener: verify bolt body: body is not solid")
	}
	b.Body = body
	return b, nil
}
