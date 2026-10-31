package spinningwheel

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

type Part struct {
	Name, Group, Motion string
	Body                *decad.Body
}

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
	tilt bool
}

func Build(ctx context.Context, doc *decad.Document, p Params) (*Assembly, error) {
	if ctx == nil {
		return nil, fmt.Errorf("build spinning wheel: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("build spinning wheel: document must not be nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("build spinning wheel: %w", err)
	}
	state, err := Motion(p)
	if err != nil {
		return nil, fmt.Errorf("build spinning wheel motion: %w", err)
	}
	b := builder{ctx: ctx, doc: doc, p: p, a: Assembly{State: state}}
	b.frame()
	b.drive()
	b.adjustment()
	b.tilt = true
	b.head()
	b.yarn()
	b.brake()
	b.tilt = false
	if b.err != nil {
		return nil, b.err
	}
	return &b.a, nil
}

func (b *builder) fail(operation string, err error) {
	if err != nil && b.err == nil {
		b.err = fmt.Errorf("spinning wheel %s: %s: %w", b.name, operation, err)
	}
}

func (b *builder) part(name, group, motion string, create func() *decad.Body) {
	if b.err != nil {
		return
	}
	b.name = name
	if err := b.ctx.Err(); err != nil {
		b.fail("create part", err)
		return
	}
	for _, existing := range b.a.Parts {
		if existing.Name == name {
			b.fail("create part", fmt.Errorf("duplicate part name"))
			return
		}
	}
	body := create()
	if b.err != nil {
		return
	}
	if body == nil || !body.IsSolid() {
		b.fail("create part", fmt.Errorf("construction did not return a solid"))
		return
	}
	if b.tilt {
		xf, err := r3.RotationAround(b.a.State.HeadPivot, r3.Vec{Y: -1}, units.Radians(b.a.State.HeadTilt))
		if err != nil {
			b.fail("create head tilt", err)
			return
		}
		body, err = decad.Transform(b.ctx, body, xf)
		if err != nil {
			b.fail("apply head tilt", err)
			return
		}
	}
	b.a.Parts = append(b.a.Parts, Part{Name: name, Group: group, Motion: motion, Body: body})
}
