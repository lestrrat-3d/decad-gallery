package meshgap

// CI: isolated-tests

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestDifferentialTreeBuildMatchesReference(t *testing.T) {
	var out mismatches
	checked := 0
	for seed := range uint64(differentialCases) {
		c := makeDifferentialCase(seed)
		a, b, ra, rb, built, why := buildPair(c)
		checked++
		if why != "" {
			out.add("%s: %s", c.desc, why)
			continue
		}
		if !built {
			continue
		}
		for _, side := range []struct {
			name string
			m    Mesh
			r    refMesh
		}{{"a", a, ra}, {"b", b, rb}} {
			checked++
			if why := treeMismatch(side.m, side.r); why != "" {
				out.add("%s mesh %s: %s", c.desc, side.name, why)
			}
		}
	}
	out.check(t, checked)
}

func TestDifferentialQueriesMatchReference(t *testing.T) {
	var out mismatches
	checked := 0
	for seed := range uint64(differentialCases) {
		c := makeDifferentialCase(seed)
		a, b, ra, rb, built, why := buildPair(c)
		if why != "" {
			out.add("%s: %s", c.desc, why)
			continue
		}
		if built {
			checked += compareQueries(c, a, b, ra, rb, &out)
		}
	}
	out.check(t, checked)
	if checked < 50000 {
		t.Fatalf("only %d comparisons ran", checked)
	}
}

func TestDifferentialLowerBoxGapMatchesReference(t *testing.T) {
	const pairs = 400000
	rng := newRNG(20261126)
	var out mismatches
	valid := 0
	for range pairs {
		a, b := randomBox(rng), randomBox(rng)
		if rng.IntN(10) == 0 {
			b = a // identical boxes touch on every axis
		}
		got, ok := lowerBoxGap(a, b)
		want, wantOK := refLowerBoxGap(refTriangle(a), refTriangle(b))
		if ok != wantOK {
			out.add("valid %v/%v for %v %v", ok, wantOK, a, b)
			continue
		}
		if !ok {
			// Callers never read the value of an invalid bound; the old code returned 0 and the new one may not.
			continue
		}
		valid++
		if math.Float64bits(got) != math.Float64bits(want) {
			out.add("bits %s/%s for %v %v", bits(got), bits(want), a, b)
		}
	}
	out.check(t, pairs)
	if valid < pairs/4 {
		t.Fatalf("only %d of %d pairs had a valid bound", valid, pairs)
	}
}

// -Inf is left out: the old helper returned it unchanged and the new one returns 0, and both feed a max with 0.
func TestDifferentialRoundDownMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 11))
	var out mismatches
	values := append([]float64{math.Inf(1)}, boxValuePool...)
	for range 200000 {
		values = append(values, randomBoxValue(rng, boxValuePool))
	}
	for _, v := range values {
		got, want := roundDownOrdered(v), refRoundDownNonnegative(v)
		if math.Float64bits(got) != math.Float64bits(want) {
			out.add("roundDown(%s): %s/%s", bits(v), bits(got), bits(want))
		}
	}
	out.check(t, len(values))
}
