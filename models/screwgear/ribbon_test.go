package screwgear_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/screwgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestRibbonDefaultIsOneVerifiedLoft lofts gear A's default ribbon, 68 teeth at ten sections a
// pitch (681 sections, a step of Length()/680 = 0.2625 mm), as one solid and checks it the way
// decad checks a twisted fitted-edge loft: the volume against the denoted solid, the verified
// mesh, the mesh volume, and a rigid motion of the body. The ribbon is the unplaced loft the
// build places, so the real producer runs into the real consumer.
func TestRibbonDefaultIsOneVerifiedLoft(t *testing.T) {
	p := screwgear.DefaultParams()
	doc := decad.New()
	body, sections, step, err := screwgear.RibbonForTest(t.Context(), doc, p, 0)
	require.NoError(t, err)
	require.Len(t, sections, p.ToothCount*10+1)
	require.Equal(t, decad.BodySolid, body.Kind())
	require.Len(t, body.Faces(), 6)

	reference := denotedRibbonReference(t, sections, step)
	volume, err := body.Volume()
	require.NoError(t, err)
	t.Logf("volume %.3f reference %.3f bound %.3f", volume.Value.Base(), reference, volume.Bound.Base())
	require.LessOrEqual(t, math.Abs(volume.Value.Base()-reference), volume.Bound.Base())
	require.Less(t, volume.Bound.Base(), 0.03*reference)

	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	require.LessOrEqual(t, mesh.Bound().Base(), 0.05)
	closed, signed := meshIsWatertight(mesh)
	require.True(t, closed)
	require.LessOrEqual(t, math.Abs(signed-reference), volume.Bound.Base())
	t.Logf("mesh: %d triangles, bound %.5f", len(mesh.Triangles()), mesh.Bound().Base())

	// A rigid motion keeps the body usable and its readings.
	motion, err := r3.RotationAround(r3.NewVec(1, 2, 3), r3.NewVec(1, 1, 0), units.Radians(0.7))
	require.NoError(t, err)
	moved, err := decad.Copy(body).Place(decad.CopyTransform{Transform: motion}).Do(t.Context())
	require.NoError(t, err)
	movedVolume, err := moved.Volume()
	require.NoError(t, err)
	require.LessOrEqual(t, math.Abs(movedVolume.Value.Base()-volume.Value.Base()),
		movedVolume.Bound.Base()+volume.Bound.Base())
	movedMesh, err := moved.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, movedMesh.BoundaryVerified())
	require.True(t, movedMesh.VolumeVerified())
}
