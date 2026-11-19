package swisslever

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

type outline []Pt

func segDist(x, a, b Pt) float64 {
	d := b.Sub(a)
	l2 := d.Dot(d)
	if l2 == 0 {
		return x.Sub(a).Len()
	}
	t := math.Max(0, math.Min(1, x.Sub(a).Dot(d)/l2))
	return x.Sub(a.Add(d.Scale(t))).Len()
}

func segCross(a, b, c, d Pt) bool {
	d1, d2 := b.Sub(a).Cross(c.Sub(a)), b.Sub(a).Cross(d.Sub(a))
	d3, d4 := d.Sub(c).Cross(a.Sub(c)), d.Sub(c).Cross(b.Sub(c))
	return d1*d2 < 0 && d3*d4 < 0
}

// inPoly reports whether x lies inside the closed outline (even-odd).
func inPoly(poly outline, x Pt) bool {
	in := false
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		if (a.Y > x.Y) == (b.Y > x.Y) {
			continue
		}
		if x.X < a.X+(x.Y-a.Y)/(b.Y-a.Y)*(b.X-a.X) {
			in = !in
		}
	}
	return in
}

// overlap reports whether the closed polygons p and q overlap; touching does not count.
func overlap(p, q outline) bool {
	for i := range p {
		a, b := p[i], p[(i+1)%len(p)]
		for j := range q {
			if segCross(a, b, q[j], q[(j+1)%len(q)]) {
				return true
			}
		}
	}
	// Containment without a crossing: test the polygons' mean points, which lie inside a convex-ish outline.
	return inPoly(q, centroid(p)) || inPoly(p, centroid(q))
}

func centroid(p outline) Pt {
	var c Pt
	for _, v := range p {
		c = c.Add(v)
	}
	return c.Scale(1 / float64(len(p)))
}

func polyDist(p, q outline) float64 {
	m := math.Inf(1)
	for _, v := range p {
		for j := range q {
			m = math.Min(m, segDist(v, q[j], q[(j+1)%len(q)]))
		}
	}
	for _, v := range q {
		for j := range p {
			m = math.Min(m, segDist(v, p[j], p[(j+1)%len(p)]))
		}
	}
	return m
}

func turned(p outline, a float64, then Pt) outline {
	out := make(outline, len(p))
	for i, v := range p {
		out[i] = v.Rot(a).Add(then)
	}
	return out
}

// checker simulates the wheel (teeth in the wheel frame) against the two stones (in the lever frame, about the pallet
// arbor at pivot).
type checker struct {
	g      *Geometry
	pivot  Pt
	teeth  []outline
	stones [2]outline // entry, exit
	steps  int
}

type drop struct {
	psi, size float64
	entry     bool // the drop lands on the entry stone
	landing   float64
}

type actionResult struct {
	advance  []float64
	drops    []drop
	maxPhi   float64 // greatest departure from the closed form away from the drops
	others   float64 // E1
	rootGap  float64 // E2
	topGap   float64 // E3
	finalPhi float64
}

// active returns the teeth near the pallets at wheel angle phi, in the model frame.
func (c *checker) active(phi float64) []outline {
	var out []outline
	for _, t := range c.teeth {
		a := centroid(t).Rot(-phi).Angle()
		if math.Abs(normAngle(a-c.g.IL.Angle())) > 40*deg && math.Abs(normAngle(a-c.g.IR.Angle())) > 40*deg {
			continue
		}
		out = append(out, turned(t, -phi, Pt{}))
	}
	return out
}

func (c *checker) world(psi float64) [2]outline {
	return [2]outline{turned(c.stones[0], -psi, c.pivot), turned(c.stones[1], -psi, c.pivot)}
}

func (c *checker) collide(phi float64, st [2]outline) bool {
	for _, t := range c.active(phi) {
		if overlap(t, st[0]) || overlap(t, st[1]) {
			return true
		}
	}
	return false
}

// free returns the smallest turn that frees the overlap, in either direction (positive is forward).
func (c *checker) free(phi float64, st [2]outline) (float64, error) {
	const step = 0.005 * deg
	best := math.Inf(1)
	for _, dir := range []float64{1, -1} {
		lo, hi := 0.0, 0.0
		ok := false
		for k := 1; k <= 1000; k++ {
			if !c.collide(phi+dir*float64(k)*step, st) {
				lo, hi, ok = float64(k-1)*step, float64(k)*step, true
				break
			}
		}
		if !ok {
			continue
		}
		for hi-lo > 1e-9 {
			m := (lo + hi) / 2
			if c.collide(phi+dir*m, st) {
				lo = m
			} else {
				hi = m
			}
		}
		if hi < best {
			best = hi
			if dir < 0 {
				best = -hi
			}
		}
		if math.Abs(best) == hi && dir == 1 {
			continue
		}
	}
	if math.IsInf(best, 1) {
		return 0, fmt.Errorf("the lever jams the wheel")
	}
	return best, nil
}

