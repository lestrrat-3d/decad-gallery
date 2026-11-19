package bevelgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// smallEightParams is the smallest verified straight member: 8/8, module 1, five samples.
func smallEightParams() bevelgear.Params {
	p := spiralDefaults()
	p.Teeth, p.MateTeeth, p.Samples = 8, 8, 5
	p.SpiralAngle = 0
	p.EnableBore = false
	return p
}

// sideFrame is a target plane at y=-5 with its normal on +Y, the centre point at x=10, z=7, and the
// pinion leaning toward +Z.
func sideFrame(t *testing.T) *r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.NewVec(10, -5, 7), r3.NewVec(0, 0, 1), r3.NewVec(1, 0, 0))
	require.NoError(t, err)
	return &f
}
