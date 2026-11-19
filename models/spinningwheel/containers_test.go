package spinningwheel_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/stretchr/testify/require"
)

func TestSpinningWheelPreviewContainers(t *testing.T) {
	doc := spinningwheel.NewDocument()
	preview, err := spinningwheel.BuildPreview(t.Context(), doc, spinningwheel.DefaultParams())
	require.NoError(t, err)
	require.Empty(t, preview.Issues)
	requirePartContainers(t, doc, preview.Assembly)
}

func TestSpinningWheelConnectorContainer(t *testing.T) {
	doc := spinningwheel.NewDocument()
	connector, err := spinningwheel.BuildConnector(t.Context(), doc, spinningwheel.DefaultParams())
	require.NoError(t, err)
	require.NoError(t, doc.VerifyContainers().Err())
	container, err := doc.ContainerOf(connector.Body)
	require.NoError(t, err)
	require.Equal(t, "Flexible Connector", container.Name())
	require.Len(t, doc.Root().Children(), 1)
}
