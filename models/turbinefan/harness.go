package turbinefan

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Nose rib sizes, in mm or radians. They are print-process sizes, not standard dimensions; instructions.md "Print pose
// and nose rib" gives each one's reason.
const (
	// RibGapZ is the vertical gap from the rib's top to the blade's nose: two 0.08 mm layers.
	RibGapZ = 0.16
	// RibTopWidth is the rib's top width across its track: three 0.4 mm lines, wider than the nose strip it holds.
	RibTopWidth = 1.2
	// RibTopBand is how far beyond the top's half width the top's height reads the blade.
	RibTopBand = 0.2
	// RibSideAngle is each side face's angle from vertical; it keeps the rib's height under 3.7 times its base width.
	RibSideAngle = 6 * math.Pi / 180
	// RibTipMargin is how far before the blade's greatest x the rib ends.
	RibTipMargin = 1.0
	// RibEndOver is how far the wedge runs past each end of the profile slab, so their end faces never coincide.
	RibEndOver = 1.0
	// RibMinHeight is the least top height the rib may have: five 0.08 mm layers.
	RibMinHeight = 0.4
	// BrimClearance is the free bed the rib leaves round each of the blade's bed spots for the slicer's brim on the
	// blade: a 5 mm brim, its 0.1 gap to the object, half a line where its last loop spills over, and 0.45 for
	// first-layer squish.
	BrimClearance = 6.0
	// SectionStep is the spacing along x of the blade sections the rib reads.
	SectionStep = 0.25
)

// spotZ is the height of the section that reads the blade's bed spots: the middle of the first layer, which is what the
// slicer prints.
const spotZ = 0.1

// BladeHarness is the nose rib in the print frame: a wedge-shaped wall on the bed whose top follows the blade's
// leading-edge nose RibGapZ under it, from BrimClearance past the leading-edge bed spot to RibTipMargin before the tip.
type BladeHarness struct {
	Body *decad.Body
	// Sketches are the wedge's trapezoid, then the profile slab's polygon.
	Sketches []*sketch.Sketch
	// Spots are the blade's bed spots, its section loops at z = 0.1: the stud flat's, then the leading edge's.
	Spots [2][]r3.Vec
	// TrackY0 and TrackSlope give the rib's centre line on the bed, y = TrackY0 + TrackSlope x: the least-squares
	// line through the blade's lowest section points.
	TrackY0, TrackSlope float64
	// Top is the rib's top profile (x, z), one point per section plane from XStart to XEnd.
	Top [][2]float64
	// XStart and XEnd are the rib's x range, both on section planes.
	XStart, XEnd float64
}

