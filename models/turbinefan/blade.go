package turbinefan

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

type Blade struct {
	Body     *decad.Body
	Sketches []*sketch.Sketch
}

const (
	designAttack          = 5 * math.Pi / 180
	refPitch              = 25 * math.Pi / 180
	DefaultBladePitch     = refPitch
	MinBladePitch         = 10 * math.Pi / 180
	MaxBladePitch         = 40 * math.Pi / 180
	BladeSocketFloorRound = 0.5
)

type bladeStation struct{ r, chord, ratio, camber, ellipse float64 }

var bladeAirfoil = [...]bladeStation{
	{19, 11.5, .60, .010, .60}, {21.5, 15.5, .38, .025, .25}, {25, 17.8, .26, .035, 0},
	{30, 16.5, .22, .040, 0}, {38, 14.5, .19, .040, 0}, {46, 12.5, .18, .040, 0},
	{55, 10.5, .17, .035, 0}, {64, 8, .22, .030, 0},
}

func bladeStations(size Size) []bladeStation {
	out := []bladeStation{
		{size.RootRadius, size.HubScale * 8, 1, 0, 1},
		{size.RootRadius + 1.9*size.HubScale, size.HubScale * 8, 1, 0, 1},
	}
	for _, a := range bladeAirfoil {
		a.r *= size.Scale
		a.chord *= size.Scale
		out = append(out, a)
	}
	return out
}

func stationsFor(size Size) []bladeStation { return bladeStations(size) }

func bladeMeanLine(x, m float64) (float64, float64) {
	p := .4
	if x < p {
		return m / (p * p) * (2*p*x - x*x), 2 * m / (p * p) * (p - x)
	}
	return m / ((1 - p) * (1 - p)) * ((1 - 2*p) + 2*p*x - x*x), 2 * m / ((1 - p) * (1 - p)) * (p - x)
}

func bladePoint(st bladeStation, beta, xi float64, suction bool) (float64, float64) {
	c := st.chord
	if xi < 0 {
		xi = 0
	}
	if xi >= 1 {
		xi = 1 - 1e-8
	}
	yc, _ := bladeMeanLine(xi, st.camber)
	t := st.ratio * c
	n := (t/2-.15)*(.2969*math.Sqrt(xi)-.1260*xi-.3516*xi*xi+.2843*xi*xi*xi-.1036*math.Pow(xi, 4))/.10002 + xi/2
	ell := t*math.Sqrt(xi*(1-xi)) + xi/2
	delta := (1-st.ellipse)*n + st.ellipse*ell
	if !suction {
		delta = -delta
	}
	eta := c*yc + delta
	xiMM := c * xi
	stackX := c * (.4 + .1*st.ellipse)
	stackYFraction, _ := bladeMeanLine(.4+.1*st.ellipse, st.camber)
	stackY := c * stackYFraction
	a, b := stackX-xiMM, eta-stackY
	return a*math.Cos(beta) - b*math.Sin(beta), -a*math.Sin(beta) - b*math.Cos(beta)
}

func bladeBeta(size Size, r, pitch float64) float64 {
	return designAttack + math.Atan(math.Tan(refPitch-designAttack)*size.RefRadius/math.Max(r, size.HoldRadius)) + (pitch - refPitch)
}

// BladeSocketTool builds a rounded socket cutter and its driving sketch.
func BladeSocketTool(ctx context.Context, doc *decad.Document, f r3.Frame, size Size) (*decad.Body, *sketch.Sketch, error) {
	if !size.valid() {
		return nil, nil, fmt.Errorf("turbinefan: invalid blade size")
	}
	w := sketch.NewWorld()
	pl, err := w.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, nil, err
	}
	s, err := w.CreateSketch(pl)
	if err != nil {
		return nil, nil, err
	}
	circle := s.CreateCircle(s.Origin(), size.BladeSocketRadius)
	s.AddConstraint(sketch.NewRadius(circle, size.BladeSocketRadius))
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, err
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid || len(profiles[0].Holes) != 0 {
		return nil, nil, fmt.Errorf("turbinefan: invalid blade socket profile")
	}
	body, err := decad.Extrude(ctx, doc, s, profiles[0], decad.Distance{D: units.Millimeters(size.HubRadius + 1 - size.BladeSocketStart), Dir: decad.Along})
	if err != nil {
		return nil, nil, err
	}
	body, err = decad.Fillet(ctx, body, decad.Edges(decad.OuterLoopOf(decad.CapStart(body))).Exactly(1), units.Millimeters(BladeSocketFloorRound))
	if err != nil {
		return nil, nil, err
	}
	return body, s, nil
}