// run simulates three periods and measures the second and third.
func (c *checker) run() (actionResult, error) {
	g := c.g
	amp := g.Banking
	periods := 3
	res := actionResult{others: math.Inf(1), rootGap: math.Inf(1), topGap: math.Inf(1)}
	psiAt := func(i int) float64 {
		return math.Max(-amp, math.Min(amp, -1.25*amp*math.Cos(2*math.Pi*float64(i)/float64(c.steps))))
	}
	phi := 10 * deg
	st := c.world(psiAt(0))
	for k := 0; k < 100000 && !c.collide(phi+0.02*deg, st); k++ {
		phi += 0.02 * deg
	}
	prev := phi
	phis := map[int]float64{0: phi}
	k := math.NaN()
	for i := 1; i <= c.steps*periods; i++ {
		psi := psiAt(i)
		st = c.world(psi)
		if c.collide(phi, st) {
			d, err := c.free(phi, st)
			if err != nil {
				return res, fmt.Errorf("step %d, psi %v deg: %w", i, psi/deg, err)
			}
			phi += d - 1e-10*math.Copysign(1, d)
		}
		const step = 0.02 * deg
		start := phi
		for !c.collide(phi+step, st) {
			phi += step
			if phi-start > 1.5*g.Pitch {
				return res, fmt.Errorf("the wheel runs through at psi %v deg", psi/deg)
			}
		}
		lo, hi := phi, phi+step
		for hi-lo > 1e-9 {
			m := (lo + hi) / 2
			if c.collide(m, st) {
				hi = m
			} else {
				lo = m
			}
		}
		phi = lo - 1e-10
		dphi := phi - prev
		prev = phi
		if i%c.steps == 0 {
			phis[i/c.steps] = phi
		}
		if i <= c.steps {
			continue
		}
		plus := psi-psiAt(i-1) > 0
		if dphi > 0.5*deg {
			res.drops = append(res.drops, c.landed(phi, st, psi, dphi))
		}
		if math.Abs(psi-psiAt(i-1)) > 1e-9 {
			res.maxPhi = math.Max(res.maxPhi, c.closedForm(phi, psi, plus, &k))
		}
		if i%5 == 0 {
			c.measure(phi, st, &res)
		}
	}
	for p := 2; p <= periods; p++ {
		res.advance = append(res.advance, phis[p]-phis[p-1])
	}
	for _, psi := range []float64{-amp, 0, amp} {
		for si, poly := range c.world(psi) {
			for j := range poly {
				res.rootGap = math.Min(res.rootGap, segDist(Pt{}, poly[j], poly[(j+1)%len(poly)])-RootRadius)
			}
			// The top corners are G and Hs of the analytic section.
			st := g.Entry
			if si == 1 {
				st = g.Exit
			}
			for _, v := range []Pt{st.G, st.Hs} {
				res.topGap = math.Min(res.topGap, c.pivot.Add(v.Rot(-psi)).Len()-g.HeelRadius)
			}
		}
	}
	res.finalPhi = phi
	return res, nil
}

// closedForm returns the departure of the simulated wheel angle from the closed form at lever angle psi, in rad, or
// zero within 0.15 deg of a drop. k holds the whole-tooth offset between them once known.
func (c *checker) closedForm(phi, psi float64, plus bool, k *float64) float64 {
	g := c.g
	if math.Abs(psi-g.PsiEf) < 0.2*deg || math.Abs(psi-g.PsiXf) < 0.2*deg {
		return 0
	}
	want, _, err := g.WheelAt(psi, plus)
	if err != nil {
		return math.Inf(1)
	}
	turns := math.Round((phi - want) / g.Pitch)
	return math.Abs(phi - want - turns*g.Pitch)
}

// landed records a drop: its size at the heel circle and which stone the toe lands on.
func (c *checker) landed(phi float64, st [2]outline, psi, dphi float64) drop {
	d := drop{psi: psi, size: dphi * c.g.HeelRadius}
	best := math.Inf(1)
	for _, t := range c.active(phi) {
		// The toe is the vertex of the tooth nearest the toe circle.
		for _, v := range t {
			if math.Abs(v.Len()-ToeRadius) > 0.05 {
				continue
			}
			for si, poly := range st {
				_ = poly
				lock := c.locking(si, psi)
				dist := segDist(v, lock[0], lock[1])
				if dist < best {
					best, d.entry, d.landing = dist, si == 0, dist
				}
			}
		}
	}
	return d
}

