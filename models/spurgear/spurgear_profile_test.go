package spurgear

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func TestDefaultGearRootFilletsAreTangentArcs(t *testing.T) {
	t.Parallel()
	p := DefaultParams()
	d := derive(p)
	w := sketch.NewWorld()
	s, _, err := buildGearProfile(t.Context(), w, w.XY(), p, d)
	require.NoError(t, err)

	wantRadius := filletRadius(p, d)
	wantLineRadius := math.Sqrt(d.root*d.root + 2*d.root*wantRadius)
	fillets := 0
	for _, entity := range s.Entities() {
		arc, ok := entity.(*sketch.Arc)
		if !ok || math.Abs(arc.R()-wantRadius) > 1e-7 {
			continue
		}
		fillets++
		require.Less(t, arc.Sweep(), math.Pi)
		cx, cy := arc.Center.X(), arc.Center.Y()
		require.InDelta(t, d.root+wantRadius, math.Hypot(cx, cy), 1e-7)
		start, end := arc.Start, arc.End
		if math.Abs(math.Hypot(start.X(), start.Y())-d.root) > 1e-7 {
			start, end = end, start
		}
		require.InDelta(t, d.root, math.Hypot(start.X(), start.Y()), 1e-7)
		require.InDelta(t, wantLineRadius, math.Hypot(end.X(), end.Y()), 1e-7)
		require.InDelta(t, 0, cx*start.Y()-cy*start.X(), 1e-7)
		require.InDelta(t, 0, (end.X()-cx)*end.X()+(end.Y()-cy)*end.Y(), 1e-7)
	}
	require.Equal(t, 2*p.Teeth, fillets)
}
