package clamparm_test

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/lestrrat-3d/decad-gallery/models/m5bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m5nut"
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

// TestClampArmFitsTongueInFork requires each tongue to stand 0.26 from the fins of its fork on both sides (Tight
// class), at J1 to J4, in poses that swing every joint.
func TestClampArmFitsTongueInFork(t *testing.T) {
	for _, name := range []string{"default", "raised", "pan-max", "shoulder-max", "elbow-min", "wrist-min", "placed"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := asmCaseNamed(t, name)
			_, m := asmBuild(t, c)
			// The tongue part, the fork part and the tongue's local x of the probe, per joint.
			joints := []struct {
				name         string
				tongue, fork clamparm.Part
				faces        [2]r3.Vec // the tongue's two faces in its local frame, 6 mm from its axis
			}{
				// J1: the swivel's tongue is z 0 to 8 above J1's axis point; the frame's jaw and upper fin are the fork.
				{"J1", m.Swivel, m.Clamp, [2]r3.Vec{r3.NewVec(6, 0, 0), r3.NewVec(6, 0, clamparm.TongueThickness)}},
				{"J2", m.Links[0], m.Swivel, [2]r3.Vec{r3.NewVec(6, -4, 0), r3.NewVec(6, 4, 0)}},
				{"J3", m.Links[1], m.Links[0], [2]r3.Vec{r3.NewVec(6, -4, 0), r3.NewVec(6, 4, 0)}},
				{"J4", m.Head, m.Links[1], [2]r3.Vec{r3.NewVec(6, -4, 0), r3.NewVec(6, 4, 0)}},
			}
			for _, j := range joints {
				for _, face := range j.faces {
					p := asmMove(t, j.tongue.Frame, face)
					requireFit(t, j.fork.Body, p, clamparm.JointFit, j.name+" tongue face")
				}
			}
		})
	}
}

// TestClampArmFitsBoltShankInHoles requires the bolt's shank to stand 0.3 mm from the wall of every hole it passes:
// the point on the shank's surface 2.5 from the axis is 0.3 from the hole's wall (diameter 5.6 against 5).
func TestClampArmFitsBoltShankInHoles(t *testing.T) {
	for _, name := range []string{"default", "raised", "thin-desk", "thick-desk", "placed"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := asmCaseNamed(t, name)
			_, m := asmBuild(t, c)
			sites := asmWorldSites(t, c, m.Dimensions)
			o := c.params.Opening
			// Each entry: bolt, the part it passes and a distance along the bolt where the part has a plain hole.
			holes := []struct {
				bolt int
				part clamparm.Part
				t    float64
			}{
				// J1: top jaw (z 3.5 to 14 above the head), swivel tongue (z 14.26 to 22.26), upper fin (22.52 to
				// 28.26), knob (28.26 to 31.26 before its nut pocket).
				{0, m.Clamp, 8}, {0, m.Swivel, 18}, {0, m.Clamp, 25}, {0, m.Knobs[0], 29.7},
				// J2 to J4: the -Y fin (y -6.5 to -4.26), the tongue (-4 to 4), the +Y fin (4.26 to 10), the knob.
				{1, m.Swivel, 4.6}, {1, m.Links[0], 14}, {1, m.Swivel, 17}, {1, m.Knobs[1], 21.5},
				{2, m.Links[0], 4.6}, {2, m.Links[1], 14}, {2, m.Links[0], 17}, {2, m.Knobs[2], 21.5},
				{3, m.Links[1], 4.6}, {3, m.Head, 14}, {3, m.Links[1], 17}, {3, m.Knobs[3], 21.5},
				// Stud: the knob (3.5 to 10 under the head), the head (z -10 to 10 in head coordinates), the adapter
				// between the recess cone and the nut pocket.
				{4, m.ScrewKnobs[1], 6.7}, {4, m.Head, 20}, {4, m.Adapter, 34.2},
				// Clamp screw: the knob, the bottom jaw between its underside and the nut pocket's floor.
				{5, m.ScrewKnobs[0], 6.7}, {5, m.Clamp, o + 8.4},
			}
			for _, h := range holes {
				s := sites[h.bolt]
				p := s.head.Add(s.dir.Scale(h.t)).Add(s.corner.Scale(m5bolt.ShankDiameter / 2))
				requireFit(t, h.part.Body, p, m5fastener.HoleRadius-m5bolt.ShankDiameter/2, "shank")
			}
		})
	}
}

// TestClampArmFitsHeadsAndNuts requires each hex head to stand 0.3 mm past its pocket's corners (0.05 in a screw knob's
// press pocket) and each nut 0.2 mm past its pocket's corners: a corner of the head or the nut at mid height. The
// measured distance is that radial clearance times cos 30, the distance to the walls beside the corner.
func TestClampArmFitsHeadsAndNuts(t *testing.T) {
	for _, name := range []string{"default", "raised", "placed"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := asmCaseNamed(t, name)
			_, m := asmBuild(t, c)
			sites := asmWorldSites(t, c, m.Dimensions)
			// A corner of the head sits 0.3 past the pocket's corner along the corner's direction, and cos 30 of that from
			// the two walls beside it: the nearest surface.
			const wall = 0.8660254037844386 // cos 30
			wantHead := [6]float64{0.3 * wall, 0.3 * wall, 0.3 * wall, 0.3 * wall, 0.05 * wall, 0.05 * wall}
			for i, s := range sites {
				p := s.head.Add(s.dir.Scale(m5bolt.HeadHeight / 2)).Add(s.corner.Scale(asmHexCorner))
				requireFit(t, boltPocketPart(m, i).Body, p, wantHead[i], "head corner")
			}
			frames := asmWorldNutFrames(t, c, m.Dimensions)
			for i, f := range frames {
				part, _ := nutPocketPart(m, i)
				p := f.origin.Add(f.dir.Scale(m5nut.Thickness / 2)).Add(f.ref.Scale(asmHexCorner))
				requireFit(t, part.Body, p, m5fastener.NutClearance*wall, "nut corner")
			}
		})
	}
}

