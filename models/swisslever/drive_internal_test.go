package swisslever

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKnob(t *testing.T) {
	b := partBuilder(t)
	knob, err := b.buildKnob()
	require.NoError(t, err)
	require.True(t, knob.IsSolid())
	requireSketchesConstrained(t, b, 3)
	requireBoundsZ(t, knob, knobLow, knobHigh, knobRadius)

	// The keyed rear counterbore is open below z 36.30 inside its D outline; its floor is 1.0 thick.
	require.False(t, partInside(t, knob, -3.5, 0, 36.0), "keyed counterbore")
	require.True(t, partInside(t, knob, -3.9, 0, 36.0), "material round the counterbore")
	require.True(t, partInside(t, knob, -3.5, 0, 36.5), "the counterbore's floor")
	require.True(t, partInside(t, knob, 3.4, 0, 36.0), "beyond the counterbore's flat")
	require.GreaterOrEqual(t, knobHigh-(keyedCounterLow+keyedCounterLong), 1.0-1e-9)
	// The D socket goes through the floor.
	require.False(t, partInside(t, knob, -1.9, 0, 36.8), "socket arc")
	require.True(t, partInside(t, knob, -2.3, 0, 36.8), "floor round the socket")
	require.False(t, partInside(t, knob, 1.4, 0, 36.8), "socket flat")
	require.True(t, partInside(t, knob, 1.7, 0, 36.8), "floor past the socket flat")

	// The front face's area stays under 400 mm² (no chamfer).
	o := partSection(t, knob, 37.0)
	require.Less(t, o.PolygonArea().Base(), 400.0)
	require.InDelta(t, math.Pi*knobRadius*knobRadius-socketArea(), o.PolygonArea().Base(), 4)
	// Wall round the counterbore at its arc.
	require.GreaterOrEqual(t, knobRadius-keyedCounterR, 1.6)
	requireCorners(t, knob)
}

// socketArea returns the area of the end-family D socket.
func socketArea() float64 {
	return dArea(EndBoreRadius, EndBoreFlat)
}

// dArea returns the area of the D outline of radius r and flat x = f.
func dArea(r, f float64) float64 {
	return math.Pi*r*r - (r*r*math.Acos(f/r) - f*math.Sqrt(r*r-f*f))
}

func TestKnobCollar(t *testing.T) {
	b := partBuilder(t)
	ring, err := b.buildKnobCollar()
	require.NoError(t, err)
	require.True(t, ring.IsSolid())
	requireSketchesConstrained(t, b, 2)
	requireBoundsZ(t, ring, collarLow, collarLow+collarLong, collarRadius)
	require.GreaterOrEqual(t, collarLong, 3.0)
	// The collar fits the knob's keyed counterbore with 0.05 round: arc 3.75 - 3.7, flat 3.35 - 3.3.
	require.InDelta(t, 0.05, keyedCounterR-collarRadius, 1e-9)
	require.InDelta(t, 0.05, keyedCounterFlat-collarFlat, 1e-9)
	require.False(t, partInside(t, ring, -1.9, 0, 34.5), "bore arc")
	require.True(t, partInside(t, ring, -2.3, 0, 34.5), "wall")
	require.False(t, partInside(t, ring, 1.4, 0, 34.5), "bore flat")
	require.True(t, partInside(t, ring, 1.7, 0, 34.5), "chord wall")
	require.False(t, partInside(t, ring, 3.4, 0, 34.5), "beyond the chord")
	requireCorners(t, ring)
}
