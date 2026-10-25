package helicalgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/helicalgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// The real-body checks are slow, so run them one at a time:
//
//	go test ./models/helicalgear -run '^TestSmallHelicalToothAndRootDiscVerifyIndividually$' -count=1 -timeout 2m
//	go test ./models/helicalgear -run '^TestSmallRealHelicalGear$' -count=1 -timeout 2m
//	go test ./models/helicalgear -run '^TestSmallRealHelicalGearThroughBore$' -count=1 -timeout 2m
//	go test ./models/helicalgear -run '^TestSmallRealHelicalGearWithCapChamfer$' -count=1 -timeout 5m
//	go test ./models/helicalgear -run '^TestSmallRealHelicalGearWithBoreAndCapChamfer$' -count=1 -timeout 5m
//	go test ./models/helicalgear -run '^TestDefaultRealHelicalGear$' -count=1 -timeout 5m
//	go test ./models/helicalgear -run '^TestDefaultRealHelicalGearThroughBore$' -count=1 -timeout 5m
//	go test ./models/helicalgear -run '^TestDefaultRealHelicalGearWithBoreAndCapChamfer$' -count=1 -timeout 10m
//
// The small case is 8 teeth, 2 mm thick, +14.5°: a lofted tooth and an extruded root disc
// verify individually, then the gear passes all tooth Unions, the optional 1 mm bore Cut
// and both outer cap Chamfers under VerifyAll at 0.1 mm. On the bored gear each cap
// selector returns the outer edges and no bore edges. The default case is the 17-tooth
// source-default gear, and its bored meshes have one through-hole. The +14.5° gear has no
// straight axial root seam, so the source root-fillet selector is empty and that step is
// skipped; a nonempty root fillet still needs a real geometry check.

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
