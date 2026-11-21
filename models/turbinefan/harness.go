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

// Print cradle sizes, in mm. They are print-process sizes, not standard dimensions; instructions.md "Print pose and
// cradle" gives each one's reason. CradleGapZ is in print.go.
const (
	// CradleGapXY is the horizontal support gap between the cradle and the blade: about one line width.
	CradleGapXY = 0.4
	// CradleWall is the cradle wall's width beyond each of the blade's widest lines: five lines.
	CradleWall = 2.0
	// CradleTopDrop is how far each wall's top stands under its widest point.
	CradleTopDrop = 0.3
	// CradleStep is the spacing along x of the cradle's sections.
	CradleStep = 0.5
	// CradleFitPoints is the number of fit points in each shadow section; the twisted loft allows 32 spans at most.
	CradleFitPoints = 33
	// CradleYStep is the y spacing of the lower envelope's samples.
	CradleYStep = 0.05
	// CradleTopBand is how far inside a widest point the blade's outline is read for the wall's height: the wall top is
	// set from the lowest outline point within this of the widest y, which a near-vertical face leaves steady.
	CradleTopBand = 0.1
	// CradleTopSmooth is the half-width, in mm along x, of the moving average that smooths each wall's top.
	CradleTopSmooth = 1.0
	// CradleTopPasses is the number of smoothing passes; each averages, then lowers every point back under its limit.
	CradleTopPasses = 8
	// CradleTipMargin is how far before the tip cap's lowest point the cradle ends.
	CradleTipMargin = 1.0
	// StudSlotFlats is the number of flats of the stud slot's half polygon.
	StudSlotFlats = 16
	// CradlePlanWindow is the x window over which the cradle plan's outer edge takes the widest point, so the bed face's
	// outline has no narrow notch.
	CradlePlanWindow = 1.0
	// SectionStep is the spacing along x of the blade sections the layout reads.
	SectionStep = 0.25
	// BrimClearance is the free bed the cradle leaves round each of the blade's bed spots over the first two layers, for
	// the slicer's brim on the blade: a 5 mm brim, its 0.1 gap to the object, half a line where its last loop spills
	// over, and 0.45 for first-layer squish.
	BrimClearance = 6.0
	// PocketFirstLayers is how far above the bed a brim pocket still clears BrimClearance: two layers.
	PocketFirstLayers = 0.4
	// PocketSlope is a brim pocket wall's angle from vertical, so the cradle over it prints without support.
	PocketSlope = 45 * math.Pi / 180
	// CollarWidth is the width of the ring round each brim pocket that closes it in the cradle's first layer.
	CollarWidth = 3.0
	// CollarHeight is a collar's height: four layers, under the caps' z = 1 plane.
	CollarHeight = 0.8
	// CollarInset is how far the collar's inner edge stands outside the pocket's edge at the bed, so the pocket's wall
	// meets no collar face there.
	CollarInset = 0.2
)

// spotZ is the height of the section that reads the blade's bed spots: the middle of the first layer, which is what the
// slicer prints.
const spotZ = 0.1

// CradleSection is one sampled cross-section of the cradle at plane x: the offset lower outline (y, z) from the left
// widest point to the right one, and those points.
type CradleSection struct {
	X              float64
	Outline        [][2]float64 // CradleFitPoints points, y strictly increasing
	LeftY, LeftZ   float64      // offset widest point, min y side
	RightY, RightZ float64      // offset widest point, max y side
	// LeftLimit and RightLimit are the lowest blade outline points within CradleTopBand of each widest y: the highest a
	// wall may stand there.
	LeftLimit, RightLimit float64
}

