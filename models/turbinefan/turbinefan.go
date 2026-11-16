// Package turbinefan builds a fan of three replaceable blades on one printed hub. The hub is a plain round cylinder with a
// D bore for a D-flat shaft. One radial M2 x 6 bolt acts as a set screw: it threads through a nut in a hex slot open to the
// back face, its tip presses the shaft's flat and its head floats, recessed, in a counterbore in the round face. Each blade
// is this model's blade; its round root stud slides into a radial socket in the hub and one M2 bolt, parallel to the
// shaft, holds it in a nut that sits in a hex slot open to the round face.
//
// The hub has two M2 x 10 bolt holes in its back face at 30 and 150 degrees, each with a hex nut slot open to the round
// face. They are cut in both modes. With Params.Mount = ClutchMount the hub does not grip the shaft: the adapter ring of
// a onewayclutch (onewayclutch.BuildAdapter, built by the caller in the clutch's document) sits against the back face,
// one seam behind it, and two M2 x 10 bolts from the adapter's rear face hold the hub in those nuts; the set screw is
// left out. The fan frame is turned 60 degrees about the axis from the clutch frame. The shaft drives the fan through the
// clutch, and the fan coasts when the shaft stops or turns back.
//
// The local frame has its origin on the shaft axis in the hub's back face, the face toward the mechanism. +Z runs along
// the shaft axis away from the mechanism to the front face at z = HubLength, and the bore's flat faces -Y, where the set
// screw enters. Blade 1 lies along +Y, blades 2 and 3 at 210 and 330 degrees. Seen from the front, a right-hand fan turns
// clockwise and a left-hand fan counter-clockwise; either blows air forward, away from the mechanism.
package turbinefan

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
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
	// BladeSites are the three blade bolts in the world frame; SetScrewSite is the set screw in the world frame (the hub
	// has its cuts in both modes) and AdapterSites are the two adapter bolts in the world frame.
	BladeSites   [BladeCount]m2fastener.Site
	SetScrewSite m2fastener.Site
	AdapterSites [2]m2fastener.Site
}

// Model is a built fan: its parts, the three blade bolts and nuts, and the set screw and its nut (ShaftMount) or the
// two adapter bolts and nuts (ClutchMount).
type Model struct {
	Parts
	BladeBolts  [BladeCount]*m2bolt.Model
	BladeNuts   [BladeCount]*m2nut.Model
	SetScrew    *m2bolt.Model
	SetScrewNut *m2nut.Model
	// AdapterBolts and AdapterNuts are the clutch mount's adapter bolts and nuts; nil in ShaftMount, where SetScrew and
	// SetScrewNut are set instead.
	AdapterBolts [2]*m2bolt.Model
	AdapterNuts  [2]*m2nut.Model
}

