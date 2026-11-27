package meshgap

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/r3"
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
