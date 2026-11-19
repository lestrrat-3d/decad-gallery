// Package turbinefan builds a fan with three replaceable blades for rotor diameters from 128 to 224 mm, with a 192 mm
// default. The hub radius caps at 18.9 mm to leave access to clutch ear bolts. Its D bore fits a shaft, and a radial
// set screw seats in a nut held by an open slot. Axial M2 bolts retain the blades' radial studs; at the default size,
// the set screw is M2 × 10 and the blade bolts are M2 × 25. Two common adapter holes at 30° and 150° take M2 × 10
// bolts for the clutch mount, whose fan frame turns 180° from the clutch frame. The caller builds the adapter
// and clutch separately.
//
// The local frame has its origin on the shaft at the hub's back face, with +Z toward the front, -Y toward the shaft
// flat, and the first blade along +Y. The other blades lie at 210° and 330°. Viewed from the front, a right-hand
// fan turns clockwise and a left-hand fan counterclockwise to blow forward.
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
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

const (
	// HubContainerName is the body container name for the hub.
	HubContainerName = "Hub"
	// BladesContainerName is the body container name for the blades.
	BladesContainerName = "Blades"
	// BoltsContainerName is the body container name for bolts.
	BoltsContainerName = "Bolts"
	// NutsContainerName is the body container name for nuts.
	NutsContainerName = "Nuts"
)

// Parts contains the placed printed hub and three blades.
type Parts struct {
	// Hub is the placed printed hub body.
	Hub *decad.Body
	// Blades are the three placed blade bodies in blade order.
	Blades [BladeCount]*decad.Body
	// Sketches are the 29 fully constrained construction sketches for the hub and blades.
	Sketches []*sketch.Sketch
	// Dimensions contains the derived dimensions in the local frame.
	Dimensions Dimensions
	// BladeSites are the blade fastener sites in the world frame, in both mounts.
	BladeSites [BladeCount]m2fastener.Site
	// SetScrewSite is the set-screw site in the world frame, in both mounts.
	SetScrewSite m2fastener.Site
	// AdapterSites are the two adapter fastener sites in the world frame, in both mounts.
	AdapterSites [2]m2fastener.Site
}

// Model embeds Parts and adds the three blade bolt and nut pairs, plus mount-specific hardware.
// Fastener models own their construction sketches.
type Model struct {
	Parts
	// BladeBolts are the three axial bolts that retain the blade studs.
	BladeBolts [BladeCount]*m2bolt.Model
	// BladeNuts are the nuts retained in the hub's open blade slots.
	BladeNuts [BladeCount]*m2nut.Model
	// SetScrew and SetScrewNut are present in ShaftMount and nil in ClutchMount.
	SetScrew    *m2bolt.Model
	SetScrewNut *m2nut.Model
	// AdapterBolts and AdapterNuts are present in ClutchMount and nil in ShaftMount.
	AdapterBolts [2]*m2bolt.Model
	AdapterNuts  [2]*m2nut.Model
}

// BuildParts adds and returns the placed hub and three blades. The caller controls their containers.
// ctx and doc must not be nil.
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
	frame, err := p.buildFrame(dm.Size)
	if err != nil {
		return nil, fmt.Errorf("turbinefan: placement frame: %w", err)
	}
	r := &run{ctx: ctx, doc: doc, p: p, dm: dm}
	hub, err := r.hub()
	if err != nil {
		return nil, fmt.Errorf("turbinefan: hub: %w", err)
	}
	blades, err := r.blades()
	if err != nil {
		return nil, fmt.Errorf("turbinefan: blades: %w", err)
	}
	if !hub.IsSolid() {
		return nil, errors.New("turbinefan: hub is not a solid")
	}
	placedHub, err := placement.Place(ctx, hub, frame)
	if err != nil {
		return nil, fmt.Errorf("turbinefan: place hub: %w", err)
	}
	out := &Parts{Hub: placedHub, Sketches: r.sketches, Dimensions: dm}
	for i, body := range blades {
		if !body.IsSolid() {
			return nil, fmt.Errorf("turbinefan: blade %d is not a solid", i+1)
		}
		if out.Blades[i], err = placement.Place(ctx, body, frame); err != nil {
			return nil, fmt.Errorf("turbinefan: place blade %d: %w", i+1, err)
		}
	}
	for i, site := range dm.BladeSites {
		if out.BladeSites[i], err = placeSite(site, frame); err != nil {
			return nil, fmt.Errorf("turbinefan: place blade site %d: %w", i+1, err)
		}
	}
	if out.SetScrewSite, err = placeSite(dm.SetScrewSite, frame); err != nil {
		return nil, fmt.Errorf("turbinefan: place set screw site: %w", err)
	}
	for i, site := range dm.AdapterSites {
		if out.AdapterSites[i], err = placeSite(site, frame); err != nil {
			return nil, fmt.Errorf("turbinefan: place adapter site %d: %w", i+1, err)
		}
	}
	return out, nil
}

