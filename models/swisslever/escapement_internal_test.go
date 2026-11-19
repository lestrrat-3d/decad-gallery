package swisslever

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEscapeWheel(t *testing.T) {
	body, teeth, off := realWheel(t)
	require.True(t, body.IsSolid())
	bb, err := body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, wheelLow, bb.Min.Z, 3e-3)
	require.InDelta(t, wheelHigh, bb.Max.Z, 3e-3)
	o := partSection(t, body, midHeight)
	require.Len(t, o.Loops, wheelWindows+2, "outer loop, five windows and the bore")
	require.Len(t, teeth, Teeth)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	for j, tooth := range teeth {
		require.NotEmpty(t, tooth, "tooth %d", j)
		tip := 0.0
		toe := math.Inf(1)
		a := 90*deg + float64(j)*g.Pitch
		for _, v := range tooth {
			tip = math.Max(tip, v.Len())
			if math.Abs(normAngle(v.Angle()-a)) < 0.2*deg {
				toe = math.Min(toe, math.Abs(v.Len()-ToeRadius))
			}
		}
		require.InDelta(t, g.HeelRadius, tip, 0.03+off, "heel radius of tooth %d", j)
		require.Less(t, toe, 0.03+off, "toe radius of tooth %d", j)
	}
}

func TestEscapeWheelSketches(t *testing.T) {
	b := partBuilder(t)
	_, err := b.buildEscapeWheel()
	require.NoError(t, err)
	requireSketchesConstrained(t, b, 3)
}

func TestEscapeWheelCorners(t *testing.T) {
	// The club teeth are 71 degrees or blunter and their roots are rounded, so no corner needs an exemption.
	body, _, _ := realWheel(t)
	requireCorners(t, body)
}

func TestStones(t *testing.T) {
	b, bodies, sections, off := realStones(t)
	for i, st := range []StoneSection{b.g.Entry, b.g.Exit} {
		bb, err := bodies[i].Bounds()
		require.NoError(t, err)
		require.InDelta(t, stoneLow, bb.Min.Z, 3e-3)
		require.InDelta(t, stoneLow+stoneLength, bb.Max.Z, 3e-3)
		for name, want := range map[string]Pt{"C": st.C, "F": st.F, "G": st.G, "Hs": st.Hs} {
			best := math.Inf(1)
			for _, v := range sections[i] {
				best = math.Min(best, v.Sub(want).Len())
			}
			require.Less(t, best, 0.02+off, "stone %d corner %s", i, name)
		}
		a := 0.0
		for j, v := range sections[i] {
			a += v.Cross(sections[i][(j+1)%len(sections[i])]) / 2
		}
		corners := st.Corners()
		want := 0.0
		for j, v := range corners {
			want += v.Cross(corners[(j+1)%4]) / 2
		}
		require.InDelta(t, math.Abs(want), math.Abs(a), 0.05, "stone %d section area", i)
	}
	requireSketchesConstrained(t, b, 2)
}

func TestStoneCorners(t *testing.T) {
	// Every stone corner is 64.9 degrees or blunter, so none needs an exemption.
	_, bodies, _, _ := realStones(t)
	requireCorners(t, bodies[0])
	requireCorners(t, bodies[1])
}

func TestEscapementAction(t *testing.T) {
	_, teeth, wheelOff := realWheel(t)
	_, _, stones, stoneOff := realStones(t)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	c := newChecker(&g, teeth, stones, 2000)
	res, err := c.run()
	require.NoError(t, err)
	bound := wheelOff + stoneOff
	requireAction(t, &g, res, 0.05*deg, 1.70, 0.9)
	for _, d := range res.drops {
		require.Less(t, d.landing, 0.05+bound, "the toe lands on a locking face at psi %v", d.psi/deg)
	}
	require.Less(t, res.maxPhi, 0.05*deg+bound/ToeRadius, "closed form")
	t.Logf("section bound %.4f advance %v E1 %.3f E2 %.3f", bound, res.advance, res.others, res.rootGap)
}