// locking returns the locking face of stone si (C to Hs) in the model frame at lever angle psi, from the analytic
// section, so a real section is held to the design's face.
func (c *checker) locking(si int, psi float64) [2]Pt {
	st := c.g.Entry
	if si == 1 {
		st = c.g.Exit
	}
	return [2]Pt{c.pivot.Add(st.C.Rot(-psi)), c.pivot.Add(st.Hs.Rot(-psi))}
}

// measure updates E1: the least gap of every tooth to both stones, leaving out a pair that touches.
func (c *checker) measure(phi float64, st [2]outline, res *actionResult) {
	for _, t := range c.active(phi) {
		for _, s := range st {
			d := polyDist(t, s)
			if d < 0.01 {
				continue
			}
			res.others = math.Min(res.others, d)
		}
	}
}

// analyticTeeth returns the tooth polygons of the wheel from the Layout points, tooth j turned by j pitches.
func analyticTeeth(g *Geometry) []outline {
	var out []outline
	for j := 0; j < Teeth; j++ {
		a := float64(j) * g.Pitch
		out = append(out, outline{g.ToothL.Rot(a), g.ToothT.Rot(a), g.ToothH.Rot(a), g.ToothK.Rot(a)})
	}
	return out
}

// analyticStones returns the stone sections as polygons in the lever frame, corners in the order C, F, G, Hs.
func analyticStones(g *Geometry) [2]outline {
	e, x := g.Entry.Corners(), g.Exit.Corners()
	return [2]outline{e[:], x[:]}
}

func newChecker(g *Geometry, teeth []outline, stones [2]outline, steps int) *checker {
	return &checker{g: g, pivot: g.P, teeth: teeth, stones: stones, steps: steps}
}

// closedDrops returns the closed-form drops in mm at the heel circle: the entry release on swing + and the exit release
// on swing -.
func closedDrops(t *testing.T, g *Geometry) map[bool]float64 {
	t.Helper()
	out := map[bool]float64{}
	for _, c := range []struct {
		psi  float64
		plus bool
	}{{g.PsiEf, true}, {g.PsiXf, false}} {
		before, _, err := g.WheelAt(c.psi, c.plus)
		require.NoError(t, err)
		after, _, err := g.WheelAt(c.psi+math.Copysign(1e-9, c.psi), c.plus)
		require.NoError(t, err)
		out[c.plus] = (after - before) * g.HeelRadius
	}
	return out
}

// requireAction asserts the requirements of "Geometry checks" on a simulation result: one tooth per period, two drops
// per period of at least minDrop mm landing on a locking face, and the least gaps.
func requireAction(t *testing.T, g *Geometry, res actionResult, advanceTol, minDrop, minE1 float64) {
	t.Helper()
	require.Len(t, res.advance, 2)
	for _, a := range res.advance {
		require.InDelta(t, g.Pitch, a, advanceTol, "advance per period")
	}
	require.Len(t, res.drops, 4, "two drops in each measured period")
	for _, d := range res.drops {
		require.GreaterOrEqual(t, d.size, minDrop, "drop at psi %v", d.psi/deg)
	}
	require.GreaterOrEqual(t, res.others, minE1, "E1")
	require.GreaterOrEqual(t, res.rootGap, 2.0, "E2")
}

// printError is one case of the print-error study: the toe and heel radius error dr, the pallet impulse-face length
// error dw and the centre distance error dd, all in mm.
type printError struct{ dr, dw, dd float64 }

// errorCase returns the checker for the analytic outlines with the print errors applied.
func errorCase(g *Geometry, e printError, steps int) *checker {
	teeth := analyticTeeth(g)
	scale := func(v Pt, dr float64) Pt { return v.Scale((v.Len() + dr) / v.Len()) }
	for j := range teeth {
		a := float64(j) * g.Pitch
		teeth[j] = outline{g.ToothL.Rot(a), scale(g.ToothT, e.dr).Rot(a), scale(g.ToothH, e.dr).Rot(a), g.ToothK.Rot(a)}
	}
	var stones [2]outline
	for i, st := range []StoneSection{g.Entry, g.Exit} {
		u := Polar(1, g.UEAngle)
		if i == 1 {
			u = Polar(1, g.UXAngle)
		}
		face := st.F.Sub(st.C)
		face = face.Scale(1 / face.Len())
		f := st.F.Add(face.Scale(e.dw))
		hs := st.Hs
		gg, _ := lineIntersect(f, u.Rot(StoneKey), hs, face)
		stones[i] = outline{st.C, f, gg, hs}
	}
	c := newChecker(g, teeth, stones, steps)
	c.pivot = Pt{g.P.X, g.P.Y + e.dd}
	return c
}
