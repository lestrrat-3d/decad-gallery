package screwgear

import (
	"math"
	"sort"

	"github.com/lestrrat-3d/r3"
)

// sleeve is the frame that holds the two gears: one thick-walled tube about the pair
// frame's Z axis (the common perpendicular n̂), inner radius Ri and outer radius Ro,
// standing CageRise either side of the middle plane, with four twisted bores cut through
// its wall and up to two windows between them. Everything here is in the pair frame.
type sleeve struct {
	p          Params
	gears      [2]gear
	ri, ro, zb float64
	sIn, sOut  float64             // each bore's cut, as a distance from the middle on its own axis
	sections   [4][]channelStation // each bore's channel in the tube, as wallGap walks it
	windows    [2]sleeveWindow     // one per facing; room() says whether it is cut
}

// newPlainSleeve is the tube and its four bores, with no windows.
func newPlainSleeve(pf pairFrame) sleeve {
	p := pf.p
	sIn, sOut := p.SleeveCut()
	return sleeve{
		p:     p,
		gears: pf.gears,
		ri:    p.SleeveInner(),
		ro:    p.SleeveOuter(),
		zb:    p.CageRise,
		sIn:   sIn,
		sOut:  sOut,
	}
}

// newSleeve builds the sleeve for a pair, including the windows. Each window is placed
// from the tube's radii and height and from the bores' channels, so it follows whatever
// inputs the sleeve is built from.
func newSleeve(pf pairFrame) sleeve {
	f := newPlainSleeve(pf)
	for _, b := range f.bores() {
		f.sections[b.index] = f.channelSections(b)
	}
	for i, facing := range windowFacings(f.p) {
		f.windows[i] = f.newWindow(facing)
	}
	return f
}

// windowFacings are the two level directions the windows are cut toward: across the two
// wider gaps between neighbouring bores. Neighbouring bores are Sigma apart across +ê and
// -ê, and 180 - Sigma apart across +k̂ and -k̂, so the windows face +k̂ and -k̂ up to a
// right-angled crossing and +ê and -ê past it.
func windowFacings(p Params) [2]r3.Vec {
	if p.CrossAngle <= math.Pi/2 {
		return [2]r3.Vec{r3.NewVec(0, 1, 0), r3.NewVec(0, -1, 0)}
	}
	return [2]r3.Vec{r3.NewVec(1, 0, 0), r3.NewVec(-1, 0, 0)}
}

// bore is one of the four channels: a gear and which side of the middle its cut is on.
type bore struct {
	g     gear
	sign  float64 // +1 for the +R bore, -1 for the -R bore
	name  string
	index int // its place in bores()
}

// bores returns gear A's -R and +R bores, then gear B's.
func (f sleeve) bores() [4]bore {
	var out [4]bore
	for gi, g := range f.gears {
		for si, sign := range []float64{-1, 1} {
			name := [2]string{"gear A", "gear B"}[gi] + [2]string{" -R", " +R"}[si]
			out[gi*2+si] = bore{g: g, sign: sign, name: name, index: gi*2 + si}
		}
	}
	return out
}

// span is the stretch of the gear's axis the bore's cut covers.
func (b bore) span(f sleeve) (float64, float64) {
	if b.sign > 0 {
		return f.sIn, f.sOut
	}
	return -f.sOut, -f.sIn
}

// crossing is where the bore's axis meets the frame's circle.
func (b bore) crossing() r3.Vec {
	return b.g.origin.Add(b.g.ez.Scale(b.sign * b.g.p.CageRadius))
}

// roofSide is which long face of a gear's bore is its roof when the sleeve stands on its
// -n̂ end: +1 when the +v face is the upper one at the bore's crossing station, -1 when
// the -v face is. v's component along n̂ is -sin(theta)*(û·n̂) + cos(theta)*(v̂·n̂), and
// v̂·n̂ is zero for these gears, but the full product is kept.
func roofSide(g gear, sign float64) float64 {
	th := g.angle(sign * g.p.CageRadius)
	if -math.Sin(th)*g.ex.Z+math.Cos(th)*g.ey.Z > 0 {
		return 1
	}
	return -1
}

