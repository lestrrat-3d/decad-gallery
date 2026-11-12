package escapement

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestEscapeWheel(t *testing.T) {
	b := testBuilder(t)
	body, err := b.buildEscapeWheel()
	require.NoError(t, err)
	require.True(t, body.IsSolid())

	f, err := r3.NewFrame(r3.NewVec(0, 0, 23.5), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	o, err := body.SectionOutline(t.Context(), f, units.Millimeters(0.02))
	require.NoError(t, err)
	require.Len(t, o.Loops, 8, "outer, six windows and the bore")
	off := o.OffSurface.Base()
	require.Less(t, off, 0.05)

	// 30 tips on r 40.
	outer := outerLoop(o.Loops)
	teeth, err := toothPolys(outer)
	require.NoError(t, err)
	require.Len(t, teeth, Teeth)
	for _, tooth := range teeth {
		require.InDelta(t, TipRadius, tipOf(tooth).Len(), 0.03+off)
	}

	// The pinion at z 10 has 13 teeth reaching the tip radius.
	f10, err := r3.NewFrame(r3.NewVec(0, 0, 10), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	po, err := body.SectionOutline(t.Context(), f10, units.Millimeters(0.02))
	require.NoError(t, err)
	pin := outerLoop(po.Loops)
	tip, root := 0.0, math.Inf(1)
	count := 0
	for i, p := range pin {
		r := p.Len()
		tip, root = math.Max(tip, r), math.Min(root, r)
		high := func(q Pt) bool { return q.Len() > (11.23+7.73)/2 }
		if high(p) && !high(pin[(i+len(pin)-1)%len(pin)]) {
			count++
		}
	}
	require.Equal(t, PinionTeeth, count)
	require.InDelta(t, 11.23, tip, 0.03+po.OffSurface.Base())
	require.InDelta(t, 7.73, root, 0.05+po.OffSurface.Base())
}
