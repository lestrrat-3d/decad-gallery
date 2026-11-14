// Package crankhandle builds a printed crank for a D-flat shaft: an arm pressed onto the shaft, a pin clamped in the
// arm, and a handle that turns freely on the pin, held on by the pin's head. One M2 bolt and nut clamp the pin. The
// hand-crank siren imports it for its crank.
//
// The local frame has its origin on the shaft axis in the arm's back face, the face toward the mechanism. +Z runs
// along the shaft axis away from the mechanism, the arm runs along +Y to the pin, and the bore's flat faces +Y.
package crankhandle

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Container names, one per kind of body.
const (
	ArmContainerName    = "Arm"
	PinContainerName    = "Pin"
	HandleContainerName = "Handle"
	BoltsContainerName  = "Bolts"
	NutsContainerName   = "Nuts"
)

// Parts are the crank's printed bodies, placed, without the bolt and nut.
type Parts struct {
	Arm, Pin, Handle *decad.Body
	// Sketches are every sketch the parts were built from, each fully constrained.
	Sketches   []*sketch.Sketch
	Dimensions Dimensions
	// Site is the pin bolt in the world frame.
	Site m2fastener.Site
}

// Model is a built crank: its parts, and the pin bolt and nut.
type Model struct {
	Parts
	Bolt *m2bolt.Model
	Nut  *m2nut.Model
}

// BuildParts adds the arm, the pin and the handle to doc and returns them. It puts them in no container, so an
// assembly can file them with its own parts. ctx and doc must not be nil.
func BuildParts(ctx context.Context, doc *decad.Document, p Params) (*Parts, error) {
	if ctx == nil {
		return nil, errors.New("crankhandle: context must not be nil")
	}
	if doc == nil {
		return nil, errors.New("crankhandle: document must not be nil")
	}
	dm, err := p.Derive()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("crankhandle: before creating geometry: %w", err)
	}
	r := &run{ctx: ctx, doc: doc, p: p, dm: dm}
	out := &Parts{Dimensions: dm}
	for _, step := range []struct {
		name  string
		build func() (*decad.Body, error)
		dst   **decad.Body
	}{{"arm", r.arm, &out.Arm}, {"pin", r.pin, &out.Pin}, {"handle", r.handle, &out.Handle}} {
		body, err := step.build()
		if err != nil {
			return nil, fmt.Errorf("crankhandle: %s: %w", step.name, err)
		}
		if body == nil || !body.IsSolid() {
			return nil, fmt.Errorf("crankhandle: %s is not a solid", step.name)
		}
		if *step.dst, err = placement.Place(ctx, body, p.Placement); err != nil {
			return nil, fmt.Errorf("crankhandle: place %s: %w", step.name, err)
		}
	}
	out.Sketches = r.sketches
	if out.Site, err = placeSite(dm.Site, p.Placement); err != nil {
		return nil, fmt.Errorf("crankhandle: place pin bolt: %w", err)
	}
	return out, nil
}

// Build adds the arm, the pin, the handle, the pin bolt and its nut to doc, each in its container. ctx and doc must
// not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error) {
	parts, err := BuildParts(ctx, doc, p)
	if err != nil {
		return nil, err
	}
	m := &Model{Parts: *parts}
	if m.Bolt, err = m2fastener.PlaceBolt(ctx, doc, m.Site); err != nil {
		return nil, fmt.Errorf("crankhandle: %w", err)
	}
	if m.Nut, err = m2fastener.PlaceNut(ctx, doc, m.Site); err != nil {
		return nil, fmt.Errorf("crankhandle: %w", err)
	}
	for _, c := range []struct {
		name string
		body *decad.Body
	}{
		{ArmContainerName, m.Arm}, {PinContainerName, m.Pin}, {HandleContainerName, m.Handle},
		{BoltsContainerName, m.Bolt.Body}, {NutsContainerName, m.Nut.Body},
	} {
		box, err := doc.Root().ChildNamed(c.name)
		if err != nil {
			return nil, fmt.Errorf("crankhandle: container %s: %w", c.name, err)
		}
		if err := box.AddBody(c.body); err != nil {
			return nil, fmt.Errorf("crankhandle: add body to %s: %w", c.name, err)
		}
	}
	return m, nil
}

