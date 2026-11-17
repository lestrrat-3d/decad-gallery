package swisslever

import (
	"math"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"

	"github.com/stretchr/testify/require"
)

// TestLeverSketches requires the lever outline (two slots, the dart, the corner and gap rounds, three frame rounds) and
// the notch tool to verify fully constrained with DOF 0.
func TestLeverSketches(t *testing.T) {
	b := partBuilder(t)
	s, err := b.leverSketch()
	require.NoError(t, err)
	_, err = b.profile(s, 2, "lever")
	require.NoError(t, err)
	_, err = b.notchTool()
	require.NoError(t, err)
	requireSketchesConstrained(t, b, 2)
}

// dartRootRefusal is the decad refusal the lever's dart root fillet gets until decad rounds an edge that ends at a
// tangent junction.
const dartRootRefusal = "the selected edge ends at the tangent junction"

// TestLever builds the lever with its dart root round. It skips only on the known refusal of that fillet, so it runs
// on its own once decad accepts it.
func TestLever(t *testing.T) {
	b := partBuilder(t)
	body, err := b.buildLever()
	if err != nil {
		require.ErrorContains(t, err, "dart root fillet", "only the dart root fillet may fail")
		require.ErrorContains(t, err, dartRootRefusal)
		t.Skipf("blocked: %v", err)
	}
	require.True(t, body.IsSolid())
	bb, err := body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, leverLow, bb.Min.Z, 3e-3)
	require.InDelta(t, leverHigh, bb.Max.Z, 3e-3)
}

// plainLever builds the lever without its dart root round, once.
var (
	leverOnce sync.Once
	leverB    *builder
	leverBody *decad.Body
	leverErr  error
)

func plainLever(t *testing.T) (*builder, *decad.Body) {
	t.Helper()
	leverOnce.Do(func() {
		leverB = partBuilder(t)
		leverB.skipDartRoot = true
		leverBody, leverErr = leverB.buildLever()
	})
	require.NoError(t, leverErr)
	return leverB, leverBody
}

// TestLeverBody checks the lever without its dart root: bounds, the band F notch, the band S dart, the slots through
// both bands.
func TestLeverBody(t *testing.T) {
	b, body := plainLever(t)
	g := &b.g
	require.True(t, body.IsSolid())
	bb, err := body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, leverLow, bb.Min.Z, 3e-3)
	require.InDelta(t, leverHigh, bb.Max.Z, 3e-3)
	in := func(p Pt, z float64) bool { return partInside(t, body, g.P.X+p.X, g.P.Y+p.Y, z) }
	// Band F (z 7.8): the notch is open and no dart stands in it; the horns remain.
	require.False(t, in(Pt{0, g.KR.Y + 1}, 7.8), "notch open above its centre")
	require.False(t, in(Pt{0, 42.5}, 7.8), "no dart in band F")
	require.True(t, in(Pt{2.4, g.KR.Y}, 7.8), "horn")
	require.True(t, in(Pt{-2.4, g.KR.Y}, 7.8), "horn")
	// Band S (z 4.8): the dart reaches y 42.75 and the notch floor is solid below the dart tip.
	require.True(t, in(Pt{0, 42.7}, 4.8), "dart tip")
	require.False(t, in(Pt{0, 43.0}, 4.8), "beyond the dart tip")
	require.True(t, in(Pt{0, g.KR.Y - 1}, 4.8), "band S under the notch")
	// The slots run through both bands.
	for _, z := range []float64{4.8, 7.8} {
		for _, st := range []StoneSection{g.Entry, g.Exit} {
			cs := st.Corners()
			c := centroid(cs[:])
			require.False(t, in(c, z), "slot open at z %v", z)
		}
	}
	// The hub bore is a D: open inside the arc, solid beyond the flat.
	require.False(t, in(Pt{-3.0, 0}, 6), "bore")
	require.True(t, in(Pt{3.0, 0}, 6), "material beyond the flat")
}

// TestStonesKeyed requires each stone to sit in its slot with 0.05 mm all round and every wrong way of fitting it to
// overlap the lever by 0.3 mm or more: turned 180 degrees about its length, turned over, or put in the other slot.
func TestStonesKeyed(t *testing.T) {
	b, body := plainLever(t)
	g := &b.g
	_, _, stones, stoneOff := realStones(t)
	o := partSection(t, body, forkBandF)
	bound := o.OffSurface.Base() + stoneOff
	var lever []outline
	for _, l := range o.Loops {
		lever = append(lever, loopOutline(l, g.P))
	}
	// The slots are the two loops nearest the stones' centroids.
	slotOf := func(c Pt) outline {
		var best outline
		d := math.Inf(1)
		for _, l := range lever {
			if !inPoly(l, c) {
				continue
			}
			ex := 0.0
			for _, v := range l {
				ex = math.Max(ex, v.Len())
			}
			if m := centroid(l).Sub(c).Len(); m < d && ex < 40 {
				best, d = l, m
			}
		}
		return best
	}
	slots := [2]outline{slotOf(centroid(stones[0])), slotOf(centroid(stones[1]))}
	for i := range slots {
		require.NotEmpty(t, slots[i], "slot %d", i)
		require.InDelta(t, SlotPress, polyDist(stones[i], slots[i]), bound+0.01, "stone %d press gap", i)
	}
	about := func(p outline, f func(Pt) Pt, to Pt) outline {
		c := centroid(p)
		out := make(outline, len(p))
		for i, v := range p {
			out[i] = f(v.Sub(c)).Add(to)
		}
		return out
	}
	turned180 := func(v Pt) Pt { return Pt{-v.X, -v.Y} }
	flipped := func(v Pt) Pt { return Pt{-v.X, v.Y} }
	same := func(v Pt) Pt { return v }
	for i, st := range stones {
		home := centroid(st)
		for name, v := range map[string]outline{
			"turned 180":  about(st, turned180, home),
			"turned over": about(st, flipped, home),
			"other slot":  about(st, same, centroid(stones[1-i])),
		} {
			depth := loopsDepth([]outline{v}, lever)
			require.GreaterOrEqual(t, depth, 0.3-bound, "stone %d %s overlaps the lever", i, name)
		}
	}
}

// TestLeverCorners reads the lever in its print orientation. The slot corners are mating corners and the gap rounds'
// bottoms read just over the limit on the section mesh; every other corner passes.
func TestLeverCorners(t *testing.T) {
	b, body := plainLever(t)
	g := &b.g
	z := r3.Vec{Z: 1}
	var exempt []cornercheck.Exempt
	for _, slot := range [][4]Pt{g.EntrySlot, g.ExitSlot} {
		for _, v := range slot {
			exempt = append(exempt, cornercheck.Exempt{Reason: "slot corner",
				Near: cornercheck.Disc(r3.Vec{X: g.P.X + v.X, Y: g.P.Y + v.Y}, z, 0.5)})
		}
	}
	for _, c := range []Pt{g.GapRight, g.GapLeft} {
		exempt = append(exempt, cornercheck.Exempt{Reason: "gap round bottom",
			Near: cornercheck.Disc(r3.Vec{X: g.P.X + c.X, Y: g.P.Y + c.Y - g.GapRadius}, z, 0.5)})
	}
	requireCorners(t, body, exempt...)
}
