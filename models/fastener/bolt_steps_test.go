package fastener_test

// CI: isolated-tests

import (
	"context"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/fastener"
	"github.com/stretchr/testify/require"
)

func TestHexHeadBoltBuildsInsideStep(t *testing.T) {
	params := fastener.BoltParams{Preset: "M5", Length: 20}
	preset := mustLookup(t, "M5")
	require.Equal(t, fastener.HeadHex, preset.Bolt.Head)

	plain, err := fastener.BuildBolt(t.Context(), decad.New(), params)
	require.NoError(t, err)

	var stepped *fastener.Bolt
	doc := decad.New()
	err = decad.RunSteps(t.Context(), doc, []decad.Step{{
		Name: "bolt",
		Build: func(ctx context.Context) error {
			var err error
			stepped, err = fastener.BuildBolt(ctx, doc, params)
			return err
		},
	}})
	require.NoError(t, err)
	require.NotNil(t, stepped)
	require.Len(t, doc.Bodies(), 1)
	require.True(t, stepped.Body.IsSolid())

	require.Equal(t, faceKinds(plain.Body), faceKinds(stepped.Body))
	require.Len(t, stepped.Body.Faces(), len(plain.Body.Faces()))
	want, err := plain.Body.Volume()
	require.NoError(t, err)
	got, err := stepped.Body.Volume()
	require.NoError(t, err)
	require.InDelta(t, want.Value.Base(), got.Value.Base(), want.Bound.Base()+got.Bound.Base()+1e-9)
	wantBox, err := plain.Body.Bounds()
	require.NoError(t, err)
	gotBox, err := stepped.Body.Bounds()
	require.NoError(t, err)
	require.Equal(t, wantBox.Min, gotBox.Min)
	require.Equal(t, wantBox.Max, gotBox.Max)
}
