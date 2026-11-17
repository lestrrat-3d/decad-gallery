package swisslever

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
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

// buildPlates builds both finished plates once per test.
func buildPlates(t *testing.T) builtPlates {
	t.Helper()
	b := plateTestBuilder(t)
	back, err := b.buildBackPlate()
	require.NoError(t, err)
	front, err := b.buildFrontPlate()
	require.NoError(t, err)
	return builtPlates{b, back, front}
}

// TestBackPlateBosses is the first real case of the plate step: the back plate's extrusion with its six bosses, each
// selected through the cap face the extrusion made.
func TestBackPlateBosses(t *testing.T) {
	b := plateTestBuilder(t)
	b.plain = true
	plate, err := b.buildBackPlate()
	require.NoError(t, err)
	require.True(t, plate.IsSolid())
	bb, err := plate.Bounds()
	require.NoError(t, err)
	require.InDelta(t, backPlateLow, bb.Min.Z, 2e-3)
	require.InDelta(t, pillarHeight, bb.Max.Z, 2e-3)
}

func TestPlates(t *testing.T) {
	p := buildPlates(t)
	g := &p.b.g
	require.True(t, p.back.IsSolid())
	require.True(t, p.front.IsSolid())

	// Bounds: the plates' extents and the posts' tips.
	bb, err := p.back.Bounds()
	require.NoError(t, err)
	require.InDelta(t, -plateHalfWidth, bb.Min.X, 2e-3)
	require.InDelta(t, plateHalfWidth, bb.Max.X, 2e-3)
	require.InDelta(t, plateBottomEdge, bb.Min.Y, 2e-3)
	require.InDelta(t, plateTopEdge, bb.Max.Y, 2e-3)
	fb, err := p.front.Bounds()
	require.NoError(t, err)
	require.InDelta(t, frontPlateHigh, fb.Max.Z, 2e-3)
	require.InDelta(t, studTip, fb.Min.Z, 2e-3, "hairspring stud tip")

	// Back plate at mid-plane z -3: windows open, bars, hub rings and rims solid.
	back := plateSectionAt(t, p.back, -3)
	open := [][2]float64{{30, -20}, {0, -20}, {0, -12}, {30, 20}, {0, 30}, {0, 12}, {30, 70}, {-30, 70}, {20, 55}, {-12, 60},
		{0, 115}, {30, 110}, {-30, 110}, {0, 106}}
	solid := [][2]float64{{30, 0}, {0, -8}, {0, 8}, {0, 36}, {30, 44.5714}, {0, 70}, {10.5, 70}, {-10.5, 70}, {30, 92.5714},
		{0, 100}, {20, 90}, {50, 50}, {-50, 50}, {0, -45}, {0, 138}, {0, 52.5}}
	for _, c := range open {
		require.False(t, sectionHas(back, c[0], c[1]), "back plate window at (%v, %v)", c[0], c[1])
	}
	for _, c := range solid {
		require.True(t, sectionHas(back, c[0], c[1]), "back plate material at (%v, %v)", c[0], c[1])
	}
	// Front plate at mid-plane z 30: W3 is one window; its hub rings sit over P and B.
	front := plateSectionAt(t, p.front, 30)
	for _, c := range open {
		if math.Abs(c[0]) < 12 && c[1] > 48 && c[1] < 89 {
			continue
		}
		require.False(t, sectionHas(front, c[0], c[1]), "front plate window at (%v, %v)", c[0], c[1])
	}
	for _, c := range [][2]float64{{0, 70}, {0, 56}, {0, 80}, {-30, 70}, {10, 70}} {
		require.False(t, sectionHas(front, c[0], c[1]), "front plate W3 at (%v, %v)", c[0], c[1])
	}
	for _, c := range [][2]float64{{0, 52.5}, {0, 83.5}, {30, 44.5714}, {-32.0106, 92.5714}} {
		require.True(t, sectionHas(front, c[0], c[1]), "front plate material at (%v, %v)", c[0], c[1])
	}

	// Pockets and lips: the pocket is open over its 5 mm, the lip bore runs through, and the lip is 1 mm thick.
	for _, a := range []Pt{g.O, g.P, g.B} {
		require.False(t, solidAt(t, p.back, a.X+6, a.Y, -2), "back pocket open")
		require.True(t, solidAt(t, p.back, a.X+6, a.Y, -5.5), "back lip is solid beyond the bore")
		require.False(t, solidAt(t, p.back, a.X+4.5, a.Y, -5.5), "back lip bore through")
		require.False(t, solidAt(t, p.front, a.X+6, a.Y, 28), "front pocket open")
		require.True(t, solidAt(t, p.front, a.X+6, a.Y, 32.5), "front lip is solid beyond the bore")
		require.False(t, solidAt(t, p.front, a.X+4.5, a.Y, 32.5), "front lip bore through")
	}
	// The wall between each pocket and the window over its hub ring is 2.95.
	for _, a := range []Pt{g.O, g.P, g.B} {
		for _, sec := range []decad.SectionOutline{back, front} {
			off := sec.OffSurface.Base()
			require.GreaterOrEqual(t, windowGap(sec, a, seatRadius), 2.95-off-0.01, "pocket wall at %v", a)
		}
	}

	// Posts: pillars to z 27, banking posts to 9.80, each standing on the inner face.
	for _, c := range g.Pillars {
		require.True(t, solidAt(t, p.back, c.X+4, c.Y, 10), "pillar")
		require.True(t, solidAt(t, p.back, c.X+4, c.Y, 26.9), "pillar top")
		require.False(t, solidAt(t, p.back, c.X+4, c.Y, 27.1), "above the pillar")
		require.False(t, solidAt(t, p.back, c.X+5.2, c.Y, 10), "beside the pillar")
	}
	for _, c := range []Pt{g.BankingLeft, g.BankingRight} {
		require.True(t, solidAt(t, p.back, c.X, c.Y+1.9, 9.7), "banking post")
		require.False(t, solidAt(t, p.back, c.X, c.Y+1.9, 9.9), "above the banking post")
		require.False(t, solidAt(t, p.back, c.X, c.Y+3, 5), "beside the banking post")
	}
	require.True(t, solidAt(t, p.front, g.Stud.X, g.Stud.Y+1.4, 19.0), "stud")
	require.False(t, solidAt(t, p.front, g.Stud.X, g.Stud.Y+1.4, 18.9), "below the stud's tip")
	require.False(t, solidAt(t, p.front, g.Stud.X, g.Stud.Y+2.2, 22), "beside the stud")

	// Material round the posts: 1.5 from a pillar to W1, 2 from the stud to its bar's edges, 1.34 from a banking post
	// (with its round, |x| <= 9.66) to the spine's edge at |x| = 11.
	require.Greater(t, winSpine-(math.Abs(g.BankingLeft.X)+BankingPostRadius+rootRound), 1.3)
	require.InDelta(t, 2.0, barHalf-(studDiameter/2+rootRound), 1e-9)

	// The chamfers cut the outer edges of both faces.
	require.False(t, solidAt(t, p.back, 55.9, 0, -5.9), "back plate rear edge")
	require.True(t, solidAt(t, p.back, 55.4, 0, -5.4), "back plate wall below the chamfer")
	require.False(t, solidAt(t, p.back, 55.9, 0, -0.1), "back plate inner edge")
	require.False(t, solidAt(t, p.front, 55.9, 0, 27.1), "front plate inner edge")
	require.False(t, solidAt(t, p.front, 55.9, 0, 32.9), "front plate front edge")
	require.True(t, solidAt(t, p.front, 55.4, 0, 32.4), "front plate wall below the chamfer")
	for _, a := range []Pt{g.O, g.P, g.B} {
		require.False(t, solidAt(t, p.back, a.X+7.4, a.Y, -0.1), "back pocket rim chamfer")
		require.False(t, solidAt(t, p.back, a.X+5.1, a.Y, -5.9), "back lip rim chamfer")
		require.True(t, solidAt(t, p.back, a.X+5.6, a.Y, -5.4), "back plate below the lip rim chamfer")
		require.False(t, solidAt(t, p.front, a.X+7.4, a.Y, 27.1), "front pocket rim chamfer")
		require.False(t, solidAt(t, p.front, a.X+5.1, a.Y, 32.9), "front lip rim chamfer")
		require.True(t, solidAt(t, p.front, a.X+5.6, a.Y, 32.4), "front plate below the lip rim chamfer")
	}
}

