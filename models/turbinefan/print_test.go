package turbinefan_test

// CI: isolated-tests

import (
	"context"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/stretchr/testify/require"
)

// TestBuildBladePrint builds the default blade, its nose rib and its stud pad in both hands: three bodies, the blade in Blades and
// the other two in Harness, the blade's 13 sketches, the harness's sketches, and the rest tilt. A nil context, a nil
// document and a rotor outside the valid range are rejected before any geometry.
func TestBuildBladePrint(t *testing.T) {
	for _, hand := range []turbinefan.Hand{turbinefan.RightHand, turbinefan.LeftHand} {
		p := turbinefan.DefaultParams()
		p.Hand = hand
		doc := turbinefan.NewDocument()
		bp, err := turbinefan.BuildBladePrint(t.Context(), doc, p)
		require.NoError(t, err, "hand %v", hand)
		require.Equal(t, 3, len(doc.Bodies()))
		require.NoError(t, doc.VerifyContainers().Err())
		for name, want := range map[string][]any{
			turbinefan.BladesContainerName:  {bp.Blade},
			turbinefan.HarnessContainerName: {bp.Harness.Body, bp.Harness.Pad},
		} {
			box, err := doc.Root().ChildNamed(name)
			require.NoError(t, err)
			bodies := box.Bodies()
			require.Equal(t, len(want), len(bodies), name)
			for i := range want {
				require.True(t, any(bodies[i]) == want[i], "%s body %d", name, i)
			}
		}
		require.Equal(t, 13, len(bp.BladeSketches), "ten sections, the stud, the bolt hole and the stud flat's cutter")
		require.Equal(t, 3, len(bp.Harness.Sketches), "the rib's wedge and profile slab, and the pad's circle")

		build := turbinefan.BuildBlade
		if hand == turbinefan.LeftHand {
			build = turbinefan.BuildBladeMirrored
		}
		sz := mustSize(t, p.RotorDiameter)
		blade, err := build(t.Context(), turbinefan.NewDocument(), p.BladePitch, sz)
		require.NoError(t, err)
		tilt, _, err := turbinefan.RestTilt(t.Context(), blade.Body, sz, p.BladePitch, hand)
		require.NoError(t, err)
		require.InDelta(t, tilt, bp.Tilt, 1e-12, "rest tilt")
	}

	var nilCtx context.Context
	doc := turbinefan.NewDocument()
	_, err := turbinefan.BuildBladePrint(nilCtx, doc, turbinefan.DefaultParams())
	require.Error(t, err)
	require.Equal(t, 0, len(doc.Bodies()))
	_, err = turbinefan.BuildBladePrint(t.Context(), nil, turbinefan.DefaultParams())
	require.Error(t, err)
	p := turbinefan.DefaultParams()
	p.RotorDiameter = 100
	doc = turbinefan.NewDocument()
	_, err = turbinefan.BuildBladePrint(t.Context(), doc, p)
	require.Error(t, err)
	require.Equal(t, 0, len(doc.Bodies()))
}
