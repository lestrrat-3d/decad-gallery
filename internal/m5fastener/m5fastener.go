// Package m5fastener builds the cutters that seat M5 bolts and nuts in their host parts and places the bolts and nuts
// in them. Head, nut and press pockets are hexagons that stop the fastener turning. Each pocket's floor outline is
// rounded by PocketRound (PressPocketRound for the press pocket), so a printed part has no sharp inside corner there.
package m5fastener

import (
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/fastenertool"
	"github.com/lestrrat-3d/decad-gallery/models/m5bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m5nut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Fastener dimensions in mm.
const (
	HeadClearance     = 0.3    // radial clearance round a head and a shank
	NutClearance      = 0.2    // radial clearance past a nut's corners
	HoleRadius        = 2.8    // the clearance hole, 5.6 across
	HeadPocketRadius  = 4.9188 // corner radius of the head pocket: the head's corners (4.6188) plus HeadClearance
	HeadPocketDepth   = 3.5    // the head ends flush
	NutPocketRadius   = 4.8188 // corner radius of the nut pocket: the nut's corners plus NutClearance
	NutCavityHeight   = 5.1    // the nut's thickness plus 0.2 on each face
	PressPocketRadius = 4.6688 // corner radius of a press-fit head pocket: the head's corners plus 0.05
	PocketRound       = 0.6    // round on a head or nut pocket floor's outline
	// PressPocketRound is the round on a press pocket floor's outline. The head's 30 degree chamfer overlaps a 0.6
	// round at the flats of the 4.6688 pocket, so it is the largest round that keeps the fit, at least 0.5.
	PressPocketRound = 0.5
	Stock            = 1.0  // a tool's reach past a face it opens
	unitTolerance    = 1e-9 // how far a direction may stray from unit length
)

// roundChords is the number of chords RoundedSocketTool draws its floor round with.
const roundChords = 8

// Site is one bolt. Head is the centre of the head's top face. Dir is the unit vector from the head toward the tip.
// Corner is the unit vector across Dir on which the head's corners lie. Length is one of m5bolt.StandardLengths().
type Site struct {
	Head, Dir, Corner r3.Vec
	Length            float64
}

// Validate requires unit, perpendicular Dir and Corner and a standard Length.
func (s Site) Validate() error {
	if math.Abs(s.Dir.Len()-1) > unitTolerance || math.Abs(s.Corner.Len()-1) > unitTolerance {
		return fmt.Errorf("fastener site: Dir and Corner must be unit vectors")
	}
	if math.Abs(s.Dir.Dot(s.Corner)) > unitTolerance {
		return fmt.Errorf("fastener site: Corner must be perpendicular to Dir")
	}
	if !slices.Contains(m5bolt.StandardLengths(), s.Length) {
		return fmt.Errorf("fastener site: length %g is not a standard M5 length", s.Length)
	}
	return nil
}

// At returns the point t along the site's line from the head's top face.
func (s Site) At(t float64) r3.Vec { return s.Head.Add(s.Dir.Scale(t)) }

// Across returns the frame at distance t from the head whose normal is Dir and whose U axis is Corner.
func (s Site) Across(t float64) (r3.Frame, error) {
	return r3.NewFrame(s.At(t), s.Corner, s.Dir.Cross(s.Corner))
}

// Tip returns the distance from Head to the bolt's tip.
func (s Site) Tip() float64 { return HeadPocketDepth + s.Length }

// HeadPocketTool builds a head pocket's cutter on floor frame f: origin at the floor's centre, +Z toward the opening,
// U along a hex corner. It is a hexagon of corner radius HeadPocketRadius, HeadPocketDepth deep plus Stock past the
// opening, with the floor outline rounded by PocketRound.
func HeadPocketTool(ctx context.Context, doc *decad.Document, f r3.Frame) (*decad.Body, *sketch.Sketch, error) {
	return pocketTool(ctx, doc, f, HeadPocketRadius, HeadPocketDepth+Stock, PocketRound)
}

// NutPocketTool builds a nut pocket's cutter on floor frame f, as HeadPocketTool does, with corner radius
// NutPocketRadius. depth is the floor-to-opening depth; Stock is added past the opening.
func NutPocketTool(ctx context.Context, doc *decad.Document, f r3.Frame, depth float64) (*decad.Body, *sketch.Sketch,
	error) {
	return pocketTool(ctx, doc, f, NutPocketRadius, depth+Stock, PocketRound)
}

// PressPocketTool builds a press-fit head pocket's cutter on floor frame f, as HeadPocketTool does, with corner radius
// PressPocketRadius and the floor outline rounded by PressPocketRound.
func PressPocketTool(ctx context.Context, doc *decad.Document, f r3.Frame) (*decad.Body, *sketch.Sketch, error) {
	return pocketTool(ctx, doc, f, PressPocketRadius, HeadPocketDepth+Stock, PressPocketRound)
}

func pocketTool(ctx context.Context, doc *decad.Document, f r3.Frame, radius, depth, round float64) (*decad.Body,
	*sketch.Sketch, error) {
	body, sk, err := fastenertool.HexPrism(ctx, doc, f, radius, depth)
	if err != nil {
		return nil, nil, err
	}
	floor := decad.Edges(decad.OuterLoopOf(decad.CapStart(body)))
	if body, err = decad.Fillet(ctx, body, floor, units.Millimeters(round)); err != nil {
		return nil, nil, fmt.Errorf("round pocket floor: %w", err)
	}
	return body, sk, nil
}

// HoleTool builds the clearance hole's cutter: radius HoleRadius, from Stock before the head to Stock past the tip.
func HoleTool(ctx context.Context, doc *decad.Document, s Site) (*decad.Body, *sketch.Sketch, error) {
	f, err := s.Across(-Stock)
	if err != nil {
		return nil, nil, fmt.Errorf("fastener frame: %w", err)
	}
	return fastenertool.Cylinder(ctx, doc, f, HoleRadius, Stock+s.Tip()+Stock)
}

// RoundedSocketTool builds a blind round socket's cutter on floor frame f: radius, from the floor out to depth plus
// Stock past the opening, with the floor's corner rounded by round in roundChords chords.
func RoundedSocketTool(ctx context.Context, doc *decad.Document, f r3.Frame, radius, depth, round float64) (
	*decad.Body, *sketch.Sketch, error) {
	mf, err := r3.NewFrame(f.Origin(), f.N(), f.U())
	if err != nil {
		return nil, nil, fmt.Errorf("socket meridian frame: %w", err)
	}
	end := depth + Stock
	pts := [][2]float64{{0, 0}}
	pts = append(pts, fastenertool.QuarterRound(round, radius-round, round, math.Pi, roundChords)...)
	pts = append(pts, [2]float64{end, radius}, [2]float64{end, 0})
	return fastenertool.RoundedRevolve(ctx, doc, mf, pts)
}

// PlaceBolt builds an m5bolt of s.Length whose head's top face lies at s.Head and whose shank runs along s.Dir.
func PlaceBolt(ctx context.Context, doc *decad.Document, s Site) (*m5bolt.Model, error) {
	f, err := r3.NewFrame(s.At(HeadPocketDepth), s.Corner, s.Dir.Scale(-1).Cross(s.Corner))
	if err != nil {
		return nil, fmt.Errorf("bolt frame: %w", err)
	}
	m, err := m5bolt.Build(ctx, doc, m5bolt.Params{Length: s.Length, Placement: &f})
	if err != nil {
		return nil, fmt.Errorf("build bolt: %w", err)
	}
	return m, nil
}

// PlaceNut builds an m5nut whose lower face lies on f, with its corners on f's U.
func PlaceNut(ctx context.Context, doc *decad.Document, f r3.Frame) (*m5nut.Model, error) {
	m, err := m5nut.Build(ctx, doc, m5nut.Params{Placement: &f})
	if err != nil {
		return nil, fmt.Errorf("build nut: %w", err)
	}
	return m, nil
}
