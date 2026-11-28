package cycloidal_test

import (
	"context"
	"slices"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/cycloidal"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func chainDocument() *decad.Document {
	return decad.New(decad.WithChainTolerance(units.Millimeters(0.1)))
}

// overlap returns the volume the two bodies share, zero when they only touch or are apart. It intersects copies
// moved to the document root, because a boolean needs both operands in one container and the drive's bodies sit
// in one container per kind. It removes every body it made before it returns, so the drive's document keeps the
// bodies it had and a shared drive can be passed in.
func overlap(t *testing.T, a, b *decad.Body) float64 {
	t.Helper()
	ctx := t.Context()
	doc := a.Document()
	var made []*decad.Body
	defer func() {
		live := doc.Bodies()
		for _, body := range made {
			if slices.Contains(live, body) {
				require.NoError(t, decad.Remove(body).Do(context.WithoutCancel(ctx)))
			}
		}
	}()
	ca, err := decad.Copy(a).Do(ctx)
	require.NoError(t, err)
	made = append(made, ca)
	cb, err := decad.Copy(b).Do(ctx)
	require.NoError(t, err)
	made = append(made, cb)
	require.NoError(t, doc.Root().AddBody(ca))
	require.NoError(t, doc.Root().AddBody(cb))
	both, err := decad.Intersect(ca, cb).Do(ctx)
	if err != nil {
		require.ErrorContains(t, err, "the intersect result is empty")
		return 0
	}
	made = append(made, both)
	volume, err := both.Volume()
	require.NoError(t, err)
	return volume.Value.Base()
}

// requireBodiesVerified requires every body of the drive to pass 0.1 mm VerifyAll with a verified boundary and volume.
func requireBodiesVerified(t *testing.T, drive *cycloidal.Drive) {
	t.Helper()
	for _, nb := range drive.Named() {
		mesh, err := nb.Body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, nb.Name)
		require.True(t, mesh.BoundaryVerified(), nb.Name)
		require.True(t, mesh.VolumeVerified(), nb.Name)
	}
}