// boreOpening is one bore's opening in its gear's own section, before the twist: u from
// -hw to hw and v from vLo to vHi. sign is -1 for the -R bore and +1 for the +R bore.
// Every bore is the crest rectangle plus the clearance all round, with the roof allowance on
// the long face that is its roof (roofSide).
func boreOpening(g gear, sign float64) (float64, float64, float64) {
	p := g.p
	hw, ht := p.BoreHalfWidth(), p.BoreHalfThickness()
	if roofSide(g, sign) > 0 {
		return hw, -ht, ht + p.RoofAllowance
	}
	return hw, -ht - p.RoofAllowance, ht
}

// openingCorners is boreOpening's four corners, counter-clockwise:
// (-hw, vLo), (hw, vLo), (hw, vHi), (-hw, vHi).
func openingCorners(g gear, sign float64) [4][2]float64 {
	hw, vLo, vHi := boreOpening(g, sign)
	return [4][2]float64{{-hw, vLo}, {hw, vLo}, {hw, vHi}, {-hw, vHi}}
}

// sectionReach is how far a bore's opening at station s, turned to its angle there, reaches
// from the gear's axis along ex, which is n̂, and along ey, sideways across the frame's
// plane, each either way. The extremes are corners.
func sectionReach(g gear, s float64) (float64, float64, float64, float64) {
	c, sn := math.Cos(g.angle(s)), math.Sin(g.angle(s))
	alongLo, sideLo := math.Inf(1), math.Inf(1)
	alongHi, sideHi := math.Inf(-1), math.Inf(-1)
	for _, q := range openingCorners(g, math.Copysign(1, s)) {
		x, y := q[0]*c-q[1]*sn, q[0]*sn+q[1]*c
		alongLo, alongHi = math.Min(alongLo, x), math.Max(alongHi, x)
		sideLo, sideHi = math.Min(sideLo, y), math.Max(sideHi, y)
	}
	return alongLo, alongHi, sideLo, sideHi
}

// inWall answers whether any of the bore's section at station s lies in the tube's radii.
// A point of the section is at least |s| from the frame's axis, since the axis runs radially
// in projection and the section is square to it, and at most hypot(s, sideways reach) from it.
func (f sleeve) inWall(g gear, s float64) bool {
	_, _, lo, hi := sectionReach(g, s)
	return math.Abs(s) <= f.ro && math.Hypot(s, math.Max(-lo, hi)) >= f.ri
}

// channelTop is the furthest any channel reaches from the middle plane inside the wall, up
// or down, and the station it reaches it at. The roof allowance makes the sleeve differ
// either way up, so both ways are taken.
func (f sleeve) channelTop() (float64, float64) {
	top, at := 0.0, 0.0
	for _, g := range f.gears {
		for _, sign := range []float64{-1, 1} {
			for s := sign * f.sIn; math.Abs(s) <= f.sOut; s += sign * 0.01 {
				if !f.inWall(g, s) {
					continue
				}
				lo, hi, _, _ := sectionReach(g, s)
				for _, x := range []float64{lo, hi} {
					if z := math.Abs(g.origin.Z + x*g.ex.Z); z > top {
						top, at = z, s
					}
				}
			}
		}
	}
	return top, at
}

// flat is a convex polygon of at most eight corners, counter-clockwise, which is all a
// rectangle cut by four lines can come to.
type flat struct {
	n int
	v [8][2]float64
}

// clip keeps the part of a convex polygon where a*x + b*y <= c.
func (q flat) clip(a, b, c float64) flat {
	var out flat
	for i := range q.n {
		p, r := q.v[i], q.v[(i+1)%q.n]
		fp, fr := c-(a*p[0]+b*p[1]), c-(a*r[0]+b*r[1])
		if fp >= 0 {
			out.v[out.n] = p
			out.n++
		}
		if (fp >= 0) != (fr >= 0) {
			k := fp / (fp - fr)
			out.v[out.n] = [2]float64{p[0] + k*(r[0]-p[0]), p[1] + k*(r[1]-p[1])}
			out.n++
		}
	}
	return out
}

