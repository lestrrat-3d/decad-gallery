// Package onewayclutch builds a printed one-way clutch for a D-flat shaft: a hub with three flexing pawls pressed on the
// shaft, a ring with 24 internal sawtooth teeth that the pawl heads engage, and a back plate and a cover that hold the
// ring on the hub, clamped by three M2 bolts and nuts. The ring turns freely one way about the held shaft and is driven
// the other way. The freewheel direction is a parameter: every hub and ring sketch is drawn in one counter-clockwise
// layout and mirrored in the plane x = 0 for the clockwise one.
//
// The local frame has its origin on the shaft axis in the back plate's outer face, the face toward the mechanism. +Z
// runs along the shaft axis to the cover, and the shaft's flat faces +Y.
package onewayclutch

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Container names, one per kind of body.
const (
	HubContainerName    = "Hub"
	RingContainerName   = "Ring"
	PlatesContainerName = "Plates"
	BoltsContainerName  = "Bolts"
	NutsContainerName   = "Nuts"
)

// Parts are the clutch's printed bodies, placed, without the bolts and nuts.
type Parts struct {
	Hub, Ring, BackPlate, Cover *decad.Body
	// Sketches are every sketch the parts were built from, each fully constrained.
	Sketches   []*sketch.Sketch
	Dimensions Dimensions
	// Sites are the three bolts in the world frame.
	Sites [3]m2fastener.Site
}

// Model is a built clutch: its parts, bolts and nuts.
type Model struct {
	Parts
	Bolts [3]*m2bolt.Model
	Nuts  [3]*m2nut.Model
}

// BuildParts adds the hub, the ring, the back plate and the cover to doc and returns them. It puts them in no
// container. ctx and doc must not be nil.
func BuildParts(ctx context.Context, doc *decad.Document, p Params) (*Parts, error) {
	if ctx == nil {
		return nil, errors.New("onewayclutch: context must not be nil")
	}
	if doc == nil {
		return nil, errors.New("onewayclutch: document must not be nil")
	}
	dm, err := p.Derive()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("onewayclutch: before creating geometry: %w", err)
	}
	r := &run{ctx: ctx, doc: doc, p: p, dm: dm}
	out := &Parts{Dimensions: dm}
	for _, step := range []struct {
		name  string
		build func() (*decad.Body, error)
		dst   **decad.Body
	}{
		{"hub", r.hub, &out.Hub}, {"ring", r.ring, &out.Ring},
		{"back plate", r.backPlate, &out.BackPlate}, {"cover", r.cover, &out.Cover},
	} {
		body, err := step.build()
		if err != nil {
			return nil, fmt.Errorf("onewayclutch: %s: %w", step.name, err)
		}
		if body == nil || !body.IsSolid() {
			return nil, fmt.Errorf("onewayclutch: %s is not a solid", step.name)
		}
		if *step.dst, err = placement.Place(ctx, body, p.Placement); err != nil {
			return nil, fmt.Errorf("onewayclutch: place %s: %w", step.name, err)
		}
	}
	out.Sketches = r.sketches
	for i, site := range dm.Sites {
		if out.Sites[i], err = placeSite(site, p.Placement); err != nil {
			return nil, fmt.Errorf("onewayclutch: place bolt %d: %w", i+1, err)
		}
	}
	return out, nil
}

// Build adds the hub, the ring, the plates, and every bolt and nut to doc, each in its container. ctx and doc must not
// be nil.
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
	bodies := []filed{
		{HubContainerName, m.Hub}, {RingContainerName, m.Ring},
		{PlatesContainerName, m.BackPlate}, {PlatesContainerName, m.Cover},
	}
	for i, site := range m.Sites {
		if m.Bolts[i], err = m2fastener.PlaceBolt(ctx, doc, site); err != nil {
			return nil, fmt.Errorf("onewayclutch: bolt %d: %w", i+1, err)
		}
		if m.Nuts[i], err = m2fastener.PlaceNut(ctx, doc, site); err != nil {
			return nil, fmt.Errorf("onewayclutch: nut %d: %w", i+1, err)
		}
		bodies = append(bodies, filed{BoltsContainerName, m.Bolts[i].Body}, filed{NutsContainerName, m.Nuts[i].Body})
	}
	for _, c := range bodies {
		box, err := doc.Root().ChildNamed(c.name)
		if err != nil {
			return nil, fmt.Errorf("onewayclutch: container %s: %w", c.name, err)
		}
		if err := box.AddBody(c.body); err != nil {
			return nil, fmt.Errorf("onewayclutch: add body to %s: %w", c.name, err)
		}
	}
	return m, nil
}

// hub extrudes the hub's outline, with the D bore and the bolt holes, HubHeight along +Z. It has no Boolean; its
// faces are not chamfered (design Q1).
func (r *run) hub() (*decad.Body, error) {
	s, err := r.hubSketch()
	if err != nil {
		return nil, err
	}
	body, err := r.extrude(s, 4, HubHeight)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	return body, nil
}

