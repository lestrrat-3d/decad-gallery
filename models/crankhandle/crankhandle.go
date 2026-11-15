// Package crankhandle builds a printed crank for a D-flat shaft: an arm that pinches the shaft, a pin clamped in the
// arm, and a handle that turns freely on the pin, held on by the pin's head. A slit through the arm's hub and an M2
// bolt across it close the arm's D bore on the shaft; one M2 bolt and nut clamp the pin. The hand-crank siren imports
// it for its crank.
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
	// CrossSites are the two cross bolts that clamp the pin's halves, in the world frame.
	CrossSites [2]m2fastener.Site
	// ClampSite is the clamp bolt across the arm's hub, in the world frame.
	ClampSite m2fastener.Site
	// SplitNormal is the unit normal, in the world frame, of the plane through the pin axis that splits the pin into
	// its two printed halves. It runs along the arm, from the shaft toward the pin.
	SplitNormal r3.Vec
}

// Model is a built crank: its parts, the pin bolt and nut, the cross bolts and nuts that clamp the pin's halves, and
// the clamp bolt and nut that close the arm on the shaft.
type Model struct {
	Parts
	Bolt       *m2bolt.Model
	Nut        *m2nut.Model
	CrossBolts [2]*m2bolt.Model
	CrossNuts  [2]*m2nut.Model
	ClampBolt  *m2bolt.Model
	ClampNut   *m2nut.Model
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
	for i, site := range dm.CrossSites {
		if out.CrossSites[i], err = placeSite(site, p.Placement); err != nil {
			return nil, fmt.Errorf("crankhandle: place cross bolt %d: %w", i+1, err)
		}
	}
	if out.ClampSite, err = placeSite(dm.ClampSite, p.Placement); err != nil {
		return nil, fmt.Errorf("crankhandle: place clamp bolt: %w", err)
	}
	out.SplitNormal = r3.Vec{Y: 1}
	if p.Placement != nil {
		xf, err := r3.FromFrame(*p.Placement)
		if err != nil {
			return nil, fmt.Errorf("crankhandle: place split plane: %w", err)
		}
		out.SplitNormal = xf.ApplyDir(out.SplitNormal)
	}
	return out, nil
}

