package dovetail_test

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/features/dovetail"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// cases covers every parameter branch: both carriers, the shallowest and steepest slopes, and the smallest and a large
// joint. Each depth is a multiple of the slope's DepthStep, so every corner is an exact float.
func cases() map[string]dovetail.Params {
	second := dovetail.DefaultParams()
	second.TailOn = dovetail.TailOnSecond
	shallow := dovetail.Params{TailWidth: 10, Slope: dovetail.AllowedSlopes[0], Length: 16, Depth: 5.625}
	steep := dovetail.Params{TailWidth: 10, Slope: dovetail.AllowedSlopes[len(dovetail.AllowedSlopes)-1], Length: 16,
		Depth: 8.75}
	smallest := dovetail.Params{TailWidth: dovetail.MinTailWidth, Slope: dovetail.DefaultSlope, Length: dovetail.MinLength,
		Depth: dovetail.MinDepth}
	large := dovetail.Params{TailWidth: 20, Slope: dovetail.Slope{Rise: 9, Run: 40}, Length: 30, Depth: 10,
		TailOn: dovetail.TailOnSecond}
	return map[string]dovetail.Params{
		"default": dovetail.DefaultParams(), "tail on second": second, "shallowest slope": shallow,
		"steepest slope": steep, "smallest": smallest, "large": large,
	}
}

func build(t *testing.T, p dovetail.Params) (*decad.Document, *dovetail.Demo) {
	t.Helper()
	doc := decad.New()
	d, err := dovetail.BuildDemo(t.Context(), doc, p)
	require.NoError(t, err)
	return doc, d
}

func requireFullyConstrained(t *testing.T, name string, s *sketch.Sketch) {
	t.Helper()
	report := s.Verify(t.Context())
	require.True(t, report.Status == sketch.FullyConstrained, "%s: status %s", name, report.Status)
	require.Zero(t, report.DOF, name)
	require.Zero(t, len(report.Redundant), name)
	require.Zero(t, len(report.Conflicts), name)
	require.NoError(t, report.Check(), name)
}

// profileCorners returns the dovetail profile's four outline corners in the joint frame's (U, N) coordinates for a
// tail leaving its host at N = 0 toward direction s: the neck's two corners on the mating plane and the two at the
// wide end.
func profileCorners(p dovetail.Params, s float64) (neck, wide [2]r3.Vec) {
	nk, wd := p.TailWidth/2, p.TipWidth()/2
	neck = [2]r3.Vec{{X: -nk}, {X: nk}}
	wide = [2]r3.Vec{{X: -wd, Z: s * p.Depth}, {X: wd, Z: s * p.Depth}}
	return neck, wide
}

// chamferPoints returns the four points where each chamfer meets its two faces, in (U, N) coordinates. The tail's
// wide-end chamfers run from the wide-end face to the flank, and the socket's lip chamfers from the mating face to the
// flank. Each chamfer has the setback p.ChamferSize() on both faces.
func chamferPoints(p dovetail.Params, s float64) (tail, socket []r3.Vec) {
	h := math.Hypot(float64(p.Slope.Rise), float64(p.Slope.Run))
	c, sin, cos := p.ChamferSize(), float64(p.Slope.Rise)/h, float64(p.Slope.Run)/h
	nk, wd := p.TailWidth/2, p.TipWidth()/2
	for _, su := range []float64{-1, 1} {
		tail = append(tail, r3.Vec{X: su * (wd - c), Z: s * p.Depth}, r3.Vec{X: su * (wd - c*sin), Z: s * (p.Depth - c*cos)})
		socket = append(socket, r3.Vec{X: su * (nk + c)}, r3.Vec{X: su * (nk + c*sin), Z: s * c * cos})
	}
	return tail, socket
}

// requireOutlineHas requires the section outline at mid-length to pass within tol of each point.
func requireOutlineHas(t *testing.T, ctx context.Context, name string, body *decad.Body, length float64, pts []r3.Vec,
	tol float64) {
	t.Helper()
	outline, err := cornercheck.Outline(ctx, body, r3.Vec{Y: 1}, length/2)
	require.NoError(t, err)
	for _, pt := range pts {
		require.LessOrEqual(t, nearestVertex(outline, pt), tol, "%s: no outline vertex near (%.3f, %.3f)", name, pt.X, pt.Z)
	}
}

