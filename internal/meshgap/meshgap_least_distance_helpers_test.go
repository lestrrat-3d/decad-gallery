package meshgap

import (
	"math"
	"strings"
	"testing"

	"github.com/lestrrat-3d/r3"
)

// bruteLeast is the reference: the least of PointDistance over the moved points. It reports the error class
// LeastPointDistance must produce: "nonfinite" for a moved point that is not finite, "overflow" when no point has a

// finite distance. A point whose own distance overflows is skipped while another point has a finite one.

func bruteLeast(m Mesh, points []r3.Vec, shift r3.Vec) (float64, string) {
	if len(points) == 0 {
		return 0, "empty"
	}
	best, found := math.Inf(1), false
	for _, v := range points {
		d, err := m.PointDistance(v.Add(shift))
		switch {
		case err == nil:
			best, found = math.Min(best, d), true
		case strings.Contains(err.Error(), "nonfinite"):
			return 0, "nonfinite"
		}
	}
	if !found {
		return 0, "overflow"
	}
	return best, ""
}

func errClass(err error) string {
	switch {
	case err == nil:
		return ""
	case strings.Contains(err.Error(), "nonfinite"):
		return "nonfinite"
	case strings.Contains(err.Error(), "no query points"):
		return "empty"
	}
	return "overflow"
}

// leastPointSets returns point sets of several sizes at the given scale, with a repeat of one point and mesh vertices.

func leastPointSets(c differentialCase, scale float64, mesh []r3.Vec) [][]r3.Vec {
	rng := newRNG(c.seed ^ 0x5bd1e995)
	random := func(n int) []r3.Vec {
		out := make([]r3.Vec, n)
		for i := range out {
			out[i] = r3.Vec{X: (rng.Float64()*2 - 1) * 3 * scale, Y: (rng.Float64()*2 - 1) * 3 * scale,
				Z: (rng.Float64()*2 - 1) * 3 * scale}
		}
		return out
	}
	one := random(1)
	return [][]r3.Vec{
		one,
		random(2),
		random(17),
		random(120),
		{one[0], one[0], one[0]},              // ties: the same point repeated
		{mesh[0], mesh[len(mesh)-1], mesh[0]}, // distance zero, repeated
		append(random(5), mesh[:min(5, len(mesh))]...), // random points and mesh vertices
	}
}

// leastMinExactScale is the smallest coordinate scale at which the result is bit-exact. Below it,
// squaredPointTriangle returns 0 for a point inside a sliver triangle (its 1e-30 denominator guard) although the
// box gap is positive, so the carried best can prune a triangle that PointDistance alone would visit.
const (
	leastMinExactScale = 1e-3
	leastMaxExactScale = 1e12 // above it, products in squaredPointTriangle overflow
)

func leastTestMesh(t *testing.T) Mesh {
	t.Helper()
	vertices := []r3.Vec{{}, {X: 1}, {Y: 1}, {Z: 1}, {X: 1, Y: 1, Z: 1}}
	m, err := New(vertices, [][3]int{{0, 1, 2}, {0, 1, 3}, {1, 2, 4}}, 0)
	if err != nil {
		t.Fatalf("build mesh: %v", err)
	}
	return m
}
