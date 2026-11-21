package sevensegment

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestSevenSegmentFrame(t *testing.T) {
	for _, front := range []bool{true, false} {
		name := "rear"
		if front {
			name = "front"
		}
		t.Run(name, func(t *testing.T) {
			_, w := sharedWall(t, front)
			require.True(t, w.IsSolid())
			box, err := w.Bounds()
			require.NoError(t, err)
			if front {
				require.InDelta(t, -57.5, box.Min.Y, 1e-6)
				require.InDelta(t, -30.8, box.Max.Y, 1e-6)
			} else {
				require.InDelta(t, 33.0, box.Min.Y, 1e-6) // the ribs reach 10 mm in from the inner face at 43
				require.InDelta(t, 51.0, box.Max.Y, 1e-6)
			}
			require.InDelta(t, 81.25, box.Max.Z, 1e-6)
			require.InDelta(t, 0, box.Min.Z, 1e-6)
		})
	}
}

// TestSevenSegmentPlates checks the guide plate's slots and pin notches.
func TestSevenSegmentPlates(t *testing.T) {
	r, g := sharedBody(t, "slotted guide plate", (*run).guidePlate)
	require.True(t, g.IsSolid())
	box, err := g.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 54, box.Min.Z, 1e-6)
	require.InDelta(t, 58, box.Max.Z, 1e-6)
	// Slot a: the pin notch is open at the station's +Y side.
	loops := planSection(t, g, 56)
	y := r.dm.Station(SegA)
	require.False(t, insideLoops(loops, polar2(0, y)))
	require.False(t, insideLoops(loops, polar2(0, y+3.5)))
	require.True(t, insideLoops(loops, polar2(2.8, y+3.5)))
	require.False(t, insideLoops(loops, polar2(3.0, y)))
}

// TestSevenSegmentFlagPlates builds the cradle, top plate and guide plate of the flag mechanism and checks their bounds,
// their bores and half-holes, the window lip and the blade holes, their walls and their corners.
func TestSevenSegmentFlagPlates(t *testing.T) {
	_, cradle := sharedBody(t, "cradle plate", (*run).cradlePlate)
	_, top := sharedBody(t, "top plate", (*run).flagTopPlate)
	r, guide := sharedBody(t, "guide plate", (*run).columnGuidePlate)
	for name, b := range map[string]*decad.Body{"cradle": cradle, "top": top, "guide": guide} {
		require.True(t, b.IsSolid(), name)
	}
	bounds := func(b *decad.Body) decad.Box {
		box, err := b.Bounds()
		require.NoError(t, err)
		return box
	}
	cb, tb, gb := bounds(cradle), bounds(top), bounds(guide)
	require.InDelta(t, 80.0, cb.Min.Z, 0.01)
	require.InDelta(t, 82.6, cb.Max.Z, 0.01)
	require.InDelta(t, 77.4, tb.Min.Z, 0.01)
	require.InDelta(t, flagFaceZ, tb.Max.Z, 0.01)
	require.InDelta(t, 54.0, gb.Min.Z, 0.01)
	require.InDelta(t, columnTop+2.6, gb.Max.Z, 0.01)

	// Sections normal to u through segment g's centre (u = X): the bore, the journal half-holes, the lip and the blade hole.
	o := r3.Vec{Z: RotorAxisZ}
	// Bore in the cradle at x = 0: void 5.2 across at z 81.0 (the chord is 5.31), material beyond 5.42.
	cl := planeSection(t, cradle, o, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	at := func(y, z float64) r3.Vec { return r3.Vec{X: y, Y: z - RotorAxisZ} }
	require.False(t, insideLoops(cl, at(5.2, 81.0)), "bore")
	require.True(t, insideLoops(cl, at(5.42, 81.0)), "bore wall")
	// Journal half-holes at u = 13.5: the cradle's open upward, the top plate's open downward.
	j := r3.Vec{X: 13.5, Z: RotorAxisZ}
	cj := planeSection(t, cradle, j, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	require.False(t, insideLoops(cj, at(0, 82.2)), "cradle half-hole")
	require.True(t, insideLoops(cj, at(0, 81.6)), "cradle under the half-hole")
	require.True(t, insideLoops(cj, at(0.95, 82.2)), "cradle beside the half-hole")
	tj := planeSection(t, top, r3.Vec{X: 13.5, Z: topFrom}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	require.False(t, insideLoops(tj, r3.Vec{X: 0, Y: 0.4}), "top plate half-hole")
	require.True(t, insideLoops(tj, r3.Vec{X: 0, Y: 1.0}), "top plate over the half-hole")
	// Window lip: the window half-width is 2.95 and the bore chord at 2.95 leaves a lip of 0.704 under the display face (listed
	// in instructions.md as thinner than 0.8).
	tw := planeSection(t, top, r3.Vec{Z: topFrom}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	z := func(v float64) float64 { return v - topFrom }
	require.False(t, insideLoops(tw, r3.Vec{X: 2.85, Y: z(87.8)}), "window")
	require.True(t, insideLoops(tw, r3.Vec{X: 3.05, Y: z(87.8)}), "lip")
	require.False(t, insideLoops(tw, r3.Vec{X: 3.05, Y: z(87.3)}), "bore under the lip") // the bore surface is at 87.48 there
	// Blade hole of a, g, d: y in [-8.05, -6.35] for g (cy 0), up to z 87.45.
	require.False(t, insideLoops(tw, r3.Vec{X: -7.2, Y: z(85)}), "blade hole")
	require.True(t, insideLoops(tw, r3.Vec{X: -7.2, Y: z(87.6)}), "blade hole roof")
	require.True(t, insideLoops(tw, r3.Vec{X: -6.0, Y: z(85)}), "tube wall beside the blade hole")

	// Walls of 0.799 or more between holes, at the heights of each plate.
	for _, h := range []float64{80.02, 80.6, 81.4, 81.85, 82.2, 82.5, 82.58} {
		require.GreaterOrEqual(t, minWall(t, cradle, h, 1.5), 0.799, "cradle z %.2f", h)
	}
	for _, h := range []float64{77.45, 79.0, 81.0, 82.4, 83.5, 85.5, 86.85, 87.4, 88.2} {
		require.GreaterOrEqual(t, minWall(t, top, h, 1.5), 0.799, "top plate z %.2f", h)
	}

	// Corners: the cradle and the top plate have none; the guide plate only the hex pocket corners.
	fs, err := cornercheck.Check(t.Context(), cradle, r3.Vec{Z: 1}, nil)
	require.NoError(t, err)
	require.Empty(t, fs, cornercheck.Report(fs, 5))
	fs, err = cornercheck.Check(t.Context(), top, r3.Vec{Z: -1}, nil)
	require.NoError(t, err)
	require.Empty(t, fs, cornercheck.Report(fs, 5))
	var ex []cornercheck.Exempt
	for i := 4; i < 8; i++ {
		ex = append(ex, cornercheck.Exempt{Reason: "hex nut pocket corners match the nut's sharp corners",
			Near: cornercheck.Annulus(r.dm.Sites[i].Head, r3.Vec{Z: 1}, 2.21, 2.81)})
	}
	fs, err = cornercheck.Check(t.Context(), guide, r3.Vec{Z: 1}, ex)
	require.NoError(t, err)
	require.Empty(t, fs, cornercheck.Report(fs, 5))
}
