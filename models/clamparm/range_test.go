package clamparm_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/stretchr/testify/require"
)

// TestClampArmRangeOfMotion builds each limit case and requires the joint at its limit to keep 0.5 mm between its two
// sides, hardware included (instructions.md "Range of motion"); the other cases keep it at all four joints.
func TestClampArmRangeOfMotion(t *testing.T) {
	for _, c := range asmCases(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, m := asmBuild(t, c)
			if c.joint >= 0 {
				requireJointClear(t, c, m, c.joint)
				return
			}
			for k := range 4 {
				requireJointClear(t, c, m, k)
			}
		})
	}
}

// TestClampArmRangeLimitsMatchCases keeps the limit cases tied to the constants of params.go.
func TestClampArmRangeLimitsMatchCases(t *testing.T) {
	byName := map[string]clamparm.Params{}
	for _, c := range asmCases(t) {
		byName[c.name] = c.params
	}
	require.Equal(t, clamparm.J1Max, byName["pan-max"].Pan)
	require.Equal(t, -clamparm.J1Max, byName["pan-min"].Pan)
	require.Equal(t, clamparm.ShoulderMin, byName["shoulder-min"].Shoulder)
	require.Equal(t, clamparm.OpeningMax, byName["shoulder-min"].Opening)
	require.Equal(t, clamparm.JointMax, byName["shoulder-max"].Shoulder)
	require.Equal(t, -clamparm.JointMax, byName["elbow-min"].Elbow)
	require.Equal(t, clamparm.JointMax, byName["elbow-max"].Elbow)
	require.Equal(t, clamparm.WristMin, byName["wrist-min"].Wrist)
	require.Equal(t, clamparm.WristMax, byName["wrist-max"].Wrist)
	for name, p := range byName {
		require.NoError(t, p.Validate(), name)
	}
	// The default, raised and folded render poses stay valid down to JointMax 115.
	require.GreaterOrEqual(t, clamparm.JointMax, 115.0)
}
