package herringbonegear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/herringbonegear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestPlacementRejectedBeforeAddingBodies(t *testing.T) {
	frame, err := r3.NewFrame(r3.NewVec(1, 2, 3), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	p := herringbonegear.DefaultParams()
	p.Placement = &frame
	doc := decad.New()
	_, err = herringbonegear.Build(t.Context(), doc, p)
	require.ErrorContains(t, err, "does not support Placement")
	require.Empty(t, doc.Bodies())
}

func TestRealMirroredHerringboneTooth(t *testing.T) {
	for _, tc := range []struct {
		name                string
		teeth, samples      int
		thickness, angleDeg float64
	}{
		{"small_right", 8, 5, 2, 14.5},
		{"default_right", 17, 15, 10, 14.5},
		{"right_35", 17, 15, 10, 35},
		{"thick_40", 17, 15, 40, 14.5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := herringbonegear.DefaultParams()
			p.Teeth = tc.teeth
			p.Samples = tc.samples
			p.Thickness = tc.thickness
			p.HelixAngle = tc.angleDeg * math.Pi / 180
			doc := decad.New()
			tooth, err := herringbonegear.BuildTooth(t.Context(), doc, p)
			require.NoError(t, err)
			require.True(t, tooth.IsSolid())
			require.Len(t, doc.Bodies(), 1)
			box, err := tooth.Bounds()
			require.NoError(t, err)
			require.InDelta(t, 0, box.Min.Z, box.Bound.Base()+1e-8)
			require.InDelta(t, p.Thickness, box.Max.Z, box.Bound.Base()+1e-8)
			center, err := tooth.Centroid()
			require.NoError(t, err)
			require.InDelta(t, p.Thickness/2, center.Value.Z, center.Bound.Base()+1e-6)
			mesh, err := tooth.Tessellate(t.Context(), units.Millimeters(0.1),
				decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, mesh.BoundaryVerified())
			require.True(t, mesh.VolumeVerified())
		})
	}
}

func TestOutOfRangeHelixRejectedBeforeAddingBodies(t *testing.T) {
	for _, degrees := range []float64{-14.5, 0, 0.9, 45.1, 90} {
		p := herringbonegear.DefaultParams()
		p.HelixAngle = degrees * math.Pi / 180
		doc := decad.New()
		_, err := herringbonegear.BuildTooth(t.Context(), doc, p)
		require.Error(t, err)
		require.Empty(t, doc.Bodies())
	}
}

func TestSmallRealHerringboneGear(t *testing.T) {
	p := herringbonegear.DefaultParams()
	p.Teeth = 8
	p.Samples = 5
	p.Thickness = 2
	doc := decad.New()
	gear, err := herringbonegear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, gear.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func TestDefaultRealHerringboneGear(t *testing.T) {
	p := herringbonegear.DefaultParams()
	doc := decad.New()
	gear, err := herringbonegear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, gear.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func TestRealHerringboneGearThroughBore(t *testing.T) {
	for _, tc := range []struct {
		name           string
		teeth, samples int
		thickness      float64
		boreDiameter   float64
	}{
		{"small", 8, 5, 2, 1},
		{"default", 17, 15, 10, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := herringbonegear.DefaultParams()
			p.Teeth = tc.teeth
			p.Samples = tc.samples
			p.Thickness = tc.thickness
			p.BoreDiameter = tc.boreDiameter
			doc := decad.New()
			gear, err := herringbonegear.Build(t.Context(), doc, p)
			require.NoError(t, err)
			require.True(t, gear.IsSolid())
			require.Len(t, doc.Bodies(), 1)
			mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
				decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, mesh.BoundaryVerified())
			require.True(t, mesh.VolumeVerified())
			require.Equal(t, 0, meshEuler(mesh), "the bore pierces the gear")
		})
	}
}

func TestSmallRealHerringboneGearWithBoreAndCapChamfer(t *testing.T) {
	p := herringbonegear.DefaultParams()
	p.Teeth = 8
	p.Samples = 5
	p.Thickness = 2
	p.BoreDiameter = 1
	p.CapChamfer = 0.1
	doc := decad.New()
	gear, err := herringbonegear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, gear.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	require.Equal(t, 0, meshEuler(mesh), "the chamfered gear has one through-bore")
}

func TestDefaultRealHerringboneGearWithBoreAndCapChamfer(t *testing.T) {
	p := herringbonegear.DefaultParams()
	p.BoreDiameter = 3
	p.CapChamfer = 0.1
	doc := decad.New()
	gear, err := herringbonegear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, gear.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	require.Equal(t, 0, meshEuler(mesh), "the chamfered gear has one through-bore")
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
