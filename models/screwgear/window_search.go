package screwgear

import (
	"math"

	"github.com/lestrrat-3d/r3"
)

// A window is a prism: a hexagon drawn on the plane through the frame's axis square to the
// direction it faces, pushed straight out through the wall on that side. On the plane, t
// runs across the window and z up the frame's axis (n̂). The hexagon is the band
// lo <= z + lean*t <= hi, whose sides are 45 degree lines, cut off by two upright ends at
// t = left and t = right, with the two long corners trimmed by 45 degree lines, so every
// edge is upright or at 45 degrees and the printer bridges nothing.
//
// Every dimension comes from the sleeve's own geometry: the sides sit CollarWall beyond the
// reach of the two bores flanking the window, the trims keep the window inside the height the
// channels already reach and its roofs within 45 degrees of the middle on the inner face, and
// each end stands where its corner lines through the wall, and its upright edges on the inner
// and the outer face, come CollarWall, and the sampling slack, from the other two bores. A
// window the bores leave no room for is not cut (room).

const (
	// windowStep is the spacing at which each end's corner lines through the wall are walked.
	windowStep = 0.1
	// windowSlack is the most the true least distance from a face to a channel can fall below
	// the least sampled one: half the diagonal of a sample cell, and 5 microns for the stations
	// the channel is walked at (wallGap).
	windowSlack = windowStep/math.Sqrt2 + 5e-3
)

// sleeveWindow is one window: the hexagon, and the prism it is pushed out as.
type sleeveWindow struct {
	facing, across r3.Vec  // the level direction it is cut toward, and t's direction: n̂ × facing
	lean           float64 // +1 or -1: the band is lo <= z + lean*t <= hi
	lo, hi         float64
	bottom, top    float64 // the trims: bottom <= z - lean*t <= top
	left, right    float64 // the upright ends
	zLimit         float64 // the height the channels reach, which the trims keep it within
	corners        [][2]float64
}

// plane is a point's place on the window's plane, and how far it stands in front of the plane
// through the frame's axis.
func (w sleeveWindow) plane(pt r3.Vec) (float64, float64, float64) {
	return pt.Dot(w.across), pt.Z, pt.Dot(w.facing)
}

// at is the point of the prism at (t, z) on the window's plane, a in front.
func (w sleeveWindow) at(t, z, a float64) r3.Vec {
	return w.across.Scale(t).Add(w.facing.Scale(a)).Add(r3.NewVec(0, 0, z))
}

// chord is the stretch of a's over which the prism's line at t is in the tube: from the inner
// face to the outer.
func (f sleeve) chord(t float64) (float64, float64) {
	return math.Sqrt(math.Max(0, f.ri*f.ri-t*t)), math.Sqrt(math.Max(0, f.ro*f.ro-t*t))
}

// newWindow finds the window facing a level direction from the bores.
func (f sleeve) newWindow(facing r3.Vec) sleeveWindow {
	p := f.p
	w := sleeveWindow{facing: facing, across: r3.NewVec(0, 0, 1).Cross(facing)}

	// The flanking bores are the two whose crossings lie on the window's side.
	var flanks, far []bore
	for _, b := range f.bores() {
		if b.crossing().Dot(facing) > 0 {
			flanks = append(flanks, b)
		} else {
			far = append(far, b)
		}
	}
	if len(flanks) != 2 {
		// The bores sit at +-Sigma/2 round the tube, so each level facing has exactly two.
		panic("a window's side of the tube does not hold two bores")
	}
	low, high := flanks[0], flanks[1]
	if low.crossing().Z > high.crossing().Z {
		low, high = high, low
	}
	w.lean = 1
	if high.crossing().Dot(w.across) < low.crossing().Dot(w.across) {
		w.lean = -1
	}

	// The sides: CollarWall across the band beyond each flanking bore's reach.
	// The band's measure z + lean*t grows by sqrt(2) per mm across it.
	lowReach, highReach := math.Inf(-1), math.Inf(1)
	f.wallCorners(low, func(pt r3.Vec) {
		t, z, _ := w.plane(pt)
		lowReach = math.Max(lowReach, z+w.lean*t)
	})
	f.wallCorners(high, func(pt r3.Vec) {
		t, z, _ := w.plane(pt)
		highReach = math.Min(highReach, z+w.lean*t)
	})
	w.lo = lowReach + math.Sqrt2*p.CollarWall
	w.hi = highReach - math.Sqrt2*p.CollarWall

	// The trims: each long corner no higher than the channels reach, and each side cut off
	// where it would meet the inner face past 45 degrees round.
	w.zLimit, _ = f.channelTop()
	w.top = math.Min(2*w.zLimit-w.hi, w.hi+math.Sqrt2*f.ri)
	w.bottom = math.Max(-2*w.zLimit-w.lo, w.lo-math.Sqrt2*f.ri)

	// The ends: as far out as both corners of the end keep their distance.
	w.left = -f.windowEnd(w, far, -1)
	w.right = f.windowEnd(w, far, 1)

	// The hexagon's corners, clipped from a square that holds the tube.
	q := [][2]float64{{-2 * f.ro, -2 * f.ro}, {2 * f.ro, -2 * f.ro}, {2 * f.ro, 2 * f.ro}, {-2 * f.ro, 2 * f.ro}}
	for _, h := range [6][3]float64{
		{w.lean, 1, w.hi}, {-w.lean, -1, -w.lo},
		{-w.lean, 1, w.top}, {w.lean, -1, -w.bottom},
		{1, 0, w.right}, {-1, 0, -w.left},
	} {
		q = clipCorners(q, h[0], h[1], h[2])
	}
	w.corners = dropRepeats(q)
	return w
}

