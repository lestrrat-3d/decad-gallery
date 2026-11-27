package sevensegment

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestSevenSegmentPosedLight(t *testing.T) {
	t.Parallel()
	// The light steps run once with two extra poses, and again standalone at each pose. Every body of an extra pose
	// must equal the body the standalone build makes, and the main pose must stay what a standalone build makes.
	offset := offsetFrame(t, r3.Vec{X: 10, Y: 20, Z: 30})
	poses := []Pose{{Digit: 0}, {Digit: 5, Placement: offset}}
	extras := []*Parts{{}, {}}
	steps, m := lightPosedSteps(extras)
	main := Params{Digit: 2}
	_, _, _, err := runPosedSteps(t.Context(), NewDocument(), main, poses, steps, stepWorkers())
	require.NoError(t, err)

	want := []Params{main, {Digit: 0}, {Digit: 5, Placement: offset}}
	got := []*Parts{&m.Parts, extras[0], extras[1]}
	for i, p := range want {
		standalone, std := lightPosedSteps(nil)
		_, _, err := runSteps(t.Context(), NewDocument(), p, standalone, stepWorkers())
		require.NoError(t, err)
		require.Empty(t, printDiffs(lightPartPrints(t.Context(), t, &std.Parts), lightPartPrints(t.Context(), t, got[i])),
			"pose %d", i)
	}
}

func TestSevenSegmentPosedValidate(t *testing.T) {
	t.Parallel()
	// An invalid extra pose fails as the same parameters fail in BuildParts, before any geometry is made.
	for name, bad := range map[string]Params{
		"digit too large": {Digit: 10},
		"digit negative":  {Digit: -1},
		"empty frame":     {Placement: &r3.Frame{}},
	} {
		_, wantErr := BuildParts(t.Context(), NewDocument(), bad)
		require.Error(t, wantErr, name)
		doc := NewDocument()
		_, extras, err := BuildPartsPosed(t.Context(), doc, DefaultParams(), Pose(bad))
		require.EqualError(t, err, wantErr.Error(), name)
		require.Nil(t, extras, name)
		require.Empty(t, doc.Bodies(), name)
		_, _, err = BuildPosed(t.Context(), doc, DefaultParams(), Pose(bad))
		require.EqualError(t, err, wantErr.Error(), name)
		require.Empty(t, doc.Bodies(), name)
	}
}

func TestSevenSegmentPosedShare(t *testing.T) {
	t.Parallel()
	// A body the digit does not move is the main body when the placement is the same, and a new body otherwise. A body
	// the digit moves is a new body even at the same digit. Digits 1 and 4 both leave slider a down; digit 2 lifts it.
	offset := offsetFrame(t, r3.Vec{X: 5})
	poses := []Pose{{Digit: 1}, {Digit: 4}, {Digit: 2}, {Digit: 1, Placement: offset}}
	extras := []*Parts{{}, {}, {}, {}}
	steps, m := lightPosedSteps(extras)
	_, _, _, err := runPosedSteps(t.Context(), NewDocument(), Params{Digit: 1}, poses, steps, stepWorkers())
	require.NoError(t, err)

	for i, e := range extras[:3] {
		require.Same(t, m.GuidePlate, e.GuidePlate, "pose %d guide plate", i)
		require.Same(t, m.PivotBlocks[SegC][1], e.PivotBlocks[SegC][1], "pose %d pivot block", i)
		require.NotSame(t, m.Shaft, e.Shaft, "pose %d shaft", i)
		require.NotSame(t, m.Cams[SegA], e.Cams[SegA], "pose %d cam", i)
	}
	require.Same(t, m.Sliders[SegA], extras[0].Sliders[SegA])
	require.Same(t, m.Sliders[SegA], extras[1].Sliders[SegA])
	require.NotSame(t, m.Sliders[SegA], extras[2].Sliders[SegA])

	moved := extras[3]
	for name, pair := range map[string][2]*decad.Body{
		"guide plate": {m.GuidePlate, moved.GuidePlate}, "pivot block": {m.PivotBlocks[SegC][1], moved.PivotBlocks[SegC][1]},
		"slider": {m.Sliders[SegA], moved.Sliders[SegA]},
	} {
		require.NotNil(t, pair[1], name)
		require.NotSame(t, pair[0], pair[1], name)
	}
}