// arm extrudes the stadium, chamfers both faces' outlines, and cuts the blind D bore, the hexagonal pocket for the pin's
// stub from the outer face, and the pin bolt's head pocket in the back face.
func (r *run) arm() (*decad.Body, error) {
	s, err := r.armSketch()
	if err != nil {
		return nil, err
	}
	body, err := r.extrude(s, 1, ArmThickness)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	loops := decad.Edges(decad.OuterLoopOf(decad.CapStart(body))).Or(decad.OuterLoopOf(decad.CapEnd(body)))
	if body, err = decad.Chamfer(r.ctx, body, loops, units.Millimeters(ChamferSize)); err != nil {
		return nil, fmt.Errorf("chamfer faces: %w", err)
	}
	bore, err := r.boreTool()
	if err != nil {
		return nil, fmt.Errorf("bore tool: %w", err)
	}
	pocket, err := r.hexagon(r.p.Throw, PocketFrom, PocketFlats, PocketLength)
	if err != nil {
		return nil, fmt.Errorf("pocket tool: %w", err)
	}
	head, hs, err := m2fastener.PocketTool(r.ctx, r.doc, r.dm.Site)
	if err != nil {
		return nil, fmt.Errorf("head pocket tool: %w", err)
	}
	r.sketches = append(r.sketches, hs)
	for _, tool := range []*decad.Body{bore, pocket, head} {
		if body, err = decad.Cut(r.ctx, body, tool); err != nil {
			return nil, fmt.Errorf("cut: %w", err)
		}
	}
	// The 608 bore's rim (BoreLoop 25.3 mm) stays sharp: Decad refuses to chamfer it (steps.md, "Known decad limit").
	return body, nil
}

// pin joins the hexagonal stub, the round shank and the head, then cuts the bolt's hole and the nut's cavity.
func (r *run) pin() (*decad.Body, error) {
	y := r.p.Throw
	stub, err := r.hexagon(y, StubFrom, StubFlats, StubLength)
	if err != nil {
		return nil, err
	}
	shank, err := r.cylinder(y, ShankFrom, ShankR, ShankTo-ShankFrom)
	if err != nil {
		return nil, err
	}
	head, err := r.cylinder(y, HeadFrom, HeadR, HeadLength)
	if err != nil {
		return nil, err
	}
	body, err := decad.Union(r.ctx, stub, shank, head)
	if err != nil {
		return nil, fmt.Errorf("join: %w", err)
	}
	hole, hs, err := m2fastener.HoleTool(r.ctx, r.doc, r.dm.Site)
	if err != nil {
		return nil, err
	}
	cavity, cs, err := m2fastener.CavityTool(r.ctx, r.doc, r.dm.Site)
	if err != nil {
		return nil, err
	}
	r.sketches = append(r.sketches, hs, cs)
	for _, tool := range []*decad.Body{hole, cavity} {
		if body, err = decad.Cut(r.ctx, body, tool); err != nil {
			return nil, fmt.Errorf("cut: %w", err)
		}
	}
	return body, nil
}

// placeSite carries the local pin bolt site through the placement.
func placeSite(s m2fastener.Site, f *r3.Frame) (m2fastener.Site, error) {
	if f == nil {
		return s, nil
	}
	xf, err := r3.FromFrame(*f)
	if err != nil {
		return s, err
	}
	s.Head, s.Dir, s.Corner = xf.Apply(s.Head), xf.ApplyDir(s.Dir), xf.ApplyDir(s.Corner)
	return s, nil
}