// ring revolves the ring's meridian (RingMeridian) and cuts the tooth tool from it.
func (r *run) ring() (*decad.Body, error) {
	blank, err := r.meridian(RingMeridian[:])
	if err != nil {
		return nil, fmt.Errorf("blank: %w", err)
	}
	const from, length = 0.25, 20.0
	s, err := r.toothSketch(from)
	if err != nil {
		return nil, fmt.Errorf("tooth tool sketch: %w", err)
	}
	tool, err := r.extrude(s, 0, length)
	if err != nil {
		return nil, fmt.Errorf("tooth tool: %w", err)
	}
	body, err := decad.Cut(r.ctx, blank, tool)
	if err != nil {
		return nil, fmt.Errorf("cut teeth: %w", err)
	}
	return body, nil
}

// plate extrudes a plate's profile from z for length, chamfers the outer face's long loops, then cuts the inner-face
// tool and the fastener tools.
func (r *run) plate(z, length float64, outerZ float64, inner [][2]float64,
	tools func() ([]*decad.Body, error)) (*decad.Body, error) {
	s, err := r.plateSketch(z)
	if err != nil {
		return nil, err
	}
	body, err := r.extrude(s, 4, length)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	// Both of the outer face's long loops go in one call: the outer circle, and the D loop at the 8 mm shafts.
	if body, _, err = edgefinish.Chamfer(r.ctx, body, r3.Vec{Z: 1}, []float64{outerZ}, edgefinish.Long(edgefinish.All),
		PlateChamfer); err != nil {
		return nil, fmt.Errorf("chamfer outer face: %w", err)
	}
	cut, err := r.meridian(inner)
	if err != nil {
		return nil, fmt.Errorf("inner face tool: %w", err)
	}
	all := []*decad.Body{cut}
	more, err := tools()
	if err != nil {
		return nil, err
	}
	for _, tool := range append(all, more...) {
		if body, err = decad.Cut(r.ctx, body, tool); err != nil {
			return nil, fmt.Errorf("cut: %w", err)
		}
	}
	return body, nil
}

// backPlate is the plate on the mechanism side: z 0 to 5, with a nut cavity at each bolt.
func (r *run) backPlate() (*decad.Body, error) {
	return r.plate(0, BackPlateThickness, 0, BackPlateInner[:], func() ([]*decad.Body, error) {
		return r.fastenerTools(m2fastener.CavityTool)
	})
}

// cover is the plate on the front side: z 15.5 to 19.5, with a head pocket at each bolt.
func (r *run) cover() (*decad.Body, error) {
	return r.plate(CoverFrom, CoverThickness, CoverTo, CoverInner[:], func() ([]*decad.Body, error) {
		return r.fastenerTools(m2fastener.PocketTool)
	})
}

// fastenerTools builds one cutter per bolt site and records its sketch.
func (r *run) fastenerTools(build func(context.Context, *decad.Document, m2fastener.Site) (*decad.Body,
	*sketch.Sketch, error)) ([]*decad.Body, error) {
	var tools []*decad.Body
	for i, site := range r.dm.Sites {
		tool, sk, err := build(r.ctx, r.doc, site)
		if err != nil {
			return nil, fmt.Errorf("fastener tool %d: %w", i+1, err)
		}
		r.sketches = append(r.sketches, sk)
		tools = append(tools, tool)
	}
	return tools, nil
}

// RingMeridian is the ring blank's closed meridian (r, z), revolved a full turn about Z. The counterbores take the
// plates' rims, the 45 degree cones hold the ring axially, and the flats at BandFrom and BandTo become the teeth's end
// faces once the tooth tool cuts the band's bore away.
var RingMeridian = [...][2]float64{
	{RingOuter - RingChamfer, RingFrom}, {RingOuter, RingFrom + RingChamfer}, {RingOuter, RingTo - RingChamfer},
	{RingOuter - RingChamfer, RingTo}, {RingCounter, RingTo}, {RingCounter, BandTo + 1.5}, {ToothRoot, BandTo},
	{RingBand, BandTo}, {RingBand, BandFrom}, {ToothRoot, BandFrom}, {RingCounter, BandFrom - 1.5}, {RingCounter, RingFrom},
}

// BackPlateInner and CoverInner are the plates' inner-face tools (r, z): the boss, its bevel, the recessed annulus and
// the 45 degree cone to the rim.
var (
	BackPlateInner = [...][2]float64{{9.8, 6.0}, {9.8, 5.4}, {10.6, 4.6}, {17.0, 4.6}, {19.1, 2.5}, {19.1, 6.0}}
	CoverInner     = [...][2]float64{{9.8, 14.5}, {9.8, 15.1}, {10.6, 15.9}, {17.0, 15.9}, {19.1, 18.0}, {19.1, 14.5}}
)

// placeSite carries a local bolt site through the placement.
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
