package cycloidal_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/cycloidal"
	"github.com/lestrrat-3d/decad/parts"
	"github.com/stretchr/testify/require"
)

// The derived explode moves every part but the housing along the drive axis, gives every output pin an offset, and
// leaves no two parts sharing space.
func TestExplode(t *testing.T) {
	for _, discs := range []int{1, 2} {
		p := cycloidal.DefaultParams()
		p.DiscCount, p.DiscGap = discs, cycloidal.MinClearance
		doc := chainDocument()
		drive, err := cycloidal.Build(t.Context(), doc, p)
		require.NoError(t, err)
		set, err := drive.Set(doc, p)
		require.NoError(t, err)
		require.Len(t, set.Parts(), len(drive.Named()))
		layout, err := drive.Explode(t.Context(), set)
		require.NoError(t, err)
		require.Equal(t, cycloidal.HousingName, layout.Ground)
		require.True(t, layout.Derived, "Decad derives the removal order")
		require.Empty(t, layout.Overlaps)

		byName := map[string]parts.ExplodedPart{}
		for _, part := range layout.Parts {
			byName[part.Name] = part
			require.GreaterOrEqual(t, part.Depth, 0, "%s is joined to the housing", part.Name)
		}
		require.Len(t, byName, len(drive.Named()))
		require.Zero(t, byName[cycloidal.HousingName].Offset.Len(), "the housing stays put")
		for _, nb := range drive.Named() {
			if nb.Name == cycloidal.HousingName {
				continue
			}
			off := byName[nb.Name].Offset
			require.Positive(t, off.Len(), "%s moves", nb.Name)
			require.InDelta(t, off.Len(), math.Abs(off.Z), 1e-9, "%s moves along the axis", nb.Name)
		}
		for i := 1; i < len(drive.OutputPins); i++ {
			require.InDelta(t, byName[cycloidal.OutputPinName(0)].Offset.Z, byName[cycloidal.OutputPinName(i)].Offset.Z,
				0.01, "output pins share one offset")
		}
	}
}
