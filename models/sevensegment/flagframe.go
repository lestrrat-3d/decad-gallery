package sevensegment

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Plates of the flag mechanism, in mm. The cradle plate holds the rotors' journals from below and the top plate from
// above; the guide plate gains two columns that carry pegs registering the cradle.
const (
	flagFaceZ    = topFrom + topThick // the display face
	cradleFrom   = 120.0
	cradleThick  = 4.0 // 20 layers: the seam plane carries the rotor axes; whole layers
	topFrom      = cradleFrom + cradleThick + Seam
	topThick     = 8.0                // 40 layers; whole layers
	tubeFrom     = RotorAxisZ - 7.8   // the tubes hang from the top plate to the sliders' heads
	bladeHoleTop = flagFaceZ - 0.8    // the tube roof: 0.8 of top plate over the blade hole
	columnTop    = cradleFrom - 1.275 // the columns end this far under the cradle
	pegRadius    = 2.25
	pegHole      = pegRadius + PressFit // registration hole radius
	boreRadius   = 8.094                // 7.644 rotor swept circle + 0.45 Regular-fit play
	boreRound    = 1.2
)

// pegSites are the registration pegs: the first two stand on the guide plate's columns and the last two on the top plate.
var (
	columnPegs = [2][2]float64{{18.0, -67.5}, {-20.325, 58.5}}
	topPegs    = [2][2]float64{{-18.0, -67.5}, {21.75, 58.5}}
	// columnHalf are the columns' half widths (x, y), in the order of columnPegs. The rear column is 4.6 wide so it keeps
	// LooseFit from the rear top flange rib (x -16.4) and rotor a's pyramid never reaches it.
	columnHalf = [2][2]float64{{3.75, 3.75}, {ribInner - 20.325 - LooseFit, 3.75}}
)

// plan maps (u, s) about a segment's centre to world XY.
func plan(seg Segment, u, sv float64) (float64, float64) {
	cx, cy, horizontal := segmentCentre(seg)
	if horizontal {
		return cx + u, cy + sv
	}
	return cx - sv, cy + u
}

// rrect draws a rectangle whose edges are held horizontal or vertical at their distance from the sketch origin, with its
// corners rounded rad when rad is positive.
func rrect(s *sketch.Sketch, x0, x1, y0, y1, rad float64) error {
	o := s.Origin()
	v := [][2]float64{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}
	p := make([]*sketch.Point, 4)
	for i, q := range v {
		p[i] = s.CreatePoint(q[0], q[1])
	}
	l := make([]*sketch.Line, 4)
	for i := range p {
		l[i] = s.CreateLine(p[i], p[(i+1)%4])
		if i%2 == 0 {
			s.AddConstraint(sketch.NewHorizontal(l[i]), sketch.NewDistancePointLine(o, l[i], math.Abs(v[i][1])))
		} else {
			s.AddConstraint(sketch.NewVertical(l[i]), sketch.NewDistancePointLine(o, l[i], math.Abs(v[i][0])))
		}
	}
	if rad > 0 {
		for i := range p {
			if _, err := sketch.CreateFillet(s, l[(i+3)%4], l[i], p[i], rad); err != nil {
				return err
			}
		}
	}
	return nil
}

// circleAt adds a circle of radius rad centred at (x, y), the centre placed from the origin.
func circleAt(s *sketch.Sketch, x, y, rad float64) {
	c := sketchcheck.DimensionPoint(s, x, y)
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(c, rad), rad))
}

// flagSites returns the four T bolts of the flag mechanism: heads on the display face, M2 x 10, clamping the top plate and the
// cradle to the top flange ribs.
func flagSites() [4]m2fastener.Site {
	var out [4]m2fastener.Site
	for i, p := range [][2]float64{{-frameBoltX, -66.75}, {frameBoltX, -66.75}, {-frameBoltX, 57.0}, {frameBoltX, 57.0}} {
		out[i] = m2fastener.Site{Head: r3.Vec{X: p[0], Y: p[1], Z: flagFaceZ}, Dir: r3.Vec{Z: -1}, Corner: r3.Vec{Y: 1},
			Length: 16, Cavity: flagFaceZ - flangeFrom - m2fastener.NutCavityHeight}
	}
	return out
}

