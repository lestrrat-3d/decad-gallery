package crankhandle

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// newSketch starts a sketch on the plane f; extrusions run along f's normal U × V.
func newSketch(f r3.Frame) (*sketch.Sketch, error) {
	w := sketch.NewWorld()
	plane, err := w.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, fmt.Errorf("create sketch plane: %w", err)
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("create sketch: %w", err)
	}
	return s, nil
}

// selectProfile solves s, requires it fully constrained and returns its one valid profile with the given number of
// holes.
func selectProfile(ctx context.Context, s *sketch.Sketch, holes int) (*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("verify sketch: %w", err)
	}
	var chosen *sketch.Profile
	for _, pr := range s.Profiles() {
		if !pr.Valid || len(pr.Holes) != holes {
			continue
		}
		if chosen != nil {
			return nil, fmt.Errorf("several valid profiles with %d holes", holes)
		}
		chosen = pr
	}
	if chosen == nil {
		return nil, fmt.Errorf("no valid profile with %d holes", holes)
	}
	return chosen, nil
}

// extrudeProfile extrudes the profile of s with the given holes by length along its normal.
func extrudeProfile(ctx context.Context, doc *decad.Document, s *sketch.Sketch, holes int, length float64) (*decad.Body, error) {
	pr, err := selectProfile(ctx, s, holes)
	if err != nil {
		return nil, err
	}
	return decad.Extrude(ctx, doc, s, pr, decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
}

// BoreTool extrudes a D bore's cutter on the plane f by length along f's normal: an arc of radius radius about f's
// origin the long way round through -V, closed by its flat at v = flat (f's +V side). It returns the cutter and its
// sketch, fully constrained.
func BoreTool(ctx context.Context, doc *decad.Document, f r3.Frame, radius, flat, length float64) (*decad.Body, *sketch.Sketch, error) {
	s, err := newSketch(f)
	if err != nil {
		return nil, nil, err
	}
	w := math.Sqrt(radius*radius - flat*flat)
	o := s.Origin()
	q1, q2 := s.CreatePoint(-w, flat), s.CreatePoint(w, flat)
	bore := s.CreateArc(o, q1, q2)
	line := s.CreateLine(q2, q1)
	s.AddConstraint(sketch.NewRadius(bore, radius), sketch.NewHorizontal(line), sketch.NewDistancePointLine(o, line, flat))
	body, err := extrudeProfile(ctx, doc, s, 0, length)
	return body, s, err
}

// SlitTool extrudes a slit's cutter on f by length: u within ±SlitWidth/2 from v = from up to
// v = -boreRadius + SlitInBore, closed there by a half circle of radius SlitWidth/2. The round end lies inside the
// bore's outline, so it shows only in the bore's floor. It returns the cutter and its sketch, fully constrained.
func SlitTool(ctx context.Context, doc *decad.Document, f r3.Frame, from, boreRadius, length float64) (*decad.Body, *sketch.Sketch, error) {
	s, err := newSketch(f)
	if err != nil {
		return nil, nil, err
	}
	half := SlitWidth / 2
	y0, y1 := from, -boreRadius+SlitInBore
	yc := y1 - half
	a := sketchcheck.DimensionPoint(s, half, y0)
	// The arc's ends are placed by height: placing their x at ±half would meet the circle at a tangent, a singular solve.
	b, c := s.CreatePoint(half, yc), s.CreatePoint(-half, yc)
	s.AddConstraint(sketch.NewVerticalDistance(s.Origin(), b, yc), sketch.NewVerticalDistance(s.Origin(), c, yc))
	d := sketchcheck.DimensionPoint(s, -half, y0)
	center := sketchcheck.DimensionPoint(s, 0, yc)
	arc := s.CreateArc(center, b, c)
	s.AddConstraint(sketch.NewRadius(arc, half))
	s.CreateLine(a, b)
	s.CreateLine(c, d)
	s.CreateLine(d, a)
	body, err := extrudeProfile(ctx, doc, s, 0, length)
	return body, s, err
}

// NutPocketTool builds a site's hexagonal nut pocket cutter with its floor's outline rounded by NutPocketRound. The
// pocket runs from the site's Cavity, its floor, out through the face its bolt tip ends on: NutCavityHeight plus
// m2fastener.Stock along the bolt. The hexagon is centred on the bolt's axis with its corners on the circle of radius
// m2fastener.NutCavityRadius, the first along the site's Corner. Each side is held by its distance from the axis, the
// first flat is horizontal and each next side turns 60 degrees from the one before, so the corners lie where the sides
// meet. The corners stay sharp, matching the nut's sharp corners. The sketch lies on site.Across(Cavity): U is Corner
// and V is Dir × Corner, so it extrudes along the site's Dir.
func NutPocketTool(ctx context.Context, doc *decad.Document, site m2fastener.Site) (*decad.Body, *sketch.Sketch, error) {
	f, err := site.Across(site.Cavity)
	if err != nil {
		return nil, nil, err
	}
	s, err := newSketch(f)
	if err != nil {
		return nil, nil, err
	}
	rad := m2fastener.NutCavityRadius
	flat := rad * math.Sqrt(3) / 2
	var corners [6]*sketch.Point
	for i := range corners {
		a := float64(i) * math.Pi / 3
		corners[i] = s.CreatePoint(rad*math.Cos(a), rad*math.Sin(a))
	}
	var sides [6]*sketch.Line
	for i := range sides {
		sides[i] = s.CreateLine(corners[i], corners[(i+1)%6])
		s.AddConstraint(sketch.NewDistancePointLine(s.Origin(), sides[i], flat))
	}
	// sides[1] runs from the corner at 60 degrees to the one at 120 degrees: a flat.
	s.AddConstraint(sketch.NewHorizontal(sides[1]))
	for i := range 5 {
		s.AddConstraint(sketch.NewAngle(sides[i], sides[i+1], 60))
	}
	pocket, err := extrudeProfile(ctx, doc, s, 0, m2fastener.NutCavityHeight+m2fastener.Stock)
	if err != nil {
		return nil, s, err
	}
	floor := decad.Edges(decad.OuterLoopOf(decad.CapStart(pocket)))
	if pocket, err = decad.Fillet(ctx, pocket, floor, units.Millimeters(NutPocketRound)); err != nil {
		return nil, s, fmt.Errorf("round floor: %w", err)
	}
	return pocket, s, nil
}

// TailCenters returns the right corner arc's centre and the end arc's centre of a clamp tail whose sides run at
// u = ±halfWidth down to v = sideEnd: (halfWidth - ClampCornerR, sideEnd) and
// (0, sideEnd + sqrt((ClampEndR-ClampCornerR)^2 - (halfWidth-ClampCornerR)^2)). The left corner arc's centre is the
// right one mirrored in u = 0.
func TailCenters(halfWidth, sideEnd float64) (corner, end r3.Vec) {
	dx := halfWidth - ClampCornerR
	corner = r3.Vec{X: dx, Y: sideEnd}
	end = r3.Vec{Y: sideEnd + math.Sqrt((ClampEndR-ClampCornerR)*(ClampEndR-ClampCornerR)-dx*dx)}
	return corner, end
}

// TailReach is how far the tail's end arc reaches below the sketch origin: ClampEndR - end.Y.
func TailReach(halfWidth, sideEnd float64) float64 {
	_, end := TailCenters(halfWidth, sideEnd)
	return ClampEndR - end.Y
}

// SketchTail adds a clamp tail to s below the side lines' lower ends sR = (halfWidth, sideEnd) on right and
// sL = (-halfWidth, sideEnd) on left: the ClampCornerR corner arcs tangent to each side and the ClampEndR end arc
// tangent to both from inside. Every join is tangent, so the tail has no inside corner and no step.
func SketchTail(s *sketch.Sketch, right, left *sketch.Line, sR, sL *sketch.Point, halfWidth, sideEnd float64) {
	cR, cB := TailCenters(halfWidth, sideEnd)
	cL := r3.Vec{X: -cR.X, Y: cR.Y}
	// The corner arc meets the end arc on the line through their centres, ClampEndR from the end arc's centre.
	pR := cB.Add(cR.Sub(cB).Scale(ClampEndR / cR.Sub(cB).Len()))
	pL := r3.Vec{X: -pR.X, Y: pR.Y}
	cRp, cLp, cBp := s.CreatePoint(cR.X, cR.Y), s.CreatePoint(cL.X, cL.Y), s.CreatePoint(cB.X, cB.Y)
	pRp, pLp := s.CreatePoint(pR.X, pR.Y), s.CreatePoint(pL.X, pL.Y)
	arcR := s.CreateArc(cRp, pRp, sR) // counter-clockwise from the end arc up to the right side
	arcL := s.CreateArc(cLp, sL, pLp) // from the left side down to the end arc
	arcB := s.CreateArc(cBp, pLp, pRp)
	s.AddConstraint(
		sketch.NewRadius(arcR, ClampCornerR), sketch.NewTangent(right, arcR),
		sketch.NewRadius(arcL, ClampCornerR), sketch.NewTangent(left, arcL),
		sketch.NewRadius(arcB, ClampEndR), sketch.NewTangentCircles(arcB, arcR, true),
		sketch.NewTangentCircles(arcB, arcL, true),
	)
}
