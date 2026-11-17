package swisslever

import (
	"math"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// midHeight is where the wheel and the stones are cut: inside the wheel's layers and the stones' length.
const midHeight = 11.85

var (
	wheelOnce  sync.Once
	wheelBody  *decad.Body
	wheelTeeth []outline
	wheelOff   float64
	wheelErr   error
)

// loopOutline returns a section loop as a polygon in the XY plane, relative to origin.
func loopOutline(l []r3.Vec, origin Pt) outline {
	out := make(outline, len(l))
	for i, v := range l {
		out[i] = Pt{v.X, v.Y}.Sub(origin)
	}
	return out
}

// farthestLoop returns the loop that reaches farthest from origin.
func farthestLoop(loops [][]r3.Vec, origin Pt) outline {
	var best outline
	far := -1.0
	for _, l := range loops {
		o := loopOutline(l, origin)
		m := 0.0
		for _, p := range o {
			m = math.Max(m, p.Len())
		}
		if m > far {
			best, far = o, m
		}
	}
	return best
}

// splitTeeth cuts the wheel's outer loop into one polygon per tooth, by the 24 degree sector each tooth owns.
func splitTeeth(loop outline) []outline {
	out := make([]outline, Teeth)
	for _, v := range loop {
		if v.Len() < RootRadius-0.5 {
			continue
		}
		a := math.Mod(v.Angle()/deg-84+720, 360)
		out[int(a/24)%Teeth] = append(out[int(a/24)%Teeth], v)
	}
	return out
}

// realWheel builds the real escape wheel once and returns it with its tooth polygons at the middle of its layers.
func realWheel(t *testing.T) (*decad.Body, []outline, float64) {
	t.Helper()
	wheelOnce.Do(func() {
		g, err := Layout(DefaultParams())
		if err != nil {
			wheelErr = err
			return
		}
		b := newBuilder(t.Context(), NewDocument(), g)
		if wheelBody, wheelErr = b.buildEscapeWheel(); wheelErr != nil {
			return
		}
		o := partSection(t, wheelBody, midHeight)
		wheelOff = o.OffSurface.Base()
		wheelTeeth = splitTeeth(farthestLoop(o.Loops, Pt{}))
	})
	require.NoError(t, wheelErr)
	return wheelBody, wheelTeeth, wheelOff
}

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

// realStones builds the two stones and returns their sections about the pallet arbor.
func realStones(t *testing.T) (*builder, [2]*decad.Body, [2]outline, float64) {
	t.Helper()
	b := partBuilder(t)
	e, x, err := b.buildStones()
	require.NoError(t, err)
	var bodies [2]*decad.Body
	var out [2]outline
	off := 0.0
	for i, body := range []*decad.Body{e, x} {
		bodies[i] = body
		o := partSection(t, body, midHeight)
		off = math.Max(off, o.OffSurface.Base())
		require.Len(t, o.Loops, 1)
		out[i] = loopOutline(o.Loops[0], b.g.P)
	}
	return b, bodies, out, off
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
