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
	return &run{ctx: t.Context(), doc: NewDocument(), p: DefaultParams(), dm: dm}
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
	for r := 21.0; r < CamRadius; r += 0.005 {
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
