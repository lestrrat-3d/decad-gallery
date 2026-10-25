package bevelgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/stretchr/testify/require"
)

// spiralDefaults is DefaultParams with the verified 35° spiral teeth, the pair the spiral tests build.
func spiralDefaults() bevelgear.Params {
	p := bevelgear.DefaultParams()
	p.SpiralAngle = 35 * math.Pi / 180
	return p
}

func TestDefaultParamsHaveStraightTeeth(t *testing.T) {
	p := bevelgear.DefaultParams()
	require.Zero(t, p.SpiralAngle)
	require.Equal(t, "right", p.SpiralHand)
	require.Equal(t, 31, p.Teeth)
	require.Equal(t, 31, p.MateTeeth)
	require.True(t, p.EnableBore)
}
