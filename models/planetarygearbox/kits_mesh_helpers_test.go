package planetarygearbox_test

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/stretchr/testify/require"
)

func requireKit605Shaft608PlanetMesh(t *testing.T, index int) {
	t.Helper()
	p, err := planetarygearbox.DefaultParamsFor("planet605-shaft608")
	require.NoError(t, err)
	box, err := planetarygearbox.Build(t.Context(), planetarygearbox.NewDocument(), p)
	require.NoError(t, err)
	requirePlanetMesh(t, box, index)
}
