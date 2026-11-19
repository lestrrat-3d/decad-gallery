package main

// CI: isolated-tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSpinningWheelMotion builds the real assembly once and checks every moving part against its joints.
func TestSpinningWheelMotion(t *testing.T) {
	meshes, base, err := spinningWheelModels(t.Context())
	require.NoError(t, err)
	requireWheelAnimationJoints(t, meshes, base)
}
