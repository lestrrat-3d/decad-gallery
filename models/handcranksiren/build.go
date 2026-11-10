package handcranksiren

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Part is one solid of the assembly. Group is the top-level container name; Motion is fixed, crank, intermediate or
// rotor.
type Part struct {
	Name, Group, Motion string
	Body                *decad.Body
}

// Assembly holds the posed parts and every sketch the build consumed.
type Assembly struct {
	Parts    []Part
	Sketches []*sketch.Sketch
	State    State
	Params   Params
}

type builder struct {
	ctx  context.Context
	doc  *decad.Document
	p    Params
	a    Assembly
	err  error
	name string
}

// turnAxis is the direction every part turns about: forward turns run clockwise seen from the operator's right (+Y), about −Y.
var turnAxis = r3.Vec{Y: -1}

// Build constructs the siren at p's pose. ctx and doc must not be nil; doc comes from NewDocument.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Assembly, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("build hand-crank siren: context and document must not be nil")
	}
	state, err := Motion(p)
	if err != nil {
		return nil, fmt.Errorf("build hand-crank siren: %w", err)
	}
	b := &builder{ctx: ctx, doc: doc, p: p, a: Assembly{State: state, Params: p}}
	b.gears()
	b.shafts()
	b.bearings()
	b.gearbox()
	b.head()
	b.crank()
	b.grip()
	for _, s := range sitesNamed("gearbox-") {
		b.hardware(s)
	}
	if b.err != nil {
		return nil, b.err
	}
	if err := moveBodies(doc, b.a.Parts); err != nil {
		return nil, fmt.Errorf("hand-crank siren: move bodies into containers: %w", err)
	}
	return &b.a, nil
}

func (b *builder) fail(operation string, err error) {
	if err != nil && b.err == nil {
		b.err = fmt.Errorf("hand-crank siren %s: %s: %w", b.name, operation, err)
	}
}

// part records the body create returns, built at crank angle 0, after posing it by its motion.
func (b *builder) part(name, group, motion string, create func() *decad.Body) {
	if b.err != nil {
		return
	}
	b.name = name
	body := create()
	if b.err != nil {
		return
	}
	if motion != "fixed" {
		xf, err := motionTransform(b.a.State, motion)
		b.fail("create pose", err)
		if b.err != nil {
			return
		}
		body, err = decad.Transform(b.ctx, body, xf)
		b.fail("pose body", err)
		if b.err != nil {
			return
		}
	}
	if body == nil || !body.IsSolid() {
		b.fail("record part", fmt.Errorf("construction did not return a solid"))
		return
	}
	b.a.Parts = append(b.a.Parts, Part{Name: name, Group: group, Motion: motion, Body: body})
}

// motionTransform is the pose of a part of the given motion (fixed, crank, intermediate or rotor) in state s.
// builder.part and the tests pose through it, so both see the same placement.
func motionTransform(s State, motion string) (r3.Transform, error) {
	switch motion {
	case "fixed":
		return r3.Identity(), nil
	case "crank":
		return r3.RotationAround(s.Crank, turnAxis, units.Radians(s.CrankAngle))
	case "intermediate":
		return r3.RotationAround(s.Intermediate, turnAxis, units.Radians(s.IntermediateAngle))
	case "rotor":
		return r3.RotationAround(s.Rotor, turnAxis, units.Radians(s.RotorAngle))
	}
	return r3.Transform{}, fmt.Errorf("unknown motion %q", motion)
}
