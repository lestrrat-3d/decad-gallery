// Package m2fastener places M2 bolts and captive nuts in their host parts, as AGENTS.md "Fasteners" requires: a bolt
// head sinks in a pocket and ends flush, and a nut sits in a cavity the host encloses apart from the bolt hole.
package m2fastener

import (
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Fastener dimensions in mm.
const (
	Clearance        = 0.1                               // AGENTS.md radial clearance round a head or nut
	HoleRadius       = 1.3                               // clearance hole, 2.6 across (>= the 2.4 shank rule)
	HeadPocketRadius = m2bolt.HeadDiameter/2 + Clearance // 2.0
	HeadPocketDepth  = m2bolt.HeadHeight                 // 2.0: the head ends flush
	NutCavityFlats   = m2nut.AcrossFlats + 2*Clearance   // 4.2
	NutCavityHeight  = m2nut.Thickness + 2*Clearance     // 1.8
	Stock            = 1.0                               // a tool's reach past a face it opens
	NutSpec          = "M2 hex nut"                      // the bought part a nut is
	unitTolerance    = 1e-9                              // how far a direction may stray from unit length
)

// Site is one bolt and its captive nut. Head is the centre of the head's top face. Dir is the unit vector from the head
// toward the tip. Corner is the unit vector across Dir on which the nut's and the cavity's corners lie. Length is one of
// m2bolt.StandardLengths(). Cavity is the nut cavity's near face, measured from Head along Dir.
type Site struct {
	Head, Dir, Corner r3.Vec
	Length, Cavity    float64
}

// Validate requires unit, perpendicular Dir and Corner, a standard Length and a Cavity beyond the head pocket.
func (s Site) Validate() error {
	if math.Abs(s.Dir.Len()-1) > unitTolerance || math.Abs(s.Corner.Len()-1) > unitTolerance {
		return fmt.Errorf("fastener site: Dir and Corner must be unit vectors")
	}
	if math.Abs(s.Dir.Dot(s.Corner)) > unitTolerance {
		return fmt.Errorf("fastener site: Corner must be perpendicular to Dir")
	}
	if !slices.Contains(m2bolt.StandardLengths(), s.Length) {
		return fmt.Errorf("fastener site: length %g is not a standard M2 length", s.Length)
	}
	if !(s.Cavity > HeadPocketDepth) {
		return fmt.Errorf("fastener site: Cavity %g must lie beyond the head pocket (%g)", s.Cavity, HeadPocketDepth)
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

// CavityFar returns the distance from Head to the nut cavity's far face.
func (s Site) CavityFar() float64 { return s.Cavity + NutCavityHeight }

// Turned returns the site turned by angle about the line through center along axis.
func (s Site) Turned(center, axis r3.Vec, angle float64) (Site, error) {
	xf, err := r3.RotationAround(center, axis, units.Radians(angle))
	if err != nil {
		return s, err
	}
	s.Head, s.Dir, s.Corner = xf.Apply(s.Head), xf.ApplyDir(s.Dir), xf.ApplyDir(s.Corner)
	return s, nil
}

// BoltSpec names the bought bolt of the given length.
func BoltSpec(length float64) string { return fmt.Sprintf("M2 x %g socket head cap screw", length) }

// PocketTool builds the head pocket's cutter: radius HeadPocketRadius, from Stock before the face to HeadPocketDepth.
func PocketTool(ctx context.Context, doc *decad.Document, s Site) (*decad.Body, *sketch.Sketch, error) {
	return cylinderTool(ctx, doc, s, -Stock, HeadPocketRadius, Stock+HeadPocketDepth)
}

// HoleTool builds the clearance hole's cutter: radius HoleRadius, from Stock before the face to Stock past the tip.
func HoleTool(ctx context.Context, doc *decad.Document, s Site) (*decad.Body, *sketch.Sketch, error) {
	return cylinderTool(ctx, doc, s, -Stock, HoleRadius, Stock+s.Tip()+Stock)
}

func cylinderTool(ctx context.Context, doc *decad.Document, s Site, from, radius, length float64) (*decad.Body,
	*sketch.Sketch, error) {
	sk, err := newSketch(s, from)
	if err != nil {
		return nil, nil, err
	}
	sk.AddConstraint(sketch.NewRadius(sk.CreateCircle(sk.Origin(), radius), radius))
	body, err := extrude(ctx, doc, sk, length)
	return body, sk, err
}

// CavityTool builds the nut cavity's cutter: a hexagon NutCavityFlats across, from Cavity to CavityFar, corners on
// Corner.
func CavityTool(ctx context.Context, doc *decad.Document, s Site) (*decad.Body, *sketch.Sketch, error) {
	sk, err := newSketch(s, s.Cavity)
	if err != nil {
		return nil, nil, err
	}
	radius := NutCavityFlats / math.Sqrt(3)
	p, err := sk.CreatePolygon(0, 0, 6, radius)
	if err != nil {
		return nil, nil, fmt.Errorf("create cavity hexagon: %w", err)
	}
	sk.AddConstraint(
		sketch.NewCoincident(p.Center, sk.Origin()),
		sketch.NewHorizontal(p.Sides[1]),
		sketch.NewDistance(p.Center, p.Vertices[0], radius),
	)
	body, err := extrude(ctx, doc, sk, NutCavityHeight)
	return body, sk, err
}

func newSketch(s Site, t float64) (*sketch.Sketch, error) {
	f, err := s.Across(t)
	if err != nil {
		return nil, fmt.Errorf("fastener frame: %w", err)
	}
	w := sketch.NewWorld()
	plane, err := w.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, fmt.Errorf("create fastener plane: %w", err)
	}
	sk, err := w.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("create fastener sketch: %w", err)
	}
	return sk, nil
}

// extrude solves sk, requires it fully constrained and extrudes its one valid profile along the normal.
func extrude(ctx context.Context, doc *decad.Document, sk *sketch.Sketch, length float64) (*decad.Body, error) {
	if _, err := sk.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve fastener sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, sk); err != nil {
		return nil, fmt.Errorf("verify fastener sketch: %w", err)
	}
	var chosen *sketch.Profile
	for _, p := range sk.Profiles() {
		if !p.Valid || len(p.Holes) != 0 {
			continue
		}
		if chosen != nil {
			return nil, fmt.Errorf("select fastener profile: several valid profiles")
		}
		chosen = p
	}
	if chosen == nil {
		return nil, fmt.Errorf("select fastener profile: no valid profile")
	}
	body, err := decad.Extrude(ctx, doc, sk, chosen, decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("extrude fastener tool: %w", err)
	}
	return body, nil
}