// Build files the complete fan, including mount-specific hardware, under the four named body containers.
// The caller builds the clutch and adapter separately. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error) {
	parts, err := BuildParts(ctx, doc, p)
	if err != nil {
		return nil, err
	}
	m := &Model{Parts: *parts}
	for i, site := range m.BladeSites {
		if m.BladeBolts[i], err = m2fastener.PlaceBolt(ctx, doc, site); err != nil {
			return nil, fmt.Errorf("turbinefan: blade bolt %d: %w", i+1, err)
		}
		if m.BladeNuts[i], err = m2fastener.PlaceNut(ctx, doc, site); err != nil {
			return nil, fmt.Errorf("turbinefan: blade nut %d: %w", i+1, err)
		}
	}
	if p.Mount == ShaftMount {
		if m.SetScrew, err = m2fastener.PlaceBolt(ctx, doc, m.SetScrewSite); err != nil {
			return nil, fmt.Errorf("turbinefan: set screw: %w", err)
		}
		if m.SetScrewNut, err = m2fastener.PlaceNut(ctx, doc, m.SetScrewSite); err != nil {
			return nil, fmt.Errorf("turbinefan: set screw nut: %w", err)
		}
	} else {
		for i, site := range m.AdapterSites {
			if m.AdapterBolts[i], err = m2fastener.PlaceBolt(ctx, doc, site); err != nil {
				return nil, fmt.Errorf("turbinefan: adapter bolt %d: %w", i+1, err)
			}
			if m.AdapterNuts[i], err = m2fastener.PlaceNut(ctx, doc, site); err != nil {
				return nil, fmt.Errorf("turbinefan: adapter nut %d: %w", i+1, err)
			}
		}
	}
	bodies := []struct {
		name string
		body *decad.Body
	}{
		{HubContainerName, m.Hub},
	}
	for i := range m.Blades {
		bodies = append(bodies, struct {
			name string
			body *decad.Body
		}{BladesContainerName, m.Blades[i]})
	}
	for i := range m.BladeBolts {
		bodies = append(bodies, struct {
			name string
			body *decad.Body
		}{BoltsContainerName, m.BladeBolts[i].Body})
	}
	for i := range m.BladeNuts {
		bodies = append(bodies, struct {
			name string
			body *decad.Body
		}{NutsContainerName, m.BladeNuts[i].Body})
	}
	if p.Mount == ShaftMount {
		bodies = append(bodies,
			struct {
				name string
				body *decad.Body
			}{BoltsContainerName, m.SetScrew.Body},
			struct {
				name string
				body *decad.Body
			}{NutsContainerName, m.SetScrewNut.Body})
	} else {
		for i := range m.AdapterBolts {
			bodies = append(bodies,
				struct {
					name string
					body *decad.Body
				}{BoltsContainerName, m.AdapterBolts[i].Body},
				struct {
					name string
					body *decad.Body
				}{NutsContainerName, m.AdapterNuts[i].Body})
		}
	}
	for _, item := range bodies {
		container, err := doc.Root().ChildNamed(item.name)
		if err != nil {
			return nil, fmt.Errorf("turbinefan: container %s: %w", item.name, err)
		}
		if err := container.AddBody(item.body); err != nil {
			return nil, fmt.Errorf("turbinefan: add body to %s: %w", item.name, err)
		}
	}
	return m, nil
}

