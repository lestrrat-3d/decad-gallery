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