// boreCone is the constant of the bore's cone r + sqrt2*u = boreCone: the line stands 0.40 (the Regular fit) from the rotor's
// pyramid edge r + sqrt2*u = sqrt2*(RotorHalfWidth + rotorSquareHalf). The edge's constant is 8.244 + sqrt2*15.531 and the fit
// measured along r is boreConeFit, of which the line keeps half beyond the edge's constant.
const (
	boreConeFit = 0.4 * 1.7320508075688772 // the fit 0.40 measured along r, sqrt3 times the distance normal to the cone
	boreCone    = 8.244 + math.Sqrt2*15.531 - 0.5*boreConeFit
	// boreConeMove is how far the cone's line lies beyond the 5.496 + sqrt2*10.354 that the end wall's cone end was set from.
	boreConeMove = 2.748 + math.Sqrt2*5.177 - 0.5*boreConeFit
)

// boreTool builds segment seg's rotor bore: a revolve about the rotor axis at height axisZ. The meridian (r, u) is symmetric in
// u: the cylinder RB = 5.546, a round of 0.8 into the cone that follows the rotor's pyramid and the end wall at u 12.5.
// The pivot block's pocket (pivotPocketTool) opens through the end wall.
func (r *run) boreTool(seg Segment, axisZ float64) (*decad.Body, error) {
	rc := boreRadius - boreRound
	uc := (boreCone - (boreRadius - boreRound) - math.Sqrt(3)*boreRound) / math.Sqrt2
	half := quarterRound(rc, uc, boreRound, 0, math.Atan(math.Sqrt2)*180/math.Pi, 8)
	// The cone ends on the end wall at r 2.461 plus the cone's move, less sqrt2 per mm of the wall's offset from u 12.5.
	half = append(half, [2]float64{2.461 + boreConeMove - math.Sqrt2*(pivotBlockFrom-12.5), pivotBlockFrom})
	pts := [][2]float64{{0, -pivotBlockFrom}}
	for i := len(half) - 1; i >= 0; i-- {
		pts = append(pts, [2]float64{half[i][0], -half[i][1]})
	}
	pts = append(pts, half...)
	pts = append(pts, [2]float64{0, pivotBlockFrom})
	b, err := r.meridian(pts)
	if err != nil {
		return nil, fmt.Errorf("bore %s: %w", seg, err)
	}
	cx, cy, horizontal := segmentCentre(seg)
	var rot r3.Transform
	if horizontal {
		rot, err = r3.RotationAround(r3.Vec{}, r3.Vec{Y: 1}, units.Degrees(90))
	} else {
		rot, err = r3.RotationAround(r3.Vec{}, r3.Vec{X: 1}, units.Degrees(-90))
	}
	if err != nil {
		return nil, err
	}
	if b, err = decad.Transform(r.ctx, b, rot); err != nil {
		return nil, fmt.Errorf("bore %s: turn: %w", seg, err)
	}
	tr, err := r3.Translation(r3.Vec{X: cx, Y: cy, Z: axisZ})
	if err != nil {
		return nil, err
	}
	return decad.Transform(r.ctx, b, tr)
}

