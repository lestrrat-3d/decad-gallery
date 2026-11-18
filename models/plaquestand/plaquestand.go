// Package plaquestand builds a one-piece desk or shelf stand for a flat, heavy rectangular plaque. The stand is one
// extruded prism: a base slab, a front lip, a slot reclined from vertical that ends in a full round, and a back rest.
// The plaque leans in the slot against the back rest, and the base is deep enough that the assembly tips only after a
// tilt of 15 degrees or more.
//
// The local frame has its origin at the stand's front-bottom corner (before rounding) on the side face x = 0. +X runs
// along the stand's width, +Y from the front to the back (the viewer stands at -Y) and +Z is up; the base's bottom face
// is z = 0. The stand is printed lying on its x = 0 face.
//
// Build also adds a reference plaque body, centred on the stand and posed in the slot. The plaque is not printed: it is
// there for the fit and stability checks and for images.
package plaquestand

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Container names, one per kind of body.
const (
	StandContainerName  = "Stand"
	PlaqueContainerName = "Plaque"
)

// Model is a built stand and its reference plaque.
type Model struct {
	// Stand is the printed body, in container Stand.
	Stand *decad.Body
	// Plaque is the reference plaque, not printed, in container Plaque.
	Plaque *decad.Body
	// Profile is the stand's profile sketch.
	Profile *sketch.Sketch
	// PlaqueSketch is the plaque's rectangle sketch.
	PlaqueSketch *sketch.Sketch
	Dimensions   Dimensions
}

// Build adds the stand and the reference plaque to doc, each in its container. ctx and doc must not be nil. A
// failure before the first body adds none.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error) {
	if ctx == nil {
		return nil, errors.New("plaquestand: context must not be nil")
	}
	if doc == nil {
		return nil, errors.New("plaquestand: document must not be nil")
	}
	dm, err := p.Derive()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("plaquestand: before creating geometry: %w", err)
	}

	profileSk, profile, err := profileSketch(ctx, dm)
	if err != nil {
		return nil, fmt.Errorf("plaquestand: profile sketch: %w", err)
	}
	stand, err := decad.Extrude(ctx, doc, profileSk, profile,
		decad.Distance{D: units.Millimeters(dm.StandWidth), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("plaquestand: extrude stand: %w", err)
	}
	stand, _, err = edgefinish.Chamfer(ctx, stand, r3.Vec{X: 1}, []float64{0, dm.StandWidth}, edgefinish.Outer,
		ChamferSize)
	if err != nil {
		return nil, fmt.Errorf("plaquestand: chamfer side faces: %w", err)
	}
	if !stand.IsSolid() {
		return nil, errors.New("plaquestand: stand is not a solid")
	}

	plaqueSk, plaqueProfile, err := plaqueSketch(ctx, dm, p)
	if err != nil {
		return nil, fmt.Errorf("plaquestand: plaque sketch: %w", err)
	}
	plaque, err := decad.Extrude(ctx, doc, plaqueSk, plaqueProfile,
		decad.Distance{D: units.Millimeters(p.PlaqueThickness), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("plaquestand: extrude plaque: %w", err)
	}
	if !plaque.IsSolid() {
		return nil, errors.New("plaquestand: plaque is not a solid")
	}

	for _, c := range []struct {
		name string
		body *decad.Body
	}{{StandContainerName, stand}, {PlaqueContainerName, plaque}} {
		box, err := doc.Root().ChildNamed(c.name)
		if err != nil {
			return nil, fmt.Errorf("plaquestand: container %s: %w", c.name, err)
		}
		if err := box.AddBody(c.body); err != nil {
			return nil, fmt.Errorf("plaquestand: add body to %s: %w", c.name, err)
		}
	}
	return &Model{Stand: stand, Plaque: plaque, Profile: profileSk, PlaqueSketch: plaqueSk, Dimensions: dm}, nil
}
