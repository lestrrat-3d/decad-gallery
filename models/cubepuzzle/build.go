package cubepuzzle

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/fastenertool"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

type Params struct{}

func DefaultParams() Params { return Params{} }

type Assembly struct {
	Hub                                         *decad.Body
	Centers, Edges, Corners                     []*decad.Body
	Disks, FlangedSleeves, Sleeves, Bolts, Nuts []*decad.Body
}

func NewDocument(opts ...decad.DocumentOption) *decad.Document {
	return decad.New(append([]decad.DocumentOption{decad.WithChainTolerance(units.Millimeters(0.01))}, opts...)...)
}

var axes = []r3.Vec{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}, {Z: 1}, {Z: -1}}

func Build(ctx context.Context, doc *decad.Document, params Params) (*Assembly, error) {
	if doc == nil {
		return nil, fmt.Errorf("build Cube Puzzle: nil document")
	}
	if params != (Params{}) {
		return nil, fmt.Errorf("build Cube Puzzle: fixed-size model has no parameters")
	}
	a := &Assembly{}
	var err error
	a.Hub, err = makeHub(ctx, doc)
	if err != nil {
		return nil, err
	}
	center, err := makeBlock(ctx, doc, r3.Vec{Z: 1}, 0)
	if err != nil {
		return nil, err
	}
	edge, err := makeBlock(ctx, doc, r3.Vec{X: 1, Y: 1}, 1)
	if err != nil {
		return nil, err
	}
	corner, err := makeBlock(ctx, doc, r3.Vec{X: 1, Y: 1, Z: 1}, 2)
	if err != nil {
		return nil, err
	}
	rotor, err := makeRotor(ctx, doc, center)
	if err != nil {
		return nil, err
	}
	disk, bolt, nut, err := makeMount(ctx, doc)
	if err != nil {
		return nil, err
	}
	a.Bolts = append(a.Bolts, bolt)
	a.Nuts = append(a.Nuts, nut)
	flange, _, err := fastenertool.RoundedRevolve(ctx, doc, mustFrame(r3.Vec{}, r3.Vec{Z: 1}, r3.Vec{X: 1}), [][2]float64{{10.5, 1.5}, {15.6, 1.5}, {15.6, 3.8}, {15.05, 3.8}, {15.05, 3}, {10.5, 3}})
	if err != nil {
		return nil, fmt.Errorf("build flanged sleeve: %w", err)
	}
	sleeve, _, err := fastenertool.RoundedRevolve(ctx, doc, mustFrame(r3.Vec{}, r3.Vec{Z: 1}, r3.Vec{X: 1}), [][2]float64{{15.6, 1.5}, {26, 1.5}, {26, 2.5}, {15.6, 2.5}})
	if err != nil {
		return nil, fmt.Errorf("build sleeve: %w", err)
	}

	for _, s := range axes {
		f := faceFrame(s)
		b, e := place(ctx, doc, rotor, f)
		if e != nil {
			return nil, e
		}
		a.Centers = append(a.Centers, b)
		b, e = place(ctx, doc, disk, f)
		if e != nil {
			return nil, e
		}
		a.Disks = append(a.Disks, b)
		b, e = place(ctx, doc, flange, f)
		if e != nil {
			return nil, e
		}
		a.FlangedSleeves = append(a.FlangedSleeves, b)
		b, e = place(ctx, doc, sleeve, f)
		if e != nil {
			return nil, e
		}
		a.Sleeves = append(a.Sleeves, b)
		site := m2fastener.Site{Head: s.Scale(30), Dir: s.Scale(-1), Corner: tangent(s), Length: 25, Cavity: 19.5}
		if s != (r3.Vec{Z: 1}) {
			boltPart, e := m2fastener.PlaceBolt(ctx, doc, site)
			if e != nil {
				return nil, e
			}
			a.Bolts = append(a.Bolts, boltPart.Body)
			nutPart, e := m2fastener.PlaceNut(ctx, doc, site)
			if e != nil {
				return nil, e
			}
			a.Nuts = append(a.Nuts, nutPart.Body)
		}
	}
	for _, pair := range [][2]r3.Vec{{{X: 1}, {Y: 1}}, {{X: 1}, {Z: 1}}, {{Y: 1}, {Z: 1}}} {
		for x := -1; x <= 1; x += 2 {
			for y := -1; y <= 1; y += 2 {
				u, v := pair[0].Scale(float64(x)), pair[1].Scale(float64(y))
				b, e := place(ctx, doc, edge, mustFrame(r3.Vec{}, u, v))
				if e != nil {
					return nil, e
				}
				a.Edges = append(a.Edges, b)
			}
		}
	}
	for x := -1; x <= 1; x += 2 {
		for y := -1; y <= 1; y += 2 {
			for z := -1; z <= 1; z += 2 {
				u, v := r3.Vec{X: float64(x)}, r3.Vec{Y: float64(y)}
				if u.Cross(v).Z != float64(z) {
					u, v = v, u
				}
				b, e := place(ctx, doc, corner, mustFrame(r3.Vec{}, u, v))
				if e != nil {
					return nil, e
				}
				a.Corners = append(a.Corners, b)
			}
		}
	}
	// The reusable solids above are construction sources; only placed bodies are published.
	for _, src := range []*decad.Body{edge, corner, rotor, disk, flange, sleeve} {
		if err := decad.Remove(src).Do(ctx); err != nil {
			return nil, err
		}
	}
	for name, bodies := range map[string][]*decad.Body{"Hubs": {a.Hub}, "Centers": a.Centers, "Edges": a.Edges, "Corners": a.Corners, "Disks": a.Disks, "Flanged Sleeves": a.FlangedSleeves, "Sleeves": a.Sleeves, "Bolts": a.Bolts, "Nuts": a.Nuts} {
		c, e := doc.Root().NewChild(name)
		if e != nil {
			return nil, e
		}
		for _, b := range bodies {
			if e := c.AddBody(b); e != nil {
				return nil, e
			}
		}
	}
	if report := doc.VerifyContainers(); !report.Passed() {
		return nil, fmt.Errorf("verify Cube Puzzle containers: %v", report.Err())
	}
	return a, nil
}

