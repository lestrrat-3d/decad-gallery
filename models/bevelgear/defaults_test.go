package bevelgear_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/stretchr/testify/require"
)

func TestDefaultParamsHaveStraightTeeth(t *testing.T) {
	p := bevelgear.DefaultParams()
	require.Zero(t, p.SpiralAngle)
	require.Equal(t, "right", p.SpiralHand)
	require.Equal(t, 31, p.Teeth)
	require.Equal(t, 31, p.MateTeeth)
	require.True(t, p.EnableBore)
}
