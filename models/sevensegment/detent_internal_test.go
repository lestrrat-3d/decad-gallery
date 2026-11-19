package sevensegment

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// sectionBound is the allowance for the meshes of a section, in mm.
const sectionBound = 0.03

// densify adds points along every segment of the loops, at most step apart.
func densify(loops [][]r3.Vec, step float64) []r3.Vec {
	var out []r3.Vec
	for _, l := range loops {
		for i, a := range l {
			b := l[(i+1)%len(l)]
			n := int(math.Ceil(b.Sub(a).Len() / step))
			for k := 0; k < max(n, 1); k++ {
				out = append(out, a.Add(b.Sub(a).Scale(float64(k)/float64(max(n, 1)))))
			}
		}
	}
	return out
}

func segDist(p, a, b r3.Vec) float64 {
	e := b.Sub(a)
	l := e.Dot(e)
	f := 0.0
	if l > 0 {
		f = math.Max(0, math.Min(1, p.Sub(a).Dot(e)/l))
	}
	return p.Sub(a.Add(e.Scale(f))).Len()
}

func nearest(p r3.Vec, loops [][]r3.Vec) float64 {
	best := math.Inf(1)
	for _, l := range loops {
		for i, a := range l {
			best = math.Min(best, segDist(p, a, l[(i+1)%len(l)]))
		}
	}
	return best
}

func turnLoops(loops [][]r3.Vec, deg float64) [][]r3.Vec {
	out := make([][]r3.Vec, len(loops))
	for i, l := range loops {
		for _, p := range l {
			out[i] = append(out[i], rotate(p, deg))
		}
	}
	return out
}

// overlaps reports whether a boundary point of either section lies in the other's material, by more than inset.
func overlaps(a, b [][]r3.Vec) bool {
	for _, p := range densify(a, 0.05) {
		if insideLoops(b, p) && nearest(p, b) > 0.01 {
			return true
		}
	}
	for _, p := range densify(b, 0.05) {
		if insideLoops(a, p) && nearest(p, a) > 0.01 {
			return true
		}
	}
	return false
}

func TestSevenSegmentDetentPlate(t *testing.T) {
	r, plate := sharedBody(t, "detent plate", (*run).detentPlate)
	require.True(t, plate.IsSolid())
	_, wheel := sharedBody(t, "ratchet wheel", (*run).ratchetWheel)
	box, err := plate.Bounds()
	require.NoError(t, err)
	require.InDelta(t, -49.25, box.Min.Y, 1e-6)
	require.InDelta(t, -37.25, box.Max.Y, 1e-6)
	require.InDelta(t, 0, box.Min.Z, 1e-6)
	require.InDelta(t, 49, box.Max.Z, 1e-6)

	mid := -43.25
	pl := sectionAt(t, plate, mid)
	wh := sectionAt(t, wheel, mid)

	// The arm: inner edge on PawlArmInner, thickness linear from PawlArmTip to PawlArmRoot, read across pawl A at three
	// stations along it (pawl frame: seat on +u, arm along +v, turned by PawlAngle). The outer edge runs from the lock
	// flank's outer end to the root.
	g := r.pawlGeometry()
	for _, f := range []float64{0.1, 0.5, 0.9} {
		v := PawlBackV + f*PawlArmLength
		var edges []float64
		u0 := r.dm.PawlArmInner - 1
		prev := insideLoops(pl, rotate(r3.Vec{X: u0, Y: v}, PawlAngle))
		for u := u0; u < r.dm.PawlArmInner+PawlArmRoot+1; u += 0.002 {
			cur := insideLoops(pl, rotate(r3.Vec{X: u, Y: v}, PawlAngle))
			if cur != prev {
				edges = append(edges, u)
			}
			prev = cur
		}
		require.Len(t, edges, 2, "window void, arm, slot void at v %.2f", v)
		require.InDelta(t, r.dm.PawlArmInner, edges[0], sectionBound)
		require.InDelta(t, PawlArmTip+(PawlArmRoot-PawlArmTip)*(v-g.lk.Y)/(g.vr-g.lk.Y), edges[1]-edges[0], sectionBound)
	}
	// The window circle, on the ray half way between the pawls.
	edge := 0.0
	for rad := 10.0; rad < 25 && !insideLoops(pl, polar(rad, PawlAngle-90)); rad += 0.002 {
		edge = rad
	}
	require.InDelta(t, WindowRadius, edge, sectionBound)

	// Gaps at theta = 0: the head to the flank (PawlFlankGap), the rest of the plate to the wheel, and each arm's inner
	// edge (u = PawlArmInner past the back-flank point, pawl frame) to the tooth tips.
	head, rest, arm := math.Inf(1), math.Inf(1), math.Inf(1)
	for _, p := range densify(pl, 0.05) {
		d := nearest(p, wh)
		if p.Len() < r.dm.PawlArmInner-0.3 {
			head = math.Min(head, d)
		} else {
			rest = math.Min(rest, d)
		}
		for k := range 2 {
			q := rotate(p, -(PawlAngle + 180*float64(k)))
			if math.Abs(q.X-r.dm.PawlArmInner) < 0.05 && q.Y >= PawlBackV {
				arm = math.Min(arm, d)
			}
		}
	}
	require.GreaterOrEqual(t, head, PawlFlankGap-sectionBound)
	require.GreaterOrEqual(t, rest, TightFit)
	require.False(t, math.IsInf(arm, 1), "no arm inner edge found")
	require.GreaterOrEqual(t, arm, LooseFit-sectionBound)

	// Turning the wheel backward (counter-clockwise in front view) brings the lock face to the head; forward climbs.
	require.False(t, overlaps(pl, turnLoops(wh, 1.0)), "1.0 degrees back")
	require.True(t, overlaps(pl, turnLoops(wh, 1.5)), "1.5 degrees back")
	require.False(t, overlaps(pl, turnLoops(wh, -0.5)), "0.5 degrees forward")
	require.True(t, overlaps(pl, turnLoops(wh, -2.0)), "2.0 degrees forward")
}
