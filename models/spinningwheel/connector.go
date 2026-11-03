package spinningwheel

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch"
)

// Connector contains the posed flexible connector and every sketch consumed to build it.
type Connector struct {
	Body     *decad.Body
	Sketches []*sketch.Sketch
}

// BuildConnector rebuilds the flexible connector at p's pose. ctx and doc must not be nil.
func BuildConnector(ctx context.Context, doc *decad.Document, p Params) (*Connector, error) {
	if ctx == nil {
		return nil, fmt.Errorf("build connector: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("build connector: nil document")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("build connector: %w", err)
	}
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("build connector: %w", err)
	}
	state, err := Motion(p)
	if err != nil {
		return nil, fmt.Errorf("build connector pose: %w", err)
	}
	b := &builder{ctx: ctx, doc: doc, p: p, a: Assembly{State: state}, name: "treadle-flex-connector"}
	body := b.flexConnector()
	if b.err != nil {
		return nil, fmt.Errorf("build connector: %w", b.err)
	}
	return &Connector{Body: body, Sketches: b.a.Sketches}, nil
}

func (b *builder) flexConnector() *decad.Body {
	s := b.p.Scale
	state := b.a.State
	d := state.Toe
	lower := b.span(d.Sub(state.PedalFrame.N().Scale(16*s)), d, 3)
	// The upper leg is a hair thinner than the lower one: at equal radii the two cylinders meet tangentially at
	// the elbow, which some poses' Union cannot decide.
	upper := b.span(d, d.Add(state.RodFrame.U().Scale(32*s)), 2.9)
	body := b.union(lower, upper)
	lowerCentre := d.Sub(state.PedalFrame.N().Scale(12 * s))
	upperCentre := d.Add(state.RodFrame.U().Scale(26 * s))
	lowerAxis := state.PedalFrame.U()
	upperAxis := state.RodFrame.V()
	body = b.cut(body, b.cylinder(2.2, 24*s+4,
		axisFrame(lowerCentre.Sub(lowerAxis.Scale(12*s+2)), lowerAxis)))
	return b.refine(b.cut(body, b.cylinder(2.2, 20*s+4,
		axisFrame(upperCentre.Sub(upperAxis.Scale(10*s+2)), upperAxis))), 0.01)
}