func BuildBlade(ctx context.Context, doc *decad.Document, pitch float64, size Size) (*Blade, error) {
	return buildBlade(ctx, doc, pitch, size, false)
}
func BuildBladeMirrored(ctx context.Context, doc *decad.Document, pitch float64, size Size) (*Blade, error) {
	return buildBlade(ctx, doc, pitch, size, true)
}
func buildBlade(ctx context.Context, doc *decad.Document, pitch float64, size Size, mirror bool) (*Blade, error) {
	if doc == nil {
		return nil, fmt.Errorf("turbinefan: nil document")
	}
	if math.IsNaN(pitch) || math.IsInf(pitch, 0) || pitch < MinBladePitch || pitch > MaxBladePitch {
		return nil, fmt.Errorf("turbinefan: blade pitch out of range")
	}
	if !size.valid() {
		return nil, fmt.Errorf("turbinefan: invalid blade size")
	}
	w := sketch.NewWorld()
	xy := w.XY()
	sections := make([]decad.LoftSection, 0, 10)
	sketches := make([]*sketch.Sketch, 0, 13)
	for _, st := range bladeStations(size) {
		pl, err := w.CreateOffsetPlane(xy, st.r)
		if err != nil {
			return nil, err
		}
		s, err := w.CreateSketch(pl)
		if err != nil {
			return nil, err
		}
		beta := bladeBeta(size, st.r, 25*math.Pi/180)
		points := make([]*sketch.Point, 0, 21)
		for k := 10; k >= 1; k-- {
			x := (1 - math.Cos(math.Pi*float64(k)/10)) / 2
			u, v := bladePoint(st, beta, x, true)
			if !mirror {
				u = -u
			}
			points = append(points, sketchcheck.DimensionPoint(s, u, v))
		}
		x0, y0 := bladePoint(st, beta, 0, false)
		if !mirror {
			x0 = -x0
		}
		leading := sketchcheck.DimensionPoint(s, x0, y0)
		points = append(points, leading)
		for k := 1; k <= 10; k++ {
			x := (1 - math.Cos(math.Pi*float64(k)/10)) / 2
			u, v := bladePoint(st, beta, x, false)
			if !mirror {
				u = -u
			}
			points = append(points, sketchcheck.DimensionPoint(s, u, v))
		}
		first, last := points[0], points[len(points)-1]
		tail := s.CreatePoint((first.X()+last.X())/2, (first.Y()+last.Y())/2)
		s.AddConstraint(sketch.NewMidpointOf(tail, first, last))
		if _, err = s.CreateFitSpline(points...); err != nil {
			return nil, err
		}
		s.CreateLine(last, tail)
		s.CreateLine(tail, first)
		if _, err = s.Solve(ctx); err != nil {
			return nil, err
		}
		if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
			return nil, err
		}
		profiles := s.Profiles()
		if len(profiles) != 1 || !profiles[0].Valid || len(profiles[0].Holes) != 0 || len(profiles[0].Outer) != 3 {
			return nil, fmt.Errorf("turbinefan: invalid blade station profile")
		}
		sections = append(sections, decad.LoftSection{Sketch: s, Profile: profiles[0]})
		sketches = append(sketches, s)
	}
	loft, err := decad.LoftSections(ctx, doc, sections...)
	if err != nil {
		return nil, err
	}
	cut, err := w.CreateSketch(xy)
	if err != nil {
		return nil, err
	}
	xlo, xhi := -100*size.Scale, -size.BladeStudRadius
	if mirror {
		xlo, xhi = size.BladeStudRadius, 100*size.Scale
	}
	corners := []r3.Vec{{X: xlo, Y: -100 * size.Scale}, {X: xhi, Y: -100 * size.Scale}, {X: xhi, Y: 100 * size.Scale}, {X: xlo, Y: 100 * size.Scale}}
	pts := make([]*sketch.Point, 4)
	for i, p := range corners {
		pts[i] = sketchcheck.DimensionPoint(cut, p.X, p.Y)
	}
	for i := range pts {
		cut.CreateLine(pts[i], pts[(i+1)%len(pts)])
	}
	if _, err = cut.Solve(ctx); err != nil {
		return nil, err
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, cut); err != nil {
		return nil, err
	}
	cp := cut.Profiles()
	if len(cp) != 1 || len(cp[0].Holes) != 0 {
		return nil, fmt.Errorf("turbinefan: invalid blade trim profile")
	}
	tool, err := decad.Extrude(ctx, doc, cut, cp[0], decad.Distance{D: units.Millimeters(size.TipRadius + 1), Dir: decad.Along})
	if err != nil {
		return nil, err
	}
	sketches = append(sketches, cut)
	body, err := decad.Cut(ctx, loft, tool)
	if err != nil {
		return nil, err
	}
	if pitch != 25*math.Pi/180 {
		angle := pitch - 25*math.Pi/180
		if mirror {
			angle = -angle
		}
		rot, e := r3.RotationAround(r3.Vec{}, r3.Vec{Z: 1}, units.Radians(angle))
		if e != nil {
			return nil, e
		}
		body, err = decad.Transform(ctx, body, rot)
		if err != nil {
			return nil, err
		}
	}
	studFrame, err := r3.NewFrame(r3.Vec{Z: size.BladeStudStart}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	if err != nil {
		return nil, err
	}
	studPlane, err := w.CreatePlaneFromFrame(studFrame)
	if err != nil {
		return nil, err
	}
	studSketch, err := w.CreateSketch(studPlane)
	if err != nil {
		return nil, err
	}
	circle := studSketch.CreateCircle(studSketch.Origin(), size.BladeStudRadius)
	studSketch.AddConstraint(sketch.NewRadius(circle, size.BladeStudRadius))
	if _, err = studSketch.Solve(ctx); err != nil {
		return nil, err
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, studSketch); err != nil {
		return nil, err
	}
	sp := studSketch.Profiles()
	if len(sp) != 1 {
		return nil, fmt.Errorf("turbinefan: invalid stud profile")
	}
	stud, err := decad.Extrude(ctx, doc, studSketch, sp[0], decad.Distance{D: units.Millimeters(size.BladeStudEnd - size.BladeStudStart), Dir: decad.Along})
	if err != nil {
		return nil, err
	}
	sketches = append(sketches, studSketch)
	stud, err = decad.Chamfer(ctx, stud,
		decad.Edges(decad.OuterLoopOf(decad.CapStart(stud))).Exactly(1), units.Millimeters(.3))
	if err != nil {
		return nil, err
	}
	boltFrame, err := r3.NewFrame(r3.Vec{Y: -(size.BladeStudRadius + 2), Z: size.BladeBoltAt}, r3.Vec{Z: 1}, r3.Vec{X: 1})
	if err != nil {
		return nil, err
	}
	boltPlane, err := w.CreatePlaneFromFrame(boltFrame)
	if err != nil {
		return nil, err
	}
	boltSketch, err := w.CreateSketch(boltPlane)
	if err != nil {
		return nil, err
	}
	bc := boltSketch.CreateCircle(boltSketch.Origin(), 1.3)
	boltSketch.AddConstraint(sketch.NewRadius(bc, 1.3))
	if _, err = boltSketch.Solve(ctx); err != nil {
		return nil, err
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, boltSketch); err != nil {
		return nil, err
	}
	bp := boltSketch.Profiles()
	if len(bp) != 1 {
		return nil, fmt.Errorf("turbinefan: invalid bolt profile")
	}
	hole, err := decad.Extrude(ctx, doc, boltSketch, bp[0], decad.Distance{D: units.Millimeters(2 * (size.BladeStudRadius + 2)), Dir: decad.Along})
	if err != nil {
		return nil, err
	}
	stud, err = decad.Cut(ctx, stud, hole)
	if err != nil {
		return nil, err
	}
	sketches = append(sketches, boltSketch)
	body, err = decad.Union(ctx, body, stud)
	if err != nil {
		return nil, err
	}
	return &Blade{Body: body, Sketches: sketches}, nil
}
