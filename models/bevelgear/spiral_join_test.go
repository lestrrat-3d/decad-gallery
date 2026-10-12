package bevelgear

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestSourceDefaultSpiralFullCombine(t *testing.T) {
	for _, member := range []struct {
		name   string
		pinion bool
	}{{name: "driving"}, {name: "pinion", pinion: true}} {
		t.Run(member.name, func(t *testing.T) {
			doc, joined, _ := buildSourceDefaultFullMember(t, member.pinion)
			require.Len(t, doc.Bodies(), 1)
			require.True(t, joined.IsSolid())
			mesh, err := joined.Tessellate(t.Context(), units.Millimeters(0.1),
				decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, mesh.BoundaryVerified())
			require.True(t, mesh.VolumeVerified())
			require.Equal(t, 2, spiralMeshEuler(mesh), "the joined gear has one closed outer shell")
		})
	}
}

func buildSourceDefaultFullMember(t *testing.T, pinion bool) (*decad.Document, *decad.Body, Params) {
	t.Helper()
	p := DefaultParams()
	geometry := p
	geometry.SpiralAngle = 0
	geometry.BoreDiameter = 0
	doc := decad.New()
	joined, err := buildSpiralMember(t.Context(), doc, p, geometry, pinion)
	require.NoError(t, err)
	return doc, joined, p
}

func spiralMeshEuler(mesh *decad.Mesh) int {
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

func TestSourceDefaultSpiralFirstToothJoin(t *testing.T) {
	for _, member := range []struct {
		name   string
		pinion bool
	}{
		{name: "driving"},
		{name: "pinion", pinion: true},
	} {
		t.Run(member.name, func(t *testing.T) {
			p := DefaultParams()
			p.SpiralAngle = 35 * math.Pi / 180
			p.SpiralHand = "right"
			geometry := p
			geometry.SpiralAngle = 0
			doc := decad.New()
			blank, err := BuildBlank(t.Context(), doc, geometry)
			require.NoError(t, err)
			curved, err := buildSpiralTooth(t.Context(), doc, p, member.pinion)
			require.NoError(t, err)
			trimmed, err := trimSpiralTooth(t.Context(), doc, p, curved)
			require.NoError(t, err)
			require.Len(t, doc.Bodies(), 2)
			joined, err := decad.Union(t.Context(), blank, trimmed)
			require.NoError(t, err)
			require.Len(t, doc.Bodies(), 1)
			require.True(t, joined.IsSolid())
			mesh, err := joined.Tessellate(t.Context(), units.Millimeters(0.1),
				decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, mesh.BoundaryVerified())
			require.True(t, mesh.VolumeVerified())
		})
	}
}
