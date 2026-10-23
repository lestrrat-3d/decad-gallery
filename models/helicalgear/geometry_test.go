package helicalgear

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestSmallHelicalToothVerifies(t *testing.T) {
	p := DefaultParams()
	p.Teeth, p.Samples, p.Thickness = 8, 5, 2
	doc := decad.New()
	tooth, err := BuildTooth(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 1)
	for _, body := range []*decad.Body{tooth} {
		require.True(t, body.IsSolid())
		mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.1),
			decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		require.True(t, mesh.BoundaryVerified())
		require.True(t, mesh.VolumeVerified())
	}
}
