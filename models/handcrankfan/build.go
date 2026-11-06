package handcrankfan

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// axisHeight is the height of the apex above the table, in mm.
const axisHeight = 170

// Part is one solid of the assembly. Motion is fixed, crank, drive or fan.
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

var (
	// The crank shaft turns about -X by the crank angle (clockwise seen from the right), the drive shaft about +X by the
	// same angle (the spur pair reverses the turn) and the fan about -Y by minus twice that, so the fan turns the way
	// its mirrored blades push air toward the operator.
	crankAxis = r3.Vec{X: -1}
	driveAxis = r3.Vec{X: 1}
	fanAxis   = r3.Vec{Y: -1}
)

// Build constructs the fan at p's pose. ctx and doc must not be nil.
// It takes about half a minute and one gigabyte, almost all of it for the bevel pair.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Assembly, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("build hand-crank fan: context and document must not be nil")
	}
	state, err := Motion(p)
	if err != nil {
		return nil, fmt.Errorf("build hand-crank fan: %w", err)
	}
	b := &builder{ctx: ctx, doc: doc, p: p, a: Assembly{State: state, Params: p}}
	b.handle()
	b.gearbox()
	b.bearings()
	b.drive()
	b.fan()
	if b.err != nil {
		return nil, b.err
	}
	return &b.a, nil
}

func (b *builder) fail(operation string, err error) {
	if err != nil && b.err == nil {
		b.err = fmt.Errorf("hand-crank fan %s: %s: %w", b.name, operation, err)
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
	switch motion {
	case "crank":
		body = b.turn(body, b.a.State.Crank, crankAxis, b.a.State.CrankAngle)
	case "drive":
		body = b.turn(body, b.a.State.Apex, driveAxis, b.a.State.CrankAngle)
	case "fan":
		body = b.turn(body, b.a.State.Apex, fanAxis, b.a.State.FanAngle)
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

// alongX is the plane at x whose normal is +X and whose coordinates are (y, z - axisHeight).
func alongX(x float64) r3.Frame {
	return frame(r3.Vec{X: x, Z: axisHeight}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
}

// alongCrank is the plane at x whose normal is +X and whose coordinates are (y, z - crankHeight).
func alongCrank(x float64) r3.Frame {
	return frame(r3.Vec{X: x, Z: crankHeight}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
}

// alongMinusY is the plane at y whose normal is -Y and whose coordinates are (x, z - axisHeight).
func alongMinusY(y float64) r3.Frame {
	return frame(r3.Vec{Y: y, Z: axisHeight}, r3.Vec{X: 1}, r3.Vec{Z: 1})
}
