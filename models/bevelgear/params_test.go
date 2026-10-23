package bevelgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/stretchr/testify/require"
)

func TestBevelGearReadsDerivedParameters(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Teeth, p.MateTeeth, p.Samples = 8, 8, 5
	p.SpiralAngle, p.EnableBore = 0, false
	doc := decad.New()
	_, err := bevelgear.Build(t.Context(), doc, p)
	require.NoError(t, err)

	values := map[string]float64{}
	for _, b := range doc.Bindings() {
		values[b.Name] = b.Value.Base()
	}
	require.InDelta(t, math.Pi/4, values["cone_angle"], 1e-12)
	require.InDelta(t, 4*math.Sqrt2, values["cone_distance"], 1e-9)
	require.Greater(t, values["face_width"], 0.0)
	require.InDelta(t, 1-values["face_width"]/values["cone_distance"], values["toe_scale"], 1e-12)
}