func makeBlock(ctx context.Context, doc *decad.Document, direction r3.Vec, kind int) (*decad.Body, error) {
	var base *decad.Body
	layers := []r3.Vec{{X: 1}, {Y: 1}, {Z: 1}}
	if kind == 0 {
		layers = []r3.Vec{{Z: 1}}
	}
	switch kind {
	case 1:
		layers = []r3.Vec{{X: 1}, {Y: 1}}
	}
	for _, axis := range layers {
		layer, _, err := makeLayer(ctx, doc, axis, .25, 30)
		if err != nil {
			return nil, err
		}
		if base == nil {
			base = layer
			continue
		}
		base, err = decad.Intersect(base, layer).Do(ctx)
		if err != nil {
			return nil, fmt.Errorf("intersect layer block: %w", err)
		}
	}
	switch kind {
	case 1:
		for _, axis := range []r3.Vec{{Z: 1}, {Z: -1}} {
			tool, _, e := makeLayer(ctx, doc, axis, -.25, 60)
			if e != nil {
				return nil, e
			}
			base, e = decad.Cut(base, tool).Do(ctx)
			if e != nil {
				return nil, fmt.Errorf("cut edge layer: %w", e)
			}
		}
	case 0:
		for _, axis := range []r3.Vec{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
			tool, _, e := makeLayer(ctx, doc, axis, -.25, 60)
			if e != nil {
				return nil, e
			}
			base, e = decad.Cut(base, tool).Do(ctx)
			if e != nil {
				return nil, fmt.Errorf("cut center layer: %w", e)
			}
		}
	}
	ball, err := makeSphere(ctx, doc, 12)
	if err != nil {
		return nil, err
	}
	base, err = decad.Cut(base, ball).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("hollow layer block: %w", err)
	}
	_ = direction
	return base, nil
}

