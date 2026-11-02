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

type Mount struct {
	Part, Kind          string
	Supports, Fasteners []string
	Clearance           float64
}

type Assembly struct {
	Parts    []Part
	Sketches []*sketch.Sketch
	State    State
	Mounts   []Mount
}

type builder struct {
	ctx     context.Context
	doc     *decad.Document
	p       Params
	a       Assembly
	err     error
	name    string
	tilt    bool
	preview *Preview
}

// Build constructs the physical assembly. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Assembly, error) {
	b, err := newAssemblyBuilder(ctx, doc, p)
	if err != nil {
		return nil, err
	}
	b.buildInitialParts()
	b.finishJoinery()
	if b.err != nil {
		return nil, b.err
	}
	return &b.a, nil
}

func newAssemblyBuilder(ctx context.Context, doc *decad.Document, p Params) (*builder, error) {
	if ctx == nil {
		return nil, fmt.Errorf("build spinning wheel: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("build spinning wheel: nil document")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("build spinning wheel: %w", err)
	}
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("build spinning wheel: %w", err)
	}
	state, err := Motion(p)
	if err != nil {
		return nil, fmt.Errorf("build spinning wheel motion: %w", err)
	}
	return &builder{ctx: ctx, doc: doc, p: p, a: Assembly{State: state}}, nil
}

func (b *builder) buildInitialParts() {
	b.frame()
	b.drive()
	b.adjustment()
	b.tilt = true
	b.head()
	b.yarn()
	b.brake()
	b.tilt = false
}

func (b *builder) fail(operation string, err error) {
	if err != nil && b.err == nil {
		b.err = fmt.Errorf("spinning wheel %s: %s: %w", b.name, operation, err)
	}
}

func (b *builder) part(name, group, motion string, create func() *decad.Body) {
	if b.preview != nil {
		defer b.recordPreviewFailure(name)
	}
	if b.err != nil {
		return
	}
	b.name = name
	for _, part := range b.a.Parts {
		if part.Name == name {
			b.fail("record part", fmt.Errorf("duplicate name %q", name))
			return
		}
	}
	body := create()
	if b.err != nil {
		return
	}
	if b.tilt {
		body = b.tiltBody(body)
	}
	if b.err != nil {
		return
	}
	if body == nil || !body.IsSolid() {
		b.fail("record part", fmt.Errorf("construction did not return a solid"))
		return
	}
	b.a.Parts = append(b.a.Parts, Part{Name: name, Group: group, Motion: motion, Body: body})
}

func (b *builder) tiltBody(body *decad.Body) *decad.Body {
	if b.err != nil {
		return nil
	}
	xf, err := r3.RotationAround(b.a.State.HeadPivot, r3.Vec{Y: -1}, units.Radians(b.a.State.HeadTilt))
	if err != nil {
		b.fail("create head tilt", err)
		return nil
	}
	body, err = decad.Transform(b.ctx, body, xf)
	b.fail("apply head tilt", err)
	return body
}
