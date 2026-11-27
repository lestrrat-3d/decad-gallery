package sevensegment_test

// CI: isolated-tests

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSevenSegmentPoseSelection checks how SEVENSEGMENT_POSES selects poses.
func TestSevenSegmentPoseSelection(t *testing.T) {
	t.Parallel()
	all := []string{"digit0", "x", "offset", "quarter"}
	require.Equal(t, all, allPoseNames())
	valid := []struct {
		value string
		set   bool
		want  []string
	}{
		{"", false, all},
		{"none", true, nil},
		{"digit0", true, []string{"digit0"}},
		{"offset", true, []string{"offset"}},
		{"quarter,x", true, []string{"x", "quarter"}},
		{"digit0,x,offset,quarter", true, all},
	}
	for _, c := range valid {
		got, err := selectPoses(c.value, c.set)
		require.NoError(t, err, "%q", c.value)
		require.Equal(t, c.want, got, "%q", c.value)
	}
	invalid := map[string]string{
		"":              "empty",
		"digit":         `unknown pose "digit"`,
		"digit0,":       `unknown pose ""`,
		"x,,offset":     `unknown pose ""`,
		"Digit0":        `unknown pose "Digit0"`,
		"digit0 ":       `unknown pose "digit0 "`,
		"x,x":           `pose "x" twice`,
		"none,digit0":   `unknown pose "none"`,
		"placement:0":   `unknown pose "placement:0"`,
		"digit0|offset": `unknown pose "digit0|offset"`,
	}
	for value, message := range invalid {
		_, err := selectPoses(value, true)
		require.ErrorContains(t, err, message, "%q", value)
		require.ErrorContains(t, err, posesEnv, "%q", value)
	}
}

// TestSevenSegmentPoseMissingFails checks that a test asking for a pose the build skipped fails and names the pose.
func TestSevenSegmentPoseMissingFails(t *testing.T) {
	t.Parallel()
	d := posedModel{built: []string{"digit0"}}
	requirePose(t, d, "digit0")
	for _, name := range []string{"x", "offset", "quarter"} {
		r := &poseRecorder{}
		func() {
			defer func() { _ = recover() }()
			requirePose(r, d, name)
		}()
		require.True(t, r.failed, "pose %q", name)
		require.Contains(t, r.message, "pose not built")
	}
}

// TestSevenSegmentPoseCoverage checks, for each CI shard, that the poses it builds are the poses its tests read. A test
// that reads a pose its shard does not build would fail there, and a pose no test of the shard reads would be built for
// nothing.
func TestSevenSegmentPoseCoverage(t *testing.T) {
	t.Parallel()
	uses := externalPoseUses(t)
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	cmd := exec.CommandContext(t.Context(), "python3", "scripts/ci/run_tests.py", "sevensegment", "plan")
	cmd.Dir = root
	raw, err := cmd.Output()
	require.NoError(t, err)
	var plan map[string]struct {
		Tests []string `json:"tests"`
		Poses []string `json:"poses"`
	}
	// The runner prints its CPU line first; the plan is the last line.
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	require.NoError(t, json.Unmarshal([]byte(lines[len(lines)-1]), &plan))
	require.NotEmpty(t, plan)
	known := allPoseNames()
	for shard, entry := range plan {
		var wantDigit0, wantPlaced bool
		for _, test := range entry.Tests {
			u := uses[strings.SplitN(test, "/", 2)[0]]
			wantDigit0 = wantDigit0 || u.digit0
			wantPlaced = wantPlaced || u.placed
		}
		var gotDigit0, gotPlaced bool
		for _, pose := range entry.Poses {
			require.True(t, slices.Contains(known, pose), "shard %s lists unknown pose %q", shard, pose)
			if pose == digit0Pose {
				gotDigit0 = true
				continue
			}
			gotPlaced = true
		}
		require.Equal(t, wantDigit0, gotDigit0, "shard %s: digit0 pose built iff a test reads it", shard)
		require.Equal(t, wantPlaced, gotPlaced, "shard %s: a placed pose built iff a test reads one", shard)
		// A shard that runs one placement case builds that case and no other.
		if strings.HasPrefix(shard, "placement-") {
			require.Len(t, entry.Poses, 1, "shard %s", shard)
		}
	}
	for _, c := range placementCases {
		require.NotNil(t, plan["placement-"+c.key], "placement case %q has no shard", c.key)
		require.Equal(t, []string{c.key}, plan["placement-"+c.key].Poses)
	}
}
