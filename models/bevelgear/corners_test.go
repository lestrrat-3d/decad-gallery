package bevelgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestCorners applies "Corners and roots" to the bevel gear body about its shaft, +X in the build frame.
//
// The spiral member joins its teeth on the serial trim path, which decad.Refine does not rebuild. It builds only in
// a document with a 0.1 mm chain tolerance, and its sections still read at or below cornercheck.MaxMeshBound, so
// it is checked like the straight member and nothing is exempt.
func TestCorners(t *testing.T) {
	unbored := bevelgear.DefaultParams()
	unbored.EnableBore = false
	spiral := spiralDefaults()
	spiralUnbored := spiral
	spiralUnbored.EnableBore = false
	for _, tc := range []struct {
		name string
		p    bevelgear.Params
		doc  func() *decad.Document
	}{
		{"straight_bored", bevelgear.DefaultParams(), func() *decad.Document { return decad.New() }},
		{"straight_unbored", unbored, func() *decad.Document { return decad.New() }},
		{"spiral_bored", spiral, chainDocument},
		{"spiral_unbored", spiralUnbored, chainDocument},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body, err := bevelgear.Build(t.Context(), tc.doc(), tc.p)
			require.NoError(t, err)
			axis := r3.Vec{X: 1}
			// A silent empty list would check nothing, so require a section height.
			heights, err := cornercheck.MidHeights(body, axis)
			require.NoError(t, err)
			require.NotEmpty(t, heights)
			// The scan finds no corner on any section, gear teeth included, so no exemption is listed.
			fs, err := cornercheck.Check(t.Context(), body, axis, nil)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		})
	}
}
