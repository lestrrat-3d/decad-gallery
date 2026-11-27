package sevensegment

// CI: isolated-tests

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestSevenSegmentFrame(t *testing.T) {
	t.Parallel()
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
				require.InDelta(t, frontWallOuterY, box.Min.Y, 1e-6)
				require.InDelta(t, frontInnerY+frontLedgeDeep, box.Max.Y, 1e-6)
			} else {
				require.InDelta(t, rearInnerY-rearLedgeDeep, box.Min.Y, 1e-6) // the rear ledge ribs reach rearLedgeDeep in from the inner face
				require.InDelta(t, rearInnerY+WallThickness, box.Max.Y, 1e-6)
			}
			require.InDelta(t, wallTopZ, box.Max.Z, 1e-6)
			require.InDelta(t, wallBottomZ, box.Min.Z, 1e-6)
			require.InDelta(t, -wallHalf, box.Min.X, 1e-6)
			require.InDelta(t, wallHalf, box.Max.X, 1e-6)
		})
	}
}

// TestSevenSegmentPlates checks the guide plate's slots and pin notches.
func TestSevenSegmentPlates(t *testing.T) {
	t.Parallel()
	r, g := sharedBody(t, "slotted guide plate", (*run).guidePlate)
	require.True(t, g.IsSolid())
	box, err := g.Bounds()
	require.NoError(t, err)
	require.InDelta(t, guideFrom, box.Min.Z, 1e-6)
	require.InDelta(t, guideTop, box.Max.Z, 1e-6)
	// Slot a: the pin notch is open at the station's +Y side.
	loops := planSection(t, g, guideMidZ)
	y := r.dm.Station(SegA)
	notchX, notchY, slotX := 2.7+0.1, StemThickness/2+PinLength+0.4, 5.25+TightFit // the notch's half width and reach, the slot's half width
	require.False(t, insideLoops(loops, polar2(0, y)))
	require.False(t, insideLoops(loops, polar2(0, y+notchY-0.75)))
	require.True(t, insideLoops(loops, polar2(notchX+0.9, y+notchY-0.75)))
	require.False(t, insideLoops(loops, polar2(slotX-0.75, y)))
}

