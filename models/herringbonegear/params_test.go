package herringbonegear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/herringbonegear"
	"github.com/stretchr/testify/require"
)

func TestHerringboneMirrorPlaneIsHalfThickness(t *testing.T) {
	p := herringbonegear.DefaultParams()
	p.Teeth, p.Samples, p.Thickness = 8, 5, 2
	doc := decad.New()
	_, err := herringbonegear.Build(t.Context(), doc, p)
	require.NoError(t, err)

	values := map[string]float64{}
	for _, b := range doc.Bindings() {
		values[b.Name] = b.Value.Base()
	}
	require.InDelta(t, 1, values["helix_plane_offset"], 1e-12)
}
