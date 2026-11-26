package dovetail

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Container names of the demo's bodies.
const (
	TailBlocksContainerName   = "Tail Blocks"
	SocketBlocksContainerName = "Socket Blocks"
)

// DemoWall is the host material the demo leaves beside the tail's wide end and beyond the socket's floor, in mm.
const DemoWall = 4.0

// Demo is the minimal build of the feature: two host blocks with the joint applied, assembled at the joint frame.
type Demo struct {
	Params Params
	// Frame is the joint frame: the origin at the centre of the joint's start on the mating plane, U across the
	// joint, V along the slide direction, and N from the first block into the second.
	Frame r3.Frame
	// First lies on the -N side of the mating plane, Second on the +N side. TailBlock carries the tail and
	// SocketBlock holds the socket; each is First or Second.
	First, Second, TailBlock, SocketBlock *decad.Body
	// Sketches holds every sketch the demo made: the two block outlines, then the joint's two profiles.
	Sketches []*sketch.Sketch
	// Width is each block's size along U, and Height its size along N.
	Width, Height float64
}

// DemoFrame returns the demo's joint frame: the world origin with U = +X, V = +Y and N = +Z.
func DemoFrame() r3.Frame {
	f, err := r3.NewFrame(r3.Vec{}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	if err != nil {
		panic(err) // a constant, valid frame
	}
	return f
}

// BuildDemo builds two rectangular host blocks and joins them with the dovetail described by p. Each block is
// TipWidth + 2 DemoWall wide, Length long and Depth + DemoWall tall. The first block spans N from -Height to 0 and the
// second from 0 to Height. The bodies sit in the containers "Tail Blocks" and "Socket Blocks".
func BuildDemo(ctx context.Context, doc *decad.Document, p Params) (*Demo, error) {
	if ctx == nil {
		return nil, fmt.Errorf("dovetail demo: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("dovetail demo: document must not be nil")
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	f := DemoFrame()
	d := &Demo{Params: p, Frame: f, Width: p.TipWidth() + 2*DemoWall, Height: p.Depth + DemoWall}
	first, firstSketch, err := d.block(ctx, doc, "first", decad.Against)
	if err != nil {
		return nil, err
	}
	second, secondSketch, err := d.block(ctx, doc, "second", decad.Along)
	if err != nil {
		return nil, err
	}
	joint, err := Apply(ctx, doc, p, f, first, second)
	if err != nil {
		return nil, err
	}
	d.First, d.Second = joint.First, joint.Second
	d.TailBlock, d.SocketBlock = joint.TailBody(p), joint.SocketBody(p)
	d.Sketches = []*sketch.Sketch{firstSketch, secondSketch, joint.TailSketch, joint.SocketSketch}

	for name, body := range map[string]*decad.Body{TailBlocksContainerName: d.TailBlock,
		SocketBlocksContainerName: d.SocketBlock} {
		box, err := doc.Root().ChildNamed(name)
		if err != nil {
			return nil, fmt.Errorf("dovetail demo: container %s: %w", name, err)
		}
		if err := box.AddBody(body); err != nil {
			return nil, fmt.Errorf("dovetail demo: add body to %s: %w", name, err)
		}
	}
	return d, nil
}

// block extrudes one host block's rectangle from the mating plane. Its outline is centred on U, runs from V = 0 to
// Length, and the origin is the middle of its edge at V = 0.
func (d *Demo) block(ctx context.Context, doc *decad.Document, name string, dir decad.Direction) (*decad.Body,
	*sketch.Sketch, error) {
	frame, err := r3.NewFrame(d.Frame.Origin(), d.Frame.U(), d.Frame.V())
	if err != nil {
		return nil, nil, fmt.Errorf("dovetail demo: %s block frame: %w", name, err)
	}
	w := sketch.NewWorld()
	plane, err := w.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, nil, fmt.Errorf("dovetail demo: %s block plane: %w", name, err)
	}
	sk, err := w.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("dovetail demo: %s block sketch: %w", name, err)
	}
	half, length := d.Width/2, d.Params.Length
	p0, p1 := sk.CreatePoint(-half, 0), sk.CreatePoint(half, 0)
	p2, p3 := sk.CreatePoint(half, length), sk.CreatePoint(-half, length)
	l0, l1, l2, l3 := sk.CreateLine(p0, p1), sk.CreateLine(p1, p2), sk.CreateLine(p2, p3), sk.CreateLine(p3, p0)
	sk.AddConstraint(
		sketch.NewHorizontal(l0), sketch.NewHorizontal(l2), sketch.NewVertical(l1), sketch.NewVertical(l3),
		sketch.NewMidpoint(sk.Origin(), l0),
		sketch.NewDistance(p0, p1, d.Width), sketch.NewVerticalDistance(p1, p2, length),
	)
	if _, err := sk.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("dovetail demo: solve %s block sketch: %w", name, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, sk); err != nil {
		return nil, nil, fmt.Errorf("dovetail demo: verify %s block sketch: %w", name, err)
	}
	var chosen *sketch.Profile
	for _, candidate := range sk.Profiles() {
		if candidate.Valid && len(candidate.Holes) == 0 {
			chosen = candidate
		}
	}
	if chosen == nil {
		return nil, nil, fmt.Errorf("dovetail demo: select %s block profile: no valid profile", name)
	}
	body, err := decad.Extrude(doc, sk, chosen, decad.Distance{D: units.Millimeters(d.Height), Dir: dir}).Do(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("dovetail demo: extrude %s block: %w", name, err)
	}
	return body, sk, nil
}
