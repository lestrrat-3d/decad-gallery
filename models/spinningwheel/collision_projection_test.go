package spinningwheel_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// A separating projection proves the complete solids disjoint after charging both verified mesh bounds.
// Failure to separate is inconclusive, never a claim that the enclosed nonconvex solids collide.
type collisionShape struct {
	points, normals, edges []r3.Vec
	triangles              [][3]int
	lo, hi                 r3.Vec
	bound                  float64
}

// Faces transverse to Y must lie outside the relevant band. Every remaining wall must run along Y.
// Thus one real section encloses the complete obstacle over that band, including concave outlines.
func nativeInvariantYBand(t *testing.T, body *decad.Body, whole collisionShape, lo, hi float64) (collisionShape, bool) {
	t.Helper()
	if hi-lo < 1e-8 {
		return collisionShape{}, false
	}
	for _, face := range body.Faces() {
		minY, maxY := math.Inf(1), math.Inf(-1)
		for _, edge := range face.Edges() {
			for _, p := range []r3.Vec{edge.Start().Position().Value, edge.End().Position().Value} {
				minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
			}
			var centre, axis r3.Vec
			radius := 0.0
			switch c := edge.Curve().(type) {
			case decad.Line3:
			case decad.Circle3:
				centre, axis, radius = c.Center, c.Axis, c.Radius.Base()
			case decad.Arc3:
				centre, axis, radius = c.Center, c.Axis, c.Radius.Base()
			default:
				return collisionShape{}, false
			}
			if radius > 0 {
				span := radius * math.Sqrt(math.Max(0, 1-axis.Y*axis.Y))
				minY, maxY = math.Min(minY, centre.Y-span), math.Max(maxY, centre.Y+span)
			}
		}
		if maxY <= lo+1e-8 || minY >= hi-1e-8 {
			continue
		}
		switch surface := face.Surface().(type) {
		case decad.Plane:
			if math.Abs(surface.Frame.N().Y) > 1e-10 {
				return collisionShape{}, false
			}
		case decad.Cylinder:
			if math.Abs(surface.Axis.Y) < 1-1e-10 {
				return collisionShape{}, false
			}
		default:
			return collisionShape{}, false
		}
	}
	plane, err := r3.NewFrame(r3.Vec{Y: (lo + hi) / 2}, r3.Vec{X: 1}, r3.Vec{Z: 1})
	require.NoError(t, err)
	section, err := body.SectionOutline(t.Context(), plane, units.Millimeters(.025))
	require.NoError(t, err)
	s := collisionShape{bound: whole.bound + section.OffSurface.Base(), normals: []r3.Vec{{X: 1}, {Y: 1}, {Z: 1}}, edges: []r3.Vec{{Y: 1}}}
	for _, loop := range section.Loops {
		for i, p := range loop {
			s.points = append(s.points, r3.Vec{X: p.X, Y: lo, Z: p.Z}, r3.Vec{X: p.X, Y: hi, Z: p.Z})
			s.edges = appendDirection(s.edges, loop[(i+1)%len(loop)].Sub(p))
		}
	}
	return s.placed(r3.Identity()), true
}

func appendDirection(directions []r3.Vec, direction r3.Vec) []r3.Vec {
	n, ok := direction.Normalize()
	if !ok {
		return directions
	}
	for _, existing := range directions {
		if math.Abs(existing.Dot(n)) > 1-1e-10 {
			return directions
		}
	}
	return append(directions, n)
}

func nativeCollisionShape(body *decad.Body, mesh surfaceTriangles) collisionShape {
	s := collisionShape{points: mesh.vertices, triangles: mesh.triangles, bound: mesh.bound}
	for _, face := range body.Faces() {
		if plane, ok := face.Surface().(decad.Plane); ok {
			s.normals = appendDirection(s.normals, plane.Frame.N())
		}
	}
	for _, edge := range body.Edges() {
		if _, ok := edge.Curve().(decad.Line3); ok {
			s.edges = appendDirection(s.edges, edge.End().Position().Value.Sub(edge.Start().Position().Value))
		}
	}
	return s.placed(r3.Identity())
}

func (s collisionShape) placed(xf r3.Transform) collisionShape {
	r := collisionShape{bound: s.bound, triangles: s.triangles}
	r.lo = r3.Vec{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}
	r.hi = r3.Vec{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}
	for _, point := range s.points {
		p := xf.Apply(point)
		r.points = append(r.points, p)
		r.lo = r3.Vec{X: math.Min(r.lo.X, p.X), Y: math.Min(r.lo.Y, p.Y), Z: math.Min(r.lo.Z, p.Z)}
		r.hi = r3.Vec{X: math.Max(r.hi.X, p.X), Y: math.Max(r.hi.Y, p.Y), Z: math.Max(r.hi.Z, p.Z)}
	}
	origin := xf.Apply(r3.Vec{})
	for _, n := range s.normals {
		r.normals = append(r.normals, xf.Apply(n).Sub(origin))
	}
	for _, n := range s.edges {
		r.edges = append(r.edges, xf.Apply(n).Sub(origin))
	}
	return r
}

func (s collisionShape) separates(other collisionShape) bool {
	margin := s.bound + other.bound + 1e-6
	if boxesSeparate(s.lo, s.hi, other.lo, other.hi, margin) {
		return true
	}
	axes := append(append([]r3.Vec{}, s.normals...), other.normals...)
	for _, a := range s.edges {
		for _, b := range other.edges {
			axes = appendDirection(axes, a.Cross(b))
		}
	}
	for _, axis := range axes {
		amin, amax, bmin, bmax := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
		for _, p := range s.points {
			d := p.Dot(axis)
			amin = math.Min(amin, d)
			amax = math.Max(amax, d)
		}
		for _, p := range other.points {
			d := p.Dot(axis)
			bmin = math.Min(bmin, d)
			bmax = math.Max(bmax, d)
		}
		if amin-bmax > margin || bmin-amax > margin {
			return true
		}
	}
	return false
}

func nativeShaftPassage(obstacle spinningwheel.Part, moving collisionShape, scale float64) bool {
	var centre, axis r3.Vec
	switch obstacle.Name {
	case "head-leg", "front-wheel-leg":
		centre, axis = r3.Vec{Y: -190 * scale, Z: 60 * scale}, r3.Vec{X: 1}
	case "front-wheel-bearing-cap", "rear-wheel-bearing-cap":
		centre, axis = r3.Vec{X: 220 * scale, Z: 580 * scale}, r3.Vec{Y: 1}
	default:
		return false
	}
	found := false
	for _, face := range obstacle.Body.Faces() {
		c, ok := face.Surface().(decad.Cylinder)
		if !ok || math.Abs(c.Radius.Base()-11) > 1e-8 || math.Abs(c.Axis.Dot(axis)) < 1-1e-10 {
			continue
		}
		d := c.Origin.Sub(centre)
		if d.Sub(axis.Scale(d.Dot(axis))).Len() < 1e-8 {
			found = true
		}
	}
	if !found {
		return false
	}
	for _, p := range moving.points {
		d := p.Sub(centre)
		if d.Sub(axis.Scale(d.Dot(axis))).Len()+moving.bound >= 11-.01 {
			return false
		}
	}
	return true
}
