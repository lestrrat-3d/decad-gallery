package swisslever

import (
	"math"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// studTip is the height of the hairspring stud's tip.
const studTip = FrontPlateInner - studHeight

func plateTestBuilder(t *testing.T) *builder {
	t.Helper()
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	return newBuilder(t.Context(), NewDocument(), g)
}

// plateSectionAt cuts body with the horizontal plane at z at the finest chord tolerance the body allows. A faceted
// Boolean result refuses a chord below its own bound, so the cut retries coarser.
func plateSectionAt(t *testing.T, body *decad.Body, z float64) decad.SectionOutline {
	t.Helper()
	f, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	var o decad.SectionOutline
	for _, tol := range []float64{0.005, 0.02, 0.05} {
		if o, err = body.SectionOutline(t.Context(), f, units.Millimeters(tol)); err == nil {
			break
		}
	}
	require.NoError(t, err)
	return o
}

// plateInPoly reports whether x lies inside the closed polygon (even-odd).
func plateInPoly(poly []r3.Vec, x, y float64) bool {
	in := false
	for i, a := range poly {
		c := poly[(i+1)%len(poly)]
		if (a.Y > y) == (c.Y > y) {
			continue
		}
		if x < a.X+(y-a.Y)/(c.Y-a.Y)*(c.X-a.X) {
			in = !in
		}
	}
	return in
}

// solidAt reports whether the point is inside the solid, by the section of the body at the point's height.
func solidAt(t *testing.T, body *decad.Body, x, y, z float64) bool {
	t.Helper()
	n := 0
	for _, l := range plateSectionAt(t, body, z).Loops {
		if plateInPoly(l, x, y) {
			n++
		}
	}
	return n%2 == 1
}

// sectionHas reports whether the point is inside the section's solid (even-odd over its loops).
func sectionHas(o decad.SectionOutline, x, y float64) bool {
	n := 0
	for _, l := range o.Loops {
		if plateInPoly(l, x, y) {
			n++
		}
	}
	return n%2 == 1
}

// windowGap returns the least distance from the point to the section's loops that reach the window edge x = 45
// (the windows), less r.
func windowGap(o decad.SectionOutline, c Pt, r float64) float64 {
	m := math.Inf(1)
	for _, l := range o.Loops {
		window := false
		for _, v := range l {
			if math.Abs(math.Abs(v.X)-winSide) < 0.01 && v.Y > winW1Bottom+6 && v.Y < winW4Top-6 {
				window = true
				break
			}
		}
		if !window {
			continue
		}
		for i := range l {
			a, e := l[i], l[(i+1)%len(l)]
			m = math.Min(m, plateSegDist(c, Pt{a.X, a.Y}, Pt{e.X, e.Y}))
		}
	}
	return m - r
}

func plateSegDist(x, a, b Pt) float64 {
	d := b.Sub(a)
	l2 := d.Dot(d)
	if l2 == 0 {
		return x.Sub(a).Len()
	}
	t := math.Max(0, math.Min(1, x.Sub(a).Dot(d)/l2))
	return x.Sub(a.Add(d.Scale(t))).Len()
}

type builtPlates struct {
	b           *builder
	back, front *decad.Body
}

var (
	platesOnce   sync.Once
	platesShared builtPlates
	platesErr    error
)

// buildPlates returns both finished plates, built once for the whole test binary. Tests must only read the plates.
// The builder it returns is a fresh one over the same layout, so a test that builds more parts does not change what
// another test reads.
func buildPlates(t *testing.T) builtPlates {
	t.Helper()
	platesOnce.Do(func() {
		b := plateTestBuilder(t)
		back, err := b.buildBackPlate()
		if err != nil {
			platesErr = err
			return
		}
		front, err := b.buildFrontPlate()
		if err != nil {
			platesErr = err
			return
		}
		platesShared = builtPlates{back: back, front: front}
	})
	require.NoError(t, platesErr)
	return builtPlates{plateTestBuilder(t), platesShared.back, platesShared.front}
}

// plateEdges returns the length of the edges the finishing chamfers on one face of a plate: the outer loop and every
// window (the loops of the section at z that are not the lips), plus the pockets or lip rims given.
func plateEdges(t *testing.T, body *decad.Body, z float64) float64 {
	t.Helper()
	sum := 0.0
	for _, l := range plateSectionAt(t, body, z).Loops {
		lo, hi := l[0], l[0]
		for _, v := range l {
			lo = r3.Vec{X: math.Min(lo.X, v.X), Y: math.Min(lo.Y, v.Y)}
			hi = r3.Vec{X: math.Max(hi.X, v.X), Y: math.Max(hi.Y, v.Y)}
		}
		if hi.X-lo.X < 12 {
			continue
		}
		for i, v := range l {
			sum += math.Hypot(l[(i+1)%len(l)].X-v.X, l[(i+1)%len(l)].Y-v.Y)
		}
	}
	return sum
}

// requireNoOverlap checks that a and b share no volume by their section outlines at each z.
func requireNoOverlap(t *testing.T, a, b *decad.Body, zs ...float64) {
	t.Helper()
	for _, z := range zs {
		c, err := plateSectionAt(t, a, z).Clearance(plateSectionAt(t, b, z))
		require.NoError(t, err)
		require.False(t, c.Overlaps, "overlap at z %v", z)
	}
}
