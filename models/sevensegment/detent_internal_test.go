package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

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
		for u := u0; u < r.dm.PawlArmInner+pawlOuterOpen+0.5; u += 0.002 {
			cur := insideLoops(pl, rotate(r3.Vec{X: u, Y: v}, PawlAngle))
			if cur != prev {
				edges = append(edges, u)
			}
			prev = cur
		}
		require.GreaterOrEqual(t, len(edges), 2, "window void, arm, slot void at v %.2f", v)
		require.InDelta(t, r.dm.PawlArmInner, edges[0], sectionBound)
		require.InDelta(t, PawlArmTip+(PawlArmRoot-PawlArmTip)*(v-g.lk.Y)/(g.vr-g.lk.Y), edges[1]-edges[0], sectionBound)
		// The outer slot's side, on the line from its root corner to where it meets the window circle, where the slot
		// runs straight (past the window, before the root rounds).
		if v > g.jo.Y && v < g.vr-1.0 {
			require.Len(t, edges, 3, "window void, arm, slot void, plate at v %.2f", v)
			// The side stays where it was for the 1.75 mm root: 3.55 past the inner edge at the root, 4.75 at the window.
			u0, u1 := r.dm.PawlArmInner+3.55, r.dm.PawlArmInner+4.75
			side := u0 + (u1-u0)*(v-g.vr)/(g.jo.Y-g.vr)
			require.InDelta(t, side, edges[2], sectionBound, "the outer slot side at v %.2f", v)
		}
	}
	// The outer slots keep their sides when the arms thicken: in the section through the head pockets, the window loop
	// keeps 1 mm or more of material round each D head pocket (pawl B's outer slot passes D2).
	pockets := sectionAt(t, plate, detentFace-1.0)
	var window [][]r3.Vec
	for _, l := range pockets {
		if c, e := centroidExtent(l); c.Len() < 3 && e > 30 {
			window = append(window, l)
		}
	}
	require.Len(t, window, 1, "the window loop")
	for _, site := range r.dm.Sites[8:] {
		centre := r3.Vec{X: site.Head.X, Y: site.Head.Z - ShaftZ}
		var pocket [][]r3.Vec
		for _, l := range pockets {
			if c, e := centroidExtent(l); c.Sub(centre).Len() < 0.5 && e < 6 {
				pocket = append(pocket, l)
			}
		}
		require.Len(t, pocket, 1, "the head pocket at x %.0f", site.Head.X)
		gap := loopsGap(window, pocket)
		t.Logf("window to the head pocket at x %.0f: %.3f", site.Head.X, gap)
		require.GreaterOrEqual(t, gap, 1.0-sectionBound, "material between the window and the pocket at x %.0f", site.Head.X)
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