// BuildBladeHarness builds the nose rib for blade, which must already be in the print pose PrintPoseFor(sz, pitch,
// hand, tilt) gives. It reads the blade with SectionOutline. It puts the rib in no container. ctx, doc and blade must not
// be nil.
func BuildBladeHarness(ctx context.Context, doc *decad.Document, blade *decad.Body, sz Size, hand Hand,
	tilt float64) (*BladeHarness, error) {
	if ctx == nil || doc == nil || blade == nil {
		return nil, errors.New("turbinefan: context, document and blade must not be nil")
	}
	if !sz.valid() {
		return nil, errors.New("turbinefan: sz must come from SizeFor")
	}
	if hand != RightHand && hand != LeftHand {
		return nil, errors.New("turbinefan: hand must be RightHand or LeftHand")
	}
	// The pose is not read, but a tilt PrintPoseFor rejects is rejected here too.
	if _, err := PrintPoseFor(sz, DefaultBladePitch, hand, tilt); err != nil {
		return nil, err
	}
	b := &bladeBuilder{ctx: ctx, doc: doc, sz: sz, mirror: hand == LeftHand, what: "harness"}
	h := b.harness(blade)
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

// sections cuts the blade at every SectionStep across its x range (step 1).
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

// harness runs construction steps 1 to 8: the blade's sections and bed spots, the rib's range, track and top, the
// wedge, the profile slab and their intersection.
func (b *bladeBuilder) harness(blade *decad.Body) *BladeHarness {
	box, err := blade.Bounds()
	if err != nil {
		b.fail("measure blade", err)
		return nil
	}
	secs := b.sections(blade, box)
	h := &BladeHarness{}
	b.spots(h, blade)
	if b.err != nil {
		return nil
	}
	// 3: from the first plane BrimClearance past the leading-edge spot to the last one RibTipMargin before the tip.
	_, edge := loopBox(h.Spots[1])
	first := int(math.Ceil((edge.X + BrimClearance - secs.x0) / SectionStep))
	last := int(math.Floor((box.Max.X - RibTipMargin - secs.x0) / SectionStep))
	if last-first+1 < 8 {
		b.fail("rib range", fmt.Errorf("%d section planes", last-first+1))
		return nil
	}
	plane := func(i int) float64 { return secs.x0 + float64(i)*SectionStep }
	h.XStart, h.XEnd = plane(first), plane(last)
	b.ribTrack(h, secs, first, last, plane)
	b.ribTop(h, secs, first, last, plane)
	if b.err != nil {
		return nil
	}
	wedge := b.ribWedge(h)
	slab := b.ribSlab(h)
	if b.err != nil {
		return nil
	}
	rib, err := decad.Intersect(b.ctx, wedge, slab)
	b.fail("rib", err)
	if b.err != nil {
		return nil
	}
	if !rib.IsSolid() || len(rib.Lumps()) != 1 {
		b.fail("rib", fmt.Errorf("solid %v, %d pieces", rib.IsSolid(), len(rib.Lumps())))
		return nil
	}
	h.Body = rib
	return h
}

// ribTrack fits the rib's centre line (step 4): the least-squares line through the lowest vertex of each section from
// first to last. It fails when the nose strays more than RibTopWidth/2 + RibTopBand from the line.
func (b *bladeBuilder) ribTrack(h *BladeHarness, secs bladeSections, first, last int, plane func(int) float64) {
	var xs, ys []float64
	for i := first; i <= last; i++ {
		x := plane(i)
		low := r3.Vec{Z: math.Inf(1)}
		for _, l := range secs.near(x) {
			for _, p := range l {
				if p.Z < low.Z {
					low = p
				}
			}
		}
		if math.IsInf(low.Z, 1) {
			b.fail("rib track", fmt.Errorf("x %.2f: no blade section", x))
			return
		}
		xs, ys = append(xs, x), append(ys, low.Y)
	}
	h.TrackY0, h.TrackSlope = fitLine(xs, ys)
	dev := 0.0
	for i := range xs {
		dev = math.Max(dev, math.Abs(ys[i]-(h.TrackY0+h.TrackSlope*xs[i])))
	}
	if dev > RibTopWidth/2+RibTopBand {
		b.fail("rib track", fmt.Errorf("the nose strays %.2f mm from the rib's line", dev))
	}
}

// ribTop sets the rib's top profile (step 5): on each plane, RibGapZ under the lowest point of the blade's outline
// within RibTopWidth/2 + RibTopBand of the track, every outline segment clipped to that band.
func (b *bladeBuilder) ribTop(h *BladeHarness, secs bladeSections, first, last int, plane func(int) float64) {
	if b.err != nil {
		return
	}
	band := RibTopWidth/2 + RibTopBand
	for i := first; i <= last; i++ {
		x := plane(i)
		yc := h.TrackY0 + h.TrackSlope*x
		low := math.Inf(1)
		for _, l := range secs.near(x) {
			for k := range l {
				if z, ok := clippedLow(l[k], l[(k+1)%len(l)], yc-band, yc+band); ok {
					low = math.Min(low, z)
				}
			}
		}
		if math.IsInf(low, 1) {
			b.fail("rib top", fmt.Errorf("x %.2f: no blade over the rib", x))
			return
		}
		if z := low - RibGapZ; z < RibMinHeight {
			b.fail("rib top", fmt.Errorf("x %.2f: top %.2f mm over the bed, under %g", x, z, RibMinHeight))
			return
		}
		h.Top = append(h.Top, [2]float64{x, low - RibGapZ})
	}
}

// clippedLow clips the segment p-q to lo <= y <= hi and returns the lower z of the clipped endpoints, and whether any of
// the segment lies in the band.
func clippedLow(p, q r3.Vec, lo, hi float64) (float64, bool) {
	t0, t1 := 0.0, 1.0
	dy := q.Y - p.Y
	if dy == 0 {
		if p.Y < lo || p.Y > hi {
			return 0, false
		}
	} else {
		a, c := (lo-p.Y)/dy, (hi-p.Y)/dy
		t0, t1 = math.Max(t0, math.Min(a, c)), math.Min(t1, math.Max(a, c))
		if t0 > t1 {
			return 0, false
		}
	}
	return math.Min(p.Z+t0*(q.Z-p.Z), p.Z+t1*(q.Z-p.Z)), true
}

// fitLine returns the least-squares line v = v0 + slope u through the points (us[i], vs[i]).
func fitLine(us, vs []float64) (float64, float64) {
	var su, sv, suu, suv float64
	for i := range us {
		su, sv, suu, suv = su+us[i], sv+vs[i], suu+us[i]*us[i], suv+us[i]*vs[i]
	}
	n := float64(len(us))
	slope := (n*suv - su*sv) / (n*suu - su*su)
	return (sv - slope*su) / n, slope
}

// ribGeometry is the wedge's placement (step 6): the track direction u, the across direction v, the top line's rise
// per mm along u (su), its height z = c + s x raised by off over every top point, and the wedge's x range xa..xb.
type ribGeometry struct {
	u, v       r3.Vec
	c, s, off  float64
	su, k      float64
	xa, xb, hv float64
}

// ribPlacement computes the wedge's placement from h's track and top.
func ribPlacement(h *BladeHarness) ribGeometry {
	var g ribGeometry
	g.u, _ = r3.Vec{X: 1, Y: h.TrackSlope}.Normalize()
	g.v = r3.Vec{Z: 1}.Cross(g.u)
	xs, zs := make([]float64, len(h.Top)), make([]float64, len(h.Top))
	for i, q := range h.Top {
		xs[i], zs[i] = q[0], q[1]
	}
	g.c, g.s = fitLine(xs, zs)
	for _, q := range h.Top {
		g.off = math.Max(g.off, q[1]-(g.c+g.s*q[0]))
	}
	g.off += 0.05
	g.su = g.s * g.u.X
	g.k = math.Sqrt(1 + g.su*g.su)
	g.xa, g.xb = h.XStart-RibEndOver, h.XEnd+RibEndOver
	g.hv = g.c + g.s*g.xb + g.off + 1
	return g
}

// ribWedge builds the sheared wedge (step 6): a trapezoid RibTopWidth wide at its top, sides RibSideAngle from vertical,
// on a frame whose normal runs along the track and rises with the top line, extruded from xa to xb. Across the track it
// is the trapezoid with its top on the line c + s x + off, reaching z = -1 or lower everywhere.
func (b *bladeBuilder) ribWedge(h *BladeHarness) *decad.Body {
	g := ribPlacement(h)
	n, _ := g.u.Add(r3.Vec{Z: g.su}).Normalize()
	origin := r3.Vec{X: g.xa, Y: h.TrackY0 + h.TrackSlope*g.xa, Z: g.c + g.s*g.xa + g.off}
	s, err := newSketch(frame(origin, g.v, n.Cross(g.v)))
	if err != nil {
		b.fail("create rib wedge", err)
		return nil
	}
	b.sketches = append(b.sketches, s)
	w, depth, run := RibTopWidth, g.hv/g.k, g.hv*math.Tan(RibSideAngle)
	// The corners are created where they belong, so the solver starts on the answer; the constraints, not fixes, hold
	// them: TL anchored, the top and bottom horizontal, the top w wide, TL depth over BL, and each side running run
	// outward over that depth.
	tl := sketchcheck.DimensionPoint(s, -w/2, 0)
	tr := s.CreatePoint(w/2, 0)
	br := s.CreatePoint(w/2+run, -depth)
	bl := s.CreatePoint(-(w/2 + run), -depth)
	top, bottom := s.CreateLine(tl, tr), s.CreateLine(br, bl)
	s.CreateLine(tr, br)
	s.CreateLine(bl, tl)
	s.AddConstraint(
		sketch.NewHorizontal(top), sketch.NewHorizontalDistance(tl, tr, w),
		sketch.NewHorizontal(bottom), sketch.NewVerticalDistance(bl, tl, depth),
		sketch.NewHorizontalDistance(bl, tl, run), sketch.NewHorizontalDistance(tr, br, run),
	)
	return b.extrudeProfile(s, 0, (g.xb-g.xa)/g.u.X*g.k, "rib wedge")
}

// ribSlab builds the profile slab (step 7): the polygon from (XStart, 0) to (XEnd, 0) and back along Top, on the XZ
// plane, extruded across the wedge's y range and 2 mm more each side. Its bottom is the rib's bed face.
func (b *bladeBuilder) ribSlab(h *BladeHarness) *decad.Body {
	g := ribPlacement(h)
	ya, yb := h.TrackY0+h.TrackSlope*g.xa, h.TrackY0+h.TrackSlope*g.xb
	reach := RibTopWidth/2 + g.hv*math.Tan(RibSideAngle) + 2
	ylo, yhi := math.Min(ya, yb)-reach, math.Max(ya, yb)+reach
	poly := [][2]float64{{h.XStart, 0}, {h.XEnd, 0}}
	for i := len(h.Top) - 1; i >= 0; i-- {
		poly = append(poly, h.Top[i])
	}
	return b.extrudePolygon(frame(r3.Vec{Y: yhi}, r3.Vec{X: 1}, r3.Vec{Z: 1}), poly, yhi-ylo, "rib profile")
}

// spots reads the blade's two bed spots, its section loops at z = 0.1, into h.Spots, the stud flat's first (step 2).
func (b *bladeBuilder) spots(h *BladeHarness, blade *decad.Body) {
	if b.err != nil {
		return
	}
	f := frame(r3.Vec{Z: spotZ}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	o, err := blade.SectionOutline(b.ctx, f, units.Millimeters(0.02))
	if err != nil {
		o, err = blade.SectionOutline(b.ctx, f, units.Millimeters(0.05))
	}
	if err != nil {
		b.fail("bed spots", fmt.Errorf("section blade: %w", err))
		return
	}
	if n := len(o.Loops); n != 2 {
		b.fail("bed spots", fmt.Errorf("%d bed spots", n))
		return
	}
	loops := slices.Clone(o.Loops)
	// The stud flat's spot lies nearer the stud end, at the smaller x.
	slices.SortFunc(loops, func(p, q []r3.Vec) int { return cmp.Compare(meanX(p), meanX(q)) })
	copy(h.Spots[:], loops)
}

// loopBox returns the least and the greatest corner of a loop's bounding box in the XY plane.
func loopBox(l []r3.Vec) (r3.Vec, r3.Vec) {
	lo, hi := r3.Vec{X: math.Inf(1), Y: math.Inf(1)}, r3.Vec{X: math.Inf(-1), Y: math.Inf(-1)}
	for _, p := range l {
		lo = r3.Vec{X: math.Min(lo.X, p.X), Y: math.Min(lo.Y, p.Y)}
		hi = r3.Vec{X: math.Max(hi.X, p.X), Y: math.Max(hi.Y, p.Y)}
	}
	return lo, hi
}

// meanX returns the mean x of the points of l.
func meanX(l []r3.Vec) float64 {
	sum := 0.0
	for _, p := range l {
		sum += p.X
	}
	return sum / float64(len(l))
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
