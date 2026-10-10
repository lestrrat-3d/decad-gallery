package spurgear

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func TestDefaultGearProfileKeepsUnroundedRootCorners(t *testing.T) {
	t.Parallel()
	p := DefaultParams()
	d := derive(p)
	w := sketch.NewWorld()
	s, _, roots, err := buildGearProfile(t.Context(), w, w.XY(), p, d)
	require.NoError(t, err)
	require.Len(t, roots, 2*p.Teeth)
	for _, root := range roots {
		require.InDelta(t, d.root, math.Hypot(root.x, root.y), 1e-7)
	}
	rootArcs, tipArcs := 0, 0
	for _, entity := range s.Entities() {
		arc, ok := entity.(*sketch.Arc)
		if !ok {
			continue
		}
		switch {
		case math.Abs(arc.R()-d.root) < 1e-7:
			rootArcs++
		case math.Abs(arc.R()-d.tip) < 1e-7:
			tipArcs++
		default:
			t.Fatalf("unexpected profile arc radius %g", arc.R())
		}
	}
	require.Equal(t, p.Teeth, rootArcs)
	require.Equal(t, p.Teeth, tipArcs)
}
