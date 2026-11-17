package swisslever

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// bevel is the volume a 0.5 mm 45° chamfer removes per millimetre of straight edge.
const bevel = 0.5 * 0.5 / 2

// perimeter returns the length of the closed polygon.
func perimeter(p outline) float64 {
	sum := 0.0
	for i := range p {
		sum += p[i].Sub(p[(i+1)%len(p)]).Len()
	}
	return sum
}

// extent returns the greatest distance of a vertex of p from c.
func extent(p outline, c Pt) float64 {
	m := 0.0
	for _, v := range p {
		m = math.Max(m, v.Sub(c).Len())
	}
	return m
}

// outerAndInner splits the loops of body at z into the outer loop (the one that reaches farthest from c) and the others.
func outerAndInner(t *testing.T, body *decad.Body, z float64, c Pt) (outline, []outline) {
	t.Helper()
	loops, _ := sectionLoops(t, body, z)
	best := 0
	for i := range loops {
		if extent(loops[i], c) > extent(loops[best], c) {
			best = i
		}
	}
	var inner []outline
	for i, l := range loops {
		if i != best {
			inner = append(inner, l)
		}
	}
	return loops[best], inner
}

// volumeOf returns the body's volume in mm³.
func volumeOf(t *testing.T, body *decad.Body) float64 {
	t.Helper()
	v, err := body.Volume()
	require.NoError(t, err)
	return v.Value.Base()
}

// TestEdgeFinishing requires each chamfered part to weigh less than its unchamfered build by the volume of the 0.5 mm
// chamfers instructions.md "Edge finishing" lists, and each kept edge to stay sharp. The plates are measured by
// TestPlateEdgeFinishing, which builds the same plates. The reads tolerate 6 % of the removed volume, as the plate test does.
func TestEdgeFinishing(t *testing.T) {
	_, m := sharedDefault(t)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	sw := movementSwitches(t)
	plain := newBuilder(t.Context(), NewDocument(), g)
	plain.plain = true
	if configure := sw.configure(); configure != nil {
		configure(plain)
	}
	table, err := poses(&g, m.State)
	require.NoError(t, err)
	check := func(name string, finished *decad.Body, unfinished func() (*decad.Body, error), edges float64) {
		t.Run(name, func(t *testing.T) {
			raw, err := unfinished()
			require.NoError(t, err)
			drop := volumeOf(t, raw) - volumeOf(t, finished)
			want := edges * bevel
			// A chamfer along a curved edge removes a little more or less than the straight-edge figure.
			require.InDelta(t, want, drop, 0.06*want, "volume drop %v against %v mm of chamfered edge", drop, edges)
		})
	}

	// Escape wheel: the five windows on both faces; the teeth keep their edges.
	_, windows := outerAndInner(t, m.EscapeWheel, 11.85, g.O)
	var windowEdges float64
	for _, w := range windows {
		if extent(w, g.O) > 12 {
			windowEdges += perimeter(w)
		}
	}
	check("escape wheel", m.EscapeWheel, plain.buildEscapeWheel, 2*windowEdges)

	// Lever: the outer loop of band S at its rear face and of band F at its front face; slots and the dart top stay sharp.
	bandS, _ := outerAndInner(t, m.Lever, 4.8, g.P)
	bandF, _ := outerAndInner(t, m.Lever, 7.8, g.P)
	check("lever", m.Lever, plain.buildLever, perimeter(bandS)+perimeter(bandF))

	// Balance wheel: the rim and the four windows on both faces; the boss face stays sharp.
	rim, wins := outerAndInner(t, m.BalanceWheel, 15.9, g.B)
	edges := perimeter(rim)
	for _, w := range wins {
		if extent(w, g.B) > 12 {
			edges += perimeter(w)
		}
	}
	if sw.chamfer != "" {
		t.Logf("balance wheel chamfer not measured: decad refuses it: %s", sw.chamfer)
	} else {
		check("balance wheel", m.BalanceWheel, plain.buildBalanceWheel, 2*edges)
	}

	// Kept sharp: a point 0.2 mm into the material from a slot's locking side, 0.1 mm below the lever's rear face, would
	// be gone if the slot rim were chamfered; so would a point just inside a tooth's club face on the wheel.
	for _, slot := range [][4]Pt{g.EntrySlot, g.ExitSlot} {
		mid := slot[0].Add(slot[3]).Scale(0.5)
		centre := slot[0].Add(slot[1]).Add(slot[2]).Add(slot[3]).Scale(0.25)
		out := mid.Sub(centre)
		out = out.Scale(0.2 / out.Len())
		p := table[LeverName].turnPt(g.P.Add(mid).Add(out))
		require.True(t, partInside(t, m.Lever, p.X, p.Y, leverLow+0.1), "slot rim is sharp at the rear face")
		require.True(t, partInside(t, m.Lever, p.X, p.Y, leverHigh-0.1), "slot rim is sharp at the front face")
	}
	mid := g.ToothT.Add(g.ToothH).Scale(0.5)
	in := g.ToothL.Add(g.ToothT).Add(g.ToothH).Add(g.ToothK).Scale(0.25).Sub(mid)
	in = in.Scale(0.2 / in.Len())
	p := table[EscapeWheelName].turnPt(mid.Add(in))
	require.True(t, partInside(t, m.EscapeWheel, p.X, p.Y, wheelLow+0.1), "tooth rim is sharp at the rear face")
	require.True(t, partInside(t, m.EscapeWheel, p.X, p.Y, wheelHigh-0.1), "tooth rim is sharp at the front face")
}
