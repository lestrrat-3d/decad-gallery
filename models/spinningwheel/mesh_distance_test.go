package spinningwheel_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// These tests use the real bearing producer and verified meshes, including a hollow and a contained pair.
func TestSpinningWheelMeshSeparation(t *testing.T) {
	makeShape := func(inner, outer, width, z float64) collisionShape {
		f, err := r3.NewFrame(r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
		require.NoError(t, err)
		m, err := ballbearing.Build(t.Context(), decad.New(), ballbearing.Params{BoreDiameter: inner, OuterDiameter: outer, Width: width, Placement: &f})
		require.NoError(t, err)
		mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		return nativeCollisionShape(m.Body, surfaceTriangles{vertices: mesh.Vertices(), triangles: mesh.Triangles(), bound: mesh.Bound().Base()})
	}
	ring := makeShape(8, 22, 7, 0)
	for _, c := range []struct {
		name                   string
		inner, outer, width, z float64
		clear                  bool
	}{
		{"inside_open_bore", 2, 4, 1, 3, true},
		{"contained_in_material", 10, 12, 1, 3, false},
		{"crossing_wall", 4, 12, 1, 3, false},
		{"disjoint_axially", 8, 22, 1, 10, true},
		{"touching_caps", 8, 22, 1, 7, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, clear := ring.meshSeparation(makeShape(c.inner, c.outer, c.width, c.z))
			require.True(t, clear == c.clear, "clear=%t, want %t", clear, c.clear)
		})
	}
}

type meshTriangle struct {
	p      [3]r3.Vec
	lo, hi r3.Vec
}

func trianglesOf(s collisionShape) []meshTriangle {
	r := make([]meshTriangle, 0, len(s.triangles))
	for _, face := range s.triangles {
		t := meshTriangle{p: [3]r3.Vec{s.points[face[0]], s.points[face[1]], s.points[face[2]]}}
		t.lo, t.hi = t.p[0], t.p[0]
		for _, p := range t.p[1:] {
			t.lo = r3.Vec{X: math.Min(t.lo.X, p.X), Y: math.Min(t.lo.Y, p.Y), Z: math.Min(t.lo.Z, p.Z)}
			t.hi = r3.Vec{X: math.Max(t.hi.X, p.X), Y: math.Max(t.hi.Y, p.Y), Z: math.Max(t.hi.Z, p.Z)}
		}
		r = append(r, t)
	}
	return r
}

func squaredBoxGap(a, b meshTriangle) float64 {
	x := math.Max(0, math.Max(a.lo.X-b.hi.X, b.lo.X-a.hi.X))
	y := math.Max(0, math.Max(a.lo.Y-b.hi.Y, b.lo.Y-a.hi.Y))
	z := math.Max(0, math.Max(a.lo.Z-b.hi.Z, b.lo.Z-a.hi.Z))
	return x*x + y*y + z*z
}

// Boundary separation alone does not exclude containment. Test vertices of every possible component as well.
func (s collisionShape) meshSeparation(other collisionShape) (float64, bool) {
	if len(s.triangles) == 0 || len(other.triangles) == 0 {
		return 0, false
	}
	budget := s.bound + other.bound + 1e-6
	best := math.Inf(1)
	a, b := trianglesOf(s), trianglesOf(other)
	for _, ta := range a {
		for _, tb := range b {
			if squaredBoxGap(ta, tb) >= best {
				continue
			}
			best = math.Min(best, squaredTriangleGap(ta.p, tb.p))
			if best <= budget*budget {
				return math.Sqrt(best) - budget, false
			}
		}
	}
	for _, p := range s.points {
		if meshContains(other, p) {
			return 0, false
		}
	}
	for _, p := range other.points {
		if meshContains(s, p) {
			return 0, false
		}
	}
	return math.Sqrt(best) - budget, true
}

func meshContains(s collisionShape, p r3.Vec) bool {
	angle := 0.0
	for _, f := range s.triangles {
		a, b, c := s.points[f[0]].Sub(p), s.points[f[1]].Sub(p), s.points[f[2]].Sub(p)
		la, lb, lc := a.Len(), b.Len(), c.Len()
		denom := la*lb*lc + a.Dot(b)*lc + b.Dot(c)*la + c.Dot(a)*lb
		angle += 2 * math.Atan2(a.Dot(b.Cross(c)), denom)
	}
	return math.Abs(angle) > 2*math.Pi
}