// TestPlateWalls checks 1 mm or more of material round every bolt hole, head pocket and nut cavity.
func TestPlateWalls(t *testing.T) {
	p := buildPlates(t)
	front := plateSectionAt(t, p.front, 32)
	for _, c := range p.b.g.Pillars {
		// The head pocket (r 2.2) to the nearest window, in the front plate.
		off := front.OffSurface.Base()
		require.GreaterOrEqual(t, windowGap(front, c, 2.2), 1.0-off, "head pocket at %v to a window", c)
		// The pillar's wall round its nut cavity (r 2.51) and bolt hole (r 1.3).
		require.GreaterOrEqual(t, pillarDiameter/2-2.51, 1.0)
		require.GreaterOrEqual(t, pillarDiameter/2-1.3, 1.0)
		// 1 mm or more of plate under the head pocket, and of pillar over and under the cavity.
		require.GreaterOrEqual(t, plateThickness-2.0, 1.0)
		require.False(t, solidAt(t, p.back, c.X+2.3, c.Y, 24), "nut cavity")
		require.True(t, solidAt(t, p.back, c.X+2.6, c.Y, 24), "cavity wall")
		require.True(t, solidAt(t, p.back, c.X+2.0, c.Y, 26.5), "cavity roof")
		require.True(t, solidAt(t, p.back, c.X+2.0, c.Y, 22.5), "cavity floor")
		require.False(t, solidAt(t, p.back, c.X+1.0, c.Y, 22), "bolt hole in the pillar")
		require.False(t, solidAt(t, p.back, c.X+1.0, c.Y, 26.5), "the bolt hole runs through the roof")
		require.True(t, solidAt(t, p.back, c.X+1.5, c.Y, 22), "pillar round the bolt hole")
		require.False(t, solidAt(t, p.front, c.X+1.0, c.Y, 29), "bolt hole in the front plate")
		require.False(t, solidAt(t, p.front, c.X+2.0, c.Y, 32), "head pocket")
		require.True(t, solidAt(t, p.front, c.X+2.0, c.Y, 30), "plate under the head pocket")
		require.True(t, solidAt(t, p.front, c.X+2.4, c.Y, 32), "plate beside the head pocket")
	}
}

