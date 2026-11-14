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
)

// plateSketch sketches the plate in the local XY plane: the hub circle about the origin cut by the tail's flat
// at y = -t, the arm's end circle about (0, Throw), the two lines tangent to both circles, and the D bore. It
// returns the sketch and the plate's profile, the one with the bore as its hole.
func plateSketch(ctx context.Context, p Params, dm Dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	fail := func(step string, err error) (*sketch.Sketch, *sketch.Profile, error) {
		return nil, nil, fmt.Errorf("crankhandle: plate sketch: %s: %w", step, err)
	}
	world := sketch.NewWorld()
	s, err := world.CreateSketch(world.XY())
	if err != nil {
		return fail("create", err)
	}
	o := s.Origin()
	rh, re, t, half := dm.HubRadius, dm.EndRadius, dm.TailDistance, dm.TailHalf

	// The outward normal of the right tangent line: its distance from O is rh and from E is re.
	ny := (rh - re) / p.Throw
	nx := math.Sqrt(1 - ny*ny)

	e := s.CreatePoint(0, p.Throw)
	hr := s.CreatePoint(rh*nx, rh*ny)
	er := s.CreatePoint(re*nx, p.Throw+re*ny)
	el := s.CreatePoint(-re*nx, p.Throw+re*ny)
	hl := s.CreatePoint(-rh*nx, rh*ny)
	tl := s.CreatePoint(-half, -t)
	tr := s.CreatePoint(half, -t)

	hubRight := s.CreateArc(o, tr, hr)
	right := s.CreateLine(hr, er)
	end := s.CreateArc(e, er, el)
	left := s.CreateLine(el, hl)
	hubLeft := s.CreateArc(o, hl, tl)
	tail := s.CreateLine(tl, tr)
	s.AddConstraint(
		sketch.NewVerticalPoints(o, e), sketch.NewVerticalDistance(o, e, p.Throw),
		sketch.NewRadius(hubRight, rh), sketch.NewEqualRadius(hubRight, hubLeft), sketch.NewRadius(end, re),
		sketch.NewTangent(right, hubRight), sketch.NewTangent(right, end),
		sketch.NewTangent(left, end), sketch.NewTangent(left, hubLeft),
		sketch.NewHorizontal(tail), sketch.NewDistancePointLine(o, tail, t),
	)

	// The D bore: an arc of radius r_b the long way round through +Y, closed by its flat at y = -f_b.
	rb, fb := dm.BoreRadius, dm.BoreFlat
	w := math.Sqrt(rb*rb - fb*fb)
	q1 := s.CreatePoint(w, -fb)
	q2 := s.CreatePoint(-w, -fb)
	bore := s.CreateArc(o, q1, q2)
	flat := s.CreateLine(q2, q1)
	s.AddConstraint(sketch.NewRadius(bore, rb), sketch.NewHorizontal(flat), sketch.NewDistancePointLine(o, flat, fb))

	if _, err := s.Solve(ctx); err != nil {
		return fail("solve", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fail("verify", err)
	}
	var plate *sketch.Profile
	for _, pr := range s.Profiles() {
		if !pr.Valid || len(pr.Holes) != 1 {
			continue
		}
		if plate != nil {
			return fail("select profile", fmt.Errorf("several profiles have one hole"))
		}
		plate = pr
	}
	if plate == nil {
		return fail("select profile", fmt.Errorf("no valid profile has one hole"))
	}
	return s, plate, nil
}

// screwMeridian returns the set screw tool's meridian as (a, r) pairs: a along the site's Dir from the head's top
// face, r from the screw axis. It runs from Stock before the tail face, through the head pocket, the clearance
// hole and the nut cavity, to Stock inside the bore. A 45 degree chamfer of RimChamfer fills the concave rim of the
// pocket's floor and both rims of the cavity.
func screwMeridian(site m2fastener.Site) [][2]float64 {
	const c = RimChamfer
	rp, dp := m2fastener.HeadPocketRadius, m2fastener.HeadPocketDepth
	rh, rn := m2fastener.HoleRadius, m2fastener.NutCavityRadius
	near, far := site.Cavity, site.CavityFar()
	end := site.Tip() + m2fastener.Stock
	return [][2]float64{
		{-m2fastener.Stock, 0}, {-m2fastener.Stock, rp}, {dp - c, rp}, {dp, rp - c}, {dp, rh},
		{near, rh}, {near, rn - c}, {near + c, rn}, {far - c, rn}, {far, rn - c}, {far, rh},
		{end, rh}, {end, 0},
	}
}

// screwTool revolves screwMeridian about the set screw's axis. The sketch plane holds the axis: its U axis is the
// site's Dir and its V axis the plate normal +Z, so every vertex is dimensioned from the origin at the head.
func screwTool(ctx context.Context, doc *decad.Document, site m2fastener.Site) (*decad.Body, *sketch.Sketch, error) {
	fail := func(step string, err error) (*decad.Body, *sketch.Sketch, error) {
		return nil, nil, fmt.Errorf("crankhandle: set screw sketch: %s: %w", step, err)
	}
	f, err := r3.NewFrame(site.Head, site.Dir, r3.Vec{Z: 1})
	if err != nil {
		return fail("frame", err)
	}
	world := sketch.NewWorld()
	plane, err := world.CreatePlaneFromFrame(f)
	if err != nil {
		return fail("plane", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return fail("create", err)
	}
	pts := screwMeridian(site)
	points := make([]*sketch.Point, len(pts))
	for i, pt := range pts {
		points[i] = sketchcheck.DimensionPoint(s, pt[0], pt[1])
	}
	for i := range points {
		s.CreateLine(points[i], points[(i+1)%len(points)])
	}
	if _, err := s.Solve(ctx); err != nil {
		return fail("solve", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fail("verify", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return fail("select profile", fmt.Errorf("want one valid profile, got %d", len(profiles)))
	}
	body, err := decad.Revolve(ctx, doc, s, profiles[0], decad.ConstructionAxis{Origin: site.Head, Dir: site.Dir},
		decad.FullRevolution{})
	if err != nil {
		return fail("revolve", err)
	}
	return body, s, nil
}
