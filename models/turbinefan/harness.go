package turbinefan

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Print harness sizes, in mm or radians. They are print-process sizes, not standard dimensions; instructions.md
// "Print pose and harness" gives each one's reason.
const (
	// KeelWidth is the keel's width past the cradle: eight 0.4 mm lines.
	KeelWidth = 3.2
	// KeelStep is the spacing of the keel top's vertices along x.
	KeelStep = 0.5
	// KeelMinRise is the least keel height above the plate; the keel ends where it would be lower.
	KeelMinRise = 0.4
	// KeelGateEmbed is how far a keel pin runs into the blade above the blade's lowest point.
	KeelGateEmbed = 0.4
	// WallThickness is the wall's thickness: five lines.
	WallThickness = 2.0
	// WallCover is how far the wall's top stands above each strut's end disc.
	WallCover = 1.0
	// GateNeckRadius is the radius of a gate's neck, the narrow part that meets the blade.
	GateNeckRadius = 0.3
	// GateNeckLength is a trailing-edge gate's neck length past its contact point.
	GateNeckLength = 2.0
	// GateRadius is the radius of a strut.
	GateRadius = 0.8
	// GateTaper is the length of the cone from the neck to the strut.
	GateTaper = 3.0
	// GateEmbed is how far a trailing-edge gate's neck runs on into the blade along its axis.
	GateEmbed = 0.2
	// GateRise is a strut's angle above the bed.
	GateRise = 60 * math.Pi / 180
	// GateConeClearance is the least distance from any strut's cone to the wall's near face; it sets the wall's place.
	GateConeClearance = 1.0
	// TipGateDrop is the tip gate's contact below the tip radius.
	TipGateDrop = 1.0
	// HarnessMargin is how far the plate reaches past everything, and the wall past its end gates.
	HarnessMargin = 3.0
	// SectionStep is the spacing along x of the blade sections the layout reads.
	SectionStep = 0.25
	// HarnessGateCount is the number of gates: six on the trailing edge and four keel pins.
	HarnessGateCount = 10
)

// HarnessGate is one gate. Contact is the blade point the neck axis enters; Dir is the unit axis from Contact out of
// the blade; Root is the gate's harness end (on the wall's mid plane for a trailing-edge gate, 0.5 inside the keel for a
// pin).
type HarnessGate struct{ Contact, Dir, Root r3.Vec }

// BladeHarness is the harness in the print frame. Gates 0..5 are the trailing-edge gates (stations 5..9, tip), 6..9
// the keel pins (stud, station 2, station 7, station 9). KeelFootprint is the keel's plan outline (x, y), KeelTop its
// top profile (x, z), KeelTopAt interpolates it. WallY is the wall's mid plane, PlateMin/PlateMax the plate corners.
// CradleWidth is the keel's width under the stud and root.
type BladeHarness struct {
	Body *decad.Body
	// Sketches are the 14 sketches in build order: keel footprint, keel profile, wall, 6 struts, 4 pins, plate.
	Sketches      []*sketch.Sketch
	Gates         [HarnessGateCount]HarnessGate
	KeelFootprint [][2]float64
	KeelTop       [][2]float64
	WallY         float64
	PlateMin      r3.Vec
	PlateMax      r3.Vec
	CradleWidth   float64
}

// KeelTopAt returns the keel top's z at x, interpolated between KeelTop's vertices, and false outside its x range.
func (h *BladeHarness) KeelTopAt(x float64) (float64, bool) {
	return keelTopAt(h.KeelTop, x)
}

func keelTopAt(top [][2]float64, x float64) (float64, bool) {
	if len(top) < 2 || x < top[0][0] || x > top[len(top)-1][0] {
		return 0, false
	}
	for i := 1; i < len(top); i++ {
		if x <= top[i][0] {
			t := (x - top[i-1][0]) / (top[i][0] - top[i-1][0])
			return top[i-1][1] + t*(top[i][1]-top[i-1][1]), true
		}
	}
	return top[len(top)-1][1], true
}