// gap2 is the squared distance from (x, y) to the polygon, zero inside.
func (q flat) gap2(x, y float64) float64 {
	inside := q.n >= 3
	best := math.Inf(1)
	for i := range q.n {
		p, r := q.v[i], q.v[(i+1)%q.n]
		ex, ey := r[0]-p[0], r[1]-p[1]
		if ex*(y-p[1])-ey*(x-p[0]) < 0 {
			inside = false
		}
		k := 0.0
		if l2 := ex*ex + ey*ey; l2 > 0 {
			k = math.Max(0, math.Min(1, ((x-p[0])*ex+(y-p[1])*ey)/l2))
		}
		dx, dy := x-p[0]-k*ex, y-p[1]-k*ey
		best = math.Min(best, dx*dx+dy*dy)
	}
	if inside {
		return 0
	}
	return best
}

// sectionInWall is a bore's channel at station s cut down to the part that lies in the tube,
// in the gear's own section coordinates: x along ex and y along ey. The section plane is
// square to the gear's axis, which is level and meets the frame's axis, so a point at (x, y)
// in it stands at radius hypot(s, y) and at height origin.Z + x*ex.Z. The part in the tube
// is the turned rectangle clipped to the heights between the end faces and to the strips of
// y that lie between the tube's radii, one on each side of the gear's axis.
func (f sleeve) sectionInWall(g gear, s float64) [2]flat {
	var out [2]flat
	if math.Abs(s) >= f.ro {
		return out
	}
	c, sn := math.Cos(g.angle(s)), math.Sin(g.angle(s))
	rect := flat{n: 4}
	for i, uv := range openingCorners(g, math.Copysign(1, s)) {
		rect.v[i] = [2]float64{uv[0]*c - uv[1]*sn, uv[0]*sn + uv[1]*c}
	}
	xa, xb := (-f.zb-g.origin.Z)/g.ex.Z, (f.zb-g.origin.Z)/g.ex.Z
	if xa > xb {
		xa, xb = xb, xa
	}
	rect = rect.clip(1, 0, xb).clip(-1, 0, -xa)
	near := math.Sqrt(math.Max(0, f.ri*f.ri-s*s))
	far := math.Sqrt(f.ro*f.ro - s*s)
	for i, side := range []float64{-1, 1} {
		out[i] = rect.clip(0, side, far).clip(0, -side, -near)
	}
	return out
}

// channelStation is one station of a bore's channel in the tube: its pieces, and a circle
// round each that holds all of it.
type channelStation struct {
	s         float64
	pieces    [2]flat
	cx, cy, r [2]float64
}

// channelSections is a bore's channel in the tube at every station wallGap walks, in order:
// every 2 microns from the cut's first station to its last, and every tenth of a micron
// between two of those where a piece appears or vanishes.
func (f sleeve) channelSections(b bore) []channelStation {
	const step, fine = 0.002, 0.0001
	lo, hi := b.span(f)
	station := func(s float64) channelStation {
		c := channelStation{s: s, pieces: f.sectionInWall(b.g, s)}
		for i, piece := range c.pieces {
			if piece.n == 0 {
				continue
			}
			for k := range piece.n {
				c.cx[i] += piece.v[k][0] / float64(piece.n)
				c.cy[i] += piece.v[k][1] / float64(piece.n)
			}
			for k := range piece.n {
				c.r[i] = math.Max(c.r[i], math.Hypot(piece.v[k][0]-c.cx[i], piece.v[k][1]-c.cy[i]))
			}
		}
		return c
	}
	has := func(c channelStation) [2]bool { return [2]bool{c.pieces[0].n > 0, c.pieces[1].n > 0} }
	var out []channelStation
	for k := 0; ; k++ {
		s := lo + float64(k)*step
		if s > hi {
			break
		}
		c := station(s)
		if k > 0 && has(c) != has(out[len(out)-1]) {
			prev := out[len(out)-1].s
			for j := 1; prev+float64(j)*fine < s; j++ {
				out = append(out, station(prev+float64(j)*fine))
			}
		}
		out = append(out, c)
	}
	return out
}