// TestSevenSegmentFlagPlates builds the cradle, top plate and guide plate of the flag mechanism and checks their bounds,
// their bores and pivot pockets, the window lip and the blade holes, their walls and their corners.
func TestSevenSegmentFlagPlates(t *testing.T) {
	t.Parallel()
	// The three plates build independently, so they build together: the group returns once all three are cached.
	var cradle, top, guide *decad.Body
	var r *run
	t.Run("build", func(t *testing.T) {
		t.Run("cradle plate", func(t *testing.T) {
			t.Parallel()
			_, cradle = sharedBody(t, "cradle plate", (*run).cradlePlate)
		})
		t.Run("top plate", func(t *testing.T) {
			t.Parallel()
			_, top = sharedBody(t, "top plate", (*run).flagTopPlate)
		})
		t.Run("guide plate", func(t *testing.T) {
			t.Parallel()
			r, guide = sharedBody(t, "guide plate", (*run).columnGuidePlate)
		})
	})
	require.False(t, t.Failed(), "the plates build")
	for name, b := range map[string]*decad.Body{"cradle": cradle, "top": top, "guide": guide} {
		require.True(t, b.IsSolid(), name)
	}
	bounds := func(b *decad.Body) decad.Box {
		box, err := b.Bounds()
		require.NoError(t, err)
		return box
	}
	cb, tb, gb := bounds(cradle), bounds(top), bounds(guide)
	require.InDelta(t, cradleFrom, cb.Min.Z, 0.01)
	require.InDelta(t, RotorAxisZ, cb.Max.Z, 0.01)
	require.InDelta(t, tubeFrom, tb.Min.Z, 0.01)
	require.InDelta(t, flagFaceZ, tb.Max.Z, 0.01)
	require.InDelta(t, guideFrom, gb.Min.Z, 0.01)
	for name, box := range map[string]decad.Box{"cradle": cb, "top": tb, "guide": gb} {
		require.InDelta(t, -WallHalfWidth, box.Min.X, 0.01, name)
		require.InDelta(t, WallHalfWidth, box.Max.X, 0.01, name)
	}
	require.InDelta(t, columnTop+3.9, gb.Max.Z, 0.01)

	// Sections normal to u through segment g's centre (u = X): the bore, the pivot pockets, the lip and the blade hole.
	o := r3.Vec{Z: RotorAxisZ}
	// Bore in the cradle at x = 0: at 2.4 under the axis: void 0.11 inside the bore's chord, material 0.11 beyond it.
	cl := planeSection(t, cradle, o, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	at := func(y, z float64) r3.Vec { return r3.Vec{X: y, Y: z - RotorAxisZ} }
	dz := 2.4
	chord := math.Sqrt(boreRadius*boreRadius - dz*dz)
	require.False(t, insideLoops(cl, at(chord-0.11, RotorAxisZ-dz)), "bore")
	require.True(t, insideLoops(cl, at(chord+0.11, RotorAxisZ-dz)), "bore wall")
	// Pivot pockets at u = 13.5: the cradle's half is open upward and 1.0 deep under the axis's seam (floor at z 81.0), the top
	// plate's open downward and reaches z 84.45 (1.6 above its underside); both are 3.3 wide, with the corners rounded 0.5.
	floor := RotorAxisZ - pivotPocketLow
	up := pivotPocketHigh - Seam // the roof above the top plate's underside
	j := r3.Vec{X: pivotWall + 1.5, Z: RotorAxisZ}
	cj := planeSection(t, cradle, j, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	require.False(t, insideLoops(cj, at(0, floor+0.2)), "cradle pocket")
	require.False(t, insideLoops(cj, at(pivotPocketHalf-0.05, RotorAxisZ-0.4)), "cradle pocket side")
	require.True(t, insideLoops(cj, at(0, floor-0.1)), "cradle under the pocket")
	require.True(t, insideLoops(cj, at(pivotPocketHalf+0.05, RotorAxisZ-0.4)), "cradle beside the pocket")
	require.True(t, insideLoops(cj, at(pivotPocketHalf-0.05, floor+0.05)), "cradle in the pocket's rounded corner")
	tj := planeSection(t, top, r3.Vec{X: pivotWall + 1.5, Z: topFrom}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	require.False(t, insideLoops(tj, r3.Vec{X: 0, Y: up - 0.1}), "top plate pocket")
	require.False(t, insideLoops(tj, r3.Vec{X: pivotPocketHalf - 0.05, Y: 0.5}), "top plate pocket side")
	require.True(t, insideLoops(tj, r3.Vec{X: 0, Y: up + 0.1}), "top plate over the pocket")
	require.True(t, insideLoops(tj, r3.Vec{X: pivotPocketHalf + 0.05, Y: 0.5}), "top plate beside the pocket")
	// Window lip: the window half-width is 2.95 and the bore chord at 2.95 leaves a lip of 0.704 under the display face (listed
	// in instructions.md as thinner than 0.8).
	tw := planeSection(t, top, r3.Vec{Z: topFrom}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	z := func(v float64) float64 { return v - topFrom }
	lipBore := topFrom + math.Sqrt(boreRadius*boreRadius-(windowHalfWidth+0.1)*(windowHalfWidth+0.1)) // the bore's surface at the lip
	require.False(t, insideLoops(tw, r3.Vec{X: windowHalfWidth - 0.1, Y: z(lipBore + 0.32)}), "window")
	require.True(t, insideLoops(tw, r3.Vec{X: windowHalfWidth + 0.1, Y: z(lipBore + 0.32)}), "lip")
	require.False(t, insideLoops(tw, r3.Vec{X: windowHalfWidth + 0.1, Y: z(lipBore - 0.18)}), "bore under the lip")
	// Blade hole of a, g, d: y in [-8.05, -6.35] for g (cy 0), up to the tube roof.
	require.False(t, insideLoops(tw, r3.Vec{X: -10.8, Y: z(topFrom + 3.225)}), "blade hole")
	require.True(t, insideLoops(tw, r3.Vec{X: -10.8, Y: z(flagFaceZ - 0.65)}), "blade hole roof")
	require.True(t, insideLoops(tw, r3.Vec{X: -9.0, Y: z(topFrom + 3.225)}), "tube wall beside the blade hole")

	// Walls of 0.799 or more between holes, at the heights of each plate.
	for _, off := range []float64{0.02, 0.6, 1.4, 1.85, 2.2, 2.5, 2.58} { // heights over the cradle's underside, as fractions of 2.6 of its thickness
		h := cradleFrom + off*cradleThick/2.6
		t.Run(fmt.Sprintf("cradle z %.2f", h), func(t *testing.T) {
			t.Parallel()
			require.GreaterOrEqual(t, minWall(t, cradle, h, 1.5), 0.799, "cradle z %.2f", h)
		})
	}
	for _, off := range []float64{0.05, 1.6, 3.6, 5.0, 6.1, 8.1, 9.45, 10.0, 10.8} { // over the tubes' bottom, as fractions of 10.85 of their length to the face
		h := tubeFrom + off*(flagFaceZ-tubeFrom)/10.85
		t.Run(fmt.Sprintf("top plate z %.2f", h), func(t *testing.T) {
			t.Parallel()
			require.GreaterOrEqual(t, minWall(t, top, h, 1.5), 0.799, "top plate z %.2f", h)
		})
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
