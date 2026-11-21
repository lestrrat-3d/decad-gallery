package sevensegment

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/fastenertool"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

func indexSketch(f r3.Frame) (*sketch.Sketch, error) {
	w := sketch.NewWorld()
	p, err := w.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, err
	}
	return w.CreateSketch(p)
}

func indexProfile(ctx context.Context, s *sketch.Sketch) (*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("verify: %w", err)
	}
	var found *sketch.Profile
	for _, p := range s.Profiles() {
		if !p.Valid || len(p.Holes) != 0 {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("multiple profiles")
		}
		found = p
	}
	if found == nil {
		return nil, fmt.Errorf("missing profile")
	}
	return found, nil
}

func indexExtrude(ctx context.Context, doc *decad.Document, s *sketch.Sketch, depth float64) (*decad.Body, error) {
	p, err := indexProfile(ctx, s)
	if err != nil {
		return nil, err
	}
	return decad.Extrude(ctx, doc, s, p, decad.Distance{D: units.Millimeters(depth), Dir: decad.Along})
}

func indexOffsetCylinder(ctx context.Context, doc *decad.Document, frame r3.Frame, x, y, radius, depth float64) (*decad.Body, error) {
	s, err := indexSketch(frame)
	if err != nil {
		return nil, err
	}
	c := sketchcheck.DimensionPoint(s, x, y)
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(c, radius), radius))
	return indexExtrude(ctx, doc, s, depth)
}