// TestCentreLine checks that the three axes and both plates stand symmetric about the line x = 0.
func TestCentreLine(t *testing.T) {
	p := buildPlates(t)
	g := &p.b.g
	for name, a := range map[string]Pt{"O": g.O, "P": g.P, "B": g.B} {
		require.InDelta(t, 0, a.X, 1e-9, name)
	}
	require.InDelta(t, -g.BankingLeft.X, g.BankingRight.X, 1e-9)
	require.InDelta(t, g.BankingLeft.Y, g.BankingRight.Y, 1e-9)
	for _, body := range []*decad.Body{p.back, p.front} {
		bb, err := body.Bounds()
		require.NoError(t, err)
		require.InDelta(t, -bb.Min.X, bb.Max.X, 2e-3)
	}
	// Probe a grid at three heights: the back plate's plate, its pillars and the front plate (the stud is below it).
	for _, c := range []struct {
		body *decad.Body
		z    float64
	}{{p.back, -3}, {p.back, 5}, {p.front, 30}} {
		o := plateSectionAt(t, c.body, c.z)
		for y := -49.0; y < 146; y += 3.7 {
			for x := 0.5; x < 56; x += 3.3 {
				require.Equal(t, sectionHas(o, x, y), sectionHas(o, -x, y), "z %v at (%v, %v)", c.z, x, y)
			}
		}
	}
}

// TestPlateSketches requires every sketch of the plates and the hardware to be fully constrained.
func TestPlateSketches(t *testing.T) {
	b := plateTestBuilder(t)
	_, err := b.buildBackPlate()
	require.NoError(t, err)
	_, err = b.buildFrontPlate()
	require.NoError(t, err)
	_, err = b.buildHardware()
	require.NoError(t, err)
	require.Greater(t, len(b.sketches), 40)
	for i, s := range b.sketches {
		rep := s.Verify(t.Context())
		require.Equal(t, sketch.FullyConstrained, rep.Status, "sketch %d", i)
		require.Equal(t, 0, rep.DOF, "sketch %d", i)
		require.Empty(t, rep.Redundant, "sketch %d", i)
		require.Empty(t, rep.Conflicts, "sketch %d", i)
	}
}