// BladeHarness is the cradle in the print frame. The cradle is one solid standing on the bed: a block minus the blade's
// upward shadow, offset from the blade by CradleGapZ and CradleGapXY, so it fits the blade's contour below its widest
// lines and the blade lifts straight up out of it. Where the blade is less than a few layers over the bed, near its
// stud end corner and its leading edge, the shadow cuts the cradle down to the bed. Round each of the blade's two bed
// spots the cradle leaves a brim pocket, BrimClearance clear of the spot over the first two layers, so the slicer's brim
// on the blade fits there; its walls rise inward at PocketSlope, except a side reaching past the block's outer face,
// which stays vertical, and a collar ring closes it in the first layer.
type BladeHarness struct {
	Body *decad.Body
	// Sketches are in build order: shadow sections, stud slot, block plan, stud collar, leading-edge collar, block
	// profile, left cap plan, left cap profile, right cap plan, right cap profile, bed box, then each pocket's XZ and YZ
	// trapezoids, the stud pocket's first.
	Sketches []*sketch.Sketch
	// Spots are the blade's bed spots, the loops of its section at z = 0.1: the stud corner's, then the leading edge's.
	Spots [2][]r3.Vec
	// Pockets are the brim pockets' rectangles at the bed, each as its least and greatest corner, in the order of Spots.
	Pockets [2][2]r3.Vec
	// Sections are every sampled section, x ascending; Sections[0].X is the shadow's start past the stud end.
	Sections []CradleSection
	// XStart and XEnd are the block's x range: XStart lies half a CradleStep past the first section beyond the stud
	// pocket, and XEnd is Sections[len-2].X.
	XStart, XEnd float64
	// StepZone is the x range where the shadow tool copies its first clean section, past the root step.
	StepZone [2]float64
	// WallTop is the top profile (x, z) of the left (min y) and the right wall over the block's sections, from XStart:
	// CradleTopDrop under the smoothed limit, the step zone's sections taking the first clean section's values.
	WallTop [2][][2]float64
}

// BuildBladeHarness builds the cradle for blade, which must already be in the print pose PrintPoseFor(sz, pitch, hand,
// tilt) gives. It reads the blade with SectionOutline. It puts the harness in no container. ctx, doc and blade must not be nil.
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
	// The posed loft does not depend on the pitch, so the analytic points use the default pitch's pose.
	pose, err := PrintPoseFor(sz, DefaultBladePitch, hand, tilt)
	if err != nil {
		return nil, err
	}
	b := &bladeBuilder{ctx: ctx, doc: doc, sz: sz, mirror: hand == LeftHand, what: "harness"}
	h := b.harness(blade, pose, hand, tilt)
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

// low returns the loop of the section nearest x that holds its lowest point: the outer loop, or below a through bolt
// hole the piece under the hole.
func (s bladeSections) low(x float64) []r3.Vec {
	var best []r3.Vec
	lowest := math.Inf(1)
	for _, l := range s.near(x) {
		for _, p := range l {
			if p.Z < lowest {
				best, lowest = l, p.Z
			}
		}
	}
	return best
}

