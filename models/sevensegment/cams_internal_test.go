package sevensegment

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// newTestRun returns a run for the default parameters in a fresh document.
func newTestRun(t *testing.T) *run {
	t.Helper()
	dm, err := DefaultParams().Derive()
	require.NoError(t, err)
	return &run{ctx: t.Context(), doc: decad.New(), p: DefaultParams(), dm: dm}
}

// sectionAt cuts body with the plane y = y, with U = +X and V = +Z about the shaft axis, and returns its loops: the
// points are (x, z - ShaftZ).
func sectionAt(t *testing.T, body *decad.Body, y float64) [][]r3.Vec {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{Y: y, Z: ShaftZ}, r3.Vec{X: 1}, r3.Vec{Z: 1})
	require.NoError(t, err)
	var out decad.SectionOutline
	for _, tol := range []float64{0.005, 0.01, 0.02, 0.03} {
		out, err = body.SectionOutline(t.Context(), f, units.Millimeters(tol))
		if err == nil {
			break
		}
	}
	require.NoError(t, err)
	require.NotEmpty(t, out.Loops)
	local := make([][]r3.Vec, len(out.Loops))
	for i, loop := range out.Loops {
		for _, p := range loop {
			local[i] = append(local[i], r3.Vec{X: p.X, Y: p.Z - ShaftZ})
		}
	}
	return local
}

// insideLoops reports whether p lies in the material of the section: an odd number of loops contain it.
func insideLoops(loops [][]r3.Vec, p r3.Vec) bool {
	in := false
	for _, loop := range loops {
		for i, a := range loop {
			b := loop[(i+1)%len(loop)]
			if (a.Y > p.Y) != (b.Y > p.Y) && p.X < (b.X-a.X)*(p.Y-a.Y)/(b.Y-a.Y)+a.X {
				in = !in
			}
		}
	}
	return in
}

// voidRun returns the radial interval of the void between the cam's island and rim along the ray at angle psi.
func voidRun(loops [][]r3.Vec, psi float64) (float64, float64, bool) {
	lo, hi, found := 0.0, 0.0, false
	prev := true
	for r := 14.0; r < 25.0; r += 0.005 {
		p := polar(r, psi)
		cur := insideLoops(loops, p)
		if prev && !cur && !found {
			lo = r
		}
		if !prev && cur && !found {
			hi, found = r, true
		}
		prev = cur
	}
	return lo, hi, found
}

func TestSevenSegmentCam(t *testing.T) {
	for _, seg := range []Segment{SegB, SegE, SegA, SegC, SegD, SegF, SegG} {
		t.Run(seg.String(), func(t *testing.T) {
			r, body := sharedCam(t, seg)
			require.True(t, body.IsSolid())
			// Every sketch was verified fully constrained by r.profile before its geometry was used.
			require.NotEmpty(t, r.sketches)
			y := r.dm.camDiscFace(seg) + 0.95
			loops := sectionAt(t, body, y)
			for k := range 10 {
				lit, _ := Lit(k, seg)
				want := OffRadius
				if lit {
					want = OnRadius
				}
				lo, hi, ok := voidRun(loops, 90+36*float64(k))
				if k == 8 {
					// The entry channel is at the follower: the void runs out to the rim.
					require.False(t, ok)
					require.InDelta(t, OnRadius-GrooveWidth/2, lo, 0.03)
					continue
				}
				require.True(t, ok, "digit %d", k)
				require.InDelta(t, want, (lo+hi)/2, 0.03, "digit %d", k)
				require.InDelta(t, GrooveWidth, hi-lo, 0.05, "digit %d", k)
			}
			// The entry channel is open to the rim at 18 degrees.
			require.False(t, insideLoops(loops, polar(24.5, EntryAngle)))
			require.True(t, insideLoops(loops, polar(24.5, EntryAngle+20)))
			// Marks.
			n, radius := camMarks(seg)
			count := 0
			for _, loop := range loops {
				c, ext := centroidExtent(loop)
				if ext < 1.6 && math.Abs(c.Len()-radius) < 0.1 {
					count++
				}
			}
			require.Equal(t, n, count)
			// Bounds: the cam spans the groove face to the disc back and the hub.
			box, err := body.Bounds()
			require.NoError(t, err)
			require.InDelta(t, r.dm.Station(seg)+2.35-camHubTop, box.Min.Y, 1e-6)
			require.InDelta(t, r.dm.Station(seg)+2.35+CamThickness, box.Max.Y, 1e-6)
		})
	}
}

