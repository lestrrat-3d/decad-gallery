package spinningwheel

import (
	"context"
	"fmt"
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch"
)

// Part identifies one final native solid and its animation motion.
type Part struct {
	Name, Group, Motion string
	Body                *decad.Body
}

// Assembly contains all final solids and all actual consumed sketches.
type Assembly struct {
	Parts    []Part
	Sketches []*sketch.Sketch
	State    State
}
type builder struct {
	ctx  context.Context
	doc  *decad.Document
	p    Params
	a    Assembly
	err  error
	name string
}

// Build constructs the assembly. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Assembly, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spinningwheel: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("spinningwheel: document must not be nil")
	}
	st, err := Motion(p)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	b := builder{ctx: ctx, doc: doc, p: p, a: Assembly{State: st}}
	b.drive()
	b.frame()
	b.adjustment()
	b.head()
	b.yarn()
	b.brake()
	if b.err != nil {
		return nil, b.err
	}
	seen := map[string]struct{}{}
	for _, part := range b.a.Parts {
		if _, ok := seen[part.Name]; ok {
			return nil, fmt.Errorf("spinningwheel: duplicate part %s", part.Name)
		}
		seen[part.Name] = struct{}{}
		if part.Body == nil || !part.Body.IsSolid() {
			return nil, fmt.Errorf("spinningwheel: %s: final body is not solid", part.Name)
		}
	}
	return &b.a, nil
}

func (b *builder) fail(op string, err error) {
	if b.err == nil && err != nil {
		b.err = fmt.Errorf("spinningwheel: %s: %s: %w", b.name, op, err)
	}
}
func (b *builder) part(name, group, motion string, create func() *decad.Body) {
	if b.err != nil {
		return
	}
	b.name = name
	if err := b.ctx.Err(); err != nil {
		b.fail("before construction", err)
		return
	}
	body := create()
	if b.err != nil {
		return
	}
	if body == nil || !body.IsSolid() {
		b.fail("verify solid", fmt.Errorf("body is not solid"))
		return
	}
	b.a.Parts = append(b.a.Parts, Part{name, group, motion, body})
}