func makeLayer(ctx context.Context, doc *decad.Document, axis r3.Vec, offset, cap float64) (*decad.Body, *sketch.Sketch, error) {
	r0, r1 := math.Sqrt(209), 19.0
	m := 6 / (r1 - r0)
	shift := offset * math.Sqrt(1+m*m)
	low, high := 4+offset, 10+offset
	rl, rh := r0+(offset-shift)/m, r1+(offset-shift)/m
	pts := [][2]float64{{low, 0}, {cap, 0}, {cap, 43}, {high, 43}, {high, rh}, {low, rl}}
	f := mustFrame(r3.Vec{}, axis, tangent(axis))
	w := sketch.NewWorld()
	plane, e := w.CreatePlaneFromFrame(f)
	if e != nil {
		return nil, nil, e
	}
	sk, e := w.CreateSketch(plane)
	if e != nil {
		return nil, nil, e
	}
	ps := make([]*sketch.Point, len(pts))
	for i, p := range pts {
		ps[i] = sk.CreatePoint(p[0], p[1])
	}
	lines := make([]*sketch.Line, len(ps))
	for i := range ps {
		lines[i] = sk.CreateLine(ps[i], ps[(i+1)%len(ps)])
	}
	angle := math.Atan(1/m)*180/math.Pi - 180
	distance := math.Abs(rl-low/m) / math.Sqrt(1+1/(m*m))
	sk.AddConstraint(
		sketch.NewHorizontal(lines[0]), sketch.NewVerticalDistance(sk.Origin(), ps[0], 0),
		sketch.NewVertical(lines[1]), sketch.NewHorizontalDistance(sk.Origin(), ps[1], cap),
		sketch.NewHorizontal(lines[2]), sketch.NewVerticalDistance(sk.Origin(), ps[2], 43),
		sketch.NewVertical(lines[3]), sketch.NewHorizontalDistance(sk.Origin(), ps[3], high),
		sketch.NewAngle(lines[0], lines[4], angle), sketch.NewDistancePointLine(sk.Origin(), lines[4], distance),
		sketch.NewVertical(lines[5]), sketch.NewHorizontalDistance(sk.Origin(), ps[0], low),
	)
	if report := sk.Verify(ctx); report.DOF != 0 {
		return nil, nil, fmt.Errorf("layer before fillets has DOF %d", report.DOF)
	}
	if _, e := sketch.CreateFillet(sk, lines[3], lines[4], ps[4], .75+offset); e != nil {
		return nil, nil, fmt.Errorf("fillet upper layer boundary: %w", e)
	}
	if report := sk.Verify(ctx); report.DOF != 0 {
		return nil, nil, fmt.Errorf("layer after upper fillet has DOF %d", report.DOF)
	}
	if _, e := sketch.CreateFillet(sk, lines[4], lines[5], ps[5], .75-offset); e != nil {
		return nil, nil, fmt.Errorf("fillet lower layer boundary: %w", e)
	}
	if report := sk.Verify(ctx); report.DOF != 0 {
		return nil, nil, fmt.Errorf("layer after lower fillet has DOF %d", report.DOF)
	}
	profile, e := validProfile(ctx, sk)
	if e != nil {
		return nil, nil, e
	}
	body, e := decad.Revolve(doc, sk, profile, decad.ConstructionAxis{Origin: f.Origin(), Dir: axis}, decad.FullRevolution{}).Do(ctx)
	if e != nil {
		return nil, nil, fmt.Errorf("revolve layer: %w", e)
	}
	return body, sk, nil
}

func makeHub(ctx context.Context, doc *decad.Document) (*decad.Body, error) {
	ball, err := makeSphere(ctx, doc, 11.5)
	if err != nil {
		return nil, err
	}
	for _, axis := range axes {
		u := tangent(axis)
		f := mustFrame(axis.Scale(10.5), u, axis.Cross(u))
		tool, _, e := fastenertool.Cylinder(ctx, doc, f, 30, 15)
		if e != nil {
			return nil, e
		}
		ball, e = decad.Cut(ball, tool).Do(ctx)
		if e != nil {
			return nil, fmt.Errorf("flatten hub: %w", e)
		}
	}
	for _, out := range axes {
		tan := tangent(out)
		site := m2fastener.Site{Head: out.Scale(30), Dir: out.Scale(-1), Corner: tan, Length: 25, Cavity: 19.5}
		if err := site.Validate(); err != nil {
			return nil, err
		}
		pocketSite := site
		pocketSite.Head = site.At(site.CavityFar())
		pocketSite.Dir = site.Dir.Scale(-1)
		pocketSite.Cavity = 0
		tool, _, e := m2fastener.RoundedHexPocketTool(ctx, doc, pocketSite, 4, .5)
		if e != nil {
			return nil, e
		}
		ball, e = decad.Cut(ball, tool).Do(ctx)
		if e != nil {
			return nil, fmt.Errorf("cut hub nut pocket: %w", e)
		}
		hole, _, e := m2fastener.RoundedHoleTool(ctx, doc, site)
		if e != nil {
			return nil, e
		}
		ball, e = decad.Cut(ball, hole).Do(ctx)
		if e != nil {
			return nil, fmt.Errorf("cut hub bolt hole: %w", e)
		}
	}
	return ball, nil
}

func makeRotor(ctx context.Context, doc *decad.Document, center *decad.Body) (*decad.Body, error) {
	cut := func(r, z, l float64) error {
		t, _, e := fastenertool.Cylinder(ctx, doc, mustFrame(r3.NewVec(0, 0, z), r3.Vec{X: 1}, r3.Vec{Y: 1}), r, l)
		if e != nil {
			return e
		}
		center, e = decad.Cut(center, t).Do(ctx)
		return e
	}
	if err := cut(45, -40, 56); err != nil {
		return nil, err
	}
	if err := cut(2.9, 15, 16); err != nil {
		return nil, err
	}
	if err := cut(6.4, 25.6, 5.4); err != nil {
		return nil, err
	}
	return center, nil
}

