package escapement

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestEscapeWheel(t *testing.T) {
	b := testBuilder(t)
	body, err := b.buildEscapeWheel()
	require.NoError(t, err)
	require.True(t, body.IsSolid())

	f, err := r3.NewFrame(r3.NewVec(0, 0, 23.5), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	o, err := outlineAt(t, body, f)
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
	po, err := outlineAt(t, body, f10)
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

	// The front counterbore: radius 3.75 from the floor at z 23.95 to the face at 25, its rim chamfered from z 24.5.
	require.False(t, inside(t, body, 3.5, 0, 24.2), "the front counterbore is open at z 24.2")
	require.True(t, inside(t, body, 3.9, 0, 24.2), "material round the front counterbore")
	require.True(t, inside(t, body, 3.5, 0, 23.5), "the front counterbore's floor")
	require.True(t, inside(t, body, 3.5, 0, 22.5), "material under the front counterbore")
	require.False(t, inside(t, body, 3.9, 0, 24.9), "the rim chamfer opens the counterbore at the face")
	// No rear counterbore: at the pinion's rear end only the D bore (arc r 3.35, chord x 2.60) is open.
	require.False(t, inside(t, body, -3.2, 0, 2.4), "the D bore is open at z 2.4")
	require.True(t, inside(t, body, -3.5, 0, 2.4), "material round the bore at the rear end")
	require.True(t, inside(t, body, 3.5, 0, 2.4), "no counterbore at the chord side")
	// No boss stands behind the pinion.
	bb, err := body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 1.5, bb.Min.Z, 2e-3, "the pinion's rear face is the wheel's rearmost face")
}
