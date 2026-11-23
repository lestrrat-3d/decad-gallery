package sevensegment

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// Section heights that the tests read: the middle of the guide plate, and a level in the tubes between their bottom and the
// top plate (z 56 and 81 at Scale 1), and the extent that tells a cam's bore from its other loops.
const (
	guideMidZ    = guideFrom + guideThick/2
	tubeSectionZ = tubeFrom + Scale*3.6
	boreExtent   = 2*camBoreRadius + 1
)

func polar2(x, y float64) r3.Vec { return r3.Vec{X: x, Y: y} }

// planSection cuts body with the horizontal plane z and returns its loops as (x, y).
func planSection(t *testing.T, body *decad.Body, z float64) [][]r3.Vec {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	var out decad.SectionOutline
	for _, tol := range []float64{0.01, 0.02, 0.03} {
		if out, err = body.SectionOutline(t.Context(), f, units.Millimeters(tol)); err == nil {
			break
		}
	}
	require.NoError(t, err)
	loops := make([][]r3.Vec, len(out.Loops))
	for i, l := range out.Loops {
		for _, p := range l {
			loops[i] = append(loops[i], r3.Vec{X: p.X, Y: p.Y})
		}
	}
	return loops
}

// minWall returns the least distance between two different loops of the section of body at height z, over loops whose
// boxes lie within limit of each other (the wall between two holes).
func minWall(t *testing.T, body *decad.Body, z, limit float64) float64 {
	t.Helper()
	o, err := cornercheck.Outline(t.Context(), body, r3.Vec{Z: 1}, z)
	require.NoError(t, err, "section at z %.2f", z)
	best := math.Inf(1)
	box := func(l []r3.Vec) (lo, hi r3.Vec) {
		lo, hi = l[0], l[0]
		for _, p := range l {
			lo = r3.Vec{X: math.Min(lo.X, p.X), Y: math.Min(lo.Y, p.Y)}
			hi = r3.Vec{X: math.Max(hi.X, p.X), Y: math.Max(hi.Y, p.Y)}
		}
		return lo, hi
	}
	for i := range o.Loops {
		lo1, hi1 := box(o.Loops[i])
		for j := i + 1; j < len(o.Loops); j++ {
			lo2, hi2 := box(o.Loops[j])
			if lo1.X-limit > hi2.X || lo2.X-limit > hi1.X || lo1.Y-limit > hi2.Y || lo2.Y-limit > hi1.Y {
				continue
			}
			for _, p := range o.Loops[i] {
				best = math.Min(best, nearest(r3.Vec{X: p.X, Y: p.Y}, flatten(o.Loops[j])))
			}
			for _, p := range o.Loops[j] {
				best = math.Min(best, nearest(r3.Vec{X: p.X, Y: p.Y}, flatten(o.Loops[i])))
			}
		}
	}
	return best
}

func flatten(l []r3.Vec) [][]r3.Vec {
	out := make([]r3.Vec, len(l))
	for i, p := range l {
		out[i] = r3.Vec{X: p.X, Y: p.Y}
	}
	return [][]r3.Vec{out}
}

// planeSection cuts body with the plane through origin spanned by u and v and returns its loops as (u, v) pairs.
func planeSection(t *testing.T, body *decad.Body, origin, u, v r3.Vec) [][]r3.Vec {
	t.Helper()
	f, err := r3.NewFrame(origin, u, v)
	require.NoError(t, err)
	var out decad.SectionOutline
	for _, tol := range []float64{0.01, 0.02, 0.03, 0.05} {
		if out, err = body.SectionOutline(t.Context(), f, units.Millimeters(tol)); err == nil {
			break
		}
	}
	require.NoError(t, err)
	loops := make([][]r3.Vec, len(out.Loops))
	for i, l := range out.Loops {
		for _, p := range l {
			d := p.Sub(origin)
			loops[i] = append(loops[i], r3.Vec{X: d.Dot(u), Y: d.Dot(v)})
		}
	}
	return loops
}