// BuildBladeHarness builds the harness for blade, which must already be in the print pose PrintPoseFor(sz, pitch, hand)
// gives. It reads the blade with SectionOutline. It puts the harness in no container. ctx, doc and blade must not be nil.
func BuildBladeHarness(ctx context.Context, doc *decad.Document, blade *decad.Body, sz Size, hand Hand) (*BladeHarness, error) {
	if ctx == nil || doc == nil || blade == nil {
		return nil, errors.New("turbinefan: context, document and blade must not be nil")
	}
	if !sz.valid() {
		return nil, errors.New("turbinefan: sz must come from SizeFor")
	}
	if hand != RightHand && hand != LeftHand {
		return nil, errors.New("turbinefan: hand must be RightHand or LeftHand")
	}
	// The posed loft does not depend on the pitch, so the analytic points use the default pitch's pose.
	pose, err := PrintPoseFor(sz, DefaultBladePitch, hand)
	if err != nil {
		return nil, err
	}
	b := &bladeBuilder{ctx: ctx, doc: doc, sz: sz, mirror: hand == LeftHand, what: "harness"}
	h := b.harness(blade, pose, hand)
	if b.err != nil {
		return nil, b.err
	}
	if h.Body == nil || !h.Body.IsSolid() {
		return nil, errors.New("turbinefan: harness is not a solid")
	}
	h.Sketches = b.sketches
	return h, nil
}

// bladeSections are the posed blade's sections by planes x = x0 + i SectionStep, each a list of loops in the print
// frame.
type bladeSections struct {
	x0    float64
	loops [][][]r3.Vec
}

// near returns the loops of the section nearest x.
func (s bladeSections) near(x float64) [][]r3.Vec {
	i := int(math.Round((x - s.x0) / SectionStep))
	if i < 0 || i >= len(s.loops) {
		return nil
	}
	return s.loops[i]
}

// ridge returns the lowest outline point of the section nearest x.
func (s bladeSections) ridge(x float64) r3.Vec {
	best := r3.Vec{Z: math.Inf(1)}
	for _, l := range s.near(x) {
		for _, p := range l {
			if p.Z < best.Z {
				best = p
			}
		}
	}
	return best
}

