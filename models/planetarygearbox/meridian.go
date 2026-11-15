package planetarygearbox

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// coneRun is the axial length of a 30° half-angle cone per unit of radius change, 1/tan(30°). A cone surface at 30° to
// the shaft axis reads 30° or less in every section parallel to the split plane, see "Split shafts" in
// docs/model-pipeline.md.
var coneRun = math.Sqrt(3)

// vertex is one corner of a meridian outline: the distance x from the axis (the sketch's U direction) and the height z
// (its V direction). expr, when set, is a table expression bound to the corner's distance from the sketch origin.
type vertex struct {
	x, z float64
	expr string
}

// edgeKind is the direction of one outline edge.
type edgeKind int

const (
	edgeVertical edgeKind = iota
	edgeHorizontal
	edgeCone
)

// kindOf returns the direction of the edge from a to b, or an error when it is none of vertical, horizontal or a
// 30° half-angle cone.
func kindOf(a, b vertex) (edgeKind, error) {
	const eps = 1e-9
	dx, dz := b.x-a.x, b.z-a.z
	switch {
	case math.Abs(dx) < eps && math.Abs(dz) < eps:
		return 0, errors.New("zero-length edge")
	case math.Abs(dx) < eps:
		return edgeVertical, nil
	case math.Abs(dz) < eps:
		return edgeHorizontal, nil
	case math.Abs(math.Abs(dz)/math.Abs(dx)-coneRun) < 1e-6:
		return edgeCone, nil
	}
	return 0, fmt.Errorf("edge (%v, %v) to (%v, %v) is not vertical, horizontal or at 30° to the axis", a.x, a.z, b.x, b.z)
}

// meridian draws the closed outline through pts on the world's XZ plane and returns its sketch and its one valid
// profile. The first vertex is dimensioned from the sketch origin. Each later vertex is placed by its edge: a vertical
// edge from the one before it takes a vertical distance, a horizontal or 30° edge takes the vertex's distance from
// the origin, and a 30° edge also takes its angle against the first vertical edge. The last edge back to the first
// vertex takes no constraint of its own. A sketch that is not fully constrained is an error.
func (r *run) meridian(ctx context.Context, what string, pts []vertex) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := r.world.CreateSketch(r.world.XZ())
	if err != nil {
		return nil, nil, fmt.Errorf("%s: sketch: %w", what, err)
	}
	points := make([]*sketch.Point, len(pts))
	points[0] = sketchcheck.DimensionPoint(s, pts[0].x, pts[0].z)
	for i := 1; i < len(pts); i++ {
		points[i] = s.CreatePoint(pts[i].x, pts[i].z)
	}
	lines := make([]*sketch.Line, len(pts))
	kinds := make([]edgeKind, len(pts))
	for i := range pts {
		next := (i + 1) % len(pts)
		lines[i] = s.CreateLine(points[i], points[next])
		if kinds[i], err = kindOf(pts[i], pts[next]); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", what, err)
		}
	}
	// The angle of a 30° edge is measured from the first vertical edge, in the direction both lines are drawn.
	ref := -1
	for i, k := range kinds {
		if k == edgeVertical {
			ref = i
			break
		}
	}
	for i := 1; i < len(pts); i++ {
		edge := i - 1
		switch kinds[edge] {
		case edgeVertical:
			s.AddConstraint(sketch.NewVertical(lines[edge]),
				sketch.NewVerticalDistance(points[edge], points[i], pts[i].z-pts[edge].z))
			continue
		case edgeHorizontal:
			s.AddConstraint(sketch.NewHorizontal(lines[edge]))
		case edgeCone:
			if ref < 0 {
				return nil, nil, fmt.Errorf("%s: a 30° edge needs a vertical edge to measure from", what)
			}
			rx, rz := pts[(ref+1)%len(pts)].x-pts[ref].x, pts[(ref+1)%len(pts)].z-pts[ref].z
			cx, cz := pts[i].x-pts[edge].x, pts[i].z-pts[edge].z
			angle := math.Atan2(rx*cz-rz*cx, rx*cx+rz*cz) * 180 / math.Pi
			s.AddConstraint(sketch.NewAngle(lines[ref], lines[edge], angle))
		}
		dim := sketch.NewHorizontalDistance(s.Origin(), points[i], pts[i].x)
		s.AddConstraint(dim)
		if pts[i].expr != "" {
			if err := s.Bind(dim, r.world.Params(), pts[i].expr); err != nil {
				return nil, nil, fmt.Errorf("%s: bind %q: %w", what, pts[i].expr, err)
			}
		}
	}
	profile, err := profileWithHoles(ctx, s, 0, what)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// revolveMeridian revolves the outline through pts, which starts and ends on the axis, a full turn about the sketch's
// vertical axis through the origin, which is the gearbox axis.
func (r *run) revolveMeridian(ctx context.Context, what string, pts []vertex) (*decad.Body, error) {
	s, profile, err := r.meridian(ctx, what, pts)
	if err != nil {
		return nil, err
	}
	axis := decad.SketchLine{Start: decad.Point2{U: 0, V: 0}, End: decad.Point2{U: 0, V: 1}}
	body, err := decad.Revolve(ctx, r.doc, s, profile, axis, decad.FullRevolution{})
	if err != nil {
		return nil, fmt.Errorf("%s: revolve: %w", what, err)
	}
	return body, nil
}

// extrudeMeridian extrudes the outline through pts half each way along Y from the XZ plane.
func (r *run) extrudeMeridian(ctx context.Context, what string, pts []vertex, half float64) (*decad.Body, error) {
	s, profile, err := r.meridian(ctx, what, pts)
	if err != nil {
		return nil, err
	}
	body, err := decad.Extrude(ctx, r.doc, s, profile, decad.Symmetric{D: units.Millimeters(half)})
	if err != nil {
		return nil, fmt.Errorf("%s: extrude: %w", what, err)
	}
	return body, nil
}
