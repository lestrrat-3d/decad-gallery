// Package crankhandle builds a printed crank handle for a D-flat shaft: a flat plate whose hub has a D bore, a
// tapered arm, and a round grip standing on the arm's end. An M2 set screw enters the hub's flat tail face and
// presses on the shaft's flat; its nut sits in a cavity the hub encloses.
//
// The local frame has its origin on the shaft axis in the crank's back face, the face toward the mechanism. +Z
// runs along the shaft axis out of the mechanism, the arm runs along +Y to the grip, and the bore's flat and the
// set screw face -Y.
package crankhandle

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
	"github.com/lestrrat-3d/units"
)

// Container names, one per kind of body.
const (
	CrankContainerName = "Crank"
	BoltsContainerName = "Bolts"
	NutsContainerName  = "Nuts"
)

// Model is a built crank handle.
type Model struct {
	// Crank is the printed crank: hub, arm and grip in one body.
	Crank *decad.Body
	// Bolt is the M2 set screw and Nut its captive nut.
	Bolt *m2bolt.Model
	Nut  *m2nut.Model
	// Plate is the plate's outline sketch and Screw the set screw tool's meridian sketch.
	Plate, Screw *sketch.Sketch
	Dimensions   Dimensions
	// Site is the set screw in the world frame.
	Site m2fastener.Site
}

// Build adds the crank, its set screw and its nut to doc, each in its container. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error) {
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
	m := &Model{Dimensions: dm}
	crank, err := m.buildCrank(ctx, doc, p)
	if err != nil {
		return nil, err
	}
	if m.Crank, err = placement.Place(ctx, crank, p.Placement); err != nil {
		return nil, fmt.Errorf("crankhandle: place crank: %w", err)
	}
	if m.Site, err = placeSite(dm.Site, p.Placement); err != nil {
		return nil, fmt.Errorf("crankhandle: place set screw: %w", err)
	}
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
		{CrankContainerName, m.Crank}, {BoltsContainerName, m.Bolt.Body}, {NutsContainerName, m.Nut.Body},
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

// buildCrank builds the crank in the local frame.
func (m *Model) buildCrank(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	dm := m.Dimensions
	s, profile, err := plateSketch(ctx, p, dm)
	if err != nil {
		return nil, err
	}
	m.Plate = s
	body, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: units.Millimeters(p.Thickness), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("crankhandle: extrude plate: %w", err)
	}
	// The set screw channel is cut before the faces are finished: Decad composes no boolean with a body whose
	// cap-loop chamfer has arcs.
	tool, ts, err := screwTool(ctx, doc, dm.Site)
	if err != nil {
		return nil, err
	}
	m.Screw = ts
	if body, err = decad.Cut(ctx, body, tool); err != nil {
		return nil, fmt.Errorf("crankhandle: cut set screw channel: %w", err)
	}
	// The only hole in either face is the bore. Its length comes from the dimensions: edgefinish.Long measures an
	// arc a Boolean cut leaves the shorter way round, and the bore's arc runs more than half a turn.
	keep := func(loop *decad.Loop, edges []*decad.Edge) bool {
		return edgefinish.Outer(loop, edges) || dm.BoreLoop >= edgefinish.LongEdgeRun
	}
	if body, _, err = edgefinish.Chamfer(ctx, body, r3.Vec{Z: 1}, []float64{0, p.Thickness}, keep,
		edgefinish.MinChamfer); err != nil {
		return nil, fmt.Errorf("crankhandle: finish plate edges: %w", err)
	}
	body, err = decad.Boss(ctx, body, decad.BossSpec{
		Face:       decad.Faces(decad.Facing(r3.Vec{Z: 1})).Exactly(1),
		At:         r3.Vec{Y: p.Throw, Z: p.Thickness},
		Diameter:   units.Millimeters(p.GripDiameter),
		Height:     units.Millimeters(p.GripLength),
		RootFillet: units.Millimeters(dm.RootRadius),
	})
	if err != nil {
		return nil, fmt.Errorf("crankhandle: raise grip: %w", err)
	}
	top, err := gripTopEdge(body, p)
	if err != nil {
		return nil, err
	}
	if body, err = decad.Fillet(ctx, body, decad.Edges(top...).Exactly(1), units.Millimeters(dm.TipRound)); err != nil {
		return nil, fmt.Errorf("crankhandle: round grip top: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("crankhandle: crank is not a solid")
	}
	return body, nil
}

// gripTopEdge names the circular edge round the grip's top face.
func gripTopEdge(body *decad.Body, p Params) ([]decad.EdgePredicate, error) {
	top := p.Thickness + p.GripLength
	for _, e := range body.Edges() {
		c, ok := e.Curve().(decad.Circle3)
		if !ok || math.Abs(c.Center.Z-top) > 1e-6 || math.Abs(c.Radius.Base()-p.GripDiameter/2) > 1e-6 {
			continue
		}
		return []decad.EdgePredicate{decad.EndpointAt(e.Start().Position().Value), decad.Circular()}, nil
	}
	return nil, errors.New("crankhandle: no circular edge round the grip's top face")
}

// placeSite carries the local set screw site through the placement.
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