// wallGap is the distance from q to the part of a bore's channel that lies in the tube, or
// reach when that part is at least reach away. The channel is the union of its sections, so
// the distance is the least, over stations, of the distance along the axis to the station's
// plane combined with the distance within that plane to the section. Every point of a section
// lies within BoreCorner of the gear's axis, which rules most points out before any station
// is walked.
//
// The stations are those of channelSections. Wherever a piece appears or vanishes between two
// stations, the stretch between them is sampled again at a tenth of a micron, because the
// least distance is often where the channel's corner first reaches into the wall.
//
// The walk starts at the station nearest q's own and goes outward both ways. A station further
// along the axis than the least distance found so far cannot come nearer, and neither can any
// beyond it, so each way stops there; a piece whose bounding circle is further off than that
// is passed over. The least distance is the same one a walk of every station would find.
func (f sleeve) wallGap(b bore, q r3.Vec, reach float64) float64 {
	g := b.g
	lo, hi := b.span(f)
	d := q.Sub(g.origin)
	x, y, sq := d.Dot(g.ex), d.Dot(g.ey), d.Dot(g.ez)
	axis := math.Hypot(math.Hypot(x, y), sq-math.Max(lo, math.Min(hi, sq)))
	if axis-g.p.BoreCorner() >= reach {
		return reach
	}
	best := reach * reach
	st := f.sections[b.index]
	// visit takes one station into account and says whether the walk goes on past it:
	// whether a station that far along the axis could still come nearer than best.
	visit := func(c *channelStation) bool {
		ds := (sq - c.s) * (sq - c.s)
		if ds >= best {
			return false
		}
		for i := range c.pieces {
			if c.pieces[i].n == 0 {
				continue
			}
			if o := math.Hypot(x-c.cx[i], y-c.cy[i]) - c.r[i]; o > 0 && ds+o*o >= best {
				continue
			}
			best = math.Min(best, ds+c.pieces[i].gap2(x, y))
		}
		return true
	}
	at := sort.Search(len(st), func(i int) bool { return st[i].s >= sq })
	for i := at; i < len(st) && visit(&st[i]); i++ {
	}
	for i := at - 1; i >= 0 && visit(&st[i]); i-- {
	}
	return math.Sqrt(best)
}

// wallCorners calls fn with every corner of every section of a bore's channel in the tube,
// in the pair frame, at stations a millimeter's thousandth apart. Any linear measure of
// the channel's part in the tube is extreme at one of them.
func (f sleeve) wallCorners(b bore, fn func(r3.Vec)) {
	g := b.g
	lo, hi := b.span(f)
	for s := lo; s <= hi; s += 0.001 {
		for _, piece := range f.sectionInWall(g, s) {
			for i := range piece.n {
				fn(g.origin.Add(g.ex.Scale(piece.v[i][0])).Add(g.ey.Scale(piece.v[i][1])).Add(g.ez.Scale(s)))
			}
		}
	}
}

// channelOutline samples a bore's channel where it runs through the wall: the rectangle's
// four sides at seventeen points each, at stations 0.1 mm apart from the cut's inner end
// outward, kept where they stand within half a millimeter of the tube's radii.
func (f sleeve) channelOutline(b bore) []r3.Vec {
	hw, vLo, vHi := boreOpening(b.g, b.sign)
	var out []r3.Vec
	for s := b.sign * f.sIn; math.Abs(s) <= f.sOut; s += b.sign * 0.1 {
		for i := range 17 {
			k := float64(i) / 16
			v := vLo + (vHi-vLo)*k
			for _, q := range [4][2]float64{
				{hw, v}, {-hw, v}, {-hw + 2*hw*k, vHi}, {-hw + 2*hw*k, vLo},
			} {
				pt := b.g.world(q[0], q[1], s)
				if r := math.Hypot(pt.X, pt.Y); r < f.ri-0.5 || r > f.ro+0.5 {
					continue
				}
				out = append(out, pt)
			}
		}
	}
	return out
}

