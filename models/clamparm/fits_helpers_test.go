package clamparm_test

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// asmDistance returns the distance from p to the body's mesh and the mesh's bound, so the distance to the body's
// surface lies within the bound of it. The placed bodies hold meshes after their Transform, so decad's own
// DistanceToPoint reports them only to within a bound of 0.05 to 0.2 mm: too coarse for a 0.26 mm fit.
func asmDistance(t *testing.T, body *decad.Body, p r3.Vec) (float64, float64) {
	t.Helper()
	mg, _ := asmFineMesh(t, body)
	d, err := mg.PointDistance(p)
	require.NoError(t, err)
	return d, mg.Bound()
}

// requireFit requires the distance from p to the body to be want, within the mesh's bound.
func requireFit(t *testing.T, body *decad.Body, p r3.Vec, want float64, what string) {
	t.Helper()
	got, bound := asmDistance(t, body, p)
	require.InDelta(t, want, got, bound+1e-3, "%s: point %v", what, p)
	require.Less(t, bound, want/2, "%s: the mesh is too coarse to show the fit", what)
}

// asmFineMesh tessellates the body as finely as it allows, from 0.005 mm, and returns its mesh for meshgap.
func asmFineMesh(t testing.TB, body *decad.Body) (meshgap.Mesh, *decad.Mesh) {
	t.Helper()
	mesh := asmFineTessellation(t, body)
	mg, err := meshgap.New(mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err)
	return mg, mesh
}

// asmFineTessellation selects the first tolerance decad accepts, in the existing order.
func asmFineTessellation(t testing.TB, body *decad.Body) *decad.Mesh {
	t.Helper()
	for _, tol := range []float64{0.005, 0.0125, 0.02, 0.0225, 0.03, 0.05} {
		mesh := asmTryTessellate(body, tol)
		if mesh == nil {
			continue
		}
		return mesh
	}
	require.FailNow(t, "no tolerance tessellates the body")
	return nil
}

// asmValidateMesh preserves meshgap.New's source checks without building its unused spatial tree.
func asmValidateMesh(mesh *decad.Mesh) error {
	vertices, faces, bound := mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base()
	if len(vertices) == 0 || len(faces) == 0 {
		return fmt.Errorf("meshgap: empty mesh")
	}
	if math.IsNaN(bound) || math.IsInf(bound, 0) || bound < 0 {
		return fmt.Errorf("meshgap: invalid displacement bound")
	}
	for i, p := range vertices {
		if math.IsNaN(p.X) || math.IsInf(p.X, 0) || math.IsNaN(p.Y) || math.IsInf(p.Y, 0) ||
			math.IsNaN(p.Z) || math.IsInf(p.Z, 0) {
			return fmt.Errorf("meshgap: nonfinite vertex %d", i)
		}
	}
	for i, face := range faces {
		for _, index := range face {
			if index < 0 || index >= len(vertices) {
				return fmt.Errorf("meshgap: invalid vertex index at face %d", i)
			}
		}
	}
	return nil
}

// asmTryTessellate tessellates the body at tol through the shared cache, or returns nil when decad refuses tol.
func asmTryTessellate(body *decad.Body, tol float64) *decad.Mesh {
	key := asmMeshKey{body, tol}
	if v, ok := asmMeshCache.Load(key); ok {
		return v.(*decad.Mesh)
	}
	mesh, err := body.Tessellate(context.Background(), units.Millimeters(tol))
	if err != nil {
		return nil
	}
	v, _ := asmMeshCache.LoadOrStore(key, mesh)
	return v.(*decad.Mesh)
}

// requireGapAtLeast requires the mesh gap between two bodies to be at least want, after charging the meshes' bounds
// the way the meshgap lower bound does. A design gap of exactly want shows as a lower bound short by those bounds, so
// the check allows twice the bounds.
func requireGapAtLeast(t *testing.T, a, b *decad.Body, want float64, what string) {
	t.Helper()
	ma, _ := asmFineMesh(t, a)
	mb, _ := asmFineMesh(t, b)
	gap, _, err := ma.Exceeds(mb, 0)
	require.NoError(t, err)
	slack := 2 * (ma.Bound() + mb.Bound())
	require.GreaterOrEqual(t, gap, want-slack, "%s: gap lower bound %g", what, gap)
}

func asmCaseNamed(t testing.TB, name string) asmCase {
	t.Helper()
	for _, c := range asmCases(t) {
		if c.name == name {
			return c
		}
	}
	require.FailNow(t, "no case "+name)
	return asmCase{}
}

// asmMove applies the frame's transform to a local point.
func asmMove(t testing.TB, f r3.Frame, v r3.Vec) r3.Vec {
	t.Helper()
	xf, err := r3.FromFrame(f)
	require.NoError(t, err)
	return xf.Apply(v)
}
