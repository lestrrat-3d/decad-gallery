package sevensegment_test

// CI: isolated-tests

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestYellowFirstCountMatchesReference checks that yellowFirstCount counts the rays the full-grid reference counts, on
// random, sliver, axis-aligned and coincident triangles of all three kinds.
func TestYellowFirstCountMatchesReference(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(7, 11))
	pt := func(scale float64) r3.Vec {
		return r3.Vec{X: (rng.Float64() - 0.5) * scale, Y: (rng.Float64() - 0.5) * scale, Z: (rng.Float64() - 0.5) * scale}
	}
	mk := func(a, b, c r3.Vec, kind int) rayTri {
		return rayTri{a: a, e1: b.Sub(a), e2: c.Sub(a),
			lo:   r3.Vec{X: min(a.X, b.X, c.X), Y: min(a.Y, b.Y, c.Y), Z: min(a.Z, b.Z, c.Z)},
			hi:   r3.Vec{X: max(a.X, b.X, c.X), Y: max(a.Y, b.Y, c.Y), Z: max(a.Z, b.Z, c.Z)},
			kind: kind}
	}
	var tris []rayTri
	for i := range 600 {
		kind := i % 3
		a := pt(12)
		switch i % 4 {
		case 0: // random
			tris = append(tris, mk(a, a.Add(pt(3)), a.Add(pt(3)), kind))
		case 1: // sliver
			e := pt(4)
			tris = append(tris, mk(a, a.Add(e), a.Add(e.Scale(0.5)).Add(pt(0.001)), kind))
		case 2: // axis aligned
			tris = append(tris, mk(a, a.Add(r3.Vec{X: 2}), a.Add(r3.Vec{Y: 2}), kind))
		default: // coincident copies of the triangle before, of another kind
			prev := tris[len(tris)-1]
			tris = append(tris, mk(prev.a, prev.a.Add(prev.e1), prev.a.Add(prev.e2), (kind+1)%3))
		}
	}
	tris = indexTris(tris)
	hits := 0
	for range 20 {
		d := r3.Vec{X: rng.Float64() - 0.5, Y: rng.Float64() - 0.5, Z: -0.2 - rng.Float64()}
		d = d.Scale(1 / math.Sqrt(d.Dot(d)))
		origins := make([]r3.Vec, 400)
		for i := range origins {
			origins[i] = r3.Vec{X: (rng.Float64() - 0.5) * 12, Y: (rng.Float64() - 0.5) * 12}.Sub(d.Scale(8))
		}
		want := yellowFirstCountReference(tris, d, origins)
		hits += want
		require.Equal(t, want, yellowFirstCount(tris, d, origins))
	}
	require.Positive(t, hits, "the input must make some rays end on yellow")
}
