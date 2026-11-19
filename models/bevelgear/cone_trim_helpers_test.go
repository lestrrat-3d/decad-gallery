package bevelgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func buildToeTrimmedTooth(t *testing.T, p bevelgear.Params) (*decad.Document, *decad.Body) {
	t.Helper()
	doc := decad.New()
	tooth, err := bevelgear.BuildPointTooth(t.Context(), doc, p)
	require.NoError(t, err)
	cone, err := bevelgear.BuildToeCone(t.Context(), doc, p)
	require.NoError(t, err)
	trimmed, err := decad.Cut(t.Context(), tooth, cone)
	require.NoError(t, err)
	require.True(t, trimmed.IsSolid())
	return doc, trimmed
}

func requireAnalyticTrim(t *testing.T, body *decad.Body) {
	t.Helper()
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	var coneFaces, freeformFaces int
	for _, face := range body.Faces() {
		switch face.Surface().(type) {
		case decad.Cone:
			coneFaces++
		case decad.NURBSSurface:
			freeformFaces++
		}
	}
	require.Positive(t, coneFaces)
	require.Positive(t, freeformFaces)
}