func indexWheel(ctx context.Context, doc *decad.Document) (*decad.Body, error) {
	f, err := r3.NewFrame(r3.Vec{Y: -61.45, Z: 28}, r3.Vec{Y: -1}, r3.Vec{X: 1})
	if err != nil {
		return nil, err
	}
	s, err := indexSketch(f)
	if err != nil {
		return nil, err
	}
	xy := [][2]float64{{0, 0}, {0, 6}, {3.9, 6}, {3.9, 25.75}, {7.9, 25.75}, {7.9, 6}, {12, 6}, {12, 0}}
	pts := make([]*sketch.Point, len(xy))
	lines := make([]*sketch.Line, len(xy))
	pts[0] = s.Origin()
	for i := 1; i < len(xy); i++ {
		pts[i] = s.CreatePoint(xy[i][0], xy[i][1])
	}
	for i := range xy {
		lines[i] = s.CreateLine(pts[i], pts[(i+1)%len(xy)])
	}
	for _, i := range []int{0, 2, 4, 6} {
		s.AddConstraint(sketch.NewVertical(lines[i]))
	}
	for _, i := range []int{1, 3, 5, 7} {
		s.AddConstraint(sketch.NewHorizontal(lines[i]))
	}
	s.AddConstraint(sketch.NewVerticalDistance(pts[0], pts[1], 6), sketch.NewVerticalDistance(pts[0], pts[3], 25.75),
		sketch.NewHorizontalPoints(pts[1], pts[5]), sketch.NewHorizontalDistance(pts[0], pts[2], 3.9),
		sketch.NewHorizontalDistance(pts[0], pts[4], 7.9), sketch.NewHorizontalDistance(pts[0], pts[6], 12))
	p, err := indexProfile(ctx, s)
	if err != nil {
		return nil, fmt.Errorf("wheel meridian: %w", err)
	}
	body, err := decad.Revolve(ctx, doc, s, p, decad.ConstructionAxis{Origin: f.Origin(), Dir: f.U()}, decad.FullRevolution{})
	if err != nil {
		return nil, fmt.Errorf("wheel revolve: %w", err)
	}
	bf, err := r3.NewFrame(r3.Vec{Y: -60.45, Z: 28}, r3.Vec{Z: 1}, r3.Vec{X: -1})
	if err != nil {
		return nil, err
	}
	bore, err := indexSketch(bf)
	if err != nil {
		return nil, err
	}
	h := math.Sqrt(2.55*2.55 - 2.05*2.05)
	a, b := bore.CreatePoint(2.05, h), bore.CreatePoint(2.05, -h)
	arc := bore.CreateArc(bore.Origin(), a, b)
	flat := bore.CreateLine(b, a)
	bore.AddConstraint(sketch.NewRadius(arc, 2.55), sketch.NewVertical(flat), sketch.NewDistancePointLine(bore.Origin(), flat, 2.05))
	tool, err := indexExtrude(ctx, doc, bore, 14)
	if err != nil {
		return nil, fmt.Errorf("wheel bore: %w", err)
	}
	body, err = decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("wheel bore cut: %w", err)
	}
	base := math.Atan2(24, 10)
	for i := range 10 {
		angle := base - math.Pi/10 + float64(i)*math.Pi/5
		radial := r3.Vec{X: math.Cos(angle), Z: math.Sin(angle)}
		tangent := r3.Vec{X: -math.Sin(angle), Z: math.Cos(angle)}
		sf, e := r3.NewFrame(r3.Vec{Y: -64.45, Z: 28}, radial, tangent)
		if e != nil {
			return nil, e
		}
		slot, e := indexSketch(sf)
		if e != nil {
			return nil, e
		}
		c := sketchcheck.DimensionPoint(slot, 17.966, 0)
		u, l := slot.CreatePoint(17.966, 2), slot.CreatePoint(17.966, -2)
		ou, ol := slot.CreatePoint(27, 2), slot.CreatePoint(27, -2)
		round := slot.CreateArc(c, u, l)
		bottom := slot.CreateLine(l, ol)
		cap := slot.CreateLine(ol, ou)
		top := slot.CreateLine(ou, u)
		slot.AddConstraint(sketch.NewRadius(round, 2), sketch.NewHorizontal(bottom), sketch.NewHorizontal(top),
			sketch.NewVertical(cap), sketch.NewDistancePointLine(slot.Origin(), cap, 27), sketch.NewTangent(bottom, round), sketch.NewTangent(top, round))
		tool, e := indexExtrude(ctx, doc, slot, 6)
		if e != nil {
			return nil, fmt.Errorf("slot %d: %w", i, e)
		}
		body, e = decad.Cut(ctx, body, tool)
		if e != nil {
			return nil, fmt.Errorf("slot cut %d: %w", i, e)
		}
	}
	for i := range 10 {
		angle := base + float64(i)*math.Pi/5
		radial := r3.Vec{X: math.Cos(angle), Z: math.Sin(angle)}
		tangent := r3.Vec{X: -math.Sin(angle), Z: math.Cos(angle)}
		hf, e := r3.NewFrame(r3.Vec{Y: -64.45, Z: 28}, radial, tangent)
		if e != nil {
			return nil, e
		}
		tool, e := indexOffsetCylinder(ctx, doc, hf, 20, 0, 2, 6)
		if e != nil {
			return nil, fmt.Errorf("hole %d: %w", i, e)
		}
		body, e = decad.Cut(ctx, body, tool)
		if e != nil {
			return nil, fmt.Errorf("hole cut %d: %w", i, e)
		}
	}
	return body, nil
}

func indexFrame(phase float64) (r3.Frame, error) {
	a := math.Atan2(24, 10)
	A := a + 3*math.Pi/5
	B := math.Atan2(20*math.Sin(A)-24, 20*math.Cos(A)-10) + phase*math.Pi/180
	u := r3.Vec{X: math.Cos(B), Z: math.Sin(B)}
	v := r3.Vec{X: -math.Sin(B), Z: math.Cos(B)}
	return r3.NewFrame(r3.Vec{X: 10, Y: -69.75, Z: 52}, u, v)
}

