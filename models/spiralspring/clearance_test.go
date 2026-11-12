package spiralspring_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spiralspring"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// meshTolerance is the chord tolerance, in mm, the body is meshed at. A chord sags into air on one wall of each gap
// or strip only, so a slice understates a gap or wall by at most this much; the extra 0.001 mm absorbs rounding.
const meshTolerance = 0.02

const sliceMargin = meshTolerance + 0.001

type vec2 [2]float64

func (a vec2) sub(b vec2) vec2       { return vec2{a[0] - b[0], a[1] - b[1]} }
func (a vec2) dot(b vec2) float64    { return a[0]*b[0] + a[1]*b[1] }
func (a vec2) len() float64          { return math.Hypot(a[0], a[1]) }
func (a vec2) scale(s float64) vec2  { return vec2{a[0] * s, a[1] * s} }
func (a vec2) add(b vec2) vec2       { return vec2{a[0] + b[0], a[1] + b[1]} }
func (a vec2) cross(b vec2) float64  { return a[0]*b[1] - a[1]*b[0] }
func (a vec2) mid(b vec2) vec2       { return a.add(b).scale(.5) }
func (a vec2) distTo(b vec2) float64 { return a.sub(b).len() }

// segment is one piece of the slice's outline. n is the unit normal, pointing out of the material.
type segment struct{ a, b, n vec2 }

func (s segment) midpoint() vec2 { return s.a.mid(s.b) }

func pointSegmentDistance(p vec2, s segment) float64 {
	d := s.b.sub(s.a)
	l2 := d.dot(d)
	if l2 == 0 {
		return p.distTo(s.a)
	}
	u := math.Max(0, math.Min(1, p.sub(s.a).dot(d)/l2))
	return p.distTo(s.a.add(d.scale(u)))
}

func segmentsCross(p, q segment) bool {
	d1, d2 := q.a.sub(p.a).cross(p.b.sub(p.a)), q.b.sub(p.a).cross(p.b.sub(p.a))
	d3, d4 := p.a.sub(q.a).cross(q.b.sub(q.a)), p.b.sub(q.a).cross(q.b.sub(q.a))
	return d1*d2 < 0 && d3*d4 < 0
}

func segmentDistance(p, q segment) float64 {
	if segmentsCross(p, q) {
		return 0
	}
	return math.Min(
		math.Min(pointSegmentDistance(p.a, q), pointSegmentDistance(p.b, q)),
		math.Min(pointSegmentDistance(q.a, p), pointSegmentDistance(q.b, p)),
	)
}

// sliceSegments cuts the mesh at z and returns the outline segments with their outward normals.
func sliceSegments(mesh *decad.Mesh, z float64) []segment {
	vs, tris := mesh.Vertices(), mesh.Triangles()
	var out []segment
	for _, tri := range tris {
		a, b, c := vs[tri[0]], vs[tri[1]], vs[tri[2]]
		var pts []vec2
		for _, e := range [][2]struct{ X, Y, Z float64 }{
			{{a.X, a.Y, a.Z}, {b.X, b.Y, b.Z}}, {{b.X, b.Y, b.Z}, {c.X, c.Y, c.Z}}, {{c.X, c.Y, c.Z}, {a.X, a.Y, a.Z}},
		} {
			if (e[0].Z < z) == (e[1].Z < z) {
				continue
			}
			u := (z - e[0].Z) / (e[1].Z - e[0].Z)
			pts = append(pts, vec2{e[0].X + u*(e[1].X-e[0].X), e[0].Y + u*(e[1].Y-e[0].Y)})
		}
		if len(pts) != 2 || pts[0].distTo(pts[1]) < 1e-12 {
			continue
		}
		ab, ac := b.Sub(a), c.Sub(a)
		nx, ny := ab.Y*ac.Z-ab.Z*ac.Y, ab.Z*ac.X-ab.X*ac.Z
		l := math.Hypot(nx, ny)
		if l < 1e-12 {
			continue
		}
		out = append(out, segment{a: pts[0], b: pts[1], n: vec2{nx / l, ny / l}})
	}
	return out
}

