package bevelgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestSourceDefaultAutoBoreOnRealBlank(t *testing.T) {
	p := spiralDefaults()
	p.SpiralAngle = 35 * math.Pi / 180
	p.SpiralHand = "right"
	diameter, err := bevelgear.ResolveBoreDiameter(p)
	require.NoError(t, err)
	require.InDelta(t, 7.75, diameter, 1e-12)

	p.SpiralAngle = 0
	p.BoreDiameter = diameter
	doc := decad.New()
	body, err := bevelgear.BuildBoredBlank(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, body.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	require.Equal(t, 0, meshEuler(mesh))
}

func TestSourceObliqueAutoBoreCapOnRealBlank(t *testing.T) {
	p := spiralDefaults()
	p.ShaftAngle = 35 * math.Pi / 180
	p.SpiralAngle = 35 * math.Pi / 180
	p.SpiralHand = "right"
	diameter, err := bevelgear.ResolveBoreDiameter(p)
	require.NoError(t, err)
	expected := 1.9 * (15.5 - 3.875/math.Tan(17.5*math.Pi/180))
	require.InDelta(t, expected, diameter, 1e-10)
	require.Less(t, diameter, 7.75, "the heel bound must cap the source auto bore")
	oversized := p
	oversized.BoreDiameter = 7.75
	_, err = bevelgear.ResolveBoreDiameter(oversized)
	require.ErrorContains(t, err, "exceeds maximum")

	p.SpiralAngle = 0
	p.BoreDiameter = diameter
	doc := decad.New()
	body, err := bevelgear.BuildBoredBlank(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, body.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	require.Equal(t, 0, meshEuler(mesh))
}

func TestReferenceBlankThroughBore(t *testing.T) {
	for _, tc := range []struct {
		name   string
		module float64
		teeth  int
		bore   float64
	}{
		{name: "small", module: 1, teeth: 8, bore: 1},
		{name: "reference default pair", module: 4, teeth: 31, bore: 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := spiralDefaults()
			p.Module, p.Teeth, p.MateTeeth, p.Samples = tc.module, tc.teeth, tc.teeth, 5
			plain, err := bevelgear.BuildBlank(t.Context(), decad.New(), p)
			require.NoError(t, err)
			plainVolume, err := plain.Volume()
			require.NoError(t, err)
			plainMesh, err := plain.Tessellate(t.Context(), units.Millimeters(0.1),
				decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.Equal(t, 2, meshEuler(plainMesh), "the uncut blank must have no tunnel")

			p.BoreDiameter = tc.bore
			doc := decad.New()
			body, err := bevelgear.BuildBoredBlank(t.Context(), doc, p)
			require.NoError(t, err)
			require.True(t, body.IsSolid())
			require.Len(t, doc.Bodies(), 1)
			volume, err := body.Volume()
			require.NoError(t, err)
			require.Greater(t, plainVolume.Value.Base()-volume.Value.Base(),
				plainVolume.Bound.Base()+volume.Bound.Base())
			mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.1),
				decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, mesh.BoundaryVerified())
			require.True(t, mesh.VolumeVerified())
			require.Equal(t, 0, meshEuler(mesh), "the bore must make a through-hole")
		})
	}
}

func meshEuler(mesh *decad.Mesh) int {
	edges := make(map[[2]int]struct{})
	triangles := mesh.Triangles()
	for _, triangle := range triangles {
		for i := range 3 {
			a, b := triangle[i], triangle[(i+1)%3]
			if a > b {
				a, b = b, a
			}
			edges[[2]int{a, b}] = struct{}{}
		}
	}
	return len(mesh.Vertices()) - len(edges) + len(triangles)
}

func meshComponents(mesh *decad.Mesh) int {
	parent := make([]int, len(mesh.Vertices()))
	for i := range parent {
		parent[i] = i
	}
	root := func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	for _, triangle := range mesh.Triangles() {
		first := root(triangle[0])
		parent[root(triangle[1])] = first
		parent[root(triangle[2])] = first
	}
	components := make(map[int]struct{})
	for i := range parent {
		components[root(i)] = struct{}{}
	}
	return len(components)
}