func (r *run) hub() (*decad.Body, error) {
	s, err := r.hubSketch()
	if err != nil {
		return nil, fmt.Errorf("outline sketch: %w", err)
	}
	body, err := r.extrudeOutline(s, 4, r.dm.Size.HubLength)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	body, _, err = edgefinish.Chamfer(r.ctx, body, r3.NewVec(0, 0, 1),
		[]float64{0, r.dm.Size.HubLength}, edgefinish.Long(edgefinish.All), ChamferSize)
	if err != nil {
		return nil, fmt.Errorf("face chamfers: %w", err)
	}
	cut := func(label string, tool *decad.Body) error {
		var cutErr error
		body, cutErr = decad.Cut(r.ctx, body, tool)
		if cutErr != nil {
			return fmt.Errorf("%s: %w", label, cutErr)
		}
		return nil
	}
	tools, err := r.socketTools()
	if err != nil {
		return nil, fmt.Errorf("socket tools: %w", err)
	}
	for i, tool := range tools {
		if err := cut(fmt.Sprintf("socket %d", i+1), tool); err != nil {
			return nil, err
		}
	}
	for k, site := range r.dm.BladeSites {
		tool, sk, err := m2fastener.PocketTool(r.ctx, r.doc, site)
		if err != nil {
			return nil, fmt.Errorf("blade pocket %d: %w", k+1, err)
		}
		r.sketches = append(r.sketches, sk)
		if err := cut(fmt.Sprintf("blade pocket %d", k+1), tool); err != nil {
			return nil, err
		}
		slot, err := r.bladeSlotTool(k + 1)
		if err != nil {
			return nil, fmt.Errorf("blade nut slot %d: %w", k+1, err)
		}
		if err := cut(fmt.Sprintf("blade nut slot %d", k+1), slot); err != nil {
			return nil, err
		}
	}
	tool, err := r.counterboreTool()
	if err != nil {
		return nil, fmt.Errorf("counterbore: %w", err)
	}
	if err := cut("counterbore", tool); err != nil {
		return nil, err
	}
	tool, sk, err := m2fastener.HoleTool(r.ctx, r.doc, r.dm.SetScrewSite)
	if err != nil {
		return nil, fmt.Errorf("set screw hole: %w", err)
	}
	r.sketches = append(r.sketches, sk)
	if err := cut("set screw hole", tool); err != nil {
		return nil, err
	}
	tool, err = r.setScrewSlotTool()
	if err != nil {
		return nil, fmt.Errorf("set screw nut slot: %w", err)
	}
	if err := cut("set screw nut slot", tool); err != nil {
		return nil, err
	}
	for k, site := range r.dm.AdapterSites {
		tool, sk, err := m2fastener.HoleTool(r.ctx, r.doc, site)
		if err != nil {
			return nil, fmt.Errorf("adapter hole %d: %w", k+1, err)
		}
		r.sketches = append(r.sketches, sk)
		if err := cut(fmt.Sprintf("adapter hole %d", k+1), tool); err != nil {
			return nil, err
		}
		slot, err := r.slotTool(site, site.Corner, r.dm.Size.HubRadius+1-9.5)
		if err != nil {
			return nil, fmt.Errorf("adapter nut slot %d: %w", k+1, err)
		}
		if err := cut(fmt.Sprintf("adapter nut slot %d", k+1), slot); err != nil {
			return nil, err
		}
	}
	if !body.IsSolid() {
		return nil, errors.New("hub result is not a solid")
	}
	return body, nil
}

func (r *run) socketTools() ([]*decad.Body, error) {
	frame, err := r3.NewFrame(r3.NewVec(0, r.dm.Size.BladeSocketStart, r.dm.Size.BladeSocketZ()),
		r3.NewVec(0, 0, 1), r3.NewVec(1, 0, 0))
	if err != nil {
		return nil, err
	}
	tool, sk, err := BladeSocketTool(r.ctx, r.doc, frame, r.dm.Size)
	if err != nil {
		return nil, err
	}
	r.sketches = append(r.sketches, sk)
	return r.ring(tool)
}

func (r *run) bladeSlotTool(k int) (*decad.Body, error) {
	if k < 1 || k > BladeCount {
		return nil, fmt.Errorf("blade index %d is outside [1, %d]", k, BladeCount)
	}
	site := r.dm.BladeSites[k-1]
	angle := BladeAngle(k)
	u := r3.NewVec(math.Cos(angle), math.Sin(angle), 0)
	return r.slotTool(site, u, r.dm.Size.HubRadius+1-r.dm.Size.BladeBoltAt)
}

func (r *run) ring(body *decad.Body) ([]*decad.Body, error) {
	copies, err := decad.PatternCopies(r.ctx, body, decad.CircularPattern{
		Center: r3.NewVec(0, 0, 0), Axis: r3.NewVec(0, 0, 1), Count: BladeCount,
	})
	if err != nil {
		return nil, fmt.Errorf("circular pattern: %w", err)
	}
	return append([]*decad.Body{body}, copies...), nil
}

func (r *run) blades() ([]*decad.Body, error) {
	var blade *Blade
	var err error
	if r.p.Hand == LeftHand {
		blade, err = BuildBladeMirrored(r.ctx, r.doc, r.p.BladePitch, r.dm.Size)
	} else {
		blade, err = BuildBlade(r.ctx, r.doc, r.p.BladePitch, r.dm.Size)
	}
	if err != nil {
		return nil, err
	}
	r.sketches = append(r.sketches, blade.Sketches...)
	frame, err := r3.NewFrame(r3.NewVec(0, 0, r.dm.Size.BladeSocketZ()), r3.NewVec(-1, 0, 0), r3.NewVec(0, 0, 1))
	if err != nil {
		return nil, fmt.Errorf("assembly frame: %w", err)
	}
	transform, err := r3.FromFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("assembly transform: %w", err)
	}
	body, err := decad.Transform(r.ctx, blade.Body, transform)
	if err != nil {
		return nil, fmt.Errorf("place first blade: %w", err)
	}
	return r.ring(body)
}

func placeSite(site m2fastener.Site, frame *r3.Frame) (m2fastener.Site, error) {
	if frame == nil {
		return site, nil
	}
	transform, err := r3.FromFrame(*frame)
	if err != nil {
		return site, fmt.Errorf("placement transform: %w", err)
	}
	site.Head = transform.Apply(site.Head)
	site.Dir = transform.ApplyDir(site.Dir)
	site.Corner = transform.ApplyDir(site.Corner)
	return site, nil
}