// TestClampArmFitsPad requires the pad socket to stand 0.35 from the screw, the pad's -X face 0.5 from the spine,
// and the pad to clear the frame by 0.5 at both Opening limits.
func TestClampArmFitsPad(t *testing.T) {
	for _, name := range []string{"thin-desk", "default", "thick-desk", "placed"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := asmCaseNamed(t, name)
			_, m := asmBuild(t, c)
			o := c.params.Opening
			screw := asmWorldSites(t, c, m.Dimensions)[5]
			// The socket is 8 deep: its mid-height is 4 above the pad's underside at z = -o - 12.
			p := screw.head.Add(screw.dir.Scale(69.5)).Add(screw.corner.Scale(m5bolt.ShankDiameter / 2))
			requireFit(t, m.Pad.Body, p, 0.35, "pad socket")
			// A point on the pad's -X face, half way up it.
			face := asmMove(t, m.Pad.Frame, r3.NewVec(-24.5, 0, -6))
			requireFit(t, m.Clamp.Body, face, 0.5, "pad to spine")
			if name != "placed" {
				require.InDelta(t, -o-6, face.Z, 1e-9)
			}
			requireGapAtLeast(t, m.Pad.Body, m.Clamp.Body, 0.5, "pad and frame")
		})
	}
}

// TestClampArmFitsScrewKnobToJaw requires the screw knob to hang at least 0.5 mm below the bottom jaw, 4 mm at the
// thinnest desk.
func TestClampArmFitsScrewKnobToJaw(t *testing.T) {
	for _, name := range []string{"thin-desk", "default", "thick-desk"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := asmCaseNamed(t, name)
			_, m := asmBuild(t, c)
			want := c.params.Opening - 6
			require.InDelta(t, want, m.Dimensions.ScrewKnobGap, 1e-9)
			require.GreaterOrEqual(t, want, 0.5)
			requireGapAtLeast(t, m.ScrewKnobs[0].Body, m.Clamp.Body, want, "screw knob and bottom jaw")
			// The screw bolt passes through the jaw's hole and the knob: it stays 0.3 from the hole wall (checked in
			// the shank fit), and the bolt's head stays above the knob's outer face.
		})
	}
}

// TestClampArmFitsConeBossInRecess requires the head's cone boss to stand 0.35 mm, measured square to the cone, from
// the adapter's recess: a point on the boss half way up its cone.
func TestClampArmFitsConeBossInRecess(t *testing.T) {
	for _, name := range []string{"default", "raised", "wrist-min", "placed"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := asmCaseNamed(t, name)
			_, m := asmBuild(t, c)
			xf := asmWorld(t, c.params)
			i := m.Dimensions.Interface
			// The boss is radius 6 at the face and 4 at 2 above it: half way up it is radius 5 at height 1.
			mid := xf.Apply(i.Origin().Add(i.N().Scale(1)).Add(i.U().Scale(5)))
			bossDistance, bound := asmDistance(t, m.Head.Body, mid)
			require.InDelta(t, 0, bossDistance, bound+1e-6, "the probe lies on the boss's cone")
			requireFit(t, m.Adapter.Body, mid, clamparm.RecessClearance, "cone boss in recess")
			// The boss's tip, 4 from the axis at 2 above the face, clears the recess by the same 0.35.
			tip := xf.Apply(i.Origin().Add(i.N().Scale(clamparm.BossHeight)).Add(i.U().Scale(4)))
			requireFit(t, m.Adapter.Body, tip, clamparm.RecessClearance, "cone boss top")
		})
	}
}

// TestClampArmFitsStudProtrusion requires the stud's tip to stand StudProtrusion above the interface face.
func TestClampArmFitsStudProtrusion(t *testing.T) {
	for _, name := range []string{"default", "raised", "placed"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := asmCaseNamed(t, name)
			_, m := asmBuild(t, c)
			xf := asmWorld(t, c.params)
			s := asmWorldSites(t, c, m.Dimensions)[4]
			i := m.Dimensions.Interface
			face := xf.Apply(i.Origin())
			faceN := xf.ApplyDir(i.N())
			tip := s.head.Add(s.dir.Scale(s.tip()))
			require.InDelta(t, clamparm.StudProtrusion, tip.Sub(face).Dot(faceN), 1e-9)
			// The stud axis is the interface's axis: the tip is on it.
			require.InDelta(t, 0, tip.Sub(face).Sub(faceN.Scale(tip.Sub(face).Dot(faceN))).Len(), 1e-9)
		})
	}
}
