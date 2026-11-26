// Package fastenertool builds the size-free cutter bodies that fastener packages such as m2fastener share: a
// cylinder, a hexagonal prism and a revolved meridian, each from a fully constrained sketch.
package fastenertool

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Cylinder builds a cylinder of the given radius and length. Its circle lies on f, centred on f's origin, and it
// extrudes along f's normal.
func Cylinder(ctx context.Context, doc *decad.Document, f r3.Frame, radius, length float64) (*decad.Body,
	*sketch.Sketch, error) {
	sk, err := newSketch(f)
	if err != nil {
		return nil, nil, err
	}
	sk.AddConstraint(sketch.NewRadius(sk.CreateCircle(sk.Origin(), radius), radius))
	body, err := extrude(ctx, doc, sk, length)
	return body, sk, err
}

// HexPrism builds a hexagonal prism of the given depth along f's normal. The hexagon is centred on f's origin with its
// corners on the circle of radius cornerRadius, the first along f's U. Each side is held by its distance from the
// origin, the first flat is horizontal and each next side turns 60 degrees from the one before, so the corners lie
// where the sides meet. The corners stay sharp.
func HexPrism(ctx context.Context, doc *decad.Document, f r3.Frame, cornerRadius, depth float64) (*decad.Body,
	*sketch.Sketch, error) {
	sk, err := newSketch(f)
	if err != nil {
		return nil, nil, err
	}
	flat := cornerRadius * math.Sqrt(3) / 2
	var corners [6]*sketch.Point
	for i := range corners {
		a := float64(i) * math.Pi / 3
		corners[i] = sk.CreatePoint(cornerRadius*math.Cos(a), cornerRadius*math.Sin(a))
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
	body, err := extrude(ctx, doc, sk, depth)
	return body, sk, err
}

// RoundedRevolve revolves the closed meridian pts, each (distance along f's U, distance along f's V), about the line
// through f's origin along f's U. Every vertex is placed by its horizontal and vertical distance from the origin.
func RoundedRevolve(ctx context.Context, doc *decad.Document, f r3.Frame, pts [][2]float64) (*decad.Body,
	*sketch.Sketch, error) {
	sk, err := newSketch(f)
	if err != nil {
		return nil, nil, err
	}
	points := make([]*sketch.Point, len(pts))
	for i, p := range pts {
		points[i] = sketchcheck.DimensionPoint(sk, p[0], p[1])
	}
	for i := range points {
		sk.CreateLine(points[i], points[(i+1)%len(points)])
	}
	chosen, err := profile(ctx, sk)
	if err != nil {
		return nil, nil, err
	}
	body, err := decad.Revolve(doc, sk, chosen, decad.ConstructionAxis{Origin: f.Origin(), Dir: f.U()},
		decad.FullRevolution{}).Do(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("revolve fastener tool: %w", err)
	}
	return body, sk, nil
}

// QuarterRound returns the chords+1 points (along, radius) of the quarter round of the given radius about (t, r). For
// start 0 it runs from the round's top down to its far side's radial end. For start π it runs from the near side's
// radial end up to the top. The points come in the order a meridian walks them from the axis outward and back.
func QuarterRound(t, r, radius, start float64, chords int) [][2]float64 {
	out := make([][2]float64, 0, chords+1)
	for i := 0; i <= chords; i++ {
		f := float64(i) / float64(chords)
		var a float64
		if start == 0 {
			a = math.Pi / 2 * (1 - f) // from the top (90°) down to the far end (0°)
		} else {
			a = math.Pi - math.Pi/2*f // from the near end (180°) up to the top (90°)
		}
		out = append(out, [2]float64{t + radius*math.Cos(a), r + radius*math.Sin(a)})
	}
	return out
}

func newSketch(f r3.Frame) (*sketch.Sketch, error) {
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

// profile solves sk, requires it fully constrained and returns its one valid profile without holes.
func profile(ctx context.Context, sk *sketch.Sketch) (*sketch.Profile, error) {
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
	return chosen, nil
}

// extrude extrudes sk's one valid profile along the normal.
func extrude(ctx context.Context, doc *decad.Document, sk *sketch.Sketch, length float64) (*decad.Body, error) {
	chosen, err := profile(ctx, sk)
	if err != nil {
		return nil, err
	}
	body, err := decad.Extrude(doc, sk, chosen, decad.Distance{D: units.Millimeters(length), Dir: decad.Along}).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("extrude fastener tool: %w", err)
	}
	return body, nil
}
