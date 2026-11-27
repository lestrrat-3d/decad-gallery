package cubepuzzle_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/cubepuzzle"
	"github.com/stretchr/testify/require"
)

func TestBuildAssembly(t *testing.T) {
	doc := cubepuzzle.NewDocument()
	a, err := cubepuzzle.Build(t.Context(), doc, cubepuzzle.DefaultParams())
	require.NoError(t, err)
	require.NotNil(t, a.Hub)
	require.Equal(t, 6, len(a.Centers))
	require.Equal(t, 12, len(a.Edges))
	require.Equal(t, 8, len(a.Corners))
	require.Equal(t, 6, len(a.Disks))
	require.Equal(t, 6, len(a.FlangedSleeves))
	require.Equal(t, 6, len(a.Sleeves))
	require.Equal(t, 6, len(a.Bolts))
	require.Equal(t, 6, len(a.Nuts))
	require.Equal(t, 57, len(doc.Bodies()))
	require.NoError(t, doc.VerifyContainers().Err())
}
