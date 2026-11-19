package sevensegment

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// extent returns the largest |x| and the largest z (local) of the section loops, and the largest radius.
func extent(loops [][]r3.Vec) (maxX, maxZ, maxR float64) {
	for _, l := range loops {
		for _, p := range l {
			maxX = math.Max(maxX, math.Abs(p.X))
			maxZ = math.Max(maxZ, p.Y)
			maxR = math.Max(maxR, p.Len())
		}
	}
	return
}

func buildShaft(t *testing.T) (*run, *decad.Body) {
	t.Helper()
	r, b := sharedBody(t, "shaft", (*run).shaft)
	require.True(t, b.IsSolid())
	return r, b
}

func TestSevenSegmentShaft(t *testing.T) {
	_, b := buildShaft(t)
	box, err := b.Bounds()
	require.NoError(t, err)
	require.InDelta(t, -72.2, box.Min.Y, 1e-6)
	require.InDelta(t, 48.0, box.Max.Y, 1e-6)
	// The cam section is a D: round 3.2 below the flat, flat 2.7 above, until the flat's square end at 39.2.
	_, maxZ, maxR := extent(sectionAt(t, b, 39.0))
	require.InDelta(t, shaftCamFlat, maxZ, 0.02)
	require.InDelta(t, 3.2, maxR, 0.02)
	_, maxZ, _ = extent(sectionAt(t, b, 39.4))
	require.InDelta(t, 3.2, maxZ, 0.02)
	_, maxZ, _ = extent(sectionAt(t, b, -58.1))
	require.InDelta(t, shaftStubFlat, maxZ, 0.02)
	_, maxZ, _ = extent(sectionAt(t, b, -57.6))
	require.InDelta(t, 2.5, maxZ, 0.02)
	// Journals are round 2.5.
	_, maxZ, maxR = extent(sectionAt(t, b, -52))
	require.InDelta(t, 2.5, maxZ, 0.02)
	require.InDelta(t, 2.5, maxR, 0.02)
}

func TestSevenSegmentCollars(t *testing.T) {
	_, b := buildShaft(t)
	// Front step: diameter 6.4 just behind -49.25.
	maxX, _, _ := extent(sectionAt(t, b, -49.0))
	require.InDelta(t, 3.2, maxX, 0.02)
	_, _, maxR := extent(sectionAt(t, b, -49.5))
	require.InDelta(t, 2.5, maxR, 0.02)
	// Rear collar: cone 39.7 to 40.739, cylinder of diameter 7.6 to 42.75: 3.05 long.
	_, _, maxR = extent(sectionAt(t, b, 39.65))
	require.InDelta(t, 3.2, maxR, 0.02)
	_, _, maxR = extent(sectionAt(t, b, 41.5))
	require.InDelta(t, 7.6/2, maxR, 0.02)
	_, _, maxR = extent(sectionAt(t, b, 42.8))
	require.InDelta(t, 2.5, maxR, 0.02)
	require.InDelta(t, 3.05, 42.75-39.7, 1e-9)
	// Collar ring: 3.5 long, diameter 12.
	_, ring := sharedBody(t, "collar ring", (*run).collarRing)
	box, err := ring.Bounds()
	require.NoError(t, err)
	require.InDelta(t, CollarRingLength, box.Max.Y-box.Min.Y, 1e-6)
	require.InDelta(t, -61.4, box.Min.Y, 1e-6)
	require.InDelta(t, -57.9, box.Max.Y, 1e-6)
	require.InDelta(t, 12, box.Max.X-box.Min.X, 0.01)
}

func TestSevenSegmentKnob(t *testing.T) {
	_, k := sharedBody(t, "knob", (*run).knob)
	require.True(t, k.IsSolid())
	box, err := k.Bounds()
	require.NoError(t, err)
	require.InDelta(t, -73.45, box.Min.Y, 1e-6)
	require.InDelta(t, -61.45, box.Max.Y, 1e-6)
	// The blind bore's floor is at -72.25: a section just in front of it is solid across the axis.
	loops := sectionAt(t, k, -72.5)
	require.True(t, insideLoops(loops, r3.Vec{}))
	loops = sectionAt(t, k, -70)
	require.False(t, insideLoops(loops, r3.Vec{}))
}

func TestSevenSegmentRatchet(t *testing.T) {
	r, w := sharedBody(t, "ratchet wheel", (*run).ratchetWheel)
	require.True(t, w.IsSolid())
	mid := -37.85 - RatchetThickness/2
	loops := sectionAt(t, w, mid)
	minR, maxR := math.Inf(1), 0.0
	for _, l := range loops {
		c, _ := centroidExtent(l)
		if c.Len() > 1 { // skip the D bore
			continue
		}
		_ = c
	}
	var outline []r3.Vec
	for _, l := range loops {
		for _, p := range l {
			if p.Len() > 5 {
				outline = append(outline, p)
			}
		}
	}
	require.NotEmpty(t, outline)
	for _, p := range outline {
		minR, maxR = math.Min(minR, p.Len()), math.Max(maxR, p.Len())
	}
	require.GreaterOrEqual(t, minR, RatchetRoot-0.02)
	// The 0.6 round on the 50 degree flank lifts the lowest outline point about 0.9 above the sharp corner at 6.5.
	require.LessOrEqual(t, minR, RatchetRoot+1.0)
	require.InDelta(t, RatchetTip, maxR, 0.02)
	// Ten lock faces: radial runs where the outline radius drops from the tip to the root within 0.5 degrees.
	drops := 0
	prevR := 0.0
	for a := 0.0; a < 360; a += 0.05 {
		rr, _ := voidEdge(loops, a)
		if prevR-rr > 2 {
			drops++
		}
		prevR = rr
	}
	require.Equal(t, RatchetTeeth, drops)
	// Away from the root rounds the outline radius matches RatchetRadius.
	for a := 0.0; a < 360; a += 3 {
		rel := math.Mod(a-r.dm.LockPhase+720, 36)
		if rel < 6 || rel > 35 {
			continue
		}
		rr, ok := voidEdge(loops, a)
		require.True(t, ok)
		require.InDelta(t, r.dm.RatchetRadius(a), rr, 0.02, "angle %.1f", a)
	}
}

// voidEdge returns the first radius along the ray at angle a, from 5 outward, where the section leaves its material.
func voidEdge(loops [][]r3.Vec, a float64) (float64, bool) {
	for rad := 5.0; rad < 14; rad += 0.002 {
		if !insideLoops(loops, polar(rad, a)) {
			return rad, true
		}
	}
	return 14, false
}
