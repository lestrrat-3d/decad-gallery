package spinningwheel

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

type brakeCordReading struct {
	volume, volumeBound  float64
	minY, maxY, boxBound float64
}

// readBrakeCord builds the real brake cord alone with the arguments brake() passes it.
func readBrakeCord(t *testing.T, p Params) brakeCordReading {
	t.Helper()
	doc := NewDocument()
	b, err := newAssemblyBuilder(t.Context(), doc, p)
	require.NoError(t, err)
	s, travel, shift := p.Scale, p.BrakeTravel, b.a.State.HeadShift
	x, z := -140*s, 600*s
	radius := brakePitch(s)
	d := brakeShift(s)
	leftEnd := -285*s + d + shift + travel/2
	rightEnd := -285*s + d + shift - travel/2
	phase := 2 * math.Pi * travel / 0.7
	body := b.brakeCord(x-radius, x+radius, leftEnd, rightEnd, z, radius, phase)
	require.NoError(t, b.err)
	require.NotNil(t, body)
	require.True(t, body.IsSolid())
	volume, err := body.Volume()
	require.NoError(t, err)
	box, err := body.Bounds()
	require.NoError(t, err)
	return brakeCordReading{
		volume: volume.Value.Base(), volumeBound: volume.Bound.Base(),
		minY: box.Min.Y, maxY: box.Max.Y, boxBound: box.Bound.Base(),
	}
}
