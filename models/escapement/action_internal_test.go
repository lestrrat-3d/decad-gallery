package escapement

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// This file holds the escapement checker: a quasi-static simulation of the wheel and anchor outlines as closed
// polylines. The anchor swings by ψ = -A cos 2πt; at each step the wheel advances as far as the outlines allow
// without overlapping, and backs off if the anchor has moved into it.

func normAng(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}

func cross(a, b Pt) float64 { return a.X*b.Y - a.Y*b.X }

// toothFootAt returns the foot of the tooth face at angle from the inward radial, on the circle of radius r.
func toothFootAt(angle, r float64) Pt {
	dir := Pt{-math.Sin(angle), -math.Cos(angle)}
	b := TipRadius * dir.Y
	s := -b - math.Sqrt(b*b-(TipRadius*TipRadius-r*r))
	return Pt{s * dir.X, TipRadius + s*dir.Y}
}

// arcPoints samples the arc about the origin of radius r from angle a0 to a1 (either direction) including both ends,
// with a chord sagitta of at most tol.
func arcPoints(r, a0, a1, tol float64) []Pt {
	step := 2 * math.Acos(1-tol/r)
	n := int(math.Ceil(math.Abs(a1-a0) / step))
	if n < 1 {
		n = 1
	}
	out := make([]Pt, 0, n+1)
	for i := 0; i <= n; i++ {
		out = append(out, Polar(r, a0+(a1-a0)*float64(i)/float64(n)))
	}
	return out
}

// analyticAnchor samples the anchor outline of instructions.md, relative to the pallet arbor.
func analyticAnchor(g *Geometry, tol float64) []Pt {
	var out []Pt
	add := func(pts ...Pt) { out = append(out, pts...) }
	arc := func(r float64, from, to Pt) {
		a0, a1 := from.Angle(), to.Angle()
		// Keep the sweep's direction: the arcs of the loop run the short way except the hub arcs.
		pts := arcPoints(r, a0, a1, tol)
		add(pts[:len(pts)-1]...)
	}
	// The 14 pieces of "Anchor outline".
	add(g.CE)
	add(g.FE)
	arc(g.RIn, g.FE, g.GE) // cw
	add(g.GE, g.HE)
	arc(HubRadius, g.HE, g.HX) // ccw through -90°
	add(g.HX, g.GX)
	arc(g.RIn, g.GX, g.CX) // cw
	add(g.CX, g.FX)
	arc(g.ROut, g.FX, g.KX) // ccw
	add(g.KX, g.MX, g.NX)
	// Hub arc ccw over the top from 13.05° to 166.94°.
	pts := arcPoints(HubRadius, g.NX.Angle(), g.NE.Angle(), tol)
	add(pts[:len(pts)-1]...)
	add(g.NE, g.ME, g.KE)
	arc(g.ROut, g.KE, g.CE) // ccw
	return out
}

// analyticToothPolys returns the tooth polygons of the wheel, tooth j with its tip at 90° + jτ.
func analyticToothPolys() [][]Pt {
	l, b := toothFootAt(Rake, RootRadius), toothFootAt(BackAngle, RootRadius)
	var out [][]Pt
	for j := 0; j < Teeth; j++ {
		a := float64(j) * 2 * math.Pi / Teeth
		out = append(out, []Pt{l.Rot(a), Pt{0, TipRadius}.Rot(a), b.Rot(a)})
	}
	return out
}