// clipCorners keeps the part of a convex polygon of any size where a*t + b*z <= c.
func clipCorners(q [][2]float64, a, b, c float64) [][2]float64 {
	var out [][2]float64
	for i := range q {
		p, r := q[i], q[(i+1)%len(q)]
		fp, fr := c-(a*p[0]+b*p[1]), c-(a*r[0]+b*r[1])
		if fp >= 0 {
			out = append(out, p)
		}
		if (fp >= 0) != (fr >= 0) {
			k := fp / (fp - fr)
			out = append(out, [2]float64{p[0] + k*(r[0]-p[0]), p[1] + k*(r[1]-p[1])})
		}
	}
	return out
}

// minCornerGap is the shortest edge a window's hexagon may have: a sketch line cannot have
// zero length, so a corner this close to the one before it is dropped.
const minCornerGap = 0.001

// dropRepeats drops each corner that lies within minCornerGap of the one before it, and the
// last one when it lies that close to the first.
func dropRepeats(q [][2]float64) [][2]float64 {
	var out [][2]float64
	for _, c := range q {
		if len(out) > 0 && math.Hypot(c[0]-out[len(out)-1][0], c[1]-out[len(out)-1][1]) < minCornerGap {
			continue
		}
		out = append(out, c)
	}
	for len(out) > 1 && math.Hypot(out[0][0]-out[len(out)-1][0], out[0][1]-out[len(out)-1][1]) < minCornerGap {
		out = out[:len(out)-1]
	}
	return out
}

// windowEnd is how far out on the side dir the window's upright end can stand: the farthest
// t, no further out than Ri or Ro/sqrt(2), at which every point below keeps CollarWall and the
// slack from each bore that does not flank the window. The points are both of the end's
// corners, walked along their lines through the wall at windowStep, and the end's upright
// edges on the inner and the outer face at the heights a full check of the window would
// sample them at: its walk along the edge, and its walk of the openings, which steps up from
// -zLimit. The corners alone let an end's edge on the inner face come too near a far bore at
// small clearances. An end the band and the trims have already closed off is taken as too
// far. The search bisects 24 times between 0 and the limit.
func (f sleeve) windowEnd(w sleeveWindow, far []bore, dir float64) float64 {
	need := f.p.CollarWall + windowSlack
	clear := func(te float64) bool {
		t := dir * te
		zLow := math.Max(w.lo-w.lean*t, w.bottom+w.lean*t)
		zHigh := math.Min(w.hi-w.lean*t, w.top+w.lean*t)
		if zLow > zHigh {
			return false
		}
		a0, a1 := f.chord(t)
		near := func(z, a float64) bool {
			for _, b := range far {
				if f.wallGap(b, w.at(t, z, a), need) < need {
					return true
				}
			}
			return false
		}
		for _, z := range [2]float64{zLow, zHigh} {
			for a := a0; ; a = math.Min(a+windowStep, a1) {
				if near(z, a) {
					return false
				}
				if a >= a1 {
					break
				}
			}
		}
		n := int(math.Ceil((zHigh - zLow) / windowStep))
		for _, a := range [2]float64{a0, a1} {
			for k := 1; k < n; k++ {
				if near(zLow+(zHigh-zLow)*float64(k)/float64(n), a) {
					return false
				}
			}
			for z := -w.zLimit; z <= w.zLimit; z += windowStep {
				if z > zLow && z < zHigh && near(z, a) {
					return false
				}
			}
		}
		return true
	}
	lo, hi := 0.0, math.Min(f.ri, f.ro/math.Sqrt2)*(1-1e-9)
	for range 24 {
		m := (lo + hi) / 2
		if clear(m) {
			lo = m
		} else {
			hi = m
		}
	}
	return lo
}

// area is the hexagon's area, by the shoelace formula.
func (w sleeveWindow) area() float64 {
	a := 0.0
	for i := range w.corners {
		p, q := w.corners[i], w.corners[(i+1)%len(w.corners)]
		a += p[0]*q[1] - q[0]*p[1]
	}
	return a / 2
}

// room is why the window is not cut, or "" when it is.
func (w sleeveWindow) room() string {
	if w.hi <= w.lo {
		return "the flanking bores leave no band between them"
	}
	if w.right <= w.left || len(w.corners) < 3 || w.area() <= 0 {
		return "the far bores leave the band no length"
	}
	return ""
}
