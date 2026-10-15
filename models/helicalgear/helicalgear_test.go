package helicalgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/helicalgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestHelixAngleRange(t *testing.T) {
	for _, degrees := range []float64{1, 14.5, 45} {
		require.NoError(t, helicalgear.ValidateHelixAngle(degrees*math.Pi/180))
	}
	for _, degrees := range []float64{-14.5, 0.9, 45.1} {
		require.Error(t, helicalgear.ValidateHelixAngle(degrees*math.Pi/180))
	}
	require.ErrorContains(t, helicalgear.ValidateHelixAngle(math.NaN()), "finite")
}

func TestZeroHelixRejectedBeforeAddingBodies(t *testing.T) {
	p := helicalgear.DefaultParams()
	p.HelixAngle = 0
	doc := decad.New()
	_, err := helicalgear.Build(t.Context(), doc, p)
	require.ErrorContains(t, err, "use spurgear.Build")
	require.Empty(t, doc.Bodies())
}

func TestPlacementRejectedBeforeAddingBodies(t *testing.T) {
	frame, err := r3.NewFrame(r3.NewVec(1, 2, 3), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	p := helicalgear.DefaultParams()
	p.Placement = &frame
	doc := decad.New()
	_, err = helicalgear.Build(t.Context(), doc, p)
	require.ErrorContains(t, err, "does not support Placement")
	require.Empty(t, doc.Bodies())
}

func TestSmallRealHelicalGear(t *testing.T) {
	p := helicalgear.DefaultParams()
	p.Teeth = 8
	p.Samples = 5
	p.Thickness = 2
	doc := decad.New()
	gear, err := helicalgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, gear.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	box, err := gear.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 0, box.Min.Z, box.Bound.Base()+1e-8)
	require.InDelta(t, p.Thickness, box.Max.Z, box.Bound.Base()+1e-8)
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func TestSmallRealHelicalGearThroughBore(t *testing.T) {
	p := helicalgear.DefaultParams()
	p.Teeth = 8
	p.Samples = 5
	p.Thickness = 2
	p.BoreDiameter = 1
	doc := decad.New()
	gear, err := helicalgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, gear.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func TestSmallRealHelicalGearWithBoreAndCapChamfer(t *testing.T) {
	p := helicalgear.DefaultParams()
	p.Teeth = 8
	p.Samples = 5
	p.Thickness = 2
	p.BoreDiameter = 1
	p.CapChamfer = 0.1
	doc := decad.New()
	gear, err := helicalgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, gear.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func TestSmallRealHelicalGearWithCapChamfer(t *testing.T) {
	p := helicalgear.DefaultParams()
	p.Teeth = 8
	p.Samples = 5
	p.Thickness = 2
	p.CapChamfer = 0.1
	doc := decad.New()
	gear, err := helicalgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, gear.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func TestDefaultRealHelicalGear(t *testing.T) {
	doc := decad.New()
	gear, err := helicalgear.Build(t.Context(), doc, helicalgear.DefaultParams())
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 1)
	require.True(t, gear.IsSolid())
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func TestDefaultRealHelicalGearThroughBore(t *testing.T) {
	p := helicalgear.DefaultParams()
	p.BoreDiameter = 1
	doc := decad.New()
	gear, err := helicalgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 1)
	require.True(t, gear.IsSolid())
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	require.Equal(t, 0, helicalMeshEuler(mesh), "the completed gear has one through-bore")
}

func TestDefaultRealHelicalGearWithBoreAndCapChamfer(t *testing.T) {
	p := helicalgear.DefaultParams()
	p.BoreDiameter = 3
	p.CapChamfer = 0.1
	doc := decad.New()
	gear, err := helicalgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 1)
	require.True(t, gear.IsSolid())
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	require.Equal(t, 0, helicalMeshEuler(mesh), "the chamfered gear retains its through-bore")
}

func helicalMeshEuler(mesh *decad.Mesh) int {
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
