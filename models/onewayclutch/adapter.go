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

// AdapterContainerName is the container of the adapter ring.
const AdapterContainerName = "Adapter"

// Adapter is the standard adapter ring with its three bolts and nuts, placed. It bolts to the clutch's ears from the
// front. A part of another shape takes its place when it has a flat back face at AdapterFrom, is AdapterThickness thick
// at each ear, and cuts m2fastener.PocketTool and a clearance hole at Dimensions.MountSites.
type Adapter struct {
	Body  *decad.Body
	Bolts [3]*m2bolt.Model
	Nuts  [3]*m2nut.Model
	// Sites are the three bolts in the world frame.
	Sites [3]m2fastener.Site
	// Sketches are every sketch the adapter was built from, each fully constrained.
	Sketches []*sketch.Sketch
}

// BuildAdapter adds the adapter ring and its three M2 bolts and nuts to doc, filed in the Adapter, Bolts and Nuts
// containers. Each nut seats in the nut pocket of the ring's rear face, so the parts meet only in a document that also
// holds the clutch built from the same Params. doc needs a chain tolerance only when it holds the clutch (see
// NewDocument). ctx and doc must not be nil.
func BuildAdapter(ctx context.Context, doc *decad.Document, p Params) (*Adapter, error) {
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
	body, err := r.adapter()
	if err != nil {
		return nil, fmt.Errorf("onewayclutch: adapter: %w", err)
	}
	if body == nil || !body.IsSolid() {
		return nil, errors.New("onewayclutch: adapter is not a solid")
	}
	out := &Adapter{Sketches: r.sketches}
	if out.Body, err = placement.Place(ctx, body, p.Placement); err != nil {
		return nil, fmt.Errorf("onewayclutch: place adapter: %w", err)
	}
	type filed struct {
		name string
		body *decad.Body
	}
	bodies := []filed{{AdapterContainerName, out.Body}}
	for i, site := range dm.MountSites {
		if out.Sites[i], err = placeSite(site, p.Placement); err != nil {
			return nil, fmt.Errorf("onewayclutch: place mount bolt %d: %w", i+1, err)
		}
		if out.Bolts[i], err = m2fastener.PlaceBolt(ctx, doc, out.Sites[i]); err != nil {
			return nil, fmt.Errorf("onewayclutch: mount bolt %d: %w", i+1, err)
		}
		if out.Nuts[i], err = m2fastener.PlaceNut(ctx, doc, out.Sites[i]); err != nil {
			return nil, fmt.Errorf("onewayclutch: mount nut %d: %w", i+1, err)
		}
		bodies = append(bodies, filed{BoltsContainerName, out.Bolts[i].Body}, filed{NutsContainerName, out.Nuts[i].Body})
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
	return out, nil
}

// adapter extrudes the adapter's annulus (adapterSketch) AdapterThickness along +Z, chamfers the long loops of both
// faces, then cuts a head pocket at each mount site. The clearance holes are in the profile.
func (r *run) adapter() (*decad.Body, error) {
	s, err := r.adapterSketch()
	if err != nil {
		return nil, err
	}
	body, err := r.extrude(s, Ears+1, AdapterThickness)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	if body, _, err = edgefinish.Chamfer(r.ctx, body, r3.Vec{Z: 1}, []float64{AdapterFrom, AdapterTo},
		edgefinish.Long(edgefinish.All), PlateChamfer); err != nil {
		return nil, fmt.Errorf("chamfer faces: %w", err)
	}
	for i, site := range r.dm.MountSites {
		tool, sk, err := m2fastener.PocketTool(r.ctx, r.doc, site)
		if err != nil {
			return nil, fmt.Errorf("head pocket %d: %w", i+1, err)
		}
		r.sketches = append(r.sketches, sk)
		if body, err = decad.Cut(r.ctx, body, tool); err != nil {
			return nil, fmt.Errorf("cut head pocket %d: %w", i+1, err)
		}
	}
	return body, nil
}