// Build adds the arm, the pin, the handle, and every bolt and nut to doc, each in its container. ctx and doc must not
// be nil.
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
	type filed struct {
		name string
		body *decad.Body
	}
	bodies := []filed{
		{ArmContainerName, m.Arm}, {PinContainerName, m.Pin}, {HandleContainerName, m.Handle},
		{BoltsContainerName, m.Bolt.Body}, {NutsContainerName, m.Nut.Body},
	}
	for i, site := range m.CrossSites {
		if m.CrossBolts[i], err = m2fastener.PlaceBolt(ctx, doc, site); err != nil {
			return nil, fmt.Errorf("crankhandle: cross bolt %d: %w", i+1, err)
		}
		if m.CrossNuts[i], err = m2fastener.PlaceNut(ctx, doc, site); err != nil {
			return nil, fmt.Errorf("crankhandle: cross nut %d: %w", i+1, err)
		}
		bodies = append(bodies, filed{BoltsContainerName, m.CrossBolts[i].Body}, filed{NutsContainerName, m.CrossNuts[i].Body})
	}
	if m.ClampBolt, err = m2fastener.PlaceBolt(ctx, doc, m.ClampSite); err != nil {
		return nil, fmt.Errorf("crankhandle: clamp bolt: %w", err)
	}
	if m.ClampNut, err = m2fastener.PlaceNut(ctx, doc, m.ClampSite); err != nil {
		return nil, fmt.Errorf("crankhandle: clamp nut: %w", err)
	}
	bodies = append(bodies, filed{BoltsContainerName, m.ClampBolt.Body}, filed{NutsContainerName, m.ClampNut.Body})
	for _, c := range bodies {
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

// arm extrudes the arm's outline, chamfers both faces' outlines, and cuts, in this order: the slit, the blind D bore,
// the hexagonal pocket for the pin's stub from the outer face, the pin bolt's head pocket in the back face, and the
// clamp bolt's hexagonal nut pocket in the -X side face, head pocket and hole. The slit goes first: the bore cut on the
// chamfered body holds a coarser mesh bound round its rim than the slit's cut admits through the document's chain
// tolerance, while the bore's cut admits the slit's finer rim. The nut pocket's floor is rounded on its tool
// (NutPocketRound).
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
	slit, err := r.slitTool()
	if err != nil {
		return nil, fmt.Errorf("slit tool: %w", err)
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
	clamp, err := r.clampTools()
	if err != nil {
		return nil, err
	}
	for _, tool := range append([]*decad.Body{slit, bore, pocket, head}, clamp...) {
		if body, err = decad.Cut(r.ctx, body, tool); err != nil {
			return nil, fmt.Errorf("cut: %w", err)
		}
	}
	// The 608 bore's rim (BoreLoop 25.3 mm) stays sharp: Decad refuses to chamfer it (steps.md, "Known decad limit").
	return body, nil
}

// nutPocketCutter builds a site's hexagonal nut pocket tool with its floor's outline rounded by NutPocketRound.
func (r *run) nutPocketCutter(site m2fastener.Site) (*decad.Body, error) {
	pocket, err := r.nutPocketTool(site)
	if err != nil {
		return nil, err
	}
	floor := decad.Edges(decad.OuterLoopOf(decad.CapStart(pocket)))
	if pocket, err = decad.Fillet(r.ctx, pocket, floor, units.Millimeters(NutPocketRound)); err != nil {
		return nil, fmt.Errorf("round floor: %w", err)
	}
	return pocket, nil
}

// clampTools builds the clamp bolt's cutters: the hexagonal nut pocket with its floor's outline rounded by
// NutPocketRound, the head pocket, and the clearance hole, which runs out through the nut pocket.
func (r *run) clampTools() ([]*decad.Body, error) {
	site := r.dm.ClampSite
	pocket, err := r.nutPocketCutter(site)
	if err != nil {
		return nil, fmt.Errorf("clamp nut pocket tool: %w", err)
	}
	head, hs, err := m2fastener.PocketTool(r.ctx, r.doc, site)
	if err != nil {
		return nil, fmt.Errorf("clamp head pocket tool: %w", err)
	}
	hole, ls, err := m2fastener.HoleTool(r.ctx, r.doc, site)
	if err != nil {
		return nil, fmt.Errorf("clamp hole tool: %w", err)
	}
	r.sketches = append(r.sketches, hs, ls)
	return []*decad.Body{pocket, head, hole}, nil
}

// pin joins the hexagonal stub to the turned shank and head, then cuts the double flats, the pin bolt's rounded hole
// and nut cavity, and each cross bolt's hole, head pocket and hexagonal nut pocket open to the -Y flat.
func (r *run) pin() (*decad.Body, error) {
	y := r.p.Throw
	stub, err := r.hexagon(y, StubFrom, StubFlats, StubLength)
	if err != nil {
		return nil, err
	}
	turned, err := r.turnedPin(y)
	if err != nil {
		return nil, fmt.Errorf("turned shank: %w", err)
	}
	body, err := decad.Union(r.ctx, stub, turned)
	if err != nil {
		return nil, fmt.Errorf("join: %w", err)
	}
	var tools []*decad.Body
	for _, z := range CrossZ {
		for _, side := range []float64{1, -1} {
			flat, err := r.flatTool(y, z, side)
			if err != nil {
				return nil, fmt.Errorf("flat tool: %w", err)
			}
			tools = append(tools, flat)
		}
	}
	hole, hs, err := m2fastener.RoundedHoleTool(r.ctx, r.doc, r.dm.Site)
	if err != nil {
		return nil, err
	}
	cavity, cs, err := m2fastener.RoundedCavityTool(r.ctx, r.doc, r.dm.Site)
	if err != nil {
		return nil, err
	}
	r.sketches = append(r.sketches, hs, cs)
	tools = append(tools, hole, cavity)
	for _, site := range r.dm.CrossSites {
		for _, build := range []func(context.Context, *decad.Document, m2fastener.Site) (*decad.Body, *sketch.Sketch, error){
			m2fastener.HoleTool, m2fastener.PocketTool,
		} {
			tool, sk, err := build(r.ctx, r.doc, site)
			if err != nil {
				return nil, err
			}
			r.sketches = append(r.sketches, sk)
			tools = append(tools, tool)
		}
		pocket, err := r.nutPocketCutter(site)
		if err != nil {
			return nil, fmt.Errorf("cross nut pocket tool: %w", err)
		}
		tools = append(tools, pocket)
	}
	for _, tool := range tools {
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