func makeMount(ctx context.Context, doc *decad.Document) (*decad.Body, *decad.Body, *decad.Body, error) {
	disk, _, err := fastenertool.Cylinder(ctx, doc, mustFrame(r3.NewVec(0, 0, 26), r3.Vec{X: 1}, r3.Vec{Y: 1}), 6, 4)
	if err != nil {
		return nil, nil, nil, err
	}
	s := m2fastener.Site{Head: r3.NewVec(0, 0, 30), Dir: r3.Vec{Z: -1}, Corner: r3.Vec{X: 1}, Length: 25, Cavity: 19.5}
	for _, fn := range []func(context.Context, *decad.Document, m2fastener.Site) (*decad.Body, *sketch.Sketch, error){m2fastener.PocketTool, m2fastener.HoleTool} {
		tool, _, e := fn(ctx, doc, s)
		if e != nil {
			return nil, nil, nil, e
		}
		disk, e = decad.Cut(disk, tool).Do(ctx)
		if e != nil {
			return nil, nil, nil, e
		}
	}
	b, e := m2fastener.PlaceBolt(ctx, doc, s)
	if e != nil {
		return nil, nil, nil, e
	}
	n, e := m2fastener.PlaceNut(ctx, doc, s)
	if e != nil {
		return nil, nil, nil, e
	}
	return disk, b.Body, n.Body, nil
}

func place(ctx context.Context, doc *decad.Document, src *decad.Body, f r3.Frame) (*decad.Body, error) {
	xf, e := r3.FromFrame(f)
	if e != nil {
		return nil, e
	}
	c := decad.Copy(src).Place(decad.CopyTransform{Transform: xf})
	b, e := c.Do(ctx)
	if e != nil {
		return nil, fmt.Errorf("place body: %w", e)
	}
	return b, nil
}
func makeSphere(ctx context.Context, doc *decad.Document, radius float64) (*decad.Body, error) {
	w := sketch.NewWorld()
	plane, e := w.CreatePlaneFromFrame(mustFrame(r3.Vec{}, r3.Vec{X: 1}, r3.Vec{Y: 1}))
	if e != nil {
		return nil, e
	}
	sk, e := w.CreateSketch(plane)
	if e != nil {
		return nil, e
	}
	start, end := sk.CreatePoint(-radius, 0), sk.CreatePoint(radius, 0)
	arc := sk.CreateArc(sk.Origin(), start, end)
	axis := sk.CreateLine(start, end)
	sk.AddConstraint(sketch.NewRadius(arc, radius), sketch.NewVerticalDistance(sk.Origin(), start, 0), sketch.NewHorizontal(axis))
	p, e := validProfile(ctx, sk)
	if e != nil {
		return nil, fmt.Errorf("verify sphere sketch: %w", e)
	}
	b, e := decad.Revolve(doc, sk, p, decad.ConstructionAxis{Origin: r3.Vec{}, Dir: r3.Vec{X: 1}}, decad.FullRevolution{}).Do(ctx)
	if e != nil {
		return nil, fmt.Errorf("revolve sphere: %w", e)
	}
	return b, nil
}
func tangent(v r3.Vec) r3.Vec {
	if math.Abs(v.X) > .5 {
		return r3.Vec{Y: 1}
	}
	if math.Abs(v.Y) > .5 {
		return r3.Vec{Z: 1}
	}
	return r3.Vec{X: 1}
}
func mustFrame(o, u, v r3.Vec) r3.Frame {
	f, e := r3.NewFrame(o, u, v)
	if e != nil {
		panic(e)
	}
	return f
}
func faceFrame(n r3.Vec) r3.Frame { u := tangent(n); v := n.Cross(u); return mustFrame(r3.Vec{}, u, v) }
func validProfile(ctx context.Context, sk *sketch.Sketch) (*sketch.Profile, error) {
	if _, e := sk.Solve(ctx); e != nil {
		return nil, e
	}
	if e := sketchcheck.VerifyFullyConstrained(ctx, sk); e != nil {
		return nil, e
	}
	for _, p := range sk.Profiles() {
		if p.Valid && len(p.Holes) == 0 {
			return p, nil
		}
	}
	return nil, fmt.Errorf("layer sketch has no valid profile")
}
