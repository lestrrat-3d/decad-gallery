package swisslever

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// Section heights of the checker: band F of the lever with the pin, band S with the dart and the safety roller.
const (
	forkBandF = 7.8
	forkBandS = 4.8
	// forkStep is the balance angle step of the sweep, 0.03 degrees.
	forkStep = 0.03 * deg
	// forkSweep is the balance angle range on each side of zero, 60 degrees.
	forkSweep = 60 * deg
	// forkMargin is the balance angle either side of an event that the touch interval check leaves open.
	forkMargin = 1 * deg
)

// sectionLoops returns the loops of body at z as outlines in the XY plane, and the section's mesh bound.
func sectionLoops(t *testing.T, body *decad.Body, z float64) ([]outline, float64) {
	t.Helper()
	o := partSection(t, body, z)
	loops := make([]outline, len(o.Loops))
	for i, l := range o.Loops {
		for _, v := range l {
			loops[i] = append(loops[i], Pt{v.X, v.Y})
		}
	}
	return loops, o.OffSurface.Base()
}

// turnedAbout returns loops turned by a about c.
func turnedAbout(loops []outline, a float64, c Pt) []outline {
	out := make([]outline, len(loops))
	for i, l := range loops {
		out[i] = make(outline, len(l))
		for j, v := range l {
			out[i][j] = v.Sub(c).Rot(a).Add(c)
		}
	}
	return out
}

// movedBy returns loops moved by d.
func movedBy(loops []outline, d Pt) []outline {
	out := make([]outline, len(loops))
	for i, l := range loops {
		out[i] = turned(l, 0, d)
	}
	return out
}

// loopsGap returns the least distance between two sets of loops.
func loopsGap(a, b []outline) float64 {
	m := math.Inf(1)
	for _, p := range a {
		for _, q := range b {
			m = math.Min(m, polyDist(p, q))
		}
	}
	return m
}

// loopsInside reports whether x is in the solid the loops enclose, by even-odd over every loop.
func loopsInside(loops []outline, x Pt) bool {
	n := 0
	for _, l := range loops {
		if inPoly(l, x) {
			n++
		}
	}
	return n%2 == 1
}

// loopsDepth returns how deep the solid of a reaches into the solid of b: the greatest distance from a vertex of a
// inside b to b's boundary, or 0 when none is inside.
func loopsDepth(a, b []outline) float64 {
	d := 0.0
	for _, p := range a {
		for _, v := range p {
			if !loopsInside(b, v) {
				continue
			}
			near := math.Inf(1)
			for _, q := range b {
				for j := range q {
					near = math.Min(near, segDist(v, q[j], q[(j+1)%len(q)]))
				}
			}
			d = math.Max(d, near)
		}
	}
	return d
}

// rayDown returns the highest y below yFrom at which the loops cross the vertical line at x, or false.
func rayDown(loops []outline, x, yFrom float64) (float64, bool) {
	best, found := math.Inf(-1), false
	for _, l := range loops {
		for i, a := range l {
			c := l[(i+1)%len(l)]
			if (a.X > x) == (c.X > x) {
				continue
			}
			y := a.Y + (x-a.X)/(c.X-a.X)*(c.Y-a.Y)
			if y < yFrom && y > best {
				best, found = y, true
			}
		}
	}
	return best, found
}

// forkBodies holds the sections the sweep poses: lever bands F and S in the world at lever angle 0, and the roller
// table pin and the safety roller in the local frame about the origin.
type forkBodies struct {
	leverF, leverS, pin, rollerF, rollerS []outline
	off                                   float64
}

func newForkBodies(t *testing.T, b *builder) forkBodies {
	t.Helper()
	lever, err := b.buildLever()
	require.NoError(t, err)
	table, err := b.buildRollerTable()
	require.NoError(t, err)
	roller, err := b.buildSafetyRoller()
	require.NoError(t, err)
	var f forkBodies
	var o [5]float64
	f.leverF, o[0] = sectionLoops(t, lever, forkBandF)
	f.leverS, o[1] = sectionLoops(t, lever, forkBandS)
	f.pin, o[2] = sectionLoops(t, table, forkBandF)
	f.rollerF, o[3] = sectionLoops(t, roller, forkBandF)
	f.rollerS, o[4] = sectionLoops(t, roller, forkBandS)
	require.Len(t, f.pin, 1, "the pin alone at the lever's band F height")
	for _, v := range o {
		f.off = math.Max(f.off, v)
	}
	return f
}

// pose returns the lever bands about P and the pin and rollers about B for lever angle psi and balance angle theta.
func (f forkBodies) pose(g *Geometry, psi, theta float64) (leverF, leverS, pin, rollerF, rollerS []outline) {
	p := Pt{g.P.X, g.P.Y}
	b := Pt{g.B.X, g.B.Y}
	turn := func(l []outline) []outline { return movedBy(turnedAbout(l, g.GammaF+theta, Pt{}), b) }
	return turnedAbout(f.leverF, -psi, p), turnedAbout(f.leverS, -psi, p), turn(f.pin), turn(f.rollerF), turn(f.rollerS)
}

// segDistLoop returns the distance from x to the closed loop.
func segDistLoop(x Pt, l outline) float64 {
	m := math.Inf(1)
	for i, a := range l {
		m = math.Min(m, segDist(x, a, l[(i+1)%len(l)]))
	}
	return m
}
