package meshgap

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/lestrrat-3d/r3"
)

// mismatches counts failures and keeps a few samples, so a broken change prints bounded output.
type mismatches struct {
	count   int
	samples []string
}

const maxMismatchSamples = 4

func (m *mismatches) add(format string, args ...any) {
	m.count++
	if len(m.samples) < maxMismatchSamples {
		m.samples = append(m.samples, fmt.Sprintf(format, args...))
	}
}

func (m *mismatches) check(t *testing.T, checked int) {
	t.Helper()
	if m.count != 0 {
		t.Fatalf("%d of %d comparisons differ from the reference; first %d:\n%s",
			m.count, checked, len(m.samples), strings.Join(m.samples, "\n"))
	}
}

func bits(x float64) string { return fmt.Sprintf("%#016x", math.Float64bits(x)) }

func newRNG(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)) }

// coordinateScales spans subnormal products, tiny, ordinary, near-limit, ineligible and overflowing magnitudes.
var coordinateScales = []float64{5e-324, 1e-310, 1e-160, 1e-9, 1, 1, 1, 10, 1e3, 6e4, 1e5, 1e12, 1e100, 1e154, 1e200}

// pickScale returns the scale for one mesh pair.
func pickScale(rng *rand.Rand) float64 { return coordinateScales[rng.IntN(len(coordinateScales))] }

type soupKind int

const (
	soupRandom     soupKind = iota // triangles of random size at random places
	soupSnapped                    // vertices on a coarse grid: coincident vertices and tied centres
	soupDegenerate                 // points, collinear triangles, repeated vertices
	soupTied                       // centres tied on the split axis, other axes random
	soupZeros                      // coordinates drawn from {-0, +0, +-scale, +-subnormals}
	soupKindCount
)

func (k soupKind) String() string {
	return [...]string{"random", "snapped", "degenerate", "tied", "zeros"}[k]
}

// randomSoup returns a triangle soup of n triangles with vertices around the origin at the given scale.
func randomSoup(rng *rand.Rand, n int, kind soupKind, scale float64) ([]r3.Vec, [][3]int) {
	vertices := make([]r3.Vec, 0, 3*n)
	faces := make([][3]int, 0, n)
	point := func() r3.Vec {
		return r3.Vec{X: (rng.Float64()*2 - 1) * scale, Y: (rng.Float64()*2 - 1) * scale, Z: (rng.Float64()*2 - 1) * scale}
	}
	snap := func(x float64) float64 { return math.Round(x/scale*2) / 2 * scale }
	zeros := []float64{0, math.Copysign(0, -1), scale, -scale, math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64, scale / 2}
	for range n {
		var tri [3]r3.Vec
		switch kind {
		case soupRandom:
			base := point()
			size := []float64{0, 1e-3, 0.1, 1}[rng.IntN(4)] * scale
			for j := range tri {
				d := point()
				tri[j] = r3.Vec{X: base.X + d.X*size/scale, Y: base.Y + d.Y*size/scale, Z: base.Z + d.Z*size/scale}
			}
		case soupSnapped:
			for j := range tri {
				p := point()
				tri[j] = r3.Vec{X: snap(p.X), Y: snap(p.Y), Z: snap(p.Z)}
			}
		case soupDegenerate:
			base := point()
			tri = [3]r3.Vec{base, base, base}
			switch rng.IntN(3) {
			case 1:
				tri[1] = r3.Vec{X: base.X + scale, Y: base.Y, Z: base.Z}
				tri[2] = r3.Vec{X: base.X + 2*scale, Y: base.Y, Z: base.Z}
			case 2:
				tri[2] = point()
			}
		case soupTied:
			centre := float64(rng.IntN(3)) * scale
			for j := range tri {
				r := float64(rng.IntN(4)+1) * scale / 4
				sign := 1.0
				if j == 1 {
					sign = -1
				}
				p := point()
				tri[j] = r3.Vec{X: centre + sign*r, Y: p.Y, Z: p.Z}
			}
		default:
			for j := range tri {
				tri[j] = r3.Vec{X: zeros[rng.IntN(len(zeros))], Y: zeros[rng.IntN(len(zeros))], Z: zeros[rng.IntN(len(zeros))]}
			}
		}
		faces = append(faces, [3]int{len(vertices), len(vertices) + 1, len(vertices) + 2})
		vertices = append(vertices, tri[:]...)
	}
	return vertices, faces
}

// differentialCase is one random pair of meshes with the queries to run on it.
type differentialCase struct {
	seed           uint64
	desc           string
	va, vb         []r3.Vec
	fa, fb         [][3]int
	boundA, boundB float64
	clearances     []float64
	points         []r3.Vec
}

// differentialCases is the number of random mesh pairs each differential test runs.
const differentialCases = 1500

var meshSizes = []int{1, 2, 3, 8, 9, 10, 17, 33, 64, 150, 400}

