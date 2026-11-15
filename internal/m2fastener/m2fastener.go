// Package m2fastener places M2 bolts and nuts in their host parts, as AGENTS.md "Fasteners" requires: a bolt head sinks
// in a pocket and ends flush, and a nut sits either in a cavity the host encloses apart from the bolt hole or in a hex
// pocket open to a face, where its tightened bolt holds it.
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
	HeadClearance    = 0.3                                    // AGENTS.md radial clearance round a head and a shank
	NutClearance     = 0.2                                    // AGENTS.md clearance round a nut: radial past its corners, axial at each face
	HoleRadius       = m2bolt.ShankDiameter/2 + HeadClearance // 1.3, the clearance hole 2.6 across
	HeadPocketRadius = m2bolt.HeadDiameter/2 + HeadClearance  // 2.2
	HeadPocketDepth  = m2bolt.HeadHeight                      // 2.0: the head ends flush
	NutCavityHeight  = m2nut.Thickness + 2*NutClearance       // 2.0
	Stock            = 1.0                                    // a tool's reach past a face it opens
	NutSpec          = "M2 hex nut"                           // the bought part a nut is
	unitTolerance    = 1e-9                                   // how far a direction may stray from unit length
)

// NutCavityRadius is the radius of the nut cavity: the circle through the nut's corners, plus NutClearance (2.51).
var NutCavityRadius = m2nut.AcrossFlats/math.Sqrt(3) + NutClearance

// Site is one bolt and its captive nut. Head is the centre of the head's top face. Dir is the unit vector from the head
// toward the tip. Corner is the unit vector across Dir on which the nut's corners lie; the cavity is round. Length is one of
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

// CavityTool builds the nut cavity's cutter: a circle of radius NutCavityRadius, from Cavity to CavityFar.
func CavityTool(ctx context.Context, doc *decad.Document, s Site) (*decad.Body, *sketch.Sketch, error) {
	return cylinderTool(ctx, doc, s, s.Cavity, NutCavityRadius, NutCavityHeight)
}

// HexPocketTool builds a hexagonal nut pocket's cutter from the site's Cavity out through the face its bolt tip ends
// on: NutCavityHeight plus Stock along Dir. The hexagon is centred on the site's line with its corners on the circle of
// radius NutCavityRadius, the first along Corner. Each side is held by its distance from the line, the first flat is
// horizontal and each next side turns 60 degrees from the one before, so the corners lie where the sides meet. The
// corners stay sharp, matching the nut's sharp corners. The sketch lies on Across(Cavity): U is Corner and V is
// Dir x Corner, so it extrudes along Dir.
func HexPocketTool(ctx context.Context, doc *decad.Document, s Site) (*decad.Body, *sketch.Sketch, error) {
	sk, err := newSketch(s, s.Cavity)
	if err != nil {
		return nil, nil, err
	}
	rad := NutCavityRadius
	flat := rad * math.Sqrt(3) / 2
	var corners [6]*sketch.Point
	for i := range corners {
		a := float64(i) * math.Pi / 3
		corners[i] = sk.CreatePoint(rad*math.Cos(a), rad*math.Sin(a))
	}
	var sides [6]*sketch.Line
	for i := range sides {
		sides[i] = sk.CreateLine(corners[i], corners[(i+1)%6])
		sk.AddConstraint(sketch.NewDistancePointLine(sk.Origin(), sides[i], flat))
	}
	// sides[1] runs from the corner at 60 degrees to the one at 120 degrees: a flat.
	sk.AddConstraint(sketch.NewHorizontal(sides[1]))
	for i := range 5 {
		sk.AddConstraint(sketch.NewAngle(sides[i], sides[i+1], 60))
	}
	body, err := extrude(ctx, doc, sk, NutCavityHeight+Stock)
	return body, sk, err
}

// RoundRadius is the radius of the rounds RoundedHoleTool and RoundedCavityTool put on a tool's end corners, and
// roundChords is the number of chords each round is drawn with. Eight chords of a 0.55 mm round turn 11.25° each,
// 0.11 mm apart, so no 0.4 mm of a section through the line turns more than 45° ("Corners and roots" in
// docs/model-pipeline.md). The tools are for a part printed with the site's line parallel to the bed, such as a split
// shaft's half, where a section through the line shows the tool's ends as corners. A section off the line crosses a
// round nearer its edge and turns more sharply; RoundRadius is the largest round that keeps the nut's corner clearance.
const (
	RoundRadius = 0.55
	roundChords = 8
)

