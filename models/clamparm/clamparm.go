// Package clamparm builds a printable, parametric desk-edge clamp arm: a C-frame clamp with a screw and a swivel pad,
// a pan joint (J1, vertical axis), a swivel bar, three friction knuckles (J2 shoulder, J3 elbow, J4 wrist) joined by two
// links, a head, and the Mount interface at the head's free end. The interface is the deliverable at the free end: a
// 45 degree cone boss and an M5 stud on the head's top face, and a documented recess and nut that any adapter
// provides. One placeholder pad adapter ships with the model; real tool holders are separate items that import this
// package's interface constants and [InterfaceRecessTool].
//
// Every pivot and fastener is an M5 hex bolt and nut (the M5 preset of [fastener]). Joints are knuckles: a tongue in a fork,
// closed by a lobed knob on a bolt, so the bolt never turns in its nut.
//
// [BuildParts] builds the 13 printed bodies, posed in the world frame; [Build] adds the six bolts and six nuts and files
// all 25 bodies in named containers. Every error starts with "clamparm: ". See instructions.md for the contract and
// steps.md for the construction.
package clamparm

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/decad-gallery/models/fastener"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Part is one printed body in the world frame, with the pose it was built at and what it needs to print.
type Part struct {
	// Body is the finished, placed body.
	Body *decad.Body
	// Frame maps the part's local frame to the world, Params.Placement included.
	Frame r3.Frame
	// PrintUp is the world direction that points up when the part is on the print bed.
	PrintUp r3.Vec
	// Sketches are every sketch the part's construction used.
	Sketches []*sketch.Sketch
}

// Parts are the 13 printed bodies, posed, in no container.
type Parts struct {
	// Clamp is the clamp frame.
	Clamp Part
	// Pad is the swivel pad on the clamp screw's tip.
	Pad Part
	// Swivel is the bar between J1 and J2.
	Swivel Part
	// Links are link A and link B.
	Links [2]Part
	// Head is the head with the interface's boss.
	Head Part
	// Adapter is the placeholder pad adapter.
	Adapter Part
	// Knobs are the J1 to J4 knobs.
	Knobs [4]Part
	// ScrewKnobs are the clamp screw's knob and the stud's knob.
	ScrewKnobs [2]Part
	// Dimensions are the derived poses, before Placement.
	Dimensions Dimensions
}

// Model is the finished assembly: the printed parts and the purchased hardware.
type Model struct {
	Parts
	// Bolts are the J1, J2, J3, J4, stud and clamp screw bolts, in that order.
	Bolts [6]*fastener.Bolt
	// Nuts are the J1 to J4 knob nuts, the jaw nut and the adapter nut, in that order.
	Nuts [6]*fastener.Nut
}

// placedFrame is f moved by at, or f when at is nil.
func placedFrame(at *r3.Frame, f r3.Frame) (r3.Frame, error) {
	if at == nil {
		return f, nil
	}
	b := &builder{}
	xf, err := r3.FromFrame(*at)
	if err != nil {
		return r3.Frame{}, fmt.Errorf("clamparm: placement transform: %w", err)
	}
	lf, err := r3.FromFrame(f)
	if err != nil {
		return r3.Frame{}, fmt.Errorf("clamparm: part transform: %w", err)
	}
	placed := b.frameOf(b.compose(xf, lf))
	if b.err != nil {
		return r3.Frame{}, fmt.Errorf("clamparm: place frame: %w", b.err)
	}
	return placed, nil
}

// worldTransform is the transform Placement applies to the world frame.
func worldTransform(at *r3.Frame) (r3.Transform, error) {
	if at == nil {
		return r3.Identity(), nil
	}
	xf, err := r3.FromFrame(*at)
	if err != nil {
		return r3.Transform{}, fmt.Errorf("clamparm: placement transform: %w", err)
	}
	return xf, nil
}

