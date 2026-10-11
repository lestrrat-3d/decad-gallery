package bevelgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

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
			p := bevelgear.DefaultParams()
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

func TestReferenceBlankRejectsOversizedBore(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Teeth, p.MateTeeth = 8, 8
	p.BoreDiameter = 10
	doc := decad.New()
	_, err := bevelgear.BuildBoredBlank(t.Context(), doc, p)
	require.ErrorContains(t, err, "bore diameter")
	require.Empty(t, doc.Bodies())
}

func TestJoinedEightToothPairThroughBore(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Teeth, p.MateTeeth, p.Samples = 8, 8, 5
	p.BoreDiameter = 1
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

func TestBuildPairRejectsUnsupportedCaseBeforeAddingBodies(t *testing.T) {
	doc := decad.New()
	_, err := bevelgear.BuildPair(t.Context(), doc, bevelgear.DefaultParams())
	require.ErrorContains(t, err, "verified 8/8")
	require.Empty(t, doc.Bodies())
}

func TestBuildRejectsOversizedBoreBeforeAddingBodies(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Teeth, p.MateTeeth, p.Samples = 8, 8, 5
	p.BoreDiameter = 10
	doc := decad.New()
	_, err := bevelgear.Build(t.Context(), doc, p)
	require.ErrorContains(t, err, "exceeds maximum")
	require.Empty(t, doc.Bodies())
}