// makeDifferentialCase derives every choice from the seed.
func makeDifferentialCase(seed uint64) differentialCase {
	rng := newRNG(seed)
	scale := pickScale(rng)
	kindA, kindB := soupKind(rng.IntN(int(soupKindCount))), soupKind(rng.IntN(int(soupKindCount)))
	na, nb := meshSizes[rng.IntN(len(meshSizes))], meshSizes[rng.IntN(len(meshSizes))]
	va, fa := randomSoup(rng, na, kindA, scale)
	var vb []r3.Vec
	var fb [][3]int
	layout := rng.IntN(5)
	switch layout {
	case 0: // coincident: the same soup
		vb, fb = va, fa
		nb = na
	default:
		vb, fb = randomSoup(rng, nb, kindB, scale)
	}
	// Move the second soup: far apart, touching the first's bounds, a hair apart, or overlapping.
	lo, hi := va[0], va[0]
	for _, p := range va {
		lo, hi = extend(lo, hi, p)
	}
	shift := r3.Vec{}
	switch layout {
	case 1:
		shift = r3.Vec{X: 100 * scale, Y: 3 * scale}
	case 2: // touching: the second soup starts where the first ends
		shift = r3.Vec{X: hi.X - lo.X, Y: 0, Z: 0}
	case 3: // a hair apart along every axis
		eps := scale * []float64{1e-12, 1e-6, 1e-3}[rng.IntN(3)]
		shift = r3.Vec{X: hi.X - lo.X + eps, Y: hi.Y - lo.Y + eps, Z: hi.Z - lo.Z + eps}
	}
	if shift != (r3.Vec{}) {
		moved := make([]r3.Vec, len(vb))
		for i, p := range vb {
			moved[i] = r3.Vec{X: p.X + shift.X, Y: p.Y + shift.Y, Z: p.Z + shift.Z}
		}
		vb = moved
	}
	bounds := []float64{0, 0, 1e-9, 0.01, 1, 70000, scale / 10}
	c := differentialCase{
		seed: seed, va: va, vb: vb, fa: fa, fb: fb,
		boundA: bounds[rng.IntN(len(bounds))], boundB: bounds[rng.IntN(len(bounds))],
		clearances: []float64{0, 1e-6, scale * 1e-3, scale, 1, 65536, 100000, scale * 10},
	}
	for range 3 {
		c.points = append(c.points, r3.Vec{X: (rng.Float64()*2 - 1) * scale, Y: (rng.Float64()*2 - 1) * scale,
			Z: (rng.Float64()*2 - 1) * scale})
	}
	c.points = append(c.points, va[0], vb[len(vb)-1], r3.Vec{})
	c.desc = fmt.Sprintf("seed %d scale %g kinds %s/%s sizes %d/%d layout %d bounds %g/%g",
		seed, scale, kindA, kindB, na, nb, layout, c.boundA, c.boundB)
	return c
}

// treeMismatch compares the triangle order and node table of a mesh with the reference, bit for bit.
func treeMismatch(m Mesh, r refMesh) string {
	if len(m.triangles) != len(r.triangles) || len(m.nodes) != len(r.nodes) {
		return fmt.Sprintf("sizes %d/%d triangles %d/%d nodes", len(m.triangles), len(r.triangles), len(m.nodes), len(r.nodes))
	}
	vecBits := func(v r3.Vec) [3]uint64 {
		return [3]uint64{math.Float64bits(v.X), math.Float64bits(v.Y), math.Float64bits(v.Z)}
	}
	for i := range m.triangles {
		a, b := m.triangles[i], r.triangles[i]
		if vecBits(a.p[0]) != vecBits(b.p[0]) || vecBits(a.p[1]) != vecBits(b.p[1]) || vecBits(a.p[2]) != vecBits(b.p[2]) ||
			vecBits(a.lo) != vecBits(b.lo) || vecBits(a.hi) != vecBits(b.hi) {
			return fmt.Sprintf("triangle %d differs", i)
		}
	}
	for i := range m.nodes {
		a, b := m.nodes[i], r.nodes[i]
		if vecBits(a.lo) != vecBits(b.lo) || vecBits(a.hi) != vecBits(b.hi) ||
			a.start != b.start || a.end != b.end || a.left != b.left || a.right != b.right {
			return fmt.Sprintf("node %d differs", i)
		}
	}
	if vecBits(m.lo) != vecBits(r.lo) || vecBits(m.hi) != vecBits(r.hi) || m.clearanceEligible != r.clearanceEligible ||
		math.Float64bits(m.bound) != math.Float64bits(r.bound) {
		return "mesh bounds differ"
	}
	return ""
}