// RoundedHoleTool builds the clearance hole's cutter, as HoleTool does, with its far end's corner rounded by
// RoundRadius: a revolved meridian about the site's line.
func RoundedHoleTool(ctx context.Context, doc *decad.Document, s Site) (*decad.Body, *sketch.Sketch, error) {
	end := s.Tip() + Stock
	pts := [][2]float64{{-Stock, 0}, {-Stock, HoleRadius}}
	pts = append(pts, quarterRound(end-RoundRadius, HoleRadius-RoundRadius, 0)...)
	pts = append(pts, [2]float64{end, 0})
	return revolveTool(ctx, doc, s, pts)
}

// RoundedCavityTool builds the nut cavity's cutter, as CavityTool does, with both outer corners rounded by
// RoundRadius. At the nut's faces, 0.2 mm inside the cavity's faces, the rounds leave 0.07 mm of the 0.2 mm radial
// clearance past the nut's corners, and the flat end faces reach radius 1.96, inside the nut's 2.0 mm half width.
func RoundedCavityTool(ctx context.Context, doc *decad.Document, s Site) (*decad.Body, *sketch.Sketch, error) {
	near, far := s.Cavity, s.CavityFar()
	pts := [][2]float64{{near, 0}}
	pts = append(pts, quarterRound(near+RoundRadius, NutCavityRadius-RoundRadius, math.Pi)...)
	pts = append(pts, quarterRound(far-RoundRadius, NutCavityRadius-RoundRadius, 0)...)
	pts = append(pts, [2]float64{far, 0})
	return revolveTool(ctx, doc, s, pts)
}

// quarterRound returns the roundChords+1 points (along, radius) of the quarter round of RoundRadius about (t, r) that
// runs from its far side's radial end to its top for start 0, or from its near side's radial end up to its top for
// start π, in the order a meridian walks them from the axis outward and back.
func quarterRound(t, r, start float64) [][2]float64 {
	out := make([][2]float64, 0, roundChords+1)
	for i := 0; i <= roundChords; i++ {
		f := float64(i) / roundChords
		var a float64
		if start == 0 {
			a = math.Pi / 2 * (1 - f) // from the top (90°) down to the far end (0°)
		} else {
			a = math.Pi - math.Pi/2*f // from the near end (180°) up to the top (90°)
		}
		out = append(out, [2]float64{t + RoundRadius*math.Cos(a), r + RoundRadius*math.Sin(a)})
	}
	return out
}

// revolveTool revolves the closed meridian pts, each (distance along the site's line from Head, radius), about the
// site's line. The sketch plane holds the line: U along Dir, V along Corner, its origin at Head. Every vertex is placed
// by its horizontal and vertical distance from the origin.
func revolveTool(ctx context.Context, doc *decad.Document, s Site, pts [][2]float64) (*decad.Body, *sketch.Sketch, error) {
	f, err := r3.NewFrame(s.Head, s.Dir, s.Corner)
	if err != nil {
		return nil, nil, fmt.Errorf("fastener meridian frame: %w", err)
	}
	w := sketch.NewWorld()
	plane, err := w.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, nil, fmt.Errorf("create fastener plane: %w", err)
	}
	sk, err := w.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create fastener sketch: %w", err)
	}
	points := make([]*sketch.Point, len(pts))
	for i, p := range pts {
		points[i] = sketchcheck.DimensionPoint(sk, p[0], p[1])
	}
	for i := range points {
		sk.CreateLine(points[i], points[(i+1)%len(points)])
	}
	if _, err := sk.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve fastener sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, sk); err != nil {
		return nil, nil, fmt.Errorf("verify fastener sketch: %w", err)
	}
	var chosen *sketch.Profile
	for _, p := range sk.Profiles() {
		if !p.Valid || len(p.Holes) != 0 {
			continue
		}
		if chosen != nil {
			return nil, nil, fmt.Errorf("select fastener profile: several valid profiles")
		}
		chosen = p
	}
	if chosen == nil {
		return nil, nil, fmt.Errorf("select fastener profile: no valid profile")
	}
	body, err := decad.Revolve(ctx, doc, sk, chosen, decad.ConstructionAxis{Origin: s.Head, Dir: s.Dir},
		decad.FullRevolution{})
	if err != nil {
		return nil, nil, fmt.Errorf("revolve fastener tool: %w", err)
	}
	return body, sk, nil
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

// PlaceNut builds an m2nut whose lower face lies NutClearance beyond the cavity's near face, corners on s.Corner.
func PlaceNut(ctx context.Context, doc *decad.Document, s Site) (*m2nut.Model, error) {
	f, err := s.Across(s.Cavity + NutClearance)
	if err != nil {
		return nil, fmt.Errorf("nut frame: %w", err)
	}
	m, err := m2nut.Build(ctx, doc, m2nut.Params{Placement: &f})
	if err != nil {
		return nil, fmt.Errorf("build nut: %w", err)
	}
	return m, nil
}
