package spurgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/stretchr/testify/require"
)

func TestDefaultGear(t *testing.T) {
	doc := decad.New()
	body, err := spurgear.Build(t.Context(), doc, spurgear.DefaultParams())
	require.NoError(t, err)
	require.True(t, body.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	volume, err := body.Volume()
	require.NoError(t, err)
	require.InDelta(t, 2218.77, volume.Value.Base(), volume.Bound.Base()+0.02)
	require.Less(t, volume.Bound.Base(), 0.5)
	box, err := body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 0, box.Min.Z, box.Bound.Base()+1e-8)
	require.InDelta(t, 10, box.Max.Z, box.Bound.Base()+1e-8)
	require.Greater(t, box.Max.X, 9.4)
	require.Less(t, box.Max.X, 9.6)
}