// lowerAt returns the lowest z where the vertical line y meets loop's boundary, and whether it meets it.
func lowerAt(loop []r3.Vec, y float64) (float64, bool) {
	best, ok := math.Inf(1), false
	for i := range loop {
		a, c := loop[i], loop[(i+1)%len(loop)]
		if (a.Y-y)*(c.Y-y) > 0 || a.Y == c.Y {
			continue
		}
		z := a.Z + (y-a.Y)/(c.Y-a.Y)*(c.Z-a.Z)
		if z < best {
			best, ok = z, true
		}
	}
	return best, ok
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

// harness runs construction steps 2 to 10.
func (b *bladeBuilder) harness(blade *decad.Body, pose r3.Frame, hand Hand, tilt float64) *BladeHarness {
	box, err := blade.Bounds()
	if err != nil {
		b.fail("measure blade", err)
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
	sz := b.sz
	// 3: the cradle runs from 0.5 past the stud end rim to CradleTipMargin before the tip cap's lowest edge point.
	xs := math.Inf(-1)
	for k := range 72 {
		a := 2 * math.Pi * float64(k) / 72
		p := xf.Apply(r3.Vec{X: sz.BladeStudRadius * math.Cos(a), Y: sz.BladeStudRadius * math.Sin(a), Z: sz.BladeStudStart})
		xs = math.Max(xs, p.X)
	}
	xs += 0.5
	tip := stationsFor(sz)[9]
	xe := math.Inf(1)
	for _, x := range []float64{0, 1} {
		p := sectionLocal(tip, sz.Beta(tip.r, refPitch), x, 0)
		if hand == RightHand {
			p[0] = -p[0]
		}
		xe = math.Min(xe, xf.Apply(r3.Vec{X: p[0], Y: p[1], Z: sz.TipRadius}).X)
	}
	xe -= CradleTipMargin
	zTop := box.Max.Z + 3
	z0 := box.Max.Z + 2

	h := &BladeHarness{}
	h.Sections = b.cradleSections(secs, xs, xe, z0)
	if b.err != nil {
		return nil
	}
	if len(h.Sections) < 4 {
		b.fail("cradle sections", fmt.Errorf("%d sections from x %.2f to %.2f", len(h.Sections), xs, xe))
		return nil
	}
	h.XEnd = h.Sections[len(h.Sections)-2].X
	// 5: the root step, where the stud meets the root face.
	xRoot := xf.Apply(r3.Vec{Z: sz.RootRadius}).X
	half := 4 * sz.HubScale * math.Sin(tilt)
	h.StepZone = [2]float64{xRoot - half - CradleStep, xRoot + half + CradleStep}
	b.spots(h, blade)
	blk := blockSections(h, wallSections(h))
	if b.err != nil {
		return nil
	}
	if len(blk) < 4 {
		b.fail("block sections", fmt.Errorf("%d sections past the stud pocket", len(blk)))
		return nil
	}
	h.XStart = blk[0].X

	shadow := b.shadowTool(h, z0, zTop)
	stud := b.studTool(xf, zTop, tilt)
	block := b.block(h, blk, zTop)
	// 9: the cradle.
	if b.err != nil {
		return nil
	}
	cradle, err := decad.Cut(b.ctx, block, stud)
	b.fail("cut stud slot", err)
	if b.err != nil {
		return nil
	}
	cradle, err = decad.Cut(b.ctx, cradle, shadow)
	b.fail("cut blade shadow", err)
	if b.err != nil {
		return nil
	}
	if n := len(cradle.Lumps()); n != 1 {
		b.fail("cradle", fmt.Errorf("%d pieces", n))
		return nil
	}
	h.Body = cradle
	return h
}

// cradleSections samples the cradle's sections from xs to xe every CradleStep (step 4): the offset lower envelope of the
// blade over the slices within CradleStep, from its min-y to its max-y point, resampled at equal polar angles about
// (0, z0).
func (b *bladeBuilder) cradleSections(secs bladeSections, xs, xe, z0 float64) []CradleSection {
	var out []CradleSection
	n := int(math.Floor((xe-xs)/CradleStep + 1e-9))
	for k := 0; k <= n; k++ {
		x := xs + float64(k)*CradleStep
		var win [][]r3.Vec
		left, right := r3.Vec{Y: math.Inf(1)}, r3.Vec{Y: math.Inf(-1)}
		for xx := x - CradleStep; xx <= x+CradleStep+1e-9; xx += SectionStep {
			l := secs.low(xx)
			if l == nil {
				b.fail("cradle sections", fmt.Errorf("no blade section near x %.2f", xx))
				return nil
			}
			win = append(win, l)
			for _, p := range l {
				if p.Y < left.Y {
					left = p
				}
				if p.Y > right.Y {
					right = p
				}
			}
		}
		chain := [][2]float64{{left.Y, left.Z}}
		for y := left.Y; y <= right.Y+1e-9; y += CradleYStep {
			z := math.Inf(1)
			for _, l := range win {
				if v, ok := lowerAt(l, y); ok {
					z = math.Min(z, v)
				}
			}
			if !math.IsInf(z, 1) {
				chain = append(chain, [2]float64{y, z})
			}
		}
		chain = append(chain, [2]float64{right.Y, right.Z})
		chain = increasing(chain)
		// Offset along the outward normal (down for a left-to-right walk), by CradleGapXY across and CradleGapZ up; the
		// two ends straight out.
		off := make([][2]float64, len(chain))
		for i := range chain {
			a, c := chain[max(i-1, 0)], chain[min(i+1, len(chain)-1)]
			ty, tz := c[0]-a[0], c[1]-a[1]
			l := math.Hypot(ty, tz)
			off[i] = [2]float64{chain[i][0] + tz/l*CradleGapXY, chain[i][1] - ty/l*CradleGapZ}
		}
		last := len(chain) - 1
		off[0] = [2]float64{chain[0][0] - CradleGapXY, chain[0][1]}
		off[last] = [2]float64{chain[last][0] + CradleGapXY, chain[last][1]}
		mono := [][2]float64{off[0]}
		for _, q := range off[1:last] {
			if q[0] > mono[len(mono)-1][0]+1e-6 && q[0] < off[last][0]-1e-6 {
				mono = append(mono, q)
			}
		}
		mono = append(mono, off[last])
		pts := polarResample(mono, z0)
		// Each wall's limit is the lowest point of the window's outlines within CradleTopBand of that side's widest y,
		// read along every outline segment, so a near-vertical face gives a steady height on both sides alike.
		leftLimit, rightLimit := math.Inf(1), math.Inf(1)
		for _, l := range win {
			for i := range l {
				p0, p1 := l[i], l[(i+1)%len(l)]
				for k := 0; k <= 10; k++ {
					p := p0.Add(p1.Sub(p0).Scale(float64(k) / 10))
					if p.Y <= left.Y+CradleTopBand {
						leftLimit = math.Min(leftLimit, p.Z)
					}
					if p.Y >= right.Y-CradleTopBand {
						rightLimit = math.Min(rightLimit, p.Z)
					}
				}
			}
		}
		out = append(out, CradleSection{X: x, Outline: pts, LeftY: pts[0][0], LeftZ: pts[0][1],
			RightY: pts[len(pts)-1][0], RightZ: pts[len(pts)-1][1], LeftLimit: leftLimit, RightLimit: rightLimit})
	}
	return out
}

// increasing drops chain points within 1e-6 of the previous one or with a smaller y.
func increasing(chain [][2]float64) [][2]float64 {
	var out [][2]float64
	for _, q := range chain {
		if n := len(out); n > 0 && (math.Hypot(q[0]-out[n-1][0], q[1]-out[n-1][1]) < 1e-6 || q[0] < out[n-1][0]) {
			continue
		}
		out = append(out, q)
	}
	return out
}

// polarResample resamples the polyline at CradleFitPoints equal polar angles about (0, z0), so every section's point k
// lies near one ray and the twisted loft's interpolated sections stay star-shaped.
func polarResample(line [][2]float64, z0 float64) [][2]float64 {
	ang := make([]float64, len(line))
	for i, q := range line {
		ang[i] = math.Atan2(q[1]-z0, q[0])
		if i == 0 {
			continue
		}
		for ang[i] < ang[i-1]-math.Pi {
			ang[i] += 2 * math.Pi
		}
		for ang[i] > ang[i-1]+math.Pi {
			ang[i] -= 2 * math.Pi
		}
	}
	out := make([][2]float64, CradleFitPoints)
	j := 0
	for k := range out {
		th := ang[0] + (ang[len(ang)-1]-ang[0])*float64(k)/float64(CradleFitPoints-1)
		for j+1 < len(ang)-1 && ang[j+1] < th {
			j++
		}
		t := 0.0
		if ang[j+1] != ang[j] {
			t = (th - ang[j]) / (ang[j+1] - ang[j])
		}
		t = math.Max(0, math.Min(1, t))
		out[k] = [2]float64{line[j][0] + t*(line[j+1][0]-line[j][0]), line[j][1] + t*(line[j+1][1]-line[j][1])}
	}
	return out
}

// shadowTool lofts the blade's upward shadow from the root to the tip (step 6) through the sections from the step zone's
// lead-in on; those before the step zone's end copy the first section past it. Each section is on a world XY plane at
// its x, its offset lower outline closed upward to zTop, lofted with Z along world x and turned into place.
func (b *bladeBuilder) shadowTool(h *BladeHarness, z0, zTop float64) *decad.Body {
	if b.err != nil {
		return nil
	}
	var clean *CradleSection
	for i := range h.Sections {
		if h.Sections[i].X >= h.StepZone[1] {
			clean = &h.Sections[i]
			break
		}
	}
	if clean == nil {
		b.fail("shadow tool", errors.New("no section past the root step"))
		return nil
	}
	var loft []decad.LoftSection
	for _, c := range h.Sections {
		if c.X < h.StepZone[0]-1.0-1e-9 {
			continue
		}
		src := c
		if c.X < h.StepZone[1] {
			src = *clean
		}
		loft = append(loft, b.shadowSection(c.X, src, z0, zTop))
		if b.err != nil {
			return nil
		}
	}
	body, err := decad.LoftSections(b.ctx, b.doc, loft...)
	if err != nil {
		b.fail("loft blade shadow", err)
		return nil
	}
	xf, err := r3.FromFrame(frame(r3.Vec{Z: z0}, r3.Vec{Y: 1}, r3.Vec{Z: 1}))
	if err != nil {
		b.fail("blade shadow frame", err)
		return nil
	}
	out, err := decad.Transform(b.ctx, body, xf)
	b.fail("place blade shadow", err)
	return out
}

// shadowSection sketches one shadow section on the world XY plane at height x: a fit spline through src's outline,
// shifted down by z0, closed by lines up to zTop and across.
func (b *bladeBuilder) shadowSection(x float64, src CradleSection, z0, zTop float64) decad.LoftSection {
	w := sketch.NewWorld()
	plane, err := w.CreateOffsetPlane(w.XY(), x)
	if err != nil {
		b.fail("create shadow plane", err)
		return decad.LoftSection{}
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		b.fail("create shadow section", err)
		return decad.LoftSection{}
	}
	b.sketches = append(b.sketches, s)
	fit := make([]*sketch.Point, len(src.Outline))
	for i, p := range src.Outline {
		fit[i] = sketchcheck.DimensionPoint(s, p[0], p[1]-z0)
	}
	right := sketchcheck.DimensionPoint(s, src.RightY, zTop-z0)
	left := sketchcheck.DimensionPoint(s, src.LeftY, zTop-z0)
	if _, err := s.CreateFitSpline(fit...); err != nil {
		b.fail("create shadow spline", err)
		return decad.LoftSection{}
	}
	s.CreateLine(fit[len(fit)-1], right)
	s.CreateLine(right, left)
	s.CreateLine(left, fit[0])
	if _, err := s.Solve(b.ctx); err != nil {
		b.fail("solve shadow section", err)
		return decad.LoftSection{}
	}
	if err := sketchcheck.VerifyFullyConstrained(b.ctx, s); err != nil {
		b.fail("verify shadow section", fmt.Errorf("x %.2f: %w", x, err))
		return decad.LoftSection{}
	}
	for _, pr := range s.Profiles() {
		if pr.Valid && len(pr.Holes) == 0 && len(pr.Outer) == 4 {
			return decad.LoftSection{Sketch: s, Profile: pr}
		}
	}
	b.fail("select shadow profile", fmt.Errorf("x %.2f: no valid profile of four edges", x))
	return decad.LoftSection{}
}

// studTool is the stud's U slot (step 7): a half polygon of StudSlotFlats flats circumscribing a half circle of radius
// BladeStudRadius + CradleGapXY, centred CradleGapXY - CradleGapZ under the stud axis, carried straight up, extruded along
// the axis from 1 mm before the stud end to 0.5 past the root.
func (b *bladeBuilder) studTool(xf r3.Transform, zTop, t float64) *decad.Body {
	if b.err != nil {
		return nil
	}
	sz := b.sz
	f := frame(xf.Apply(r3.Vec{Z: sz.BladeStudStart - 1}), r3.Vec{X: -math.Sin(t), Z: math.Cos(t)}, r3.Vec{Y: -1})
	rr := sz.BladeStudRadius + CradleGapXY
	h0 := -(CradleGapXY - CradleGapZ)
	rv := rr / math.Cos(math.Pi/(2*StudSlotFlats))
	vertices := [][2]float64{{h0, rr}}
	for k := range StudSlotFlats {
		a := math.Pi/2 + math.Pi*(float64(k)+0.5)/StudSlotFlats
		vertices = append(vertices, [2]float64{h0 + rv*math.Cos(a), rv * math.Sin(a)})
	}
	vertices = append(vertices, [2]float64{h0, -rr}, [2]float64{zTop + 5, -rr}, [2]float64{zTop + 5, rr})
	return b.extrudePolygon(f, vertices, sz.RootRadius+0.5-sz.BladeStudStart+1, "stud slot")
}

// wallSections returns the block's sections (all but the first and last) for the walls: those in the shadow tool's
// copied range, the step zone and its lead-in, take the first clean section's widest points and limits, as the shadow
// tool takes its outline. A raw section there mixes the stud's and the root's slices and would raise a fin on each
// wall.
func wallSections(h *BladeHarness) []CradleSection {
	blk := slices.Clone(h.Sections[1 : len(h.Sections)-1])
	var clean *CradleSection
	for i := range h.Sections {
		if h.Sections[i].X >= h.StepZone[1] {
			clean = &h.Sections[i]
			break
		}
	}
	if clean == nil {
		return blk
	}
	for i, c := range blk {
		if c.X >= h.StepZone[0]-1.0-1e-9 && c.X < h.StepZone[1] {
			blk[i] = *clean
			blk[i].X = c.X
		}
	}
	return blk
}

// wallTop returns a wall's top profile over blk (step 8): the limits limit(c), smoothed under themselves, less
// CradleTopDrop. Each of CradleTopPasses passes averages the profile over CradleTopSmooth and then lowers every point
// that rose above its own limit back to it, so the top stays CradleTopDrop under each section's limit and between
// sections under the straight line between their limits.
func wallTop(blk []CradleSection, limit func(CradleSection) float64) [][2]float64 {
	lim := make([]float64, len(blk))
	for i, c := range blk {
		lim[i] = limit(c)
	}
	z := slices.Clone(lim)
	for range CradleTopPasses {
		next := make([]float64, len(z))
		for i := range blk {
			sum, n := 0.0, 0
			for j := range blk {
				if math.Abs(blk[i].X-blk[j].X) <= CradleTopSmooth+1e-9 {
					sum, n = sum+z[j], n+1
				}
			}
			next[i] = math.Min(sum/float64(n), lim[i])
		}
		z = next
	}
	out := make([][2]float64, len(blk))
	for i, c := range blk {
		out[i] = [2]float64{c.X, z[i] - CradleTopDrop}
	}
	return out
}

// block builds the cradle's block (step 8): the plan from CradleWall outside each widest line, intersected with a slab
// 0.5 over the higher wall top, then cut by a cap on each side that removes what lies beyond that side's widest line and
// above its wall top.
func (b *bladeBuilder) block(h *BladeHarness, blk []CradleSection, zTop float64) *decad.Body {
	if b.err != nil {
		return nil
	}
	h.WallTop[0] = wallTop(blk, func(c CradleSection) float64 { return c.LeftLimit })
	h.WallTop[1] = wallTop(blk, func(c CradleSection) float64 { return c.RightLimit })
	// The plan's outer edge takes the widest point within CradlePlanWindow, so the bed face's outline has no narrow
	// notch.
	outer := func(x, side float64) float64 {
		v := math.Inf(-1)
		for _, c := range blk {
			if math.Abs(c.X-x) > CradlePlanWindow+1e-9 {
				continue
			}
			y := c.RightY
			if side < 0 {
				y = -c.LeftY
			}
			v = math.Max(v, y)
		}
		return side * v
	}
	var plan, bandL, bandR, topMax [][2]float64
	capL, capR := slices.Clone(h.WallTop[0]), slices.Clone(h.WallTop[1])
	for i, c := range blk {
		plan = append(plan, [2]float64{c.X, outer(c.X, -1) - CradleWall})
		bandL = append(bandL, [2]float64{c.X, outer(c.X, -1) - CradleWall - 1})
		bandR = append(bandR, [2]float64{c.X, c.RightY - 0.3})
		topMax = append(topMax, [2]float64{c.X, math.Max(capL[i][1], capR[i][1]) + 0.5})
	}
	for i := len(blk) - 1; i >= 0; i-- {
		plan = append(plan, [2]float64{blk[i].X, outer(blk[i].X, 1) + CradleWall})
		bandL = append(bandL, [2]float64{blk[i].X, blk[i].LeftY + 0.3})
		bandR = append(bandR, [2]float64{blk[i].X, outer(blk[i].X, 1) + CradleWall + 1})
	}
	ymin, ymax, zmax := math.Inf(1), math.Inf(-1), 0.0
	for _, q := range plan {
		ymin, ymax = math.Min(ymin, q[1]), math.Max(ymax, q[1])
	}
	for _, q := range topMax {
		zmax = math.Max(zmax, q[1])
	}
	// The slabs reach 20 mm past the plan's y range, so the profile slab does not trim the collars.
	ymin, ymax = ymin-20, ymax+20
	// slab is the XZ polygon between the lower and upper polylines, extruded across the plan's y range.
	slab := func(lo, hi [][2]float64, name string) *decad.Body {
		poly := append([][2]float64{}, lo...)
		for i := len(hi) - 1; i >= 0; i-- {
			poly = append(poly, hi[i])
		}
		return b.extrudePolygon(frame(r3.Vec{Y: ymax + 2}, r3.Vec{X: 1}, r3.Vec{Z: 1}), poly, ymax-ymin+4, name)
	}
	flat := func(z float64, src [][2]float64) [][2]float64 {
		out := make([][2]float64, len(src))
		for i, q := range src {
			out[i] = [2]float64{q[0], z}
		}
		return out
	}
	// The prism and the collars start under the bed, and the profile slab under them, reaching 20 mm past the block's
	// ends at the end heights, so it trims no collar; the bed box gives the single bed face after the caps. No faces
	// coincide.
	prism := b.extrudePolygon(frame(r3.Vec{Z: -1}, r3.Vec{X: 1}, r3.Vec{Y: 1}), plan, zmax+2, "block plan")
	for i := range h.Spots {
		collar := b.collar(h, i)
		if b.err != nil {
			return nil
		}
		var err error
		prism, err = decad.Union(b.ctx, prism, collar)
		b.fail(fmt.Sprintf("join collar %d", i), err)
	}
	n := len(topMax)
	reach := append([][2]float64{{topMax[0][0] - 20, topMax[0][1]}}, topMax...)
	reach = append(reach, [2]float64{topMax[n-1][0] + 20, topMax[n-1][1]})
	top := slab(flat(-2, reach), reach, "block profile")
	if b.err != nil {
		return nil
	}
	body, err := decad.Intersect(b.ctx, prism, top)
	b.fail("intersect block", err)
	// The caps reach 1 mm past the block's ends, so no end faces coincide.
	stretch := func(band [][2]float64) [][2]float64 {
		n := len(band) / 2
		band[0][0] -= 1
		band[n-1][0] += 1
		band[n][0] += 1
		band[len(band)-1][0] -= 1
		return band
	}
	extend := func(l [][2]float64) [][2]float64 {
		out := append([][2]float64{{l[0][0] - 1, l[0][1]}}, l...)
		return append(out, [2]float64{l[len(l)-1][0] + 1, l[len(l)-1][1]})
	}
	for _, c := range []struct {
		band, line [][2]float64
		name       string
	}{{stretch(bandL), extend(capL), "left cap"}, {stretch(bandR), extend(capR), "right cap"}} {
		capPlan := b.extrudePolygon(frame(r3.Vec{Z: 1}, r3.Vec{X: 1}, r3.Vec{Y: 1}), c.band, zTop+2, c.name+" plan")
		capTop := slab(c.line, flat(zTop+1, c.line), c.name+" profile")
		if b.err != nil {
			return nil
		}
		cp, err := decad.Intersect(b.ctx, capPlan, capTop)
		b.fail("intersect "+c.name, err)
		if b.err != nil {
			return nil
		}
		body, err = decad.Cut(b.ctx, body, cp)
		b.fail("cut "+c.name, err)
		if b.err != nil {
			return nil
		}
	}
	// The bed box keeps what lies over the bed: its own extent is the block's bounds grown by 5 mm.
	box, err := body.Bounds()
	if err != nil {
		b.fail("measure block", err)
		return nil
	}
	bedBox := b.extrudePolygon(frame(r3.Vec{}, r3.Vec{X: 1}, r3.Vec{Y: 1}), [][2]float64{
		{box.Min.X - 5, box.Min.Y - 5}, {box.Max.X + 5, box.Min.Y - 5}, {box.Max.X + 5, box.Max.Y + 5},
		{box.Min.X - 5, box.Max.Y + 5},
	}, box.Max.Z+5, "bed box")
	if b.err != nil {
		return nil
	}
	body, err = decad.Intersect(b.ctx, body, bedBox)
	b.fail("intersect bed box", err)
	for i := range h.Spots {
		// A pocket side whose bed edge lies past the plan's outer edge anywhere over the pocket's x range is open: a
		// sloped wall there would start outside the block and meet its outer face above the bed, leaving a fin with
		// nothing under its lowest edge.
		var open [2]bool
		lo, hi := h.Pockets[i][0], h.Pockets[i][1]
		for _, c := range blk {
			if c.X < lo.X || c.X > hi.X {
				continue
			}
			open[0] = open[0] || outer(c.X, -1)-CradleWall > lo.Y
			open[1] = open[1] || outer(c.X, 1)+CradleWall < hi.Y
		}
		pocket := b.pocket(h, i, open)
		if b.err != nil {
			return nil
		}
		body, err = decad.Cut(b.ctx, body, pocket)
		b.fail(fmt.Sprintf("cut brim pocket %d", i), err)
		if b.err != nil {
			return nil
		}
	}
	return body
}

// spots reads the blade's two bed spots, its section loops at z = 0.1, into h.Spots, the stud corner's first, and sets
// h.Pockets: each spot's bounding box grown by BrimClearance + PocketFirstLayers.
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
	// The stud corner's spot lies nearer the stud end, at the smaller x.
	slices.SortFunc(loops, func(p, q []r3.Vec) int { return cmp.Compare(meanX(p), meanX(q)) })
	d := BrimClearance + PocketFirstLayers
	for i, l := range loops {
		lo, hi := loopBox(l)
		h.Spots[i] = l
		h.Pockets[i] = [2]r3.Vec{{X: lo.X - d, Y: lo.Y - d}, {X: hi.X + d, Y: hi.Y + d}}
	}
}

// blockSections returns the block's sections from blk: those from the first one at least 0.5 past the stud pocket's
// end, the first moved on by CradleStep/2, so the block leaves no wall sliver over the pocket's slope and its start
// face lies between the shadow loft's section planes.
func blockSections(h *BladeHarness, blk []CradleSection) []CradleSection {
	x0 := h.Pockets[0][1].X + 0.5
	for len(blk) > 0 && blk[0].X < x0 {
		blk = blk[1:]
	}
	if len(blk) == 0 {
		return nil
	}
	blk = slices.Clone(blk)
	blk[0].X += CradleStep / 2
	return blk
}

// collar builds the ring round brim pocket i, from z = -1 to CollarHeight: the pocket's bed rectangle grown by
// CollarInset is its hole, grown by CollarInset + CollarWidth its outline.
func (b *bladeBuilder) collar(h *BladeHarness, i int) *decad.Body {
	if b.err != nil {
		return nil
	}
	name := fmt.Sprintf("collar %d", i)
	outline := rectAround(h.Pockets[i], CollarInset+CollarWidth)
	s := b.sketchPolygon(frame(r3.Vec{Z: -1}, r3.Vec{X: 1}, r3.Vec{Y: 1}), outline, name)
	if s == nil {
		return nil
	}
	hole := rectAround(h.Pockets[i], CollarInset)
	points := make([]*sketch.Point, len(hole))
	for k, p := range hole {
		points[k] = sketchcheck.DimensionPoint(s, p[0], p[1])
	}
	for k := range points {
		s.CreateLine(points[k], points[(k+1)%len(points)])
	}
	return b.extrudeProfile(s, 1, 1+CollarHeight, name)
}

// pocket builds brim pocket i: a frustum from z = -1 up to the spot's box at height BrimClearance + PocketFirstLayers,
// whose walls rise inward at PocketSlope from the pocket's bed rectangle. A side open[0] (min y) or open[1] (max y)
// marks stays vertical at the pocket's edge instead. It is an XZ trapezoid extruded along y intersected with a YZ
// trapezoid extruded along x, so every face is planar.
func (b *bladeBuilder) pocket(h *BladeHarness, i int, open [2]bool) *decad.Body {
	if b.err != nil {
		return nil
	}
	d := BrimClearance + PocketFirstLayers
	// PocketSlope is from vertical, so a wall runs inward tan(PocketSlope) per mm of height.
	run := d * math.Tan(PocketSlope) // from the bed up to height d
	under := math.Tan(PocketSlope)   // from z = -1 up to the bed
	lo, hi := h.Pockets[i][0], h.Pockets[i][1]
	name := fmt.Sprintf("brim pocket %d", i)
	alongY := b.extrudePolygon(frame(r3.Vec{Y: hi.Y + 5}, r3.Vec{X: 1}, r3.Vec{Z: 1}), [][2]float64{
		{lo.X - under, -1}, {hi.X + under, -1}, {hi.X - run, d}, {lo.X + run, d},
	}, hi.Y-lo.Y+10, name+" xz")
	yz := [][2]float64{{lo.Y - under, -1}, {hi.Y + under, -1}, {hi.Y - run, d}, {lo.Y + run, d}}
	if open[0] {
		yz[0], yz[3] = [2]float64{lo.Y, -1}, [2]float64{lo.Y, d}
	}
	if open[1] {
		yz[1], yz[2] = [2]float64{hi.Y, -1}, [2]float64{hi.Y, d}
	}
	alongX := b.extrudePolygon(frame(r3.Vec{X: lo.X - 5}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), yz, hi.X-lo.X+10, name+" yz")
	if b.err != nil {
		return nil
	}
	out, err := decad.Intersect(b.ctx, alongY, alongX)
	b.fail("intersect "+name, err)
	return out
}

// rectAround returns the rectangle r (least and greatest corner) grown by m, counter-clockwise from its least corner.
func rectAround(r [2]r3.Vec, m float64) [][2]float64 {
	lo, hi := r[0], r[1]
	return [][2]float64{{lo.X - m, lo.Y - m}, {hi.X + m, lo.Y - m}, {hi.X + m, hi.Y + m}, {lo.X - m, hi.Y + m}}
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