// toothPolys splits a wheel's outer loop (in the wheel frame, about the origin) into one closed polygon per tooth: the
// runs of vertices that lie beyond the root circle.
func toothPolys(loop []Pt) ([][]Pt, error) {
	const above = RootRadius + 0.05
	n := len(loop)
	start := -1
	for i, p := range loop {
		if p.Len() <= above {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("the wheel outline has no root vertex")
	}
	// A tooth is a run of vertices beyond the root circle plus the root vertex on each side of it.
	var out [][]Pt
	var run []Pt
	for k := 1; k <= n; k++ {
		p := loop[(start+k)%n]
		if p.Len() > above {
			if len(run) == 0 {
				run = append(run, loop[(start+k-1)%n])
			}
			run = append(run, p)
			continue
		}
		if len(run) > 0 {
			out = append(out, append(run, p))
		}
		run = nil
	}
	if len(run) > 0 {
		out = append(out, append(run, loop[start]))
	}
	return out, nil
}

// actionInput is what the checker needs: the tooth polygons in the wheel frame (tooth 0's tip on +Y, wheel centre on
// the origin) and the anchor outline relative to its arbor at ψ = 0.
type actionInput struct {
	teeth  [][]Pt
	anchor []Pt
}

// segment distance helpers.
func distPointSeg(x, a, b Pt) float64 {
	d := b.Sub(a)
	l2 := d.X*d.X + d.Y*d.Y
	if l2 == 0 {
		return x.Sub(a).Len()
	}
	t := math.Max(0, math.Min(1, (x.Sub(a).X*d.X+x.Sub(a).Y*d.Y)/l2))
	return x.Sub(Pt{a.X + d.X*t, a.Y + d.Y*t}).Len()
}

func segCross(a, b, c, d Pt) bool {
	d1 := cross(b.Sub(a), c.Sub(a))
	d2 := cross(b.Sub(a), d.Sub(a))
	d3 := cross(d.Sub(c), a.Sub(c))
	d4 := cross(d.Sub(c), b.Sub(c))
	return d1*d2 < 0 && d3*d4 < 0
}

// inPoly reports whether x lies strictly inside the closed polygon poly (even-odd).
func inPoly(poly []Pt, x Pt) bool {
	in := false
	n := len(poly)
	for i := 0; i < n; i++ {
		a, b := poly[i], poly[(i+1)%n]
		if (a.Y > x.Y) == (b.Y > x.Y) {
			continue
		}
		if x.X < a.X+(x.Y-a.Y)/(b.Y-a.Y)*(b.X-a.X) {
			in = !in
		}
	}
	return in
}

// overlap reports whether the closed polygons p and q overlap (touching does not count). Segments of q far from p's
// bounding box are skipped, so a small p against a long q is cheap.
func overlap(p, q []Pt) bool {
	lo, hi := p[0], p[0]
	for _, v := range p {
		lo = Pt{math.Min(lo.X, v.X), math.Min(lo.Y, v.Y)}
		hi = Pt{math.Max(hi.X, v.X), math.Max(hi.Y, v.Y)}
	}
	near := func(a, b Pt) bool {
		return math.Max(a.X, b.X) >= lo.X && math.Min(a.X, b.X) <= hi.X &&
			math.Max(a.Y, b.Y) >= lo.Y && math.Min(a.Y, b.Y) <= hi.Y
	}
	for j := range q {
		c, d := q[j], q[(j+1)%len(q)]
		if !near(c, d) {
			continue
		}
		for i := range p {
			if segCross(p[i], p[(i+1)%len(p)], c, d) {
				return true
			}
		}
	}
	for _, v := range p {
		if inPoly(q, v) {
			return true
		}
	}
	for _, v := range q {
		if v.X >= lo.X && v.X <= hi.X && v.Y >= lo.Y && v.Y <= hi.Y && inPoly(p, v) {
			return true
		}
	}
	return false
}

type checker struct {
	g      *Geometry
	in     actionInput
	near   []Pt // anchor vertices near the wheel at some swing, for pruning (anchor frame)
	psiMax float64
}

// worldAnchor returns the anchor outline at anchor angle psi in the model frame.
func (c *checker) worldAnchor(psi float64) []Pt {
	out := make([]Pt, len(c.in.anchor))
	for i, v := range c.in.anchor {
		out[i] = c.g.P.Add(v.Rot(-psi))
	}
	return out
}

// activeTeeth returns the tooth polygons at wheel angle phi that lie near the pallets, in the model frame.
func (c *checker) activeTeeth(phi float64) [][]Pt {
	var out [][]Pt
	for _, t := range c.in.teeth {
		var cen Pt
		for _, v := range t {
			cen = cen.Add(v)
		}
		a := cen.Rot(-phi).Angle()
		if math.Abs(normAng(a-3*math.Pi/4)) > 35*deg && math.Abs(normAng(a-math.Pi/4)) > 35*deg {
			continue
		}
		w := make([]Pt, len(t))
		for i, v := range t {
			w[i] = v.Rot(-phi)
		}
		out = append(out, w)
	}
	return out
}

// pruneAnchor keeps the anchor polygon's vertices as a whole; overlap tests use the full polygon, which is small.
func (c *checker) collide(phi float64, anchor []Pt) bool {
	for _, t := range c.activeTeeth(phi) {
		if overlap(t, anchor) {
			return true
		}
	}
	return false
}

func tipOf(t []Pt) Pt {
	best := t[0]
	for _, v := range t {
		if v.Len() > best.Len() {
			best = v
		}
	}
	return best
}

func distPolys(p, q []Pt) float64 {
	m := math.Inf(1)
	for _, v := range p {
		for j := range q {
			m = math.Min(m, distPointSeg(v, q[j], q[(j+1)%len(q)]))
		}
	}
	for _, v := range q {
		for j := range p {
			m = math.Min(m, distPointSeg(v, p[j], p[(j+1)%len(p)]))
		}
	}
	return m
}

type drop struct {
	psi, size float64
	entry     bool
	landR     float64 // distance of the landing tip from the pallet arbor
}

type actionResult struct {
	periodAdvance []float64
	maxRecoil     float64
	drops         []drop
	lock, lift    []float64
	others        float64 // E1: every non-contact tooth, and the contact tooth to the other pallet
	rootGap       float64 // E2
	topGap        float64 // E3
}

// run simulates three periods of 2000 steps and measures the second and third.
func (c *checker) run() (actionResult, error) {
	g := c.g
	amp := g.Params.Amplitude
	const steps, periods = 2000, 3
	res := actionResult{others: math.Inf(1), rootGap: math.Inf(1), topGap: math.Inf(1)}
	psi0 := -amp
	phi := 0.0
	anchor := c.worldAnchor(psi0)
	for k := 0; k < 100000 && c.collide(phi, anchor); k++ {
		phi += 0.01 * deg
	}
	prevPhi := phi
	phis := []float64{phi}
	stationary := true
	var landPsi, impulseStart float64
	impulse := false
	advanceRate := 0.0
	for i := 1; i <= steps*periods; i++ {
		psi := -amp * math.Cos(2*math.Pi*float64(i)/steps)
		anchor = c.worldAnchor(psi)
		if c.collide(phi, anchor) {
			lo, hi := 0.0, 1e-6
			for c.collide(phi-hi, anchor) {
				hi *= 2
				if hi > 0.2 {
					return res, fmt.Errorf("the anchor jams the wheel at ψ %v°", psi/deg)
				}
			}
			for hi-lo > 1e-9 {
				m := (lo + hi) / 2
				if c.collide(phi-m, anchor) {
					lo = m
				} else {
					hi = m
				}
			}
			res.maxRecoil = math.Max(res.maxRecoil, hi)
			phi -= hi
		}
		const step = 0.02 * deg
		start := phi
		for !c.collide(phi+step, anchor) {
			phi += step
			if phi-start > 1.5*g.Pitch {
				return res, fmt.Errorf("the wheel runs through at ψ %v°", psi/deg)
			}
		}
		lo, hi := phi, phi+step
		for hi-lo > 1e-9 {
			m := (lo + hi) / 2
			if c.collide(m, anchor) {
				hi = m
			} else {
				lo = m
			}
		}
		phi = lo - 1e-10
		adv := phi - prevPhi
		switch {
		case adv > 0.15*deg:
			if i > steps {
				var tip Pt
				best := math.Inf(1)
				for _, t := range c.activeTeeth(phi) {
					tp := tipOf(t)
					for j := range anchor {
						if d := distPointSeg(tp, anchor[j], anchor[(j+1)%len(anchor)]); d < best {
							best, tip = d, tp
						}
					}
				}
				res.drops = append(res.drops, drop{psi: psi, size: adv - advanceRate, entry: psi < 0,
					landR: tip.Sub(g.P).Len()})
				if impulse {
					res.lift = append(res.lift, math.Abs(psi-impulseStart))
				}
			}
			impulse = false
			stationary = true
			landPsi = psi
		case adv > 0.005*deg:
			if stationary {
				if i > steps {
					res.lock = append(res.lock, math.Abs(psi-landPsi))
				}
				impulseStart = psi
				impulse = true
				stationary = false
			}
			advanceRate = adv
		default:
			advanceRate = 0
		}
		prevPhi = phi
		if i%steps == 0 {
			phis = append(phis, phi)
		}
		if i > steps && i%5 == 0 {
			c.measure(phi, anchor, &res)
		}
	}
	for k := 1; k < len(phis); k++ {
		if k >= 2 {
			res.periodAdvance = append(res.periodAdvance, phis[k]-phis[k-1])
		}
	}
	// E2 and E3 over the swing's two ends and its middle.
	for _, psi := range []float64{-amp, 0, amp} {
		for _, v := range c.worldAnchor(psi) {
			res.rootGap = math.Min(res.rootGap, v.Len()-RootRadius)
		}
		for _, v := range c.in.anchor {
			if !c.isTop(v) {
				continue
			}
			res.topGap = math.Min(res.topGap, g.P.Add(v.Rot(-psi)).Len()-TipRadius)
		}
	}
	return res, nil
}

// simplify drops the vertices of the closed polygon p that lie within tol of the chord between their neighbours
// (Douglas-Peucker), which keeps the outline within tol and makes the checker much faster.
func simplify(p []Pt, tol float64) []Pt {
	n := len(p)
	if n < 8 {
		return p
	}
	// Split the loop at vertex 0 and the vertex farthest from it, and simplify the two chains.
	far, best := 0, 0.0
	for i, v := range p {
		if d := v.Sub(p[0]).Len(); d > best {
			far, best = i, d
		}
	}
	var chain func(pts []Pt) []Pt
	chain = func(pts []Pt) []Pt {
		if len(pts) < 3 {
			return pts
		}
		idx, worst := 0, 0.0
		for i := 1; i < len(pts)-1; i++ {
			if d := distPointSeg(pts[i], pts[0], pts[len(pts)-1]); d > worst {
				idx, worst = i, d
			}
		}
		if worst <= tol {
			return []Pt{pts[0], pts[len(pts)-1]}
		}
		left := chain(pts[:idx+1])
		right := chain(pts[idx:])
		return append(left[:len(left)-1:len(left)-1], right...)
	}
	a := chain(p[:far+1])
	b := chain(append(append([]Pt{}, p[far:]...), p[0]))
	return append(a[:len(a)-1:len(a)-1], b[:len(b)-1]...)
}

// isTop reports whether anchor-frame vertex v belongs to a pallet top or an arm: within 0.15° of the pallet's top or beyond it (the pallet's
// length past the locking corner is Lpal), outside the hub.
func (c *checker) isTop(v Pt) bool {
	if v.Len() < c.g.RIn-0.01 {
		return false
	}
	if v.X < 0 {
		return normAng(c.g.CE.Angle()-v.Angle()) >= PalletLength-0.15*deg && normAng(c.g.CE.Angle()-v.Angle()) < math.Pi
	}
	return normAng(v.Angle()-c.g.CX.Angle()) >= PalletLength-0.15*deg && normAng(v.Angle()-c.g.CX.Angle()) < math.Pi
}

// measure updates E1 at the current state.
func (c *checker) measure(phi float64, anchor []Pt, res *actionResult) {
	// Only anchor segments near the wheel matter, so prune to them for the distance scan.
	var near [][2]Pt
	n := len(anchor)
	for i := range anchor {
		if anchor[i].Len() < TipRadius+6 || anchor[(i+1)%n].Len() < TipRadius+6 {
			near = append(near, [2]Pt{anchor[i], anchor[(i+1)%n]})
		}
	}
	for _, t := range c.activeTeeth(phi) {
		tp := tipOf(t)
		contact := math.Inf(1)
		for j := range anchor {
			contact = math.Min(contact, distPointSeg(tp, anchor[j], anchor[(j+1)%n]))
		}
		if contact < 0.01 {
			// The contact tooth is the working contact: skip it.
			continue
		}
		for _, v := range t {
			for _, sg := range near {
				res.others = math.Min(res.others, distPointSeg(v, sg[0], sg[1]))
			}
		}
		for _, sg := range near {
			for _, v := range sg {
				for j := range t {
					res.others = math.Min(res.others, distPointSeg(v, t[j], t[(j+1)%len(t)]))
				}
			}
		}
	}
}

// checkAction asserts the requirements of "Geometry checks" on a simulation result.
func checkAction(t *testing.T, g *Geometry, res actionResult, tolOff float64) {
	t.Helper()
	require.Len(t, res.periodAdvance, 2)
	for i, a := range res.periodAdvance {
		require.InDelta(t, g.Pitch, a, 0.05*deg, "period %d advance", i+2)
	}
	require.Len(t, res.drops, 4, "two drops in each of two measured periods")
	for _, d := range res.drops {
		want := g.RIn
		if d.entry {
			want = g.ROut
		}
		require.InDelta(t, want, d.landR, 0.03+tolOff, "landing radius at ψ %v°", d.psi/deg)
		require.GreaterOrEqual(t, d.size*TipRadius, 1.30, "drop at ψ %v°", d.psi/deg)
	}
	lock := (g.Params.LockAngle)
	lift := g.Params.LiftAngle
	require.NotEmpty(t, res.lock)
	for _, a := range res.lock {
		require.InDelta(t, lock, a, 0.1*deg, "lock angle")
	}
	for _, a := range res.lift {
		require.InDelta(t, lift, a, 0.1*deg, "lift angle")
	}
	require.LessOrEqual(t, res.maxRecoil, 2*tolOff/TipRadius+1e-9, "recoil")
	require.GreaterOrEqual(t, res.others, 0.5, "E1")
	require.GreaterOrEqual(t, res.rootGap, 1.0, "E2")
	require.GreaterOrEqual(t, res.topGap, 2.0, "E3")
}

func TestActionAnalytic(t *testing.T) {
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	c := &checker{g: &g, in: actionInput{teeth: analyticToothPolys(), anchor: analyticAnchor(&g, 0.005)}}
	res, err := c.run()
	require.NoError(t, err)
	checkAction(t, &g, res, 0.005)
	// Numerical verification of instructions.md.
	for _, d := range res.drops {
		require.InDelta(t, 1.388, d.size*TipRadius, 0.05)
	}
	require.InDelta(t, 0.613, res.others, 0.02)
	require.InDelta(t, 1.947, res.rootGap, 0.02)
	t.Logf("advance %v drops %v lock %v lift %v others %.3f root %.3f top %.3f recoil %g", res.periodAdvance,
		res.drops, res.lock, res.lift, res.others, res.rootGap, res.topGap, res.maxRecoil)
}

func TestActionAnalyticWide(t *testing.T) {
	p := DefaultParams()
	p.Amplitude = 6 * deg
	g, err := Layout(p)
	require.NoError(t, err)
	c := &checker{g: &g, in: actionInput{teeth: analyticToothPolys(), anchor: analyticAnchor(&g, 0.005)}}
	res, err := c.run()
	require.NoError(t, err)
	checkAction(t, &g, res, 0.005)
	// instructions.md: the pallet tops and arms stay 2.3 mm outside the tips and 1.2 mm clear of the root at A = 6°.
	require.InDelta(t, 2.34, res.topGap, 0.05)
	require.InDelta(t, 1.228, res.rootGap, 0.02)
}