func indexShaft(ctx context.Context, doc *decad.Document, frame r3.Frame) (*decad.Body, *decad.Body, error) {
	axis := r3.Vec{Y: -1}
	f, err := r3.NewFrame(frame.Origin(), axis, frame.U())
	if err != nil {
		return nil, nil, err
	}
	pts := [][2]float64{{0, 0}, {0, 3.5}, {1, 3.5}, {1, 2.5}, {62.25, 2.5}, {62.25, 2.4},
		{62.95, 2.4}, {62.95, 2.5}, {79.85, 2.5}, {79.85, 2.4}, {80.55, 2.4}, {80.55, 2.5},
		{90.55, 2.5}, {90.55, 2.4}, {91.25, 2.4}, {91.25, 2.5}, {91.75, 2.5}, {91.75, 0}}
	s, err := indexSketch(f)
	if err != nil {
		return nil, nil, err
	}
	points := make([]*sketch.Point, len(pts))
	points[0] = s.Origin()
	for i := 1; i < len(pts); i++ {
		points[i] = s.CreatePoint(pts[i][0], pts[i][1])
	}
	for i := range pts {
		line := s.CreateLine(points[i], points[(i+1)%len(pts)])
		if pts[i][0] == pts[(i+1)%len(pts)][0] {
			s.AddConstraint(sketch.NewVertical(line))
		} else {
			s.AddConstraint(sketch.NewHorizontal(line))
		}
	}
	for _, i := range []int{2, 4, 6, 8, 10, 12, 14, 16} {
		s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), points[i], pts[i][0]))
	}
	for _, i := range []int{1, 3, 5} {
		s.AddConstraint(sketch.NewVerticalDistance(s.Origin(), points[i], pts[i][1]))
	}
	for _, i := range []int{7, 11, 15} {
		s.AddConstraint(sketch.NewHorizontalPoints(points[3], points[i]))
	}
	for _, i := range []int{9, 13} {
		s.AddConstraint(sketch.NewHorizontalPoints(points[5], points[i]))
	}
	profile, err := indexProfile(ctx, s)
	if err != nil {
		return nil, nil, fmt.Errorf("input shaft meridian: %w", err)
	}
	shaft, err := decad.Revolve(ctx, doc, s, profile, decad.ConstructionAxis{Origin: f.Origin(), Dir: f.U()}, decad.FullRevolution{})
	if err != nil {
		return nil, nil, fmt.Errorf("input shaft: %w", err)
	}
	for _, station := range []float64{2.6, 22.35, 37.25, 59.25} {
		origin := frame.Origin().Add(axis.Scale(station)).Add(frame.U().Scale(-3))
		hf, e := r3.NewFrame(origin, axis, frame.V().Scale(-1))
		if e != nil {
			return nil, nil, e
		}
		hole, _, e := fastenertool.Cylinder(ctx, doc, hf, 1, 6)
		if e != nil {
			return nil, nil, fmt.Errorf("shaft hole %g: %w", station, e)
		}
		shaft, e = decad.Cut(ctx, shaft, hole)
		if e != nil {
			return nil, nil, fmt.Errorf("shaft hole cut %g: %w", station, e)
		}
	}
	keyOrigin := frame.Origin().Add(axis.Scale(2.6)).Add(frame.U().Scale(-10))
	kf, err := r3.NewFrame(keyOrigin, axis, frame.V().Scale(-1))
	if err != nil {
		return nil, nil, err
	}
	key, _, err := fastenertool.Cylinder(ctx, doc, kf, 1, 20)
	if err != nil {
		return nil, nil, fmt.Errorf("crank key: %w", err)
	}
	return shaft, key, nil
}