// rootedBox is a box x [x0, x1], y [y0, y1] from zb up to the face zFace, with a built-in 45 degree root of leg c where it meets
// the face (overlap ov), made as the Intersect of two prisms. A fillet of the root loop is refused by decad.
func (r *run) rootedBox(x0, x1, y0, y1, zb, zFace, ov, c float64) (*decad.Body, error) {
	fx := local(r3.Vec{Y: y0 - c - 1}, r3.Vec{X: 1}, r3.Vec{Z: 1}) // normal -Y
	px := [][2]float64{{x0, zb}, {x1, zb}, {x1, zFace - c}, {x1 + c + ov, zFace + ov}, {x0 - c - ov, zFace + ov}, {x0, zFace - c}}
	sx, err := r.polyBody(fx, px, decad.Distance{D: units.Millimeters(y1 - y0 + 2*c + 2), Dir: decad.Against})
	if err != nil {
		return nil, fmt.Errorf("rooted box x: %w", err)
	}
	fy := local(r3.Vec{X: x0 - c - 1}, r3.Vec{Y: 1}, r3.Vec{Z: 1}) // normal +X
	py := [][2]float64{{y0, zb}, {y1, zb}, {y1, zFace - c}, {y1 + c + ov, zFace + ov}, {y0 - c - ov, zFace + ov}, {y0, zFace - c}}
	sy, err := r.polyBody(fy, py, decad.Distance{D: units.Millimeters(x1 - x0 + 2*c + 2), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("rooted box y: %w", err)
	}
	b, err := decad.Intersect(r.ctx, sx, sy)
	if err != nil {
		return nil, fmt.Errorf("rooted box: %w", err)
	}
	return b, nil
}

// rootedPost is the upward twin of rootedBox: a post standing on the face zFace up to ztop with a built-in 45 degree root.
func (r *run) rootedPost(x0, x1, y0, y1, zFace, ztop, ov, c float64) (*decad.Body, error) {
	fx := local(r3.Vec{Y: y0 - c - 1}, r3.Vec{X: 1}, r3.Vec{Z: 1})
	px := [][2]float64{{x0 - c - ov, zFace - ov}, {x1 + c + ov, zFace - ov}, {x1, zFace + c}, {x1, ztop}, {x0, ztop}, {x0, zFace + c}}
	sx, err := r.polyBody(fx, px, decad.Distance{D: units.Millimeters(y1 - y0 + 2*c + 2), Dir: decad.Against})
	if err != nil {
		return nil, fmt.Errorf("rooted post x: %w", err)
	}
	fy := local(r3.Vec{X: x0 - c - 1}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	py := [][2]float64{{y0 - c - ov, zFace - ov}, {y1 + c + ov, zFace - ov}, {y1, zFace + c}, {y1, ztop}, {y0, ztop}, {y0, zFace + c}}
	sy, err := r.polyBody(fy, py, decad.Distance{D: units.Millimeters(x1 - x0 + 2*c + 2), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("rooted post y: %w", err)
	}
	b, err := decad.Intersect(r.ctx, sx, sy)
	if err != nil {
		return nil, fmt.Errorf("rooted post: %w", err)
	}
	return b, nil
}

// mouthCone cuts a 45 degree mouth into the face z over a through hole of radius rh at (x, y), depth d.
func (r *run) mouthCone(x, y, z, rh, d float64) (*decad.Body, error) {
	s, err := r.sketchOn(xyFrame(z + 0.1))
	if err != nil {
		return nil, err
	}
	rad := rh + d + 0.1
	circleAt(s, x, y, rad)
	pr, err := r.profile(s, 0)
	if err != nil {
		return nil, err
	}
	return decad.Extrude(r.ctx, r.doc, s, pr, decad.Distance{D: units.Millimeters(d + 0.2), Dir: decad.Against},
		decad.WithTaper(units.Degrees(45)))
}

// cradlePassage is the plan footprint of a slider and its tube plus LooseFit: x0, x1, y0, y1.
func cradlePassage(seg Segment, yi float64) [4]float64 {
	switch seg {
	case SegA, SegG, SegD:
		return [4]float64{-(6.825 + LooseFit), 6.825 + LooseFit, yi - (13.275 + LooseFit), yi + 5.775 + LooseFit}
	case SegB, SegC:
		return [4]float64{-(5.25 + LooseFit), 26.85 + LooseFit, yi - (4.275 + LooseFit), yi + 6.075 + LooseFit}
	}
	return [4]float64{-(29.25 + LooseFit), 5.25 + LooseFit, yi - (4.275 + LooseFit), yi + 6.075 + LooseFit}
}

// cradlePlate builds the cradle plate: z 80.0 to 82.6 with the four T holes, four registration holes, seven passages, seven
// rotor bores cut at the axis, two mouth cones for the top plate's pegs, and the outer-loop chamfers. The bottom face's
// inner loops stay sharp until decad can chamfer them (item D1 of instructions.md).
func (r *run) cradlePlate() (*decad.Body, error) {
	s, err := r.sketchOn(xyFrame(cradleFrom))
	if err != nil {
		return nil, err
	}
	rectOn(s, -WallHalfWidth, WallHalfWidth, plateFrom, plateTo)
	for _, st := range flagSites() {
		circleAt(s, st.Head.X, st.Head.Y, m2fastener.HoleRadius)
	}
	for _, p := range [][2]float64{columnPegs[0], columnPegs[1], topPegs[0], topPegs[1]} {
		circleAt(s, p[0], p[1], pegHole)
	}
	body, err := r.extrude(s, 8, cradleThick, decad.Along)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	for _, seg := range Segments() {
		q := cradlePassage(seg, r.dm.Station(seg))
		ps, err := r.sketchOn(xyFrame(cradleFrom - 1))
		if err != nil {
			return nil, err
		}
		if err := rrect(ps, q[0], q[1], q[2], q[3], 0.5); err != nil {
			return nil, fmt.Errorf("passage %s: %w", seg, err)
		}
		tool, err := r.extrude(ps, 0, cradleThick+2.4, decad.Along)
		if err != nil {
			return nil, fmt.Errorf("passage tool %s: %w", seg, err)
		}
		if body, err = decad.Cut(r.ctx, body, tool); err != nil {
			return nil, fmt.Errorf("cut passage %s: %w", seg, err)
		}
	}
	for _, seg := range Segments() {
		b, err := r.boreTool(seg, RotorAxisZ)
		if err != nil {
			return nil, err
		}
		if body, err = decad.Cut(r.ctx, body, b); err != nil {
			return nil, fmt.Errorf("cut bore %s: %w", seg, err)
		}
		if body, err = r.cutPivotPockets(body, seg); err != nil {
			return nil, err
		}
	}
	for _, p := range topPegs {
		cone, err := r.mouthCone(p[0], p[1], cradleFrom+cradleThick, pegHole, 0.8)
		if err != nil {
			return nil, fmt.Errorf("mouth cone: %w", err)
		}
		if body, err = decad.Cut(r.ctx, body, cone); err != nil {
			return nil, fmt.Errorf("cut mouth cone: %w", err)
		}
	}
	if body, _, err = edgefinish.Chamfer(r.ctx, body, r3.Vec{Z: 1}, []float64{cradleFrom, cradleFrom + cradleThick},
		edgefinish.Long(edgefinish.Outer), wallChamfer); err != nil {
		return nil, fmt.Errorf("chamfer outer loops: %w", err)
	}
	return body, nil
}

// The blade keeps bladeAcross from the walls of its hole across its thin side (this sets the pin line's side play) and
// bladeAlong along its length; the tube keeps tubeWall around the hole. The hole's corners are rounded bladeHoleRound, the
// least allowed on an inside corner, so the square blade corners keep 0.21 from them (the Tight minimum is 0.2; see
// instructions.md "Blade corners").
const (
	bladeAcross    = 0.25
	bladeAlong     = 0.35
	bladeHoleRound = 0.50
	tubeWall       = 0.8
)

// tubeFootprint is a segment's tube plan, x0, x1, y0, y1: it hangs from the top plate and guides the slider's blade. The
// wall on the rotor side runs into the rotor's bore and is cut there.
func tubeFootprint(seg Segment, yi float64) [4]float64 {
	h := bladeHole(seg, yi)
	_, cy, _ := segmentCentre(seg)
	switch seg {
	case SegA, SegG, SegD:
		return [4]float64{h[0] - tubeWall, h[1] + tubeWall, h[2] - tubeWall, cy - 7.35}
	case SegB, SegC:
		return [4]float64{h[0] - tubeWall, 17.325, h[2] - tubeWall, h[3] + tubeWall}
	}
	return [4]float64{-17.325, h[1] + tubeWall, h[2] - tubeWall, h[3] + tubeWall}
}

// bladeHole is a segment's blade hole through the tube and the plate: the blade plus bladeAcross across it and bladeAlong
// along it.
func bladeHole(seg Segment, yi float64) [4]float64 {
	_, cy, _ := segmentCentre(seg)
	across, along := bladeAcross, bladeAlong
	switch seg {
	case SegA, SegG, SegD:
		return [4]float64{-5.25 - along, 5.25 + along, cy - 11.7 - across, cy - 10.05 + across}
	case SegB, SegC:
		return [4]float64{13.05 - across, 14.7 + across, yi - StemThickness/2 - along, yi + 4.5 + along}
	}
	return [4]float64{-14.7 - across, -13.05 + across, yi - StemThickness/2 - along, yi + 4.5 + along}
}

// flagTopPlate builds the top plate: 5.4 thick from z 82.85 to the display face, with seven tubes hanging from its underside,
// the rotor bores, the windows and T holes, the registration pegs, the blade holes, the outer chamfers and the head
// pockets.
func (r *run) flagTopPlate() (*decad.Body, error) {
	s0, err := r.sketchOn(xyFrame(topFrom))
	if err != nil {
		return nil, err
	}
	rectOn(s0, -WallHalfWidth, WallHalfWidth, plateFrom, plateTo)
	body, err := r.extrude(s0, 0, topThick, decad.Along)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	for _, seg := range Segments() {
		ts := tubeFootprint(seg, r.dm.Station(seg))
		tube, err := r.rootedBox(ts[0], ts[1], ts[2], ts[3], tubeFrom, topFrom, 0.1, 0.5)
		if err != nil {
			return nil, fmt.Errorf("tube %s: %w", seg, err)
		}
		if body, err = decad.Union(r.ctx, body, tube); err != nil {
			return nil, fmt.Errorf("union tube %s: %w", seg, err)
		}
	}
	for _, seg := range Segments() {
		b, err := r.boreTool(seg, topFrom)
		if err != nil {
			return nil, err
		}
		if body, err = decad.Cut(r.ctx, body, b); err != nil {
			return nil, fmt.Errorf("cut bore %s: %w", seg, err)
		}
		if body, err = r.cutPivotPockets(body, seg); err != nil {
			return nil, err
		}
	}
	if body, err = r.flagWindows(body); err != nil {
		return nil, err
	}
	for _, p := range topPegs {
		peg, err := r.meridian([][2]float64{{0, -2.4}, {pegRadius - 0.3, -2.4}, {pegRadius, -2.4 + 0.3},
			{pegRadius, -0.75}, {pegRadius + 0.5, 0}, {pegRadius + 0.5, 0.1}, {0, 0.1}})
		if err != nil {
			return nil, fmt.Errorf("peg: %w", err)
		}
		tr, err := r3.Translation(r3.Vec{X: p[0], Y: p[1], Z: topFrom})
		if err != nil {
			return nil, err
		}
		if peg, err = decad.Transform(r.ctx, peg, tr); err != nil {
			return nil, err
		}
		if body, err = decad.Union(r.ctx, body, peg); err != nil {
			return nil, fmt.Errorf("union peg: %w", err)
		}
	}
	for _, seg := range Segments() {
		h := bladeHole(seg, r.dm.Station(seg))
		hs, err := r.sketchOn(xyFrame(tubeFrom - 0.4))
		if err != nil {
			return nil, err
		}
		if err := rrect(hs, h[0], h[1], h[2], h[3], bladeHoleRound); err != nil {
			return nil, fmt.Errorf("blade hole sketch %s: %w", seg, err)
		}
		hole, err := r.extrude(hs, 0, bladeHoleTop-(tubeFrom-0.4), decad.Along)
		if err != nil {
			return nil, fmt.Errorf("blade hole %s: %w", seg, err)
		}
		if body, err = decad.Cut(r.ctx, body, hole); err != nil {
			return nil, fmt.Errorf("cut blade hole %s: %w", seg, err)
		}
	}
	if body, _, err = edgefinish.Chamfer(r.ctx, body, r3.Vec{Z: 1}, []float64{topFrom, flagFaceZ},
		edgefinish.Long(edgefinish.Outer), wallChamfer); err != nil {
		return nil, fmt.Errorf("chamfer outer loops: %w", err)
	}
	for i, site := range flagSites() {
		tool, sk, err := m2fastener.PocketTool(r.ctx, r.doc, site)
		if err != nil {
			return nil, fmt.Errorf("head pocket %d: %w", i+1, err)
		}
		r.sketches = append(r.sketches, sk)
		if body, err = decad.Cut(r.ctx, body, tool); err != nil {
			return nil, fmt.Errorf("cut head pocket %d: %w", i+1, err)
		}
	}
	return body, nil
}

// The window octagon in (u, s): |s| <= windowHalfWidth, |s| + |u| <= windowChamfer and |u| <= windowEnd. The long sides
// stand 0.8 inside the rotor's faces and the chamfers 1.1 inside its pyramid edges, so no yellow shows when a rotor is
// unlit (instructions.md "Visibility").
const (
	windowHalfWidth = RotorHalfWidth - 0.8                   // 0.8 inside the rotor's faces
	windowChamfer   = RotorHalfWidth + rotorSquareHalf - 1.1 // 1.1 inside its pyramid edges
	windowEnd       = 17.544
)

// windowFrom is the plane the window and T-hole sketch lies on, 0.85 under the top plate's underside.
const windowFrom = topFrom - 0.85

// flagWindows cuts the seven windows and the four T holes: one sketch holds every loop (a frame rectangle, the T circles and
// the window octagons), and each loop without a hole is extruded and cut on its own.
func (r *run) flagWindows(body *decad.Body) (*decad.Body, error) {
	s, err := r.sketchOn(xyFrame(windowFrom))
	if err != nil {
		return nil, err
	}
	rectOn(s, -45.0, 45.0, -78.0, 69.0)
	for _, st := range flagSites() {
		circleAt(s, st.Head.X, st.Head.Y, m2fastener.HoleRadius)
	}
	for _, seg := range Segments() {
		u0, ue, se := windowChamfer-windowHalfWidth, windowEnd, windowChamfer-windowEnd
		w := [][2]float64{{u0, windowHalfWidth}, {ue, se}, {ue, -se}, {u0, -windowHalfWidth}, {-u0, -windowHalfWidth},
			{-ue, -se}, {-ue, se}, {-u0, windowHalfWidth}}
		world := make([][2]float64, len(w))
		for i, q := range w {
			world[i][0], world[i][1] = plan(seg, q[0], q[1])
		}
		polygon(s, world)
	}
	if _, err := s.Solve(r.ctx); err != nil {
		return nil, fmt.Errorf("solve windows: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(r.ctx, s); err != nil {
		return nil, fmt.Errorf("verify windows sketch: %w", err)
	}
	n := 0
	for _, pr := range s.Profiles() {
		if !pr.Valid || len(pr.Holes) != 0 {
			continue
		}
		tool, err := decad.Extrude(r.ctx, r.doc, s, pr, decad.Distance{D: units.Millimeters(flagFaceZ + 0.75 - windowFrom), Dir: decad.Along})
		if err != nil {
			return nil, fmt.Errorf("window tool: %w", err)
		}
		if body, err = decad.Cut(r.ctx, body, tool); err != nil {
			return nil, fmt.Errorf("cut window or bolt hole %d: %w", n, err)
		}
		n++
	}
	if n != 11 {
		return nil, fmt.Errorf("cut %d window and bolt hole tools, want 11", n)
	}
	return body, nil
}

// columnGuidePlate builds today's guide plate with two columns and their pegs: each column is 5 x 5, the rear one 4.6 x 5,
// from z 58 to 79.15 with a 45 degree root, and carries a peg that registers the cradle.
func (r *run) columnGuidePlate() (*decad.Body, error) {
	body, err := r.guidePlate()
	if err != nil {
		return nil, err
	}
	for i, p := range columnPegs {
		h := columnHalf[i]
		col, err := r.rootedPost(p[0]-h[0], p[0]+h[0], p[1]-h[1], p[1]+h[1], guideTop, columnTop, 0.1, 0.5)
		if err != nil {
			return nil, fmt.Errorf("column: %w", err)
		}
		if body, err = decad.Union(r.ctx, body, col); err != nil {
			return nil, fmt.Errorf("union column: %w", err)
		}
		peg, err := r.meridian([][2]float64{{0, -0.1}, {pegRadius + 0.5, -0.1}, {pegRadius + 0.5, 0}, {pegRadius, 0.5},
			{pegRadius, 3.9 - 0.3}, {pegRadius - 0.3, 3.9}, {0, 3.9}})
		if err != nil {
			return nil, fmt.Errorf("column peg: %w", err)
		}
		tr, err := r3.Translation(r3.Vec{X: p[0], Y: p[1], Z: columnTop})
		if err != nil {
			return nil, err
		}
		if peg, err = decad.Transform(r.ctx, peg, tr); err != nil {
			return nil, err
		}
		if body, err = decad.Union(r.ctx, body, peg); err != nil {
			return nil, fmt.Errorf("union column peg: %w", err)
		}
	}
	return body, nil
}