// requireOutlineLacks requires the section outline at mid-length to have no vertex within clear of any point.
func requireOutlineLacks(t *testing.T, ctx context.Context, name string, body *decad.Body, length float64, pts []r3.Vec, clear float64) {
	t.Helper()
	outline, err := cornercheck.Outline(ctx, body, r3.Vec{Y: 1}, length/2)
	require.NoError(t, err)
	for _, pt := range pts {
		require.Greater(t, nearestVertex(outline, pt), clear, "%s: outline vertex at sharp corner (%.3f, %.3f)", name, pt.X, pt.Z)
	}
}

func nearestVertex(outline cornercheck.SectionOutline, pt r3.Vec) float64 {
	least := math.Inf(1)
	for _, loop := range outline.Loops {
		for _, v := range loop {
			least = math.Min(least, math.Hypot(v.X-pt.X, v.Z-pt.Z))
		}
	}
	return least
}

// edgeAlongV reports whether body has a straight edge along V through (u, n) in the demo frame, from V = 0 to length.
func edgeAlongV(body *decad.Body, pt r3.Vec, length float64) bool {
	const tol = 1e-4
	at := func(v r3.Vec, y float64) bool {
		return math.Abs(v.X-pt.X) < tol && math.Abs(v.Z-pt.Z) < tol && math.Abs(v.Y-y) < tol
	}
	for _, e := range body.Edges() {
		a, b := e.Start().Position().Value, e.End().Position().Value
		if at(a, 0) && at(b, length) || at(a, length) && at(b, 0) {
			return true
		}
	}
	return false
}

// meshOf tessellates body with a held bound and prepares it for a distance query.
func meshOf(t *testing.T, body *decad.Body) meshgap.Mesh {
	t.Helper()
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.02), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	mg, err := meshgap.New(mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err)
	return mg
}

// endFaces returns the faces of body that lie wholly in the plane V = v, found by the world position of their vertices.
func endFaces(body *decad.Body, v float64) []*decad.Face {
	var out []*decad.Face
	for _, f := range body.Faces() {
		edges := f.Edges()
		flat := len(edges) > 0
		for _, e := range edges {
			for _, pt := range []r3.Vec{e.Start().Position().Value, e.End().Position().Value} {
				flat = flat && math.Abs(pt.Y-v) < 1e-6
			}
		}
		if flat {
			out = append(out, f)
		}
	}
	return out
}

func mustTranslate(t *testing.T, v r3.Vec) r3.Transform {
	t.Helper()
	xf, err := r3.Translation(v)
	require.NoError(t, err)
	return xf
}

// hostBlock extrudes a rectangular host from the mating plane of frame f: width along U, length along V, height along
// N toward direction dir.
func hostBlock(t *testing.T, doc *decad.Document, f r3.Frame, width, length, height float64, dir decad.Direction) *decad.Body {
	t.Helper()
	w := sketch.NewWorld()
	plane, err := w.CreatePlaneFromFrame(f)
	require.NoError(t, err)
	sk, err := w.CreateSketch(plane)
	require.NoError(t, err)
	p0, p1 := sk.CreatePoint(-width/2, 0), sk.CreatePoint(width/2, 0)
	p2, p3 := sk.CreatePoint(width/2, length), sk.CreatePoint(-width/2, length)
	l0, l1, l2, l3 := sk.CreateLine(p0, p1), sk.CreateLine(p1, p2), sk.CreateLine(p2, p3), sk.CreateLine(p3, p0)
	sk.AddConstraint(sketch.NewHorizontal(l0), sketch.NewHorizontal(l2), sketch.NewVertical(l1), sketch.NewVertical(l3),
		sketch.NewMidpoint(sk.Origin(), l0), sketch.NewDistance(p0, p1, width), sketch.NewVerticalDistance(p1, p2, length))
	_, err = sk.Solve(t.Context())
	require.NoError(t, err)
	requireFullyConstrained(t, "host block", sk)
	var chosen *sketch.Profile
	for _, c := range sk.Profiles() {
		if c.Valid {
			chosen = c
		}
	}
	require.NotNil(t, chosen)
	body, err := decad.Extrude(doc, sk, chosen, decad.Distance{D: units.Millimeters(height), Dir: dir}).Do(t.Context())
	require.NoError(t, err)
	return body
}