func indexCrank(ctx context.Context, doc *decad.Document, frame r3.Frame) (*decad.Body, error) {
	disk, _, err := fastenertool.Cylinder(ctx, doc, frame, 11.2, 4.8)
	if err != nil {
		return nil, fmt.Errorf("crank disk: %w", err)
	}
	start := frame.Origin().Add(r3.Vec{Y: 1})
	cutFrame, err := r3.NewFrame(start, frame.U(), frame.V())
	if err != nil {
		return nil, err
	}
	s, err := indexSketch(cutFrame)
	if err != nil {
		return nil, err
	}
	ax := s.CreateLine(s.Origin(), sketchcheck.DimensionPoint(s, 12, 0))
	ax.SetConstruction(true)
	ay := s.CreateLine(s.Origin(), sketchcheck.DimensionPoint(s, 0, 12))
	ay.SetConstruction(true)
	x := []float64{math.Sqrt(2.55*2.55 - 1.06*1.06), 8.3, 8.8, 9.6, 10, 10.1}
	y := []float64{1.06, 1.06, 1.01, 1.01, 1.06, 1.06}
	var ur, ul, lr, ll [6]*sketch.Point
	for i := range x {
		ur[i] = s.CreatePoint(x[i], y[i])
		ul[i] = s.CreatePoint(-x[i], y[i])
		lr[i] = s.CreatePoint(x[i], -y[i])
		ll[i] = s.CreatePoint(-x[i], -y[i])
		if i == 0 {
			s.AddConstraint(sketch.NewHorizontalPoints(ur[i], ul[i]), sketch.NewSymmetric(ur[i], lr[i], ax),
				sketch.NewVerticalPoints(ul[i], ll[i]))
		} else {
			s.AddConstraint(sketch.NewSymmetric(ur[i], ul[i], ay), sketch.NewSymmetric(ur[i], lr[i], ax),
				sketch.NewSymmetric(ul[i], ll[i], ax))
		}
		if i > 0 {
			s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), ur[i], x[i]))
		}
	}
	s.AddConstraint(sketch.NewVerticalDistance(s.Origin(), ur[0], 1.06), sketch.NewVerticalDistance(s.Origin(), ur[2], 1.01),
		sketch.NewHorizontalPoints(ur[0], ur[1]), sketch.NewHorizontalPoints(ur[0], ur[4]),
		sketch.NewHorizontalPoints(ur[0], ur[5]), sketch.NewHorizontalPoints(ur[2], ur[3]))
	top := s.CreateArc(s.Origin(), ur[0], ul[0])
	s.CreateArc(s.Origin(), ll[0], lr[0])
	s.AddConstraint(sketch.NewRadius(top, 2.55))
	for i := 0; i < 5; i++ {
		s.CreateLine(ur[i], ur[i+1])
		s.CreateLine(ul[i], ul[i+1])
		s.CreateLine(lr[i], lr[i+1])
		s.CreateLine(ll[i], ll[i+1])
	}
	s.CreateLine(ur[5], lr[5])
	s.CreateLine(ul[5], ll[5])
	tool, err := indexExtrude(ctx, doc, s, 6.8)
	if err != nil {
		return nil, fmt.Errorf("crank keyed bore: %w", err)
	}
	disk, err = decad.Cut(ctx, disk, tool)
	if err != nil {
		return nil, fmt.Errorf("crank keyed bore cut: %w", err)
	}
	recessFrame, err := r3.NewFrame(frame.Origin().Add(r3.Vec{Y: .1}), frame.U(), frame.V())
	if err != nil {
		return nil, err
	}
	recess, _, err := fastenertool.Cylinder(ctx, doc, recessFrame, 3.6, 1.2)
	if err != nil {
		return nil, fmt.Errorf("crank recess: %w", err)
	}
	disk, err = decad.Cut(ctx, disk, recess)
	if err != nil {
		return nil, fmt.Errorf("crank recess cut: %w", err)
	}
	a := math.Atan2(24, 10)
	A := a + 3*math.Pi/5
	B := math.Atan2(20*math.Sin(A)-24, 20*math.Cos(A)-10)
	angle := a - B
	throw := 26 * math.Sin(math.Pi/10)
	for _, cut := range []struct{ station, radius, depth float64 }{{-1, 1.55, 6.8}, {0, 2.08, .78}, {4, 2.08, .8}} {
		cf, e := r3.NewFrame(frame.Origin().Add(r3.Vec{Y: -cut.station}), frame.U(), frame.V())
		if e != nil {
			return nil, e
		}
		hole, e := indexOffsetCylinder(ctx, doc, cf, throw*math.Cos(angle), throw*math.Sin(angle), cut.radius, cut.depth)
		if e != nil {
			return nil, e
		}
		disk, e = decad.Cut(ctx, disk, hole)
		if e != nil {
			return nil, fmt.Errorf("crank rivet cut %g: %w", cut.station, e)
		}
	}
	return disk, nil
}

