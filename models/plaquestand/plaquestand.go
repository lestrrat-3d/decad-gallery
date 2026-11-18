// Package plaquestand builds a stand for a flat, heavy rectangular plaque from separate parts: two identical printed
// brackets, one printed rail that joins them, and two M2 x 6 bolts and nuts that fix the rail to the brackets. Each
// bracket is a slice of one profile: a base slab with a rail channel in its rear top, a front lip, a slot reclined from
// vertical that ends in a full round, and a back rest. The plaque leans in both brackets' slots against the back rests,
// and the base is deep enough that the assembly tips only after a tilt of 15 degrees or more.
//
// The local frame has its origin at the front-bottom corner (before rounding) of bracket 1's outer face. +X runs along
// the span, +Y from the front to the back (the viewer stands at -Y) and +Z is up; the base's bottom face is z = 0.
// A bracket prints on a side face and the rail prints flat.
//
// Build also adds a reference plaque body, centred on the span and posed in the slots. The plaque is not printed: it is
// there for the fit and stability checks and for images. The rail is modeled JointFit proud of the base top, with
// every gap at its Regular fit; tightening the bolts pulls it down onto the channel floor.
package plaquestand

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Container names, one per kind of body.
const (
	BracketsContainerName = "Brackets"
	RailContainerName     = "Rail"
	BoltsContainerName    = "Bolts"
	NutsContainerName     = "Nuts"
	PlaqueContainerName   = "Plaque"
)

// Model is a built stand and its reference plaque.
type Model struct {
	// Brackets are the two identical printed brackets, in container Brackets. Bracket k spans x from k's start to its
	// start plus BracketWidth; bracket 1 starts at 0 and bracket 2 at Span - BracketWidth.
	Brackets [2]*decad.Body
	// Rail is the printed rail, in container Rail.
	Rail *decad.Body
	// Bolts and Nuts are the bought M2 x 6 bolts and nuts, one pair per bracket, in containers Bolts and Nuts.
	Bolts [2]*m2bolt.Model
	Nuts  [2]*m2nut.Model
	// Plaque is the reference plaque, not printed, in container Plaque.
	Plaque *decad.Body
	// Sketches are every sketch the bodies were built from, each fully constrained, in build order.
	Sketches   []*sketch.Sketch
	Dimensions Dimensions
}

// Build adds the brackets, the rail, the bolts, the nuts and the reference plaque to doc, each in its container. ctx and
// doc must not be nil. A failure before the first body adds none.
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
	m := &Model{Dimensions: dm}

	for k := range m.Brackets {
		x0 := float64(k) * (dm.Span - BracketWidth)
		if m.Brackets[k], err = buildBracket(ctx, doc, m, x0, dm.Sites[k]); err != nil {
			return nil, fmt.Errorf("plaquestand: bracket %d: %w", k+1, err)
		}
	}
	if m.Rail, err = buildRail(ctx, doc, m); err != nil {
		return nil, fmt.Errorf("plaquestand: rail: %w", err)
	}

	plaqueSk, plaqueProfile, err := plaqueSketch(ctx, dm, p)
	if err != nil {
		return nil, fmt.Errorf("plaquestand: plaque sketch: %w", err)
	}
	if m.Plaque, err = decad.Extrude(ctx, doc, plaqueSk, plaqueProfile,
		decad.Distance{D: units.Millimeters(p.PlaqueThickness), Dir: decad.Along}); err != nil {
		return nil, fmt.Errorf("plaquestand: extrude plaque: %w", err)
	}
	if !m.Plaque.IsSolid() {
		return nil, errors.New("plaquestand: plaque is not a solid")
	}
	m.Sketches = append(m.Sketches, plaqueSk)

	for k, site := range dm.Sites {
		if m.Bolts[k], err = m2fastener.PlaceBolt(ctx, doc, site); err != nil {
			return nil, fmt.Errorf("plaquestand: bolt %d: %w", k+1, err)
		}
		if m.Nuts[k], err = m2fastener.PlaceNut(ctx, doc, site); err != nil {
			return nil, fmt.Errorf("plaquestand: nut %d: %w", k+1, err)
		}
	}

	type filed struct {
		name string
		body *decad.Body
	}
	bodies := []filed{
		{BracketsContainerName, m.Brackets[0]}, {BracketsContainerName, m.Brackets[1]}, {RailContainerName, m.Rail},
		{BoltsContainerName, m.Bolts[0].Body}, {BoltsContainerName, m.Bolts[1].Body},
		{NutsContainerName, m.Nuts[0].Body}, {NutsContainerName, m.Nuts[1].Body}, {PlaqueContainerName, m.Plaque},
	}
	for _, c := range bodies {
		box, err := doc.Root().ChildNamed(c.name)
		if err != nil {
			return nil, fmt.Errorf("plaquestand: container %s: %w", c.name, err)
		}
		if err := box.AddBody(c.body); err != nil {
			return nil, fmt.Errorf("plaquestand: add body to %s: %w", c.name, err)
		}
	}
	return m, nil
}

