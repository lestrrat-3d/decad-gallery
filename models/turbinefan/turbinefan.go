// Package turbinefan builds a fan of three replaceable blades on one printed hub. The hub pinches a D-flat shaft the way
// the crank handle's arm does: a blind D bore, a 1 mm slit from the bore out through a clamp tab and one M2 bolt across
// the tab. Each blade is the hand-crank fan's blade; its round root stud slides into a radial socket in the hub and one
// M2 bolt, parallel to the shaft, holds it.
//
// The local frame has its origin on the shaft axis in the hub's back face, the face toward the mechanism. +Z runs along
// the shaft axis away from the mechanism to the front face at z = HubLength, and the bore's flat faces +Y. Blade 1 lies
// along +Y, blades 2 and 3 at 210 and 330 degrees; the slit and the clamp tab point to -Y. Seen from the front, a right-hand
// fan turns clockwise and a left-hand fan counter-clockwise; either blows air forward, away from the mechanism.
package turbinefan

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Container names, one per kind of body.
const (
	HubContainerName    = "Hub"
	BladesContainerName = "Blades"
	BoltsContainerName  = "Bolts"
	NutsContainerName   = "Nuts"
)

// Parts are the fan's printed bodies, placed, without the bolts and nuts.
type Parts struct {
	Hub    *decad.Body
	Blades [BladeCount]*decad.Body
	// Sketches are every sketch the parts were built from, each fully constrained.
	Sketches   []*sketch.Sketch
	Dimensions Dimensions
	// BladeSites are the three blade bolts in the world frame; ClampSite is the clamp bolt in the world frame.
	BladeSites [BladeCount]m2fastener.Site
	ClampSite  m2fastener.Site
}

// Model is a built fan: its parts, the three blade bolts and nuts, and the clamp bolt and nut.
type Model struct {
	Parts
	BladeBolts [BladeCount]*m2bolt.Model
	BladeNuts  [BladeCount]*m2nut.Model
	ClampBolt  *m2bolt.Model
	ClampNut   *m2nut.Model
}

// BuildParts adds the hub and the three blades to doc and returns them. It puts them in no container. ctx and doc must
// not be nil.
func BuildParts(ctx context.Context, doc *decad.Document, p Params) (*Parts, error) {
	if ctx == nil {
		return nil, errors.New("turbinefan: context must not be nil")
	}
	if doc == nil {
		return nil, errors.New("turbinefan: document must not be nil")
	}
	dm, err := p.Derive()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("turbinefan: before creating geometry: %w", err)
	}
	r := &run{ctx: ctx, doc: doc, p: p, dm: dm}
	out := &Parts{Dimensions: dm}
	hub, err := r.hub()
	if err != nil {
		return nil, fmt.Errorf("turbinefan: hub: %w", err)
	}
	blades, err := r.blades()
	if err != nil {
		return nil, fmt.Errorf("turbinefan: blades: %w", err)
	}
	for i, body := range append([]*decad.Body{hub}, blades...) {
		name := "hub"
		if i > 0 {
			name = fmt.Sprintf("blade %d", i)
		}
		if body == nil || !body.IsSolid() {
			return nil, fmt.Errorf("turbinefan: %s is not a solid", name)
		}
		placed, err := placement.Place(ctx, body, p.Placement)
		if err != nil {
			return nil, fmt.Errorf("turbinefan: place %s: %w", name, err)
		}
		if i == 0 {
			out.Hub = placed
			continue
		}
		out.Blades[i-1] = placed
	}
	out.Sketches = r.sketches
	if out.ClampSite, err = placeSite(dm.ClampSite, p.Placement); err != nil {
		return nil, fmt.Errorf("turbinefan: place clamp bolt: %w", err)
	}
	for i, site := range dm.BladeSites {
		if out.BladeSites[i], err = placeSite(site, p.Placement); err != nil {
			return nil, fmt.Errorf("turbinefan: place blade bolt %d: %w", i+1, err)
		}
	}
	return out, nil
}

// Build adds the hub, the blades, and every bolt and nut to doc, each in its container. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error) {
	parts, err := BuildParts(ctx, doc, p)
	if err != nil {
		return nil, err
	}
	m := &Model{Parts: *parts}
	type filed struct {
		name string
		body *decad.Body
	}
	bodies := []filed{{HubContainerName, m.Hub}}
	for _, blade := range m.Blades {
		bodies = append(bodies, filed{BladesContainerName, blade})
	}
	for i, site := range m.BladeSites {
		if m.BladeBolts[i], err = m2fastener.PlaceBolt(ctx, doc, site); err != nil {
			return nil, fmt.Errorf("turbinefan: blade bolt %d: %w", i+1, err)
		}
		if m.BladeNuts[i], err = m2fastener.PlaceNut(ctx, doc, site); err != nil {
			return nil, fmt.Errorf("turbinefan: blade nut %d: %w", i+1, err)
		}
		bodies = append(bodies, filed{BoltsContainerName, m.BladeBolts[i].Body}, filed{NutsContainerName, m.BladeNuts[i].Body})
	}
	if m.ClampBolt, err = m2fastener.PlaceBolt(ctx, doc, m.ClampSite); err != nil {
		return nil, fmt.Errorf("turbinefan: clamp bolt: %w", err)
	}
	if m.ClampNut, err = m2fastener.PlaceNut(ctx, doc, m.ClampSite); err != nil {
		return nil, fmt.Errorf("turbinefan: clamp nut: %w", err)
	}
	bodies = append(bodies, filed{BoltsContainerName, m.ClampBolt.Body}, filed{NutsContainerName, m.ClampNut.Body})
	for _, c := range bodies {
		box, err := doc.Root().ChildNamed(c.name)
		if err != nil {
			return nil, fmt.Errorf("turbinefan: container %s: %w", c.name, err)
		}
		if err := box.AddBody(c.body); err != nil {
			return nil, fmt.Errorf("turbinefan: add body to %s: %w", c.name, err)
		}
	}
	return m, nil
}

