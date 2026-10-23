package helicalgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/helicalgear"
	"github.com/stretchr/testify/require"
)

func TestHelicalGearReadsDerivedParameters(t *testing.T) {
	p := helicalgear.DefaultParams()
	p.Teeth, p.Samples, p.Thickness = 8, 5, 2
	doc := decad.New()
	_, err := helicalgear.Build(t.Context(), doc, p)
	require.NoError(t, err)

	values := map[string]float64{}
	for _, b := range doc.Bindings() {
		values[b.Name] = b.Value.Base()
	}
	require.InDelta(t, 2, values["helix_plane_offset"], 1e-12)
	require.InDelta(t, p.HelixAngle, values["helix_angle"], 1e-12)
}
