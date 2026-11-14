package cornercheck

import (
	"cmp"
	"math"
	"slices"
	"sort"

	"github.com/lestrrat-3d/r3"
)

// hit is one window of outline that breaks a limit.
type hit struct {
	start, end float64 // arc length positions, end = start + window
	turn       float64 // degrees, positive
}

// scanLoop returns the findings on one closed outline. The material lies to the left of the loop's direction, as
// decad's SectionOutline gives it, and n is the plane normal. The loop is read as a polyline whose vertex turns
// are spread over the stretch up to half a segment (at most maxSpread) on each side of the vertex, so a window's turn
// does not depend on where the mesh happens to put its vertices.
func scanLoop(loop []r3.Vec, n r3.Vec, height float64) []Finding {
	pts := dedupe(loop)
	m := len(pts)
	if m < 3 {
		return nil
	}
	seg := make([]float64, m) // seg[i] is the length of pts[i] -> pts[i+1]
	start := make([]float64, m)
	total := 0.0
	for i := range pts {
		start[i] = total
		seg[i] = pts[(i+1)%m].Sub(pts[i]).Len()
		total += seg[i]
	}
	turn := make([]float64, m) // signed degrees at vertex i, positive when convex
	for i := range pts {
		u := pts[i].Sub(pts[(i+m-1)%m])
		v := pts[(i+1)%m].Sub(pts[i])
		turn[i] = math.Atan2(n.Dot(u.Cross(v)), u.Dot(v)) * 180 / math.Pi
	}
	cum := newCumulative(start, seg, turn, total)
	var out []Finding
	for _, rule := range []struct {
		kind   Kind
		window float64
		limit  float64
		sign   float64 // +1 reads convex turns, -1 reads concave turns
	}{
		{Inside, InsideWindow, InsideMaxTurn, -1},
		{Tip, TipWindow, TipMaxTurn, 1},
	} {
		for _, h := range cum.hits(rule.window, rule.limit, rule.sign) {
			at, ok := sharpest(pts, start, seg, turn, total, h, rule.sign)
			if !ok {
				continue
			}
			out = append(out, Finding{Kind: rule.kind, At: at, Turn: h.turn, Height: height})
		}
	}
	return out
}

// minSpacing is the shortest outline segment in mm that scanLoop keeps. A mesh Boolean leaves clusters of vertices
// micrometres apart whose segments double back; their turns cancel in a window that holds all of them but read as a
// sharp corner in a window that holds one. Dropping vertices closer than minSpacing to the last kept one removes
// them, and moves the outline by less than minSpacing.
const minSpacing = 0.002

// dedupe drops vertices closer than minSpacing to the previous kept vertex, including a closing point near the first.
func dedupe(loop []r3.Vec) []r3.Vec {
	var out []r3.Vec
	for _, p := range loop {
		if len(out) == 0 || p.Sub(out[len(out)-1]).Len() >= minSpacing {
			out = append(out, p)
		}
	}
	for len(out) > 1 && out[0].Sub(out[len(out)-1]).Len() < minSpacing {
		out = out[:len(out)-1]
	}
	return out
}

// maxSpread is the longest stretch in mm on each side of a vertex over which its turn is spread. A vertex between
// long segments keeps a sharp turn; a vertex on a fine mesh shares its turn with its neighbours.
const maxSpread = 0.1

// cumulative is the running turn along a loop as a piecewise linear function of arc length, extended over several
// periods so a window may run past the loop's start.
type cumulative struct {
	x, c  []float64
	total float64 // loop length
	turn  float64 // net turn around the loop
}

func newCumulative(start, seg, turn []float64, total float64) cumulative {
	m := len(start)
	run := 0.0
	for _, t := range turn {
		run += t
	}
	cu := cumulative{total: total, turn: run}
	for q := -1; q < 3; q++ {
		shift, base := float64(q)*total, float64(q)*run
		for i := range start {
			before := math.Min(seg[(i+m-1)%m]/2, maxSpread)
			after := math.Min(seg[i]/2, maxSpread)
			cu.x = append(cu.x, start[i]-before+shift, start[i]+after+shift)
			cu.c = append(cu.c, base, base+turn[i])
			base += turn[i]
		}
	}
	return cu
}

// at returns the running turn at arc length x.
func (cu cumulative) at(x float64) float64 {
	i := sort.SearchFloat64s(cu.x, x)
	switch {
	case i == 0:
		return cu.c[0]
	case i == len(cu.x):
		return cu.c[len(cu.c)-1]
	}
	x0, x1 := cu.x[i-1], cu.x[i]
	if x1 == x0 {
		return cu.c[i]
	}
	f := (x - x0) / (x1 - x0)
	return cu.c[i-1] + f*(cu.c[i]-cu.c[i-1])
}

// hits returns the merged windows whose turn, read with sign, exceeds limit. sign +1 reads convex turns and -1
// reads concave turns, both reported as positive degrees.
func (cu cumulative) hits(window, limit, sign float64) []hit {
	w := math.Min(window, cu.total)
	// The window turn is piecewise linear in its start, so its largest values sit where either end meets a
	// breakpoint.
	var cands []float64
	for _, x := range cu.x {
		for _, a := range []float64{x, x - w} {
			a = math.Mod(a, cu.total)
			if a < 0 {
				a += cu.total
			}
			cands = append(cands, a)
		}
	}
	slices.Sort(cands)
	var raw []hit
	for _, a := range cands {
		t := sign * (cu.at(a+w) - cu.at(a))
		if t > limit+turnSlack {
			raw = append(raw, hit{start: a, end: a + w, turn: t})
		}
	}
	if len(raw) == 0 {
		return nil
	}
	slices.SortFunc(raw, func(a, b hit) int { return cmp.Compare(a.start, b.start) })
	merged := []hit{raw[0]}
	for _, h := range raw[1:] {
		last := &merged[len(merged)-1]
		if h.start > last.end+w {
			merged = append(merged, h)
			continue
		}
		last.end = math.Max(last.end, h.end)
		last.turn = math.Max(last.turn, h.turn)
	}
	// A run that crosses the loop's start appears at both ends.
	if len(merged) > 1 {
		first, last := merged[0], &merged[len(merged)-1]
		if first.start+cu.total <= last.end+w {
			last.end = math.Max(last.end, first.end+cu.total)
			last.turn = math.Max(last.turn, first.turn)
			merged = merged[1:]
		}
	}
	return merged
}

// sharpest returns the vertex whose turn spreads into the hit with the largest turn read with sign.
func sharpest(pts []r3.Vec, start, seg, turn []float64, total float64, h hit, sign float64) (r3.Vec, bool) {
	best, found := 0.0, false
	var at r3.Vec
	for i, s := range start {
		for _, pos := range []float64{s, s + total} {
			// The vertex's turn spreads over half the segment on each side.
			before, after := seg[(i+len(seg)-1)%len(seg)]/2, seg[i]/2
			if pos+after < h.start || pos-before > h.end {
				continue
			}
			if t := sign * turn[i]; !found || t > best {
				best, found, at = t, true, pts[i]
			}
		}
	}
	return at, found
}