func squaredTriangleGap(a, b [3]r3.Vec) float64 {
	best := math.Inf(1)
	for i := range 3 {
		if segmentPierces(a[i], a[(i+1)%3], b) || segmentPierces(b[i], b[(i+1)%3], a) {
			return 0
		}
		best = math.Min(best, squaredPointTriangle(a[i], b))
		best = math.Min(best, squaredPointTriangle(b[i], a))
		for j := range 3 {
			best = math.Min(best, squaredSegments(a[i], a[(i+1)%3], b[j], b[(j+1)%3]))
		}
	}
	return best
}

func segmentPierces(p, q r3.Vec, t [3]r3.Vec) bool {
	n := t[1].Sub(t[0]).Cross(t[2].Sub(t[0]))
	a, b := p.Sub(t[0]).Dot(n), q.Sub(t[0]).Dot(n)
	if a*b > 0 || math.Abs(a-b) < 1e-20 {
		return false
	}
	u := a / (a - b)
	if u < 0 || u > 1 {
		return false
	}
	return squaredPointTriangle(p.Add(q.Sub(p).Scale(u)), t) < 1e-16
}

func squaredPointTriangle(p r3.Vec, t [3]r3.Vec) float64 {
	a, b, c := t[0], t[1], t[2]
	ab, ac, ap := b.Sub(a), c.Sub(a), p.Sub(a)
	d1, d2 := ab.Dot(ap), ac.Dot(ap)
	if d1 <= 0 && d2 <= 0 {
		return ap.Dot(ap)
	}
	bp := p.Sub(b)
	d3, d4 := ab.Dot(bp), ac.Dot(bp)
	if d3 >= 0 && d4 <= d3 {
		return bp.Dot(bp)
	}
	vc := d1*d4 - d3*d2
	if vc <= 0 && d1 >= 0 && d3 <= 0 {
		d := p.Sub(a.Add(ab.Scale(d1 / (d1 - d3))))
		return d.Dot(d)
	}
	cp := p.Sub(c)
	d5, d6 := ab.Dot(cp), ac.Dot(cp)
	if d6 >= 0 && d5 <= d6 {
		return cp.Dot(cp)
	}
	vb := d5*d2 - d1*d6
	if vb <= 0 && d2 >= 0 && d6 <= 0 {
		d := p.Sub(a.Add(ac.Scale(d2 / (d2 - d6))))
		return d.Dot(d)
	}
	va := d3*d6 - d5*d4
	if va <= 0 && (d4-d3) >= 0 && (d5-d6) >= 0 {
		w := (d4 - d3) / ((d4 - d3) + (d5 - d6))
		d := p.Sub(b.Add(c.Sub(b).Scale(w)))
		return d.Dot(d)
	}
	denom := va + vb + vc
	if math.Abs(denom) < 1e-30 {
		return 0
	}
	d := p.Sub(a.Add(ab.Scale(vb / denom)).Add(ac.Scale(vc / denom)))
	return d.Dot(d)
}

func squaredSegments(p1, q1, p2, q2 r3.Vec) float64 {
	d1, d2, r := q1.Sub(p1), q2.Sub(p2), p1.Sub(p2)
	a, e, f := d1.Dot(d1), d2.Dot(d2), d2.Dot(r)
	s, u := 0.0, 0.0
	if a <= 1e-20 && e <= 1e-20 {
		return r.Dot(r)
	}
	if a <= 1e-20 {
		u = math.Max(0, math.Min(1, f/e))
	} else {
		c := d1.Dot(r)
		if e <= 1e-20 {
			s = math.Max(0, math.Min(1, -c/a))
		} else {
			b := d1.Dot(d2)
			denom := a*e - b*b
			if denom != 0 {
				s = math.Max(0, math.Min(1, (b*f-c*e)/denom))
			}
			u = (b*s + f) / e
			if u < 0 {
				u = 0
				s = math.Max(0, math.Min(1, -c/a))
			} else if u > 1 {
				u = 1
				s = math.Max(0, math.Min(1, (b-c)/a))
			}
		}
	}
	d := p1.Add(d1.Scale(s)).Sub(p2.Add(d2.Scale(u)))
	return d.Dot(d)
}
