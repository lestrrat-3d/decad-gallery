package sevensegment

import (
	"fmt"
	"math"
	"slices"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Cam dimensions that are not exported.
const (
	camBoreRadius = shaftCamRadius + PressFit          // D bore: shaft radius 3.2 plus PressFit
	camBoreFlat   = shaftCamFlat + PressFit            // flat 2.7 plus PressFit
	camHubTop     = StationPitch - CamThickness - 0.05 // hub end, local z: 0.05 short of the next cam's back face
	camRoundR     = 0.5                                // hub root round
	camChamfer    = 0.5                                // rim chamfer
	camMarkRadius = 0.75                               // look-alike hole radius (1.5 across)
	camMarkStep   = 25.0                               // degrees between marks
	camMarkFirst  = 270.0
	channelCap    = CamRadius + 1.0 // entry channel cap line, 1.0 outside the rim so the channel opens through it
)

// camDiscFace is the Y of a cam's groove face: the station plus the hub reach.
func (d Dimensions) camDiscFace(s Segment) float64 { return d.Station(s) + camFaceOffset }

// camFaceOffset is how far a cam's groove face stands from its slider's centre plane: half the stem and a Loose gap.
const camFaceOffset = StemThickness/2 + LooseFit

// camFrame is the cam's local frame in the world: origin on the groove face and the shaft axis, U = +X, V = +Z, so the
// cam's local +Z runs along world -Y.
func (d Dimensions) camFrame(s Segment) r3.Frame {
	f, _ := r3.NewFrame(r3.Vec{X: 0, Y: d.camDiscFace(s), Z: ShaftZ}, r3.Vec{X: 1}, r3.Vec{Z: 1})
	return f
}

// camMarks returns the number of look-alike holes and their circle radius: front cams d, e, c take 1, 2, 3 at radius
// 18; rear cams g, b, f, a take 1 to 4 at radius 12.
func camMarks(s Segment) (int, float64) {
	switch s {
	case SegD:
		return 1, 18.0
	case SegE:
		return 2, 18.0
	case SegC:
		return 3, 18.0
	case SegG:
		return 1, 12.0
	case SegB:
		return 2, 12.0
	case SegF:
		return 3, 12.0
	default:
		return 4, 12.0
	}
}

// cam builds segment s's cam in its local frame (axis on local Z, groove face at local z = 0), then places it in the
// world at theta = 0.
func (r *run) cam(s Segment) (*decad.Body, error) {
	pts := [][2]float64{{0, -CamThickness}, {CamRadius - camChamfer, -CamThickness},
		{CamRadius, -CamThickness + camChamfer}, {CamRadius, -camChamfer}, {CamRadius - camChamfer, 0},
		{HubRadius + camRoundR, 0}}
	pts = append(pts, quarterRound(HubRadius+camRoundR, camRoundR, camRoundR, -90, -180, 8)[1:]...)
	pts = append(pts, [2]float64{HubRadius, camHubTop}, [2]float64{0, camHubTop})
	body, err := r.meridian(pts)
	if err != nil {
		return nil, fmt.Errorf("revolve: %w", err)
	}
	if body, err = r.cutCamBore(body); err != nil {
		return nil, fmt.Errorf("cut D bore: %w", err)
	}
	if body, err = r.cutGroove(body, s); err != nil {
		return nil, fmt.Errorf("cut groove: %w", err)
	}
	if body, err = r.cutCamMarks(body, s); err != nil {
		return nil, fmt.Errorf("cut marks: %w", err)
	}
	if body, err = r.inscribe(body, s); err != nil {
		return nil, fmt.Errorf("inscription: %w", err)
	}
	f := r.dm.camFrame(s)
	return placement.Place(r.ctx, body, &f)
}

// cutCamBore cuts the D bore through the whole cam: profile on local z = -4.8, cut 12.95 along +z.
func (r *run) cutCamBore(body *decad.Body) (*decad.Body, error) {
	s, err := r.sketchOn(across(-CamThickness - 1))
	if err != nil {
		return nil, err
	}
	doubleDBoreLoop(s, camBoreRadius, camBoreFlat)
	return r.cutExtrude(body, s, 0, 1, CamThickness+1+camHubTop+1, decad.Along)
}

// cutCamMarks cuts segment seg's look-alike holes through the whole cam: one circle each in one sketch on local z =
// -4.8, each centre placed by its distances from the origin, cut 12.95 along +z in one call.
func (r *run) cutCamMarks(body *decad.Body, seg Segment) (*decad.Body, error) {
	n, radius := camMarks(seg)
	s, err := r.sketchOn(across(-CamThickness - 1))
	if err != nil {
		return nil, err
	}
	for i := range n {
		a := (camMarkFirst + camMarkStep*float64(i)) * math.Pi / 180
		c := sketchcheck.DimensionPoint(s, radius*math.Cos(a), radius*math.Sin(a))
		s.AddConstraint(sketch.NewRadius(s.CreateCircle(c, camMarkRadius), camMarkRadius))
	}
	return r.cutExtrude(body, s, 0, n, CamThickness+camHubTop+1+1, decad.Along)
}

// grooveWall is one wall's points of one ramp.
type grooveWall struct{ j1, t1, t2, j2 *sketch.Point }

// cutGroove cuts the groove from body: the groove profile on local z = +1.0 cut 2.9 against its normal, to z -1.9.
// The profile is the outer wall (dwell arcs, ramp arcs and lines, entry channel) with the inner wall as its hole.
func (r *run) cutGroove(body *decad.Body, seg Segment) (*decad.Body, error) {
	s, err := r.sketchOn(across(1.0))
	if err != nil {
		return nil, err
	}
	o := s.Origin()
	ps := newPolar(s)
	ramps := slices.Clone(r.dm.Ramps[seg])
	slices.SortFunc(ramps, func(a, b Ramp) int {
		switch x, y := wrap360(a.Start), wrap360(b.Start); {
		case x < y:
			return -1
		case x > y:
			return 1
		}
		return 0
	})
	half := GrooveWidth / 2
	type centres struct{ c1, c2 *sketch.Point }
	type rampGeo struct {
		g      RampGeometry
		rayA   *sketch.Line
		rayB   *sketch.Line
		d1, d2 float64 // centre distances from the origin
		before float64 // dwell radius before the ramp
		after  float64
		c      centres
	}
	rg := make([]rampGeo, len(ramps))
	for i, ra := range ramps {
		g := ra.Geometry()
		x := rampGeo{g: g, rayA: ps.axis(ra.Start), rayB: ps.axis(ra.End)}
		x.d1, x.d2, x.before, x.after = OffRadius+RampRound, OnRadius-RampRound, OffRadius, OnRadius
		if !ra.Rising {
			x.d1, x.d2, x.before, x.after = OnRadius-RampRound, OffRadius+RampRound, OnRadius, OffRadius
		}
		x.c.c1, x.c.c2 = s.CreatePoint(g.C1.X, g.C1.Y), s.CreatePoint(g.C2.X, g.C2.Y)
		s.AddConstraint(sketch.NewPointOnLine(x.c.c1, x.rayA), sketch.NewDistance(o, x.c.c1, x.d1),
			sketch.NewPointOnLine(x.c.c2, x.rayB), sketch.NewDistance(o, x.c.c2, x.d2))
		rg[i] = x
	}
	entry := ps.axis(EntryAngle)
	channelHalf := math.Asin(half/(OnRadius+half)) * 180 / math.Pi
	walls := [2][]grooveWall{}
	for w, sigma := range [2]float64{-1, 1} { // inner wall, outer wall
		walls[w] = make([]grooveWall, len(ramps))
		for i, ra := range ramps {
			x := rg[i]
			outward := func(c, t r3.Vec) r3.Vec {
				n := t.Sub(c).Scale(1 / RampRound)
				if n.Dot(t) < 0 {
					n = n.Scale(-1)
				}
				return n
			}
			n1, n2 := outward(x.g.C1, x.g.T1), outward(x.g.C2, x.g.T2)
			wp := func(v r3.Vec) *sketch.Point { return s.CreatePoint(v.X, v.Y) }
			wall := grooveWall{
				j1: wp(polar(x.before+sigma*half, ra.Start)), t1: wp(x.g.T1.Add(n1.Scale(sigma * half))),
				t2: wp(x.g.T2.Add(n2.Scale(sigma * half))), j2: wp(polar(x.after+sigma*half, ra.End)),
			}
			walls[w][i] = wall
			a1 := arcBetween(s, x.g.C1, x.c.c1, polar(x.before+sigma*half, ra.Start), x.g.T1.Add(n1.Scale(sigma*half)),
				wall.j1, wall.t1)
			line := s.CreateLine(wall.t1, wall.t2)
			a2 := arcBetween(s, x.g.C2, x.c.c2, x.g.T2.Add(n2.Scale(sigma*half)), polar(x.after+sigma*half, ra.End),
				wall.t2, wall.j2)
			s.AddConstraint(sketch.NewPointOnLine(wall.j1, x.rayA), sketch.NewPointOnLine(wall.j2, x.rayB),
				sketch.NewTangent(line, a1), sketch.NewTangent(line, a2))
		}
	}
	// Dwell arcs about the origin, from each ramp's end to the next ramp's start.
	for w, sigma := range [2]float64{-1, 1} {
		for i, ra := range ramps {
			nx := (i + 1) % len(ramps)
			rd := rg[i].after + sigma*half
			from, to := walls[w][i].j2, walls[w][nx].j1
			if w == 1 && channelInside(ra.End, ramps[nx].Start, channelHalf) {
				if err := r.channel(s, entry, from, to, rd); err != nil {
					return nil, err
				}
				continue
			}
			s.AddConstraint(sketch.NewRadius(s.CreateArc(o, from, to), rd))
		}
	}
	return r.cutExtrude(body, s, 1, 1, 1.0+GrooveDepth, decad.Against)
}

// channelInside reports whether the entry channel, centred at EntryAngle, lies inside the dwell from angle a to b.
func channelInside(a, b, half float64) bool {
	lo := wrap360(EntryAngle-half-a) + a
	hi := lo + 2*half
	end := a + wrap360(b-a)
	return hi <= end
}

// channel builds the outer wall's dwell arc from -> to, split by the entry channel: an arc to the channel's near side,
// the side line out to the cap line, the cap line, the far side line back and the arc on to `to`.
func (r *run) channel(s *sketch.Sketch, entry *sketch.Line, from, to *sketch.Point, rd float64) error {
	o := s.Origin()
	if math.Abs(rd-(OnRadius+GrooveWidth/2)) > 1e-9 {
		return fmt.Errorf("entry channel dwell at radius %g, want the ON wall", rd)
	}
	u := polar(1, EntryAngle)
	n := r3.Vec{X: -u.Y, Y: u.X}
	hw := GrooveWidth / 2
	along := math.Sqrt(rd*rd - hw*hw)
	at := func(l, lat float64) *sketch.Point {
		v := u.Scale(l).Add(n.Scale(lat))
		return s.CreatePoint(v.X, v.Y)
	}
	pl, pr := at(along, -hw), at(along, hw)
	cl, cr := at(channelCap, -hw), at(channelCap, hw)
	s.AddConstraint(sketch.NewRadius(s.CreateArc(o, from, pl), rd))
	sideL := s.CreateLine(pl, cl)
	cap := s.CreateLine(cl, cr)
	sideR := s.CreateLine(cr, pr)
	s.AddConstraint(sketch.NewRadius(s.CreateArc(o, pr, to), rd),
		sketch.NewParallel(entry, sideL), sketch.NewDistancePointLine(o, sideL, hw),
		sketch.NewParallel(entry, sideR), sketch.NewDistancePointLine(o, sideR, hw),
		sketch.NewPerpendicular(entry, cap), sketch.NewDistancePointLine(o, cap, channelCap))
	return nil
}

// Cam inscription: the segment's letter, engraved into the back face, the face opposite the groove, so a person can tell
// the seven cams apart. The letters are drawn in the seven-segment style from rectangular strokes (A b C d E F G, the
// hexadecimal letters of a seven-segment display), each outline a closed run of horizontal and vertical lines held by
// its distance from the cam's axis, so the sketch is fully constrained without fixing a point.
const (
	inscriptionStroke = 0.9  // stroke width, mm: above the 0.8 mm printable minimum
	inscriptionWidth  = 4.8  // letter width, mm
	inscriptionHeight = 6.9  // letter height, mm
	inscriptionDepth  = 0.5  // engraving depth, mm
	inscriptionRadius = 21.0 // the letter's centre from the cam axis, straight up (local +Y) from it
)

// letterSegments names the display segments each cam letter lights, a to g as in the truth table.
var letterSegments = map[Segment]string{
	SegA: "abcefg", // A
	SegB: "cdefg",  // b
	SegC: "adef",   // C
	SegD: "bcdeg",  // d
	SegE: "adefg",  // E
	SegF: "aefg",   // F
	SegG: "acdef",  // G
}

// letterLoops returns the closed rectilinear loops of segment s's letter, in sketch coordinates about the letter's
// centre (u across, v up): the outer outline and any counter. The letter is the union of its strokes on a lattice of
// x in {0, stroke, width-stroke, width} and y in {0, stroke, mid-stroke/2, mid+stroke/2, height-stroke, height}.
func letterLoops(s Segment) [][][2]float64 {
	w, h, k := inscriptionWidth, inscriptionHeight, inscriptionStroke
	xs := [4]float64{0, k, w - k, w}
	ys := [6]float64{0, k, h/2 - k/2, h/2 + k/2, h - k, h}
	// A cell (i, j) of the 3 by 5 lattice is filled when a stroke covers it.
	cover := map[byte][4]int{ // segment: first and last cell column, first and last cell row (inclusive)
		'a': {0, 2, 4, 4}, 'g': {0, 2, 2, 2}, 'd': {0, 2, 0, 0},
		'f': {0, 0, 2, 4}, 'e': {0, 0, 0, 2}, 'b': {2, 2, 2, 4}, 'c': {2, 2, 0, 2},
	}
	var filled [3][5]bool
	for _, c := range []byte(letterSegments[s]) {
		r := cover[c]
		for i := r[0]; i <= r[1]; i++ {
			for j := r[2]; j <= r[3]; j++ {
				filled[i][j] = true
			}
		}
	}
	in := func(i, j int) bool { return i >= 0 && i < 3 && j >= 0 && j < 5 && filled[i][j] }
	// Directed boundary edges with the material on the left, between lattice points (i, j) with i in 0..3, j in 0..5.
	type pt struct{ i, j int }
	next := map[pt]pt{}
	for i := range 3 {
		for j := range 5 {
			if !filled[i][j] {
				continue
			}
			if !in(i, j-1) {
				next[pt{i, j}] = pt{i + 1, j} // bottom edge, going right
			}
			if !in(i+1, j) {
				next[pt{i + 1, j}] = pt{i + 1, j + 1} // right edge, going up
			}
			if !in(i, j+1) {
				next[pt{i + 1, j + 1}] = pt{i, j + 1} // top edge, going left
			}
			if !in(i-1, j) {
				next[pt{i, j + 1}] = pt{i, j} // left edge, going down
			}
		}
	}
	var loops [][][2]float64
	for start := range next {
		if _, ok := next[start]; !ok {
			continue
		}
		var path []pt
		for p := start; ; {
			q, ok := next[p]
			if !ok {
				break
			}
			path = append(path, p)
			delete(next, p)
			p = q
			if p == start {
				break
			}
		}
		// Keep the corners only: drop a point whose neighbours lie on one line with it.
		var corners [][2]float64
		for n, p := range path {
			a, b := path[(n+len(path)-1)%len(path)], path[(n+1)%len(path)]
			if (a.i == p.i && b.i == p.i) || (a.j == p.j && b.j == p.j) {
				continue
			}
			corners = append(corners, [2]float64{xs[p.i] - w/2, ys[p.j] - h/2})
		}
		loops = append(loops, corners)
	}
	return loops
}

// inscribe engraves segment s's letter into the back face (local z = -CamThickness) of the cam body. The sketch frame
// looks at the face from outside, so the letter reads correctly from the back: U runs along local -X.
func (r *run) inscribe(body *decad.Body, s Segment) (*decad.Body, error) {
	frame, err := r3.NewFrame(r3.Vec{Z: -CamThickness}, r3.Vec{X: -1}, r3.Vec{Y: 1})
	if err != nil {
		return nil, err
	}
	sk, err := r.sketchOn(frame)
	if err != nil {
		return nil, err
	}
	o := sk.Origin()
	loops := letterLoops(s)
	for _, loop := range loops {
		pts := make([]*sketch.Point, len(loop))
		for i, c := range loop {
			pts[i] = sk.CreatePoint(c[0], c[1]+inscriptionRadius)
		}
		for i := range pts {
			a, b := loop[i], loop[(i+1)%len(loop)]
			line := sk.CreateLine(pts[i], pts[(i+1)%len(pts)])
			if a[1] == b[1] { // horizontal: held at its distance from the axis
				sk.AddConstraint(sketch.NewHorizontal(line), sketch.NewDistancePointLine(o, line, math.Abs(a[1]+inscriptionRadius)))
			} else {
				sk.AddConstraint(sketch.NewVertical(line), sketch.NewDistancePointLine(o, line, math.Abs(a[0])))
			}
		}
	}
	if err := sketchcheck.VerifyFullyConstrained(r.ctx, sk); err != nil {
		return nil, fmt.Errorf("verify lettering sketch: %w", err)
	}
	// The letter is the profile with the counter as its hole, or the one profile of a letter without a counter.
	var profiles []*sketch.Profile
	for _, pr := range sk.Profiles() {
		if pr.Valid && len(pr.Holes) > 0 {
			profiles = append(profiles, pr)
		}
	}
	if len(profiles) == 0 {
		for _, pr := range sk.Profiles() {
			if pr.Valid {
				profiles = append(profiles, pr)
			}
		}
	}
	if len(profiles) != 1 {
		return nil, fmt.Errorf("lettering has %d profiles, want 1", len(profiles))
	}
	out, err := decad.Emboss(r.ctx, body, sk, profiles, decad.Faces(decad.Facing(r3.Vec{Z: -1})), decad.EmbossEngraved,
		units.Millimeters(inscriptionDepth))
	if err != nil {
		return nil, fmt.Errorf("engrave: %w", err)
	}
	if !out.IsSolid() {
		return nil, fmt.Errorf("engraved cam is not a solid")
	}
	return out, nil
}