// buildBracket extrudes the profile on the plane x = x0 by BracketWidth, chamfers both side faces' outlines, then cuts
// the nut's hex pocket (its floor outline rounded by NutPocketRound) and the bolt's hole. It records the sketches.
func buildBracket(ctx context.Context, doc *decad.Document, m *Model, x0 float64, site m2fastener.Site) (*decad.Body,
	error) {
	sk, profile, err := profileSketch(ctx, m.Dimensions, x0)
	if err != nil {
		return nil, fmt.Errorf("profile sketch: %w", err)
	}
	m.Sketches = append(m.Sketches, sk)
	body, err := decad.Extrude(ctx, doc, sk, profile, decad.Distance{D: units.Millimeters(BracketWidth), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	body, _, err = edgefinish.Chamfer(ctx, body, r3.Vec{X: 1}, []float64{x0, x0 + BracketWidth}, edgefinish.Outer,
		ChamferSize)
	if err != nil {
		return nil, fmt.Errorf("chamfer side faces: %w", err)
	}
	pocket, pocketSk, err := m2fastener.RoundedHexPocketTool(ctx, doc, site, m2fastener.NutCavityHeight+m2fastener.Stock,
		NutPocketRound)
	if err != nil {
		return nil, fmt.Errorf("nut pocket tool: %w", err)
	}
	hole, holeSk, err := m2fastener.HoleTool(ctx, doc, site)
	if err != nil {
		return nil, fmt.Errorf("hole tool: %w", err)
	}
	m.Sketches = append(m.Sketches, pocketSk, holeSk)
	for _, tool := range []*decad.Body{pocket, hole} {
		if body, err = decad.Cut(ctx, body, tool); err != nil {
			return nil, fmt.Errorf("cut: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, errors.New("not a solid")
	}
	return body, nil
}

// buildRail extrudes the rail's section along +X by Span, then cuts each bolt's head pocket and hole. It adds no
// chamfer: see instructions.md "Surface edge finishing".
func buildRail(ctx context.Context, doc *decad.Document, m *Model) (*decad.Body, error) {
	sk, profile, err := railSketch(ctx, m.Dimensions)
	if err != nil {
		return nil, fmt.Errorf("sketch: %w", err)
	}
	m.Sketches = append(m.Sketches, sk)
	body, err := decad.Extrude(ctx, doc, sk, profile, decad.Distance{D: units.Millimeters(m.Dimensions.Span), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	var tools []*decad.Body
	for _, site := range m.Dimensions.Sites {
		pocket, pocketSk, err := m2fastener.PocketTool(ctx, doc, site)
		if err != nil {
			return nil, fmt.Errorf("head pocket tool: %w", err)
		}
		m.Sketches = append(m.Sketches, pocketSk)
		tools = append(tools, pocket)
	}
	for _, site := range m.Dimensions.Sites {
		hole, holeSk, err := m2fastener.HoleTool(ctx, doc, site)
		if err != nil {
			return nil, fmt.Errorf("hole tool: %w", err)
		}
		m.Sketches = append(m.Sketches, holeSk)
		tools = append(tools, hole)
	}
	for _, tool := range tools {
		if body, err = decad.Cut(ctx, body, tool); err != nil {
			return nil, fmt.Errorf("cut: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, errors.New("not a solid")
	}
	return body, nil
}