// hub extrudes the outline along +Z, chamfers both faces' outlines, and cuts, in this order: the slit, the blind D
// bore, the three blade sockets, each blade bolt's head pocket and nut cavity, and the clamp bolt's hexagonal nut
// pocket in the -X side face, head pocket and hole. The blade bolts' holes are in the outline.
func (r *run) hub() (*decad.Body, error) {
	s, err := r.hubSketch()
	if err != nil {
		return nil, err
	}
	body, err := r.extrudeOutline(s, BladeCount, HubLength)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	loops := decad.Edges(decad.OuterLoopOf(decad.CapStart(body))).Or(decad.OuterLoopOf(decad.CapEnd(body)))
	if body, err = decad.Chamfer(r.ctx, body, loops, units.Millimeters(ChamferSize)); err != nil {
		return nil, fmt.Errorf("chamfer faces: %w", err)
	}
	var tools []*decad.Body
	add := func(tool *decad.Body, s *sketch.Sketch, err error, what string) error {
		if err != nil {
			return fmt.Errorf("%s: %w", what, err)
		}
		r.sketches = append(r.sketches, s)
		tools = append(tools, tool)
		return nil
	}
	below := across(-1)
	slit, ss, err := crankhandle.SlitTool(r.ctx, r.doc, below, -r.dm.Reach-2, r.dm.BoreRadius, HubLength+2)
	if err := add(slit, ss, err, "slit tool"); err != nil {
		return nil, err
	}
	bore, bs, err := crankhandle.BoreTool(r.ctx, r.doc, below, r.dm.BoreRadius, r.dm.BoreFlat, 1+HubLength-BoreFloor)
	if err := add(bore, bs, err, "bore tool"); err != nil {
		return nil, err
	}
	socketFrame, err := r3.NewFrame(r3.Vec{Y: handcrankfan.BladeSocketStart, Z: HubMid}, r3.Vec{Z: 1}, r3.Vec{X: 1})
	if err != nil {
		return nil, fmt.Errorf("socket frame: %w", err)
	}
	socket, ks, err := handcrankfan.BladeSocketTool(r.ctx, r.doc, socketFrame)
	if err != nil {
		return nil, fmt.Errorf("socket tool: %w", err)
	}
	r.sketches = append(r.sketches, ks)
	sockets, err := r.ring(socket)
	if err != nil {
		return nil, fmt.Errorf("socket copies: %w", err)
	}
	tools = append(tools, sockets...)
	for k, site := range r.dm.BladeSites {
		pocket, ps, err := m2fastener.PocketTool(r.ctx, r.doc, site)
		if err := add(pocket, ps, err, fmt.Sprintf("blade %d head pocket tool", k+1)); err != nil {
			return nil, err
		}
		cavity, cs, err := m2fastener.CavityTool(r.ctx, r.doc, site)
		if err := add(cavity, cs, err, fmt.Sprintf("blade %d nut cavity tool", k+1)); err != nil {
			return nil, err
		}
	}
	nut, ns, err := crankhandle.NutPocketTool(r.ctx, r.doc, r.dm.ClampSite)
	if err := add(nut, ns, err, "clamp nut pocket tool"); err != nil {
		return nil, err
	}
	head, hs, err := m2fastener.PocketTool(r.ctx, r.doc, r.dm.ClampSite)
	if err := add(head, hs, err, "clamp head pocket tool"); err != nil {
		return nil, err
	}
	hole, ls, err := m2fastener.HoleTool(r.ctx, r.doc, r.dm.ClampSite)
	if err := add(hole, ls, err, "clamp hole tool"); err != nil {
		return nil, err
	}
	for _, tool := range tools {
		if body, err = decad.Cut(r.ctx, body, tool); err != nil {
			return nil, fmt.Errorf("cut: %w", err)
		}
	}
	return body, nil
}

// ring returns body and its two copies about the shaft axis, 120 and 240 degrees on.
func (r *run) ring(body *decad.Body) ([]*decad.Body, error) {
	copies, err := decad.PatternCopies(r.ctx, body, decad.CircularPattern{Axis: r3.Vec{Z: 1}, Count: BladeCount})
	if err != nil {
		return nil, err
	}
	return append([]*decad.Body{body}, copies...), nil
}

// blades builds blade 1 (handcrankfan.BuildBlade), turns it into place along +Y in the hub's mid plane and copies it
// about the shaft axis. Its Sketches join the run's.
func (r *run) blades() ([]*decad.Body, error) {
	build := handcrankfan.BuildBlade
	if r.p.Hand == LeftHand {
		build = handcrankfan.BuildBladeMirrored
	}
	blade, err := build(r.ctx, r.doc, r.p.BladePitch)
	if err != nil {
		return nil, err
	}
	r.sketches = append(r.sketches, blade.Sketches...)
	// Blade-local Z (the radial line) becomes +Y, local Y (downstream) becomes +Z and local X becomes -X.
	f, err := r3.NewFrame(r3.Vec{Z: HubMid}, r3.Vec{X: -1}, r3.Vec{Z: 1})
	if err != nil {
		return nil, fmt.Errorf("blade frame: %w", err)
	}
	xf, err := r3.FromFrame(f)
	if err != nil {
		return nil, fmt.Errorf("blade frame: %w", err)
	}
	body, err := decad.Transform(r.ctx, blade.Body, xf)
	if err != nil {
		return nil, fmt.Errorf("place blade: %w", err)
	}
	return r.ring(body)
}

// placeSite carries a local site through the placement.
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
