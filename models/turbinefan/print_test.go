package turbinefan_test

// CI: isolated-tests

import (
	"context"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/stretchr/testify/require"
)

// TestBuildBladePrint builds the default blade and its nose rib in both hands: two bodies, one in each of the Blades and
// Harness containers, the blade's 13 sketches, the harness's sketches, and the rest tilt. A nil context, a nil
// document and a rotor outside the valid range are rejected before any geometry.
func TestBuildBladePrint(t *testing.T) {
	for _, hand := range []turbinefan.Hand{turbinefan.RightHand, turbinefan.LeftHand} {
		p := turbinefan.DefaultParams()
		p.Hand = hand
		doc := turbinefan.NewDocument()
		bp, err := turbinefan.BuildBladePrint(t.Context(), doc, p)
		require.NoError(t, err, "hand %v", hand)
		require.Equal(t, 2, len(doc.Bodies()))
		require.NoError(t, doc.VerifyContainers().Err())
		for name, want := range map[string]any{
			turbinefan.BladesContainerName:  bp.Blade,
			turbinefan.HarnessContainerName: bp.Harness.Body,
		} {
			box, err := doc.Root().ChildNamed(name)
			require.NoError(t, err)
			bodies := box.Bodies()
			require.Equal(t, 1, len(bodies), name)
			require.True(t, any(bodies[0]) == want, name)
		}
		require.Equal(t, 13, len(bp.BladeSketches), "ten sections, the stud, the bolt hole and the stud flat's cutter")
		require.Equal(t, 2, len(bp.Harness.Sketches), "the rib's wedge and profile slab")

		build := turbinefan.BuildBlade
		if hand == turbinefan.LeftHand {
			build = turbinefan.BuildBladeMirrored
		}
		sz := mustSize(t, p.RotorDiameter)
		blade, err := build(t.Context(), turbinefan.NewDocument(), p.BladePitch, sz)
		require.NoError(t, err)
		tilt, err := turbinefan.RestTilt(t.Context(), blade.Body, sz, p.BladePitch, hand)
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