// PlaceBolt builds an m2bolt of s.Length whose head's top face lies at s.Head and whose shank runs along s.Dir. The
// bolt frame's origin is At(HeadPocketDepth), U is Corner and V is (-Dir) x Corner.
func PlaceBolt(ctx context.Context, doc *decad.Document, s Site) (*m2bolt.Model, error) {
	f, err := r3.NewFrame(s.At(HeadPocketDepth), s.Corner, s.Dir.Scale(-1).Cross(s.Corner))
	if err != nil {
		return nil, fmt.Errorf("bolt frame: %w", err)
	}
	m, err := m2bolt.Build(ctx, doc, m2bolt.Params{Length: s.Length, Placement: &f})
	if err != nil {
		return nil, fmt.Errorf("build bolt: %w", err)
	}
	return m, nil
}

// PlaceNut builds an m2nut whose lower face lies Clearance beyond the cavity's near face, corners on s.Corner.
func PlaceNut(ctx context.Context, doc *decad.Document, s Site) (*m2nut.Model, error) {
	f, err := s.Across(s.Cavity + Clearance)
	if err != nil {
		return nil, fmt.Errorf("nut frame: %w", err)
	}
	m, err := m2nut.Build(ctx, doc, m2nut.Params{Placement: &f})
	if err != nil {
		return nil, fmt.Errorf("build nut: %w", err)
	}
	return m, nil
}