// pairLimits returns the smallest distance between facing segments that border air (an air gap) and between facing
// segments that border material (a wall thickness). It looks at segments no farther apart than reach, and skips air
// pairs for which skipAir is true. A limit nothing matched is +Inf.
func pairLimits(segs []segment, reach float64, skipAir func(p, q segment) bool) (minAir, minMaterial float64) {
	minAir, minMaterial = math.Inf(1), math.Inf(1)
	cells := map[[2]int][]int{}
	cell := func(p vec2) [2]int { return [2]int{int(math.Floor(p[0] / reach)), int(math.Floor(p[1] / reach))} }
	for i, s := range segs {
		c := cell(s.midpoint())
		for dx := -2; dx <= 2; dx++ {
			for dy := -2; dy <= 2; dy++ {
				for _, j := range cells[[2]int{c[0] + dx, c[1] + dy}] {
					p, q := segs[j], s
					if p.n.dot(q.n) >= -0.94 {
						continue
					}
					d := segmentDistance(p, q)
					if d > reach {
						continue
					}
					toQ, toP := q.midpoint().sub(p.midpoint()), p.midpoint().sub(q.midpoint())
					switch {
					case p.n.dot(toQ) > 0 && q.n.dot(toP) > 0:
						if !skipAir(p, q) {
							minAir = math.Min(minAir, d)
						}
					case p.n.dot(toQ) < 0 && q.n.dot(toP) < 0:
						minMaterial = math.Min(minMaterial, d)
					}
				}
			}
		}
		cells[cell(s.midpoint())] = append(cells[cell(s.midpoint())], i)
	}
	return minAir, minMaterial
}

func TestSpiralSpringClearance(t *testing.T) {
	t.Run("pair-helper", func(t *testing.T) {
		never := func(p, q segment) bool { return false }
		air := []segment{
			{a: vec2{0, 0}, b: vec2{1, 0}, n: vec2{0, 1}},
			{a: vec2{0, 0.5}, b: vec2{1, 0.5}, n: vec2{0, -1}},
		}
		minAir, minMaterial := pairLimits(air, 2, never)
		require.InDelta(t, 0.5, minAir, 1e-12)
		require.True(t, math.IsInf(minMaterial, 1))
		wall := []segment{
			{a: vec2{0, 0}, b: vec2{1, 0}, n: vec2{0, -1}},
			{a: vec2{0, 0.5}, b: vec2{1, 0.5}, n: vec2{0, 1}},
		}
		minAir, minMaterial = pairLimits(wall, 2, never)
		require.True(t, math.IsInf(minAir, 1))
		require.InDelta(t, 0.5, minMaterial, 1e-12)
	})

	for _, c := range []struct {
		name string
		edit func(p *spiralspring.Params)
	}{
		{"default", func(*spiralspring.Params) {}},
		{"thickness-floor", func(p *spiralspring.Params) {
			p.OuterDiameter, p.Thickness, p.MinGap, p.Height = 30, 1.5, 1, 4
		}},
		{"large", func(p *spiralspring.Params) {
			p.OuterDiameter, p.HubDiameter, p.Thickness, p.MinGap, p.Height = 120, 30, 3, 3, 8
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := spiralspring.DefaultParams()
			c.edit(&p)
			m, err := spiralspring.Build(t.Context(), decad.New(), p)
			require.NoError(t, err)
			dm := m.Dimensions
			mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(meshTolerance))
			require.NoError(t, err)
			segs := sliceSegments(mesh, p.Height/2)
			require.NotEmpty(t, segs)

			inHole := func(s segment) bool {
				mid := s.midpoint()
				return mid.len() <= dm.BoreRadius+0.01 ||
					mid.distTo(vec2{dm.EyeCenterX, dm.EyeCenterY}) <= dm.EyeHoleDiameter/2+0.01
			}
			skip := func(p, q segment) bool { return inHole(p) && inHole(q) }
			reach := 1.5 * math.Max(dm.Gap, p.Thickness)
			minAir, minMaterial := pairLimits(segs, reach, skip)
			msg := fmt.Sprintf("minAir %.4f (gap %.4f), minMaterial %.4f (thickness %.4f)", minAir, dm.Gap, minMaterial, p.Thickness)
			require.GreaterOrEqual(t, minAir, dm.Gap-sliceMargin, msg)
			require.LessOrEqual(t, minAir, dm.Gap+sliceMargin, msg)
			require.GreaterOrEqual(t, minMaterial, p.Thickness-sliceMargin, msg)
			require.LessOrEqual(t, minMaterial, p.Thickness+sliceMargin, msg)
		})
	}
}