// sections cuts the blade at every SectionStep across its x range (step 2).
func (b *bladeBuilder) sections(blade *decad.Body, box decad.Box) bladeSections {
	var s bladeSections
	if b.err != nil {
		return s
	}
	s.x0 = math.Ceil(box.Min.X/SectionStep) * SectionStep
	for x := s.x0; x <= box.Max.X; x += SectionStep {
		f := frame(r3.Vec{X: x}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
		o, err := blade.SectionOutline(b.ctx, f, units.Millimeters(0.02))
		if err != nil {
			o, err = blade.SectionOutline(b.ctx, f, units.Millimeters(0.05))
		}
		if err != nil {
			b.fail("section blade", fmt.Errorf("x %.2f: %w", x, err))
			return s
		}
		s.loops = append(s.loops, o.Loops)
	}
	return s
}

// harness runs construction steps 2 to 14.
func (b *bladeBuilder) harness(blade *decad.Body, pose r3.Frame, hand Hand) *BladeHarness {
	box, err := blade.Bounds()
	if err != nil {
		b.fail("measure blade", err)
		return nil
	}
	if box.Min.Z < PrintLift-box.Bound.Base()-1e-6 {
		b.fail("check pose", errors.New("blade dips below its stud corner"))
		return nil
	}
	secs := b.sections(blade, box)
	if b.err != nil {
		return nil
	}
	xf, err := r3.FromFrame(pose)
	if err != nil {
		b.fail("pose", err)
		return nil
	}
	xr := func(r float64) float64 { return xf.Apply(r3.Vec{Z: r}).X }
	h := &BladeHarness{CradleWidth: 8 * b.sz.HubScale * math.Sin(math.Pi/4)}
	stations := stationsFor(b.sz)

	h.KeelFootprint = keelFootprint(b.keelSpine(secs, xr, stations, h.CradleWidth))
	h.KeelTop = b.keelTop(secs, h.KeelFootprint)
	if b.err != nil {
		return nil
	}
	keel := b.keel(h.KeelFootprint, h.KeelTop)

	contacts := trailingContacts(b.sz, xf, stations, hand)
	wall, struts := b.wallAndStruts(h, contacts, hand)
	pins := b.keelPins(h, secs, xr, stations)
	plate := b.plate(h, box)
	if b.err != nil {
		return nil
	}

	// Join in small chains, each branch then the base: the keel with its pins, the plate with the keel, the wall with
	// its struts, then the base with the wall.
	for i, pin := range pins {
		keel = b.join(keel, pin, fmt.Sprintf("keel pin %d", 6+i))
	}
	body := b.join(plate, keel, "keel")
	for i, strut := range struts {
		wall = b.join(wall, strut, fmt.Sprintf("strut %d", i))
	}
	h.Body = b.join(body, wall, "wall")
	return h
}

// keelSpine returns the keel track's vertices with their half widths (step 4): the stud end, stations 2 and 4 to 9 and
// the tip, with a taper vertex after station 5 where the cradle narrows to KeelWidth at 45 degrees.
func (b *bladeBuilder) keelSpine(secs bladeSections, xr func(float64) float64, stations [10]station,
	cradle float64) [][3]float64 {
	radii := []float64{b.sz.BladeStudStart, stations[1].r, stations[3].r, stations[4].r, stations[5].r, stations[6].r,
		stations[7].r, stations[8].r, b.sz.TipRadius}
	var track []r3.Vec
	for i, r := range radii {
		x := xr(r)
		switch i {
		case 0:
			x += 0.6
		case len(radii) - 1:
			x -= 0.6
		}
		y := secs.ridge(x).Y
		if i == 0 {
			y = 0 // the stud's bottom lies under its axis, which the pose puts at y = 0
		}
		track = append(track, r3.Vec{X: x, Y: y})
	}
	var spine [][3]float64
	for i, t := range track {
		hw := KeelWidth / 2
		if i <= 3 {
			hw = cradle / 2
		}
		spine = append(spine, [3]float64{t.X, t.Y, hw})
		if i == 3 {
			d, _ := track[4].Sub(t).Normalize()
			e := t.Add(d.Scale((cradle - KeelWidth) / 2))
			spine = append(spine, [3]float64{e.X, e.Y, KeelWidth / 2})
		}
	}
	return spine
}

// keelFootprint offsets the spine (x, y, half width) to both sides (step 5): the left side forward, the right side
// backward, mitred at interior vertices.
func keelFootprint(spine [][3]float64) [][2]float64 {
	off := func(i int, side float64) r3.Vec {
		var d0, d1 r3.Vec
		if i > 0 {
			d0, _ = r3.Vec{X: spine[i][0] - spine[i-1][0], Y: spine[i][1] - spine[i-1][1]}.Normalize()
		}
		if i+1 < len(spine) {
			d1, _ = r3.Vec{X: spine[i+1][0] - spine[i][0], Y: spine[i+1][1] - spine[i][1]}.Normalize()
		}
		if i == 0 {
			d0 = d1
		}
		if i+1 == len(spine) {
			d1 = d0
		}
		n0, n1 := r3.Vec{Z: 1}.Cross(d0), r3.Vec{Z: 1}.Cross(d1)
		m, _ := n0.Add(n1).Normalize()
		return r3.Vec{X: spine[i][0], Y: spine[i][1]}.Add(m.Scale(side * spine[i][2] / m.Dot(n0)))
	}
	var fp [][2]float64
	for i := range spine {
		q := off(i, 1)
		fp = append(fp, [2]float64{q.X, q.Y})
	}
	for i := len(spine) - 1; i >= 0; i-- {
		q := off(i, -1)
		fp = append(fp, [2]float64{q.X, q.Y})
	}
	return fp
}

// insidePolygon reports whether (x, y) lies inside the polygon, by the even-odd rule.
func insidePolygon(poly [][2]float64, x, y float64) bool {
	in := false
	for i := range poly {
		a, c := poly[i], poly[(i+len(poly)-1)%len(poly)]
		if (a[1] > y) != (c[1] > y) && x < (c[0]-a[0])*(y-a[1])/(c[1]-a[1])+a[0] {
			in = !in
		}
	}
	return in
}

// keelTop returns the keel top profile (step 6): at every KeelStep along the track, the lowest blade point inside the
// footprint within KeelStep of x, less KeelGap. Leading and trailing vertices with no point or too low are dropped.
func (b *bladeBuilder) keelTop(secs bladeSections, fp [][2]float64) [][2]float64 {
	if b.err != nil {
		return nil
	}
	// The track runs from the stud end's vertex to the tip's: the midpoints of the footprint's first and last pairs.
	first, last := (fp[0][0]+fp[len(fp)-1][0])/2, (fp[len(fp)/2-1][0]+fp[len(fp)/2][0])/2
	floor := HarnessPlateThickness + KeelMinRise
	var top [][2]float64
	n := int(math.Floor((last-first)/KeelStep + 1e-9))
	for i := 0; i <= n; i++ {
		x := first + float64(i)*KeelStep
		lo := math.Inf(1)
		for xx := x - KeelStep; xx <= x+KeelStep+1e-9; xx += SectionStep {
			for _, l := range secs.near(xx) {
				for pi := range l {
					p0, p1 := l[pi], l[(pi+1)%len(l)]
					for k := 0; k <= 10; k++ {
						p := p0.Add(p1.Sub(p0).Scale(float64(k) / 10))
						if insidePolygon(fp, p.X, p.Y) {
							lo = math.Min(lo, p.Z)
						}
					}
				}
			}
		}
		top = append(top, [2]float64{x, lo - KeelGap})
	}
	low := func(q [2]float64) bool { return math.IsInf(q[1], 0) || q[1] < floor }
	for len(top) > 0 && low(top[0]) {
		top = top[1:]
	}
	for len(top) > 0 && low(top[len(top)-1]) {
		top = top[:len(top)-1]
	}
	if len(top) < 2 {
		b.fail("keel top", errors.New("fewer than two keel top vertices"))
		return nil
	}
	for _, q := range top {
		if low(q) {
			b.fail("keel top", fmt.Errorf("keel top %.2f at x %.2f under the plate", q[1], q[0]))
			return nil
		}
	}
	return top
}

// keel intersects the footprint's vertical prism with the top profile's horizontal slab (steps 5 and 7). The prism
// starts at a quarter of the plate's thickness and the slab at half of it, so no faces coincide.
func (b *bladeBuilder) keel(fp, top [][2]float64) *decad.Body {
	if b.err != nil {
		return nil
	}
	maxTop, ymin, ymax := 0.0, math.Inf(1), math.Inf(-1)
	for _, q := range top {
		maxTop = math.Max(maxTop, q[1])
	}
	for _, q := range fp {
		ymin, ymax = math.Min(ymin, q[1]), math.Max(ymax, q[1])
	}
	t := HarnessPlateThickness
	prism := b.extrudePolygon(frame(r3.Vec{Z: t / 4}, r3.Vec{X: 1}, r3.Vec{Y: 1}), fp, maxTop+1, "keel footprint")
	side := [][2]float64{{top[0][0], t / 2}, {top[len(top)-1][0], t / 2}}
	for i := len(top) - 1; i >= 0; i-- {
		side = append(side, top[i])
	}
	slab := b.extrudePolygon(frame(r3.Vec{Y: ymax + 1}, r3.Vec{X: 1}, r3.Vec{Z: 1}), side, ymax-ymin+2, "keel profile")
	if b.err != nil {
		return nil
	}
	keel, err := decad.Intersect(b.ctx, prism, slab)
	b.fail("intersect keel", err)
	return keel
}

// trailingContacts returns the trailing-edge gates' contacts in the print frame (step 8): the trailing-edge midpoint of
// stations 5 to 9, and of station 10 at TipGateDrop under the tip.
func trailingContacts(sz Size, xf r3.Transform, stations [10]station, hand Hand) [6]r3.Vec {
	var out [6]r3.Vec
	for i := range out {
		st := stations[4+i]
		p := sectionLocal(st, sz.Beta(st.r, refPitch), 1, 0)
		if hand == RightHand {
			p[0] = -p[0]
		}
		r := st.r
		if i == 5 {
			r = sz.TipRadius - TipGateDrop
		}
		out[i] = xf.Apply(r3.Vec{X: p[0], Y: p[1], Z: r})
	}
	return out
}

// wallAndStruts places the wall on the pressure side and builds it and the six struts (steps 9 and 10).
func (b *bladeBuilder) wallAndStruts(h *BladeHarness, contacts [6]r3.Vec, hand Hand) (*decad.Body, []*decad.Body) {
	if b.err != nil {
		return nil, nil
	}
	head := GateEmbed + GateNeckLength + GateTaper
	dmin := GateConeClearance + head*math.Cos(GateRise) + WallThickness/2 + GateRadius*math.Sin(GateRise)
	h.WallY = math.Inf(1)
	if hand == LeftHand {
		h.WallY = math.Inf(-1)
	}
	for _, c := range contacts {
		if hand == LeftHand {
			h.WallY = math.Max(h.WallY, c.Y+dmin)
			continue
		}
		h.WallY = math.Min(h.WallY, c.Y-dmin)
	}
	var struts []*decad.Body
	tops := make([][2]float64, len(contacts))
	for i, c := range contacts {
		d := math.Abs(c.Y - h.WallY)
		if cone := d - head*math.Cos(GateRise) - WallThickness/2 - GateRadius*math.Sin(GateRise); cone < GateConeClearance-1e-9 {
			b.fail("place wall", fmt.Errorf("gate %d cone is %.2f mm from the wall, under GateConeClearance", i, cone))
			return nil, nil
		}
		s := r3.Vec{X: c.X, Y: h.WallY, Z: c.Z - d*math.Tan(GateRise)}
		w, _ := s.Sub(c).Normalize()
		t := c.Sub(w.Scale(GateEmbed))
		length := s.Sub(t).Len()
		v, _ := r3.Vec{Z: 1}.Cross(w).Normalize()
		n := GateEmbed + GateNeckLength
		struts = append(struts, b.revolvePolygon(frame(t, w, v), [][2]float64{
			{0, 0}, {0, GateNeckRadius}, {n, GateNeckRadius}, {n + GateTaper, GateRadius}, {length, GateRadius}, {length, 0},
		}, "strut"))
		h.Gates[i] = HarnessGate{Contact: c, Dir: w, Root: s}
		tops[i] = [2]float64{c.X, s.Z + GateRadius*math.Cos(GateRise) + WallCover}
	}
	t := HarnessPlateThickness
	x0, x1 := contacts[0].X-HarnessMargin, contacts[5].X+HarnessMargin
	outline := [][2]float64{{x0, t / 2}, {x1, t / 2}, {x1, tops[5][1]}}
	for i := len(tops) - 1; i >= 0; i-- {
		outline = append(outline, tops[i])
	}
	outline = append(outline, [2]float64{x0, tops[0][1]})
	wall := b.extrudePolygon(frame(r3.Vec{Y: h.WallY + WallThickness/2}, r3.Vec{X: 1}, r3.Vec{Z: 1}), outline,
		WallThickness, "wall")
	return b.chamferCaps(wall, "wall"), struts
}

// keelPins builds the four keel pins (step 11): vertical circles of GateNeckRadius from 0.5 under the keel top up to
// KeelGateEmbed above the blade's lowest point, between the bolt hole and the root, and at stations 2, 7 and 9.
func (b *bladeBuilder) keelPins(h *BladeHarness, secs bladeSections, xr func(float64) float64,
	stations [10]station) []*decad.Body {
	if b.err != nil {
		return nil
	}
	radii := []float64{(b.sz.BladeBoltAt + m2fastener.HoleRadius + b.sz.RootRadius) / 2, stations[1].r, stations[6].r,
		stations[8].r}
	var pins []*decad.Body
	for i, r := range radii {
		p := secs.ridge(xr(r))
		top, ok := h.KeelTopAt(p.X)
		if !ok || !insidePolygon(h.KeelFootprint, p.X, p.Y) {
			b.fail("keel pin", fmt.Errorf("pin %d at %v lies off the keel", 6+i, p))
			return nil
		}
		root := r3.Vec{X: p.X, Y: p.Y, Z: top - 0.5}
		pins = append(pins, b.cylinder(frame(root, r3.Vec{X: 1}, r3.Vec{Y: 1}), GateNeckRadius, p.Z+KeelGateEmbed-root.Z))
		h.Gates[6+i] = HarnessGate{Contact: p, Dir: r3.Vec{Z: -1}, Root: root}
	}
	return pins
}

// plate builds the plate (step 12): a rectangle under the blade, the keel and the wall, HarnessMargin beyond them,
// chamfered on both caps.
func (b *bladeBuilder) plate(h *BladeHarness, box decad.Box) *decad.Body {
	if b.err != nil {
		return nil
	}
	ylo := math.Min(h.WallY-WallThickness/2, box.Min.Y)
	yhi := math.Max(h.WallY+WallThickness/2, box.Max.Y)
	for _, q := range h.KeelFootprint {
		ylo, yhi = math.Min(ylo, q[1]), math.Max(yhi, q[1])
	}
	h.PlateMin = r3.Vec{X: box.Min.X - HarnessMargin, Y: ylo - HarnessMargin}
	h.PlateMax = r3.Vec{X: box.Max.X + HarnessMargin, Y: yhi + HarnessMargin, Z: HarnessPlateThickness}
	s, err := newSketch(frame(h.PlateMin, r3.Vec{X: 1}, r3.Vec{Y: 1}))
	if err != nil {
		b.fail("create plate", err)
		return nil
	}
	b.sketches = append(b.sketches, s)
	w, d := h.PlateMax.X-h.PlateMin.X, h.PlateMax.Y-h.PlateMin.Y
	o := s.Origin()
	p1, p2, p3 := s.CreatePoint(w, 0), s.CreatePoint(w, d), s.CreatePoint(0, d)
	l1, l2, l3, l4 := s.CreateLine(o, p1), s.CreateLine(p1, p2), s.CreateLine(p2, p3), s.CreateLine(p3, o)
	s.AddConstraint(sketch.NewHorizontal(l1), sketch.NewVertical(l2), sketch.NewHorizontal(l3), sketch.NewVertical(l4),
		sketch.NewHorizontalDistance(o, p1, w), sketch.NewVerticalDistance(o, p3, d))
	return b.chamferCaps(b.extrudeProfile(s, 0, HarnessPlateThickness, "plate"), "plate")
}

// chamferCaps chamfers both caps' outer loops of an extruded body by ChamferSize in one call.
func (b *bladeBuilder) chamferCaps(body *decad.Body, name string) *decad.Body {
	if b.err != nil || body == nil {
		return nil
	}
	loops := decad.Edges(decad.OuterLoopOf(decad.CapStart(body))).Or(decad.OuterLoopOf(decad.CapEnd(body)))
	out, err := decad.Chamfer(b.ctx, body, loops, units.Millimeters(ChamferSize))
	b.fail("chamfer "+name, err)
	return out
}

// join unions part into body; name tells the part in an error.
func (b *bladeBuilder) join(body, part *decad.Body, name string) *decad.Body {
	if b.err != nil {
		return nil
	}
	joined, err := decad.Union(b.ctx, body, part)
	b.fail("union "+name, err)
	return joined
}

// sketchPolygon starts a sketch on f and draws the closed polygon of vertices, each placed by
// sketchcheck.DimensionPoint.
func (b *bladeBuilder) sketchPolygon(f r3.Frame, vertices [][2]float64, name string) *sketch.Sketch {
	if b.err != nil {
		return nil
	}
	s, err := newSketch(f)
	if err != nil {
		b.fail("create "+name, err)
		return nil
	}
	b.sketches = append(b.sketches, s)
	points := make([]*sketch.Point, len(vertices))
	for i, p := range vertices {
		points[i] = sketchcheck.DimensionPoint(s, p[0], p[1])
	}
	for i := range points {
		s.CreateLine(points[i], points[(i+1)%len(points)])
	}
	return s
}

// extrudePolygon extrudes the closed polygon of vertices on f by depth along f's normal.
func (b *bladeBuilder) extrudePolygon(f r3.Frame, vertices [][2]float64, depth float64, name string) *decad.Body {
	s := b.sketchPolygon(f, vertices, name)
	if s == nil {
		return nil
	}
	return b.extrudeProfile(s, 0, depth, name)
}

// revolvePolygon revolves the closed polygon of vertices on f a full turn about the line through f's origin along U.
func (b *bladeBuilder) revolvePolygon(f r3.Frame, vertices [][2]float64, name string) *decad.Body {
	s := b.sketchPolygon(f, vertices, name)
	pr := b.profile(s, 0, name)
	if pr == nil {
		return nil
	}
	body, err := decad.Revolve(b.ctx, b.doc, s, pr, decad.ConstructionAxis{Origin: f.Origin(), Dir: f.U()},
		decad.FullRevolution{})
	b.fail("revolve "+name, err)
	return body
}

// extrudeProfile solves s, requires it fully constrained and extrudes its one valid profile with the given holes by
// length along its normal.
func (b *bladeBuilder) extrudeProfile(s *sketch.Sketch, holes int, length float64, name string) *decad.Body {
	pr := b.profile(s, holes, name)
	if pr == nil {
		return nil
	}
	body, err := decad.Extrude(b.ctx, b.doc, s, pr, decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
	b.fail("extrude "+name, err)
	return body
}

// profile solves s, requires it fully constrained and returns its one valid profile with the given holes.
func (b *bladeBuilder) profile(s *sketch.Sketch, holes int, name string) *sketch.Profile {
	if b.err != nil || s == nil {
		return nil
	}
	if _, err := s.Solve(b.ctx); err != nil {
		b.fail("solve "+name, err)
		return nil
	}
	if err := sketchcheck.VerifyFullyConstrained(b.ctx, s); err != nil {
		b.fail("verify "+name, err)
		return nil
	}
	var chosen *sketch.Profile
	for _, pr := range s.Profiles() {
		if !pr.Valid || len(pr.Holes) != holes {
			continue
		}
		if chosen != nil {
			b.fail("select "+name+" profile", errors.New("several valid profiles"))
			return nil
		}
		chosen = pr
	}
	if chosen == nil {
		b.fail("select "+name+" profile", fmt.Errorf("no valid profile with %d holes", holes))
	}
	return chosen
}