// channelSeparation is the build's check on the wall between neighbouring bores, and the
// gap it is least across. Round the tube the bores alternate between the gears, so the four
// gaps between neighbours face +k̂ (gear A's +R bore and gear B's -R), -ê (the two -R bores),
// -k̂ (gear A's -R and gear B's +R) and +ê (the two +R bores). For each gap both bores'
// outlines, as channelOutline samples them, are projected onto the plane through the frame's
// axis square to the direction the gap faces, and the convex hull of each projection is
// taken. The separation is the most, over the directions square to the two hulls' edges, by
// which one hull's projection onto that direction clears the other's. A projection brings no
// two points nearer together, so the separation never exceeds the least distance between the
// two outlines.
func channelSeparation(f sleeve) (float64, string) {
	b := f.bores()
	gaps := []struct {
		facing r3.Vec
		x, y   bore
	}{
		{r3.NewVec(0, 1, 0), b[1], b[2]},
		{r3.NewVec(-1, 0, 0), b[2], b[0]},
		{r3.NewVec(0, -1, 0), b[0], b[3]},
		{r3.NewVec(1, 0, 0), b[3], b[1]},
	}
	least, which := math.Inf(1), ""
	for _, gap := range gaps {
		across := r3.NewVec(0, 0, 1).Cross(gap.facing)
		hull := func(bb bore) [][2]float64 {
			var pts [][2]float64
			for _, pt := range f.channelOutline(bb) {
				pts = append(pts, [2]float64{pt.Dot(across), pt.Z})
			}
			return convexHull(pts)
		}
		hx, hy := hull(gap.x), hull(gap.y)
		sep := math.Inf(-1)
		for _, h := range [][][2]float64{hx, hy} {
			for i := range h {
				a, c := h[i], h[(i+1)%len(h)]
				n := [2]float64{c[1] - a[1], a[0] - c[0]}
				l := math.Hypot(n[0], n[1])
				if l == 0 {
					continue
				}
				n[0], n[1] = n[0]/l, n[1]/l
				loX, hiX := projectRange(hx, n)
				loY, hiY := projectRange(hy, n)
				sep = math.Max(sep, math.Max(loY-hiX, loX-hiY))
			}
		}
		if sep < least {
			least, which = sep, "the "+gap.x.name+" and "+gap.y.name+" bores"
		}
	}
	return least, which
}

// projectRange is the interval the points cover along the unit direction n.
func projectRange(pts [][2]float64, n [2]float64) (float64, float64) {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, q := range pts {
		d := q[0]*n[0] + q[1]*n[1]
		lo, hi = math.Min(lo, d), math.Max(hi, d)
	}
	return lo, hi
}

// convexHull is the convex hull of a set of points, counter-clockwise, by Andrew's monotone
// chain. It reorders pts.
func convexHull(pts [][2]float64) [][2]float64 {
	sort.Slice(pts, func(i, j int) bool {
		if pts[i][0] != pts[j][0] {
			return pts[i][0] < pts[j][0]
		}
		return pts[i][1] < pts[j][1]
	})
	cross := func(o, a, b [2]float64) float64 {
		return (a[0]-o[0])*(b[1]-o[1]) - (a[1]-o[1])*(b[0]-o[0])
	}
	var h [][2]float64
	for range 2 {
		start := len(h)
		for _, q := range pts {
			for len(h) >= start+2 && cross(h[len(h)-2], h[len(h)-1], q) <= 0 {
				h = h[:len(h)-1]
			}
			h = append(h, q)
		}
		h = h[:len(h)-1]
		for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
			pts[i], pts[j] = pts[j], pts[i]
		}
	}
	return h
}
