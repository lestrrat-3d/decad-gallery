package turbinefan_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestBladeSocketRejectsInvalidSize(t *testing.T) {
	size, err := turbinefan.SizeFor(128)
	require.NoError(t, err)
	size.RefRadius = 0
	frame, err := r3.NewFrame(r3.Vec{}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	doc := turbinefan.NewDocument()
	body, profile, err := turbinefan.BladeSocketTool(t.Context(), doc, frame, size)
	require.ErrorContains(t, err, "invalid blade size")
	require.Nil(t, body)
	require.Nil(t, profile)
	require.Empty(t, doc.Bodies())
}
