package meshgap

// CI: isolated-tests

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

func TestLeastPointDistanceMatchesBruteForce(t *testing.T) {
	var out mismatches
	checked := 0
	for seed := range uint64(differentialCases) {
		c := makeDifferentialCase(seed)
		m, err := New(c.va, c.fa, c.boundA)
		if err != nil {
			continue
		}
		scale := pickScale(newRNG(seed))
		if scale < leastMinExactScale || scale > leastMaxExactScale {
			continue
		}
		shifts := []r3.Vec{{}, {X: scale}, {X: -scale, Y: 2 * scale, Z: 0.5 * scale}}
		for si, points := range leastPointSets(c, scale, c.va) {
			for _, shift := range shifts {
				want, wantClass := bruteLeast(m, points, shift)
				got, err := m.LeastPointDistance(points, shift)
				checked++
				if errClass(err) != wantClass {
					out.add("%s: set %d: error class %q, want %q", c.desc, si, errClass(err), wantClass)
					continue
				}
				if wantClass == "" && math.Float64bits(got) != math.Float64bits(want) {
					out.add("%s: set %d: got %s, want %s", c.desc, si, bits(got), bits(want))
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no comparison ran")
	}
	out.check(t, checked)
}

func leastTestMesh(t *testing.T) Mesh {
	t.Helper()
	vertices := []r3.Vec{{}, {X: 1}, {Y: 1}, {Z: 1}, {X: 1, Y: 1, Z: 1}}
	m, err := New(vertices, [][3]int{{0, 1, 2}, {0, 1, 3}, {1, 2, 4}}, 0)
	if err != nil {
		t.Fatalf("build mesh: %v", err)
	}
	return m
}

func TestLeastPointDistanceEmptyPoints(t *testing.T) {
	m := leastTestMesh(t)
	for _, points := range [][]r3.Vec{nil, {}} {
		if d, err := m.LeastPointDistance(points, r3.Vec{}); err == nil {
			t.Fatalf("empty point list returned %g and no error", d)
		}
	}
}

func TestLeastPointDistanceEmptyMesh(t *testing.T) {
	if d, err := (Mesh{}).LeastPointDistance([]r3.Vec{{}}, r3.Vec{}); err == nil {
		t.Fatalf("empty mesh returned %g and no error", d)
	}
}

func TestLeastPointDistanceSinglePoint(t *testing.T) {
	m := leastTestMesh(t)
	for _, p := range []r3.Vec{{}, {X: 0.25, Y: 0.25}, {X: 3, Y: -2, Z: 5}} {
		shift := r3.Vec{X: 0.5, Z: -1}
		want, err := m.PointDistance(p.Add(shift))
		if err != nil {
			t.Fatalf("PointDistance: %v", err)
		}
		got, err := m.LeastPointDistance([]r3.Vec{p}, shift)
		if err != nil {
			t.Fatalf("LeastPointDistance: %v", err)
		}
		if got != want {
			t.Fatalf("point %v: got %g, want %g", p, got, want)
		}
	}
}

func TestLeastPointDistanceTies(t *testing.T) {
	m := leastTestMesh(t)
	// Two points at the same distance from the mesh, one on each side of the first triangle.
	points := []r3.Vec{{X: 0.25, Y: 0.25, Z: 2}, {X: 0.25, Y: 0.25, Z: 2}, {X: 0.25, Y: 0.25, Z: 2}}
	want, err := m.PointDistance(points[0])
	if err != nil {
		t.Fatalf("PointDistance: %v", err)
	}
	got, err := m.LeastPointDistance(points, r3.Vec{})
	if err != nil || got != want {
		t.Fatalf("repeated point: got %g (err %v), want %g", got, err, want)
	}
}

func TestLeastPointDistanceNonfinitePoint(t *testing.T) {
	m := leastTestMesh(t)
	bad := []r3.Vec{{X: math.NaN()}, {Y: math.Inf(1)}, {Z: math.Inf(-1)}}
	for _, b := range bad {
		for _, points := range [][]r3.Vec{{b}, {{}, b}, {b, {}}} {
			if d, err := m.LeastPointDistance(points, r3.Vec{}); err == nil {
				t.Fatalf("nonfinite point %v returned %g and no error", b, d)
			}
		}
	}
	// A finite point whose shifted position overflows is also rejected.
	if _, err := m.LeastPointDistance([]r3.Vec{{X: math.MaxFloat64}}, r3.Vec{X: math.MaxFloat64}); err == nil {
		t.Fatal("overflowing shifted point returned no error")
	}
}

func TestLeastPointDistanceKnownValue(t *testing.T) {
	m := leastTestMesh(t)
	got, err := m.LeastPointDistance([]r3.Vec{{X: 0.25, Y: 0.25, Z: 3}, {X: 0.25, Y: 0.25, Z: -2}}, r3.Vec{})
	if err != nil {
		t.Fatalf("LeastPointDistance: %v", err)
	}
	if got != 2 {
		t.Fatalf("got %g, want 2", got)
	}
}

// TestLeastPointDistanceExtremeScales checks only that the result is a finite non-negative distance or an error
// outside the bit-exact range, with failure output limited to a count.
func TestLeastPointDistanceExtremeScales(t *testing.T) {
	bad, checked := 0, 0
	for seed := range uint64(differentialCases) {
		c := makeDifferentialCase(seed)
		scale := pickScale(newRNG(seed))
		if scale >= leastMinExactScale && scale <= leastMaxExactScale {
			continue
		}
		m, err := New(c.va, c.fa, c.boundA)
		if err != nil {
			continue
		}
		for _, points := range leastPointSets(c, scale, c.va) {
			d, err := m.LeastPointDistance(points, r3.Vec{})
			checked++
			if err == nil && (d < 0 || math.IsNaN(d) || math.IsInf(d, 0)) {
				bad++
			}
		}
	}
	if checked == 0 || bad != 0 {
		t.Fatalf("%d of %d extreme-scale results are not finite non-negative distances", bad, checked)
	}
}