func TestPlateCorners(t *testing.T) {
	p := buildPlates(t)
	for name, body := range map[string]*decad.Body{"back": p.back, "front": p.front} {
		t.Run(name, func(t *testing.T) {
			fs, err := cornercheck.Check(t.Context(), body, zAxis, nil)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		})
	}
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

// TestPlateEdgeFinishing compares each finished plate with its unfinished build: the volume drop is the outer loop and
// the windows on both faces, the three pocket rims and the three lip rims, each a 0.5 mm 45 degree chamfer.
func TestPlateEdgeFinishing(t *testing.T) {
	p := buildPlates(t)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	plain := newBuilder(t.Context(), NewDocument(), g)
	plain.plain = true
	bevel := 0.5 * 0.5 / 2
	rim := func(r float64) float64 { return math.Pi * (r + 0.5/3) * 0.25 }
	for name, c := range map[string]struct {
		finished *decad.Body
		build    func() (*decad.Body, error)
		z        float64
	}{"back": {p.back, plain.buildBackPlate, -5.5}, "front": {p.front, plain.buildFrontPlate, 32.5}} {
		raw, err := c.build()
		require.NoError(t, err)
		fv, err := c.finished.Volume()
		require.NoError(t, err)
		rv, err := raw.Volume()
		require.NoError(t, err)
		drop := rv.Value.Base() - fv.Value.Base()
		want := 2*plateEdges(t, c.finished, c.z)*bevel + 3*rim(seatRadius) + 3*rim(lipRadius)
		require.InDelta(t, want, drop, 0.06*want, "%s plate volume drop", name)
	}
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

func TestFasteners(t *testing.T) {
	p := buildPlates(t)
	h, err := p.b.buildHardware()
	require.NoError(t, err)
	require.Len(t, h.Bolts, 4)
	require.Len(t, h.Nuts, 4)
	tip := p.b.sites()[0].Tip()
	for i, c := range p.b.g.Pillars {
		bolt, nut := h.Bolts[i], h.Nuts[i]
		bb, err := bolt.Bounds()
		require.NoError(t, err)
		// The head's top is flush with the front plate's face at z 33; the tip stays inside the pillar's hole.
		require.InDelta(t, HeadTop, bb.Max.Z, 2e-3, "head top")
		require.InDelta(t, HeadTop-tip, bb.Min.Z, 2e-3, "bolt tip")
		require.Greater(t, bb.Min.Z, 20.0, "the tip stays inside the pillar hole, which ends at z 20")
		require.InDelta(t, c.X, (bb.Min.X+bb.Max.X)/2, 2e-3)
		require.Less(t, math.Max(bb.Max.X-c.X, bb.Max.Y-c.Y), 2.2, "head inside its pocket")
		nb, err := nut.Bounds()
		require.NoError(t, err)
		require.GreaterOrEqual(t, nb.Min.Z, 23.0, "nut inside its cavity")
		require.LessOrEqual(t, nb.Max.Z, 25.0, "nut inside its cavity")
		for _, d := range []float64{nb.Min.X - c.X, nb.Max.X - c.X, nb.Min.Y - c.Y, nb.Max.Y - c.Y} {
			require.LessOrEqual(t, math.Abs(d), 2.51)
		}
		requireNoOverlap(t, bolt, p.back, 21.5, 22.5, 23, 26.5)
		requireNoOverlap(t, bolt, p.front, 29, 31, 32.9)
		requireNoOverlap(t, nut, p.back, 23.5, 24, 24.5)
	}
}

// TestBearings checks the six 605ZZ envelopes sit in their pockets at their heights and share no volume with the plates.
func TestBearings(t *testing.T) {
	p := buildPlates(t)
	h, err := p.b.buildHardware()
	require.NoError(t, err)
	g := &p.b.g
	for _, e := range []struct {
		body *decad.Body
		at   Pt
		z    float64
		back bool
	}{
		{h.BackEscape, g.O, backBearingZ, true}, {h.BackPallet, g.P, backBearingZ, true},
		{h.BackBalance, g.B, backBearingZ, true}, {h.FrontEscape, g.O, frontBearingZ, false},
		{h.FrontPallet, g.P, frontBearingZ, false}, {h.FrontBalance, g.B, frontBearingZ, false},
	} {
		bb, err := e.body.Bounds()
		require.NoError(t, err)
		require.InDelta(t, e.z, bb.Min.Z, 2e-3)
		require.InDelta(t, e.z+5, bb.Max.Z, 2e-3)
		require.InDelta(t, e.at.X, (bb.Min.X+bb.Max.X)/2, 2e-3)
		require.InDelta(t, e.at.Y, (bb.Min.Y+bb.Max.Y)/2, 2e-3)
		require.InDelta(t, 14, bb.Max.X-bb.Min.X, 2e-3)
		plate, mid := p.front, e.z+2.5
		if e.back {
			plate = p.back
		}
		requireNoOverlap(t, e.body, plate, mid)
	}
}