func indexRivet(ctx context.Context, doc *decad.Document, frame r3.Frame) (*decad.Body, error) {
	a := math.Atan2(24, 10)
	A := a + 3*math.Pi/5
	B := math.Atan2(20*math.Sin(A)-24, 20*math.Cos(A)-10)
	throw := 26 * math.Sin(math.Pi/10)
	center := frame.Origin().Add(frame.U().Scale(throw * math.Cos(a-B))).Add(frame.V().Scale(throw * math.Sin(a-B)))
	var parts []*decad.Body
	for _, dim := range []struct{ station, radius, length float64 }{{-4.4, 1.5, 9.14}, {.05, 1.925, .65}, {4.10, 1.925, .64}} {
		f, err := r3.NewFrame(center.Add(r3.Vec{Y: -dim.station}), frame.U(), frame.V())
		if err != nil {
			return nil, err
		}
		body, _, err := fastenertool.Cylinder(ctx, doc, f, dim.radius, dim.length)
		if err != nil {
			return nil, fmt.Errorf("rivet cylinder: %w", err)
		}
		parts = append(parts, body)
	}
	rivet, err := decad.Union(ctx, parts[0], parts[1], parts[2])
	if err != nil {
		return nil, fmt.Errorf("join rivet: %w", err)
	}
	return rivet, nil
}

type buttonIndex struct {
	Wheel, Crank, Rivet, Shaft, Key *decad.Body
}

func indexAdvance(phi float64) float64 {
	if phi <= 108 {
		return 0
	}
	if phi >= 252 {
		return 36
	}
	p := phi * math.Pi / 180
	s := math.Sin(math.Pi / 10)
	return 18 - math.Atan2(s*math.Sin(p), 1+s*math.Cos(p))*180/math.Pi
}

func buildButtonIndex(ctx context.Context, doc *decad.Document, phase float64) (*buttonIndex, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("button index requires context and document")
	}
	if math.IsNaN(phase) || math.IsInf(phase, 0) || phase < 0 || phase > 360 {
		return nil, fmt.Errorf("button index phase %g outside 0..360", phase)
	}
	frame, err := indexFrame(0)
	if err != nil {
		return nil, fmt.Errorf("input frame: %w", err)
	}
	wheel, err := indexWheel(ctx, doc)
	if err != nil {
		return nil, err
	}
	turn, err := r3.RotationAround(r3.Vec{Z: 28}, r3.Vec{Y: 1}, units.Degrees(324+indexAdvance(phase)))
	if err != nil {
		return nil, err
	}
	wheel, err = decad.Transform(ctx, wheel, turn)
	if err != nil {
		return nil, fmt.Errorf("rotate wheel: %w", err)
	}
	shaft, key, err := indexShaft(ctx, doc, frame)
	if err != nil {
		return nil, err
	}
	crank, err := indexCrank(ctx, doc, frame)
	if err != nil {
		return nil, err
	}
	rivet, err := indexRivet(ctx, doc, frame)
	if err != nil {
		return nil, err
	}
	inputTurn, err := r3.RotationAround(frame.Origin(), r3.Vec{Y: -1}, units.Degrees(phase))
	if err != nil {
		return nil, fmt.Errorf("input rotation: %w", err)
	}
	shaft, err = decad.Transform(ctx, shaft, inputTurn)
	if err != nil {
		return nil, fmt.Errorf("rotate input shaft: %w", err)
	}
	key, err = decad.Transform(ctx, key, inputTurn)
	if err != nil {
		return nil, fmt.Errorf("rotate crank key: %w", err)
	}
	crank, err = decad.Transform(ctx, crank, inputTurn)
	if err != nil {
		return nil, fmt.Errorf("rotate crank: %w", err)
	}
	rivet, err = decad.Transform(ctx, rivet, inputTurn)
	if err != nil {
		return nil, fmt.Errorf("rotate drive rivet: %w", err)
	}
	out := &buttonIndex{Wheel: wheel, Crank: crank, Rivet: rivet, Shaft: shaft, Key: key}
	container, err := doc.Root().NewUniqueChild("Button Index")
	if err != nil {
		return nil, err
	}
	for _, item := range []struct {
		name string
		body *decad.Body
	}{{"Geneva Wheel", wheel}, {"Crank", crank}, {"Drive Rivet", rivet}, {"Input Shaft", shaft}, {"Crank Key", key}} {
		group, e := container.NewChild(item.name)
		if e != nil {
			return nil, e
		}
		if e = group.AddBody(item.body); e != nil {
			return nil, fmt.Errorf("place %s: %w", item.name, e)
		}
	}
	return out, nil
}
