package sevensegment

// CI: isolated-tests

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

func TestButtonDetentDisc(t *testing.T) {
	body, _, err := buttonDetentDisc(t.Context(), decad.New())
	require.NoError(t, err)
	require.True(t, body.IsSolid())
}

func TestButtonDetentInvalid(t *testing.T) {
	for _, phase := range []float64{-1, 361, math.NaN(), math.Inf(1), math.Inf(-1)} {
		doc := decad.New()
		_, err := buildButtonDetent(t.Context(), doc, phase)
		require.Error(t, err)
		require.Empty(t, doc.Bodies())
	}
	_, err := buildButtonDetent(nil, decad.New(), 0) //nolint:staticcheck // Exercise nil-context validation.
	require.Error(t, err)
	_, err = buildButtonDetent(t.Context(), nil, 0)
	require.Error(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	doc := decad.New()
	_, err = buildButtonDetent(ctx, doc, 0)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, doc.Bodies())
}

func TestButtonDetentBuild(t *testing.T) {
	doc := decad.New()
	model, err := buildButtonDetent(t.Context(), doc, 0)
	require.NoError(t, err)
	require.True(t, model.Cam.IsSolid())
	require.True(t, model.Key.IsSolid())
	require.NoError(t, doc.VerifyContainers().Err())
}