func centroidExtent(loop []r3.Vec) (r3.Vec, float64) {
	var c r3.Vec
	minX, maxX, minY, maxY := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, p := range loop {
		c = c.Add(p)
		minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
		minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
	}
	return c.Scale(1 / float64(len(loop))), math.Max(maxX-minX, maxY-minY)
}

// TestSevenSegmentCamInscription checks each cam's engraved letter: it exists at the engraving depth and not below it, it is
// different on each cam, and it stays clear of the bore, the mark holes and the rim.
func TestSevenSegmentCamInscription(t *testing.T) {
	type shape struct{ area, width, height, loops, offX, offY float64 }
	shapes := map[Segment]shape{}
	for _, seg := range Segments() {
		r, cam := sharedCam(t, seg)
		back := r.dm.camDiscFace(seg) + CamThickness
		// Inside the engraving: the letter is a void of the section, its loops lying about the letter's centre.
		letter := func(y float64) [][]r3.Vec {
			var out [][]r3.Vec
			for _, l := range sectionAt(t, cam, y) {
				c, e := centroidExtent(l)
				if e < 8 && math.Abs(c.X) < 6 && math.Abs(c.Y-inscriptionRadius) < 6 {
					out = append(out, l)
				}
			}
			return out
		}
		in := letter(back - inscriptionDepth/2)
		require.NotEmpty(t, in, "cam %s has no letter at the engraving depth", seg)
		require.Empty(t, letter(back-inscriptionDepth-0.3), "cam %s letter is deeper than the engraving", seg)
		area := 0.0
		for _, l := range in {
			a := 0.0
			for i, p := range l {
				q := l[(i+1)%len(l)]
				a += p.X*q.Y - q.X*p.Y
			}
			area += math.Abs(a) / 2
		}
		minX, maxX, minY, maxY := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
		for _, l := range in {
			for _, p := range l {
				minX, maxX, minY, maxY = math.Min(minX, p.X), math.Max(maxX, p.X), math.Min(minY, p.Y), math.Max(maxY, p.Y)
			}
		}
		// The centroid of the outline's points against the middle of its box tells a letter from its mirror image (b and d).
		var mean r3.Vec
		count := 0.0
		for _, l := range in {
			for _, p := range l {
				mean = mean.Add(p)
				count++
			}
		}
		mean = mean.Scale(1 / count)
		shapes[seg] = shape{area, maxX - minX, maxY - minY, float64(len(in)), mean.X - (minX+maxX)/2,
			mean.Y - (minY+maxY)/2}
		// Clearances: from the axis, from the mark holes and from the rim.
		n, radius := camMarks(seg)
		for _, l := range in {
			for _, p := range l {
				require.Greater(t, p.Len(), 3.25+1.6, "cam %s letter is too near the bore", seg)
				require.Less(t, p.Len(), CamRadius-1.5, "cam %s letter is too near the rim", seg)
				for i := range n {
					require.Greater(t, p.Sub(polar(radius, camMarkFirst+camMarkStep*float64(i))).Len(), camMarkRadius+1.0,
						"cam %s letter is too near a mark hole", seg)
				}
			}
		}
	}
	// Distinct: no two letters share the same area, extent and loop count.
	for _, a := range Segments() {
		for _, b := range Segments() {
			if a >= b {
				continue
			}
			x, y := shapes[a], shapes[b]
			diff := math.Max(math.Max(math.Max(math.Abs(x.area-y.area)/3, math.Abs(x.width-y.width)), math.Max(math.Abs(x.height-y.height),
				math.Abs(x.loops-y.loops))), math.Max(math.Abs(x.offX-y.offX), math.Abs(x.offY-y.offY)))
			require.Greater(t, diff, 0.05, "letters %s and %s look alike", a, b)
		}
	}
}