// errText returns the error message, or the empty string for nil.
func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// compareQueries runs every query in both argument orders on the new and the reference meshes and reports each
// difference to out. It returns the number of comparisons made.
func compareQueries(c differentialCase, a, b Mesh, ra, rb refMesh, out *mismatches) int {
	checked := 0
	orders := []struct {
		name string
		x, y Mesh
		rx   refMesh
		ry   refMesh
	}{{"a,b", a, b, ra, rb}, {"b,a", b, a, rb, ra}}
	for _, o := range orders {
		for _, clearance := range c.clearances {
			gap, sep, err := o.x.Exceeds(o.y, clearance)
			rgap, rsep, rerr := o.rx.Exceeds(o.ry, clearance)
			checked++
			if math.Float64bits(gap) != math.Float64bits(rgap) || sep != rsep || errText(err) != errText(rerr) {
				out.add("%s Exceeds(%s, %g): gap %s/%s sep %v/%v err %q/%q", c.desc, o.name, clearance,
					bits(gap), bits(rgap), sep, rsep, errText(err), errText(rerr))
			}
			ok, err := o.x.ExceedsClearance(o.y, clearance)
			rok, rerr := o.rx.ExceedsClearance(o.ry, clearance)
			checked++
			if ok != rok || errText(err) != errText(rerr) {
				out.add("%s ExceedsClearance(%s, %g): %v/%v err %q/%q", c.desc, o.name, clearance, ok, rok,
					errText(err), errText(rerr))
			}
			if rerr != nil || !finite(rgap) {
				continue
			}
			// Clearances at the reference gap, one ulp each side, decide exactly at the rounding edges.
			for _, edge := range []float64{rgap, math.Nextafter(rgap, 0), math.Nextafter(rgap, math.Inf(1))} {
				ok, err := o.x.ExceedsClearance(o.y, edge)
				rok, rerr := o.rx.ExceedsClearance(o.ry, edge)
				checked++
				if ok != rok || errText(err) != errText(rerr) {
					out.add("%s ExceedsClearance(%s, edge %s): %v/%v", c.desc, o.name, bits(edge), ok, rok)
				}
			}
		}
		for _, p := range c.points {
			d, err := o.x.PointDistance(p)
			rd, rerr := o.rx.PointDistance(p)
			checked++
			if math.Float64bits(d) != math.Float64bits(rd) || errText(err) != errText(rerr) {
				out.add("%s PointDistance(%s, %v): %s/%s", c.desc, o.name, p, bits(d), bits(rd))
			}
		}
	}
	return checked
}

// randomBoxValue draws one box coordinate, favouring values that expose rounding edges.
func randomBoxValue(rng *rand.Rand, pool []float64) float64 {
	switch rng.IntN(6) {
	case 0:
		return pool[rng.IntN(len(pool))]
	case 1:
		v := math.Nextafter(pool[rng.IntN(len(pool))], []float64{math.Inf(-1), math.Inf(1)}[rng.IntN(2)])
		if !finite(v) { // mesh coordinates are always finite
			return 0
		}
		return v
	case 2:
		return math.Round(rng.NormFloat64()*4) / 2
	default:
		scale := coordinateScales[rng.IntN(len(coordinateScales))]
		if rng.IntN(8) == 0 {
			scale = math.MaxFloat64 / 4
		}
		return (rng.Float64()*2 - 1) * scale
	}
}

var boxValuePool = []float64{
	0, math.Copysign(0, -1), 1, -1, 0.1, -0.1, 1e-3, math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64,
	2.2250738585072014e-308, 1e154, -1e154, 1.5e154, 1 << 16, -(1 << 16), math.MaxFloat64, -math.MaxFloat64,
	math.MaxFloat64 / 2, 1e-160,
}

// randomBox returns an ordered box (lo <= hi on every axis).
func randomBox(rng *rand.Rand) triangle {
	var lo, hi [3]float64
	for i := range lo {
		x, y := randomBoxValue(rng, boxValuePool), randomBoxValue(rng, boxValuePool)
		lo[i], hi[i] = min(x, y), max(x, y)
	}
	return triangle{lo: r3.Vec{X: lo[0], Y: lo[1], Z: lo[2]}, hi: r3.Vec{X: hi[0], Y: hi[1], Z: hi[2]}}
}

// buildPair builds both meshes of a case with the production code and with the reference. It returns a description
// of a construction difference, or the empty string.
func buildPair(c differentialCase) (a, b Mesh, ra, rb refMesh, built bool, why string) {
	a, aerr := New(c.va, c.fa, c.boundA)
	ra, raerr := refNew(c.va, c.fa, c.boundA)
	b, berr := New(c.vb, c.fb, c.boundB)
	rb, rberr := refNew(c.vb, c.fb, c.boundB)
	if errText(aerr) != errText(raerr) || errText(berr) != errText(rberr) {
		return a, b, ra, rb, false, fmt.Sprintf("construction errors %q/%q %q/%q",
			errText(aerr), errText(raerr), errText(berr), errText(rberr))
	}
	return a, b, ra, rb, aerr == nil && berr == nil, ""
}