// BuildParts adds the hub and the three blades to doc and returns them. It puts them in no container. ctx and doc must not be nil.
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
	bodies := append([]*decad.Body{hub}, blades...)
	for i, body := range bodies {
		name := fmt.Sprintf("blade %d", i)
		if i == 0 {
			name = "hub"
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
	if out.SetScrewSite, err = placeSite(dm.SetScrewSite, p.Placement); err != nil {
		return nil, fmt.Errorf("turbinefan: place set screw: %w", err)
	}
	for i, site := range dm.BladeSites {
		if out.BladeSites[i], err = placeSite(site, p.Placement); err != nil {
			return nil, fmt.Errorf("turbinefan: place blade bolt %d: %w", i+1, err)
		}
	}
	for i, site := range dm.AdapterSites {
		if out.AdapterSites[i], err = placeSite(site, p.Placement); err != nil {
			return nil, fmt.Errorf("turbinefan: place adapter bolt %d: %w", i+1, err)
		}
	}
	return out, nil
}

// Build adds the hub, the blades, and every bolt and nut to doc, each in its container: the set screw in ShaftMount, the
// two adapter bolts in ClutchMount. It does not build the adapter or the clutch. ctx and doc must not be nil.
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
	if p.Mount == ClutchMount {
		for i, site := range m.AdapterSites {
			if m.AdapterBolts[i], err = m2fastener.PlaceBolt(ctx, doc, site); err != nil {
				return nil, fmt.Errorf("turbinefan: adapter bolt %d: %w", i+1, err)
			}
			if m.AdapterNuts[i], err = m2fastener.PlaceNut(ctx, doc, site); err != nil {
				return nil, fmt.Errorf("turbinefan: adapter nut %d: %w", i+1, err)
			}
			bodies = append(bodies, filed{BoltsContainerName, m.AdapterBolts[i].Body}, filed{NutsContainerName, m.AdapterNuts[i].Body})
		}
	} else {
		if m.SetScrew, err = m2fastener.PlaceBolt(ctx, doc, m.SetScrewSite); err != nil {
			return nil, fmt.Errorf("turbinefan: set screw: %w", err)
		}
		if m.SetScrewNut, err = m2fastener.PlaceNut(ctx, doc, m.SetScrewSite); err != nil {
			return nil, fmt.Errorf("turbinefan: set screw nut: %w", err)
		}
		bodies = append(bodies, filed{BoltsContainerName, m.SetScrew.Body}, filed{NutsContainerName, m.SetScrewNut.Body})
	}
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

// hub extrudes the outline along +Z, chamfers both faces' long loops, and cuts, in this order: the three sockets, each
// blade's head pocket and nut slot, then the set screw's counterbore, hole and nut slot, then each adapter bolt's hole
// and nut slot. The blade bolts' holes and the D bore are in the outline.
func (r *run) hub() (*decad.Body, error) {
	s, err := r.hubSketch()
	if err != nil {
		return nil, err
	}
	body, err := r.extrudeOutline(s, BladeCount+1, HubLength)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	// Both faces' long loops go in one call: the outer circle, and the D loop at the 8 mm shaft.
	if body, _, err = edgefinish.Chamfer(r.ctx, body, r3.Vec{Z: 1}, []float64{0, HubLength}, edgefinish.Long(edgefinish.All),
		ChamferSize); err != nil {
		return nil, fmt.Errorf("chamfer faces: %w", err)
	}
	tools, err := r.socketTools()
	if err != nil {
		return nil, err
	}
	for k, site := range r.dm.BladeSites {
		pocket, ps, err := m2fastener.PocketTool(r.ctx, r.doc, site)
		if err != nil {
			return nil, fmt.Errorf("blade %d head pocket tool: %w", k+1, err)
		}
		r.sketches = append(r.sketches, ps)
		slot, err := r.bladeSlotTool(k + 1)
		if err != nil {
			return nil, fmt.Errorf("blade %d nut slot tool: %w", k+1, err)
		}
		tools = append(tools, pocket, slot)
	}
	counterbore, err := r.counterboreTool()
	if err != nil {
		return nil, fmt.Errorf("counterbore tool: %w", err)
	}
	hole, hs, err := m2fastener.HoleTool(r.ctx, r.doc, r.dm.SetScrewSite)
	if err != nil {
		return nil, fmt.Errorf("set screw hole tool: %w", err)
	}
	r.sketches = append(r.sketches, hs)
	slot, err := r.setScrewSlotTool()
	if err != nil {
		return nil, fmt.Errorf("set screw nut slot tool: %w", err)
	}
	tools = append(tools, counterbore, hole, slot)
	for k, site := range r.dm.AdapterSites {
		ah, ahs, err := m2fastener.HoleTool(r.ctx, r.doc, site)
		if err != nil {
			return nil, fmt.Errorf("adapter bolt %d hole tool: %w", k+1, err)
		}
		r.sketches = append(r.sketches, ahs)
		as, err := r.slotTool(site, site.Corner, HubRadius+m2fastener.Stock-onewayclutch.FaceCircle)
		if err != nil {
			return nil, fmt.Errorf("adapter bolt %d nut slot tool: %w", k+1, err)
		}
		tools = append(tools, ah, as)
	}
	for _, tool := range tools {
		if body, err = decad.Cut(r.ctx, body, tool); err != nil {
			return nil, fmt.Errorf("cut: %w", err)
		}
	}
	return body, nil
}

// socketTools returns the three blade sockets' cutters and records the sketch.
func (r *run) socketTools() ([]*decad.Body, error) {
	socketFrame, err := r3.NewFrame(r3.Vec{Y: BladeSocketStart, Z: HubMid}, r3.Vec{Z: 1}, r3.Vec{X: 1})
	if err != nil {
		return nil, fmt.Errorf("socket frame: %w", err)
	}
	socket, ks, err := BladeSocketTool(r.ctx, r.doc, socketFrame)
	if err != nil {
		return nil, fmt.Errorf("socket tool: %w", err)
	}
	r.sketches = append(r.sketches, ks)
	sockets, err := r.ring(socket)
	if err != nil {
		return nil, fmt.Errorf("socket copies: %w", err)
	}
	return sockets, nil
}

// bladeSlotTool is blade k's nut slot: slotTool along the blade's radial line out through the hub's rim.
func (r *run) bladeSlotTool(k int) (*decad.Body, error) {
	u := r3.Vec{X: math.Cos(BladeAngle(k)), Y: math.Sin(BladeAngle(k))}
	return r.slotTool(r.dm.BladeSites[k-1], u, HubRadius+m2fastener.Stock-BladeBoltAt)
}

// ring returns body and its two copies about the shaft axis, 120 and 240 degrees on.
func (r *run) ring(body *decad.Body) ([]*decad.Body, error) {
	copies, err := decad.PatternCopies(r.ctx, body, decad.CircularPattern{Axis: r3.Vec{Z: 1}, Count: BladeCount})
	if err != nil {
		return nil, err
	}
	return append([]*decad.Body{body}, copies...), nil
}

// blades builds blade 1 (BuildBlade), turns it into place along +Y in the hub's mid plane and copies it
// about the shaft axis. Its Sketches join the run's.
func (r *run) blades() ([]*decad.Body, error) {
	build := BuildBlade
	if r.p.Hand == LeftHand {
		build = BuildBladeMirrored
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
