package fastener_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/fastener"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestEveryPresetBoltAndNutVerifyAll tessellates every catalog preset's bolt at each standard length, and its nut, at
// 0.01 mm under VerifyAll. The renderer meshes them without verification, so this test is the soundness check.
func TestEveryPresetBoltAndNutVerifyAll(t *testing.T) {
	for _, name := range fastener.Names() {
		preset, ok := fastener.Lookup(name)
		require.True(t, ok, name)
		for _, length := range preset.Bolt.StandardLengths {
			doc := decad.New()
			b, err := fastener.BuildBolt(t.Context(), doc, fastener.BoltParams{Preset: name, Length: length})
			require.NoError(t, err, "%s x %g bolt", name, length)
			_, err = b.Body.Tessellate(t.Context(), units.Millimeters(.01), decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err, "%s x %g bolt", name, length)
		}
		n, err := fastener.BuildNut(t.Context(), decad.New(), fastener.NutParams{Preset: name})
		require.NoError(t, err, "%s nut", name)
		_, err = n.Body.Tessellate(t.Context(), units.Millimeters(.01), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, "%s nut", name)
	}
}