// BuildParts builds the 13 printed bodies, posed, in no container. ctx and doc must not be nil. A rejected build adds
// no body: every check precedes the first geometry call.
func BuildParts(ctx context.Context, doc *decad.Document, p Params) (*Parts, error) {
	if ctx == nil {
		return nil, fmt.Errorf("clamparm: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("clamparm: document must not be nil")
	}
	dm, err := p.Derive()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("clamparm: before creating geometry: %w", err)
	}
	if dm.ScrewKnobGap < 0.5 {
		return nil, fmt.Errorf("clamparm: screw knob gap %g is under 0.5 mm", dm.ScrewKnobGap)
	}

	identity, err := frame(r3.Vec{}, axisX, axisY)
	if err != nil {
		return nil, err
	}
	up, down := axisZ, axisZ.Scale(-1)
	type builderFunc func(context.Context, *decad.Document, Params) (*decad.Body, []*sketch.Sketch, error)
	steps := []struct {
		name  string
		build builderFunc
		frame r3.Frame
		up    r3.Vec
	}{
		{"frame", buildClamp, identity, axisY},
		{"pad", buildPad, dm.PadFrame, down},
		{"swivel", buildSwivel, dm.Swivel, up},
		{"link A", buildLink, dm.LinkA, up},
		{"link B", buildLink, dm.LinkB, up},
		{"head", buildHead, dm.Head, up},
		{"adapter", buildAdapter, dm.Interface, up},
		{"knob J1", buildKnob, dm.KnobFrames[0], up},
		{"knob J2", buildKnob, dm.KnobFrames[1], up},
		{"knob J3", buildKnob, dm.KnobFrames[2], up},
		{"knob J4", buildKnob, dm.KnobFrames[3], up},
		{"screw knob", buildScrewKnob, dm.ScrewKnobFrames[0], up},
		{"stud knob", buildScrewKnob, dm.ScrewKnobFrames[1], up},
	}
	parts := make([]Part, len(steps))
	for i, st := range steps {
		body, sketches, err := st.build(ctx, doc, p)
		if err != nil {
			return nil, err
		}
		if !body.IsSolid() {
			return nil, fmt.Errorf("clamparm: %s is not a solid", st.name)
		}
		placed, err := placedFrame(p.Placement, st.frame)
		if err != nil {
			return nil, err
		}
		if body, err = placement.Place(ctx, body, &placed); err != nil {
			return nil, fmt.Errorf("clamparm: place %s: %w", st.name, err)
		}
		move, err := r3.FromFrame(placed)
		if err != nil {
			return nil, fmt.Errorf("clamparm: %s frame: %w", st.name, err)
		}
		parts[i] = Part{Body: body, Frame: placed, PrintUp: move.ApplyDir(st.up), Sketches: sketches}
	}
	return &Parts{
		Clamp: parts[0], Pad: parts[1], Swivel: parts[2], Links: [2]Part{parts[3], parts[4]}, Head: parts[5],
		Adapter: parts[6], Knobs: [4]Part{parts[7], parts[8], parts[9], parts[10]},
		ScrewKnobs: [2]Part{parts[11], parts[12]}, Dimensions: dm,
	}, nil
}

// Build builds the printed parts and the hardware and files all 25 bodies in the containers FrameContainerName to
// NutsContainerName. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error) {
	parts, err := BuildParts(ctx, doc, p)
	if err != nil {
		return nil, err
	}
	dm := parts.Dimensions
	xf, err := worldTransform(p.Placement)
	if err != nil {
		return nil, err
	}
	b := &builder{}
	sites := [6]m5fastener.Site{dm.PanSite, dm.JointSites[0], dm.JointSites[1], dm.JointSites[2], dm.StudSite,
		dm.ScrewSite}
	nutFrames := [6]r3.Frame{dm.KnobNutFrames[0], dm.KnobNutFrames[1], dm.KnobNutFrames[2], dm.KnobNutFrames[3],
		dm.JawNutFrame, dm.AdapterNutFrame}
	for i := range sites {
		sites[i] = moveSite(xf, sites[i])
		nutFrames[i] = b.moveFrame(xf, nutFrames[i])
	}
	if b.err != nil {
		return nil, fmt.Errorf("clamparm: place hardware frames: %w", b.err)
	}
	m := &Model{Parts: *parts}
	for i, site := range sites {
		if m.Bolts[i], err = m5fastener.PlaceBolt(ctx, doc, site); err != nil {
			return nil, fmt.Errorf("clamparm: bolt %d: %w", i, err)
		}
	}
	for i, f := range nutFrames {
		if m.Nuts[i], err = m5fastener.PlaceNut(ctx, doc, f); err != nil {
			return nil, fmt.Errorf("clamparm: nut %d: %w", i, err)
		}
	}

	bodiesOf := func(ps ...Part) []*decad.Body {
		out := make([]*decad.Body, len(ps))
		for i, part := range ps {
			out[i] = part.Body
		}
		return out
	}
	bolts := make([]*decad.Body, len(m.Bolts))
	for i, bolt := range m.Bolts {
		bolts[i] = bolt.Body
	}
	nuts := make([]*decad.Body, len(m.Nuts))
	for i, nut := range m.Nuts {
		nuts[i] = nut.Body
	}
	groups := []struct {
		name   string
		bodies []*decad.Body
	}{
		{FrameContainerName, bodiesOf(m.Clamp)},
		{PadContainerName, bodiesOf(m.Pad)},
		{SwivelContainerName, bodiesOf(m.Swivel)},
		{LinksContainerName, bodiesOf(m.Links[:]...)},
		{HeadContainerName, bodiesOf(m.Head)},
		{AdapterContainerName, bodiesOf(m.Adapter)},
		{KnobsContainerName, bodiesOf(m.Knobs[:]...)},
		{ScrewKnobsContainerName, bodiesOf(m.ScrewKnobs[:]...)},
		{BoltsContainerName, bolts},
		{NutsContainerName, nuts},
	}
	for _, g := range groups {
		c, err := doc.Root().ChildNamed(g.name)
		if err != nil {
			return nil, fmt.Errorf("clamparm: container %s: %w", g.name, err)
		}
		for _, body := range g.bodies {
			if err := c.AddBody(body); err != nil {
				return nil, fmt.Errorf("clamparm: file body in %s: %w", g.name, err)
			}
		}
	}
	return m, nil
}