func TestReferenceBlankRejectsOversizedBore(t *testing.T) {
	p := spiralDefaults()
	p.Teeth, p.MateTeeth = 8, 8
	p.BoreDiameter = 10
	doc := decad.New()
	_, err := bevelgear.BuildBoredBlank(t.Context(), doc, p)
	require.ErrorContains(t, err, "bore diameter")
	require.Empty(t, doc.Bodies())
}

func TestJoinedEightToothPairThroughBore(t *testing.T) {
	p := spiralDefaults()
	p.Teeth, p.MateTeeth, p.Samples = 8, 8, 5
	p.SpiralAngle = 0
	p.BoreDiameter = 1
	p.MateBoreDiameter = 1
	doc := decad.New()
	pair, err := bevelgear.BuildPair(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 2)
	require.Same(t, pair.Pinion, doc.Bodies()[0], "the source builds the pinion first")
	require.Same(t, pair.Driving, doc.Bodies()[1])
	for _, tc := range []struct {
		name string
		body *decad.Body
	}{
		{name: "pinion", body: pair.Pinion},
		{name: "driving", body: pair.Driving},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.True(t, tc.body.IsSolid())
			mesh, err := tc.body.Tessellate(t.Context(), units.Millimeters(0.1),
				decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, mesh.BoundaryVerified())
			require.True(t, mesh.VolumeVerified())
			require.Equal(t, 0, meshEuler(mesh), "each member must have one through-hole")
		})
	}
}

func TestSourceDefaultSpiralPair(t *testing.T) {
	doc := chainDocument()
	pair, err := bevelgear.BuildPair(t.Context(), doc, spiralDefaults())
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 2)
	require.Same(t, pair.Pinion, doc.Bodies()[0], "the source builds the pinion first")
	require.Same(t, pair.Driving, doc.Bodies()[1])
	for _, member := range []struct {
		name string
		body *decad.Body
	}{{name: "pinion", body: pair.Pinion}, {name: "driving", body: pair.Driving}} {
		t.Run(member.name, func(t *testing.T) {
			require.True(t, member.body.IsSolid())
			mesh, err := member.body.Tessellate(t.Context(), units.Millimeters(0.1),
				decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, mesh.BoundaryVerified())
			require.True(t, mesh.VolumeVerified())
			require.Equal(t, 1, meshComponents(mesh), "teeth and blank share one connected mesh")
			require.Equal(t, 0, meshEuler(mesh), "each member has a through-bore")
		})
	}
}

func TestBuildPairRejectsUnsupportedCaseBeforeAddingBodies(t *testing.T) {
	p := spiralDefaults()
	p.Samples = 14
	doc := decad.New()
	_, err := bevelgear.BuildPair(t.Context(), doc, p)
	require.ErrorContains(t, err, "15 flank samples")
	require.Empty(t, doc.Bodies())
}

func TestBuildPairRejectsOversizedPinionBoreBeforeAddingBodies(t *testing.T) {
	p := spiralDefaults()
	p.MateBoreDiameter = 100
	doc := decad.New()
	_, err := bevelgear.BuildPair(t.Context(), doc, p)
	require.ErrorContains(t, err, "exceeds maximum")
	require.Empty(t, doc.Bodies())
}

func TestBuildRejectsOversizedBoreBeforeAddingBodies(t *testing.T) {
	p := spiralDefaults()
	p.Teeth, p.MateTeeth, p.Samples = 8, 8, 5
	p.SpiralAngle = 0
	p.BoreDiameter = 10
	doc := decad.New()
	_, err := bevelgear.Build(t.Context(), doc, p)
	require.ErrorContains(t, err, "exceeds maximum")
	require.Empty(t, doc.Bodies())
}
