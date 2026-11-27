package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/internal/splitcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentGroovePinSplit applies "Split shafts" to the separate groove pin of a, g and d: its seam plane has normal +X
// (it holds the pin's axis and the main load direction Z), the halves are mirror images with each half 2.25 high, no
// face leans past 45 degrees with a half on its seam face, and "Corners and roots" reads no finding on the halves at the split
// heights. No exemption.
func TestSevenSegmentGroovePinSplit(t *testing.T) {
	t.Parallel()
	for _, seg := range groovePinSegments() {
		t.Run(seg.String(), func(t *testing.T) {
			r, pin := sharedGroovePin(t, seg)
			require.True(t, pin.IsSolid())
			point := r3.Vec{Y: r.dm.Station(seg) + StemThickness/2 - groovePinShank, Z: sliderPinZ}
			rep, err := splitcheck.Check(t.Context(), splitcheck.Shaft{Body: pin, Point: point, Axis: r3.Vec{Y: 1}, Normal: r3.Vec{X: 1}}, 0.09)
			require.NoError(t, err)
			require.Empty(t, rep.Violations)
			require.InDelta(t, 1.0, math.Abs(rep.Normal.Dot(r3.Vec{X: 1})), 1e-6, "the seam's normal is +X")
			require.Zero(t, rep.Bridges)
			require.Less(t, rep.Overhang, 1.0, "no downward face on a half's seam face")
			require.InDelta(t, PinRadius, rep.HalfHeights[0], 0.01)
			require.InDelta(t, PinRadius, rep.HalfHeights[1], 0.01)
			require.Less(t, rep.Asymmetry, rep.AsymmetryLimit)
			heights, err := cornercheck.SplitHeights(point, r3.Vec{X: 1}, PinRadius)
			require.NoError(t, err)
			fs, err := cornercheck.CheckAt(t.Context(), pin, r3.Vec{X: 1}, heights, nil)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 8))
			t.Logf("pin %s: overhang %.1f, half heights %.3f / %.3f, asymmetry %.4f (limit %.3f)", seg, rep.Overhang, rep.HalfHeights[0],
				rep.HalfHeights[1], rep.Asymmetry, rep.AsymmetryLimit)
		})
	}
}

// TestSevenSegmentGroovePinShape reads the pin on sections normal to Y: its volume 109.56 mm3, the round Ø4.5 over its working
// part, the wings' reach 2.397 inside the stem (the flanks' crossing of the 30 degree cone ends 0.1 inside the face) and no
// wing outside it, and the root and tip against the hole's floor and the groove's floor.
func TestSevenSegmentGroovePinShape(t *testing.T) {
	t.Parallel()
	for _, seg := range groovePinSegments() {
		t.Run(seg.String(), func(t *testing.T) {
			r, pin := sharedGroovePin(t, seg)
			vol, err := pin.Volume()
			require.NoError(t, err)
			require.InDelta(t, 109.56, vol.Value.Base(), 0.5)
			yFace := r.dm.Station(seg) + StemThickness/2
			box, err := pin.Bounds()
			require.NoError(t, err)
			// The root is 0.2 from the hole's floor and the tip 0.40 from the groove's floor.
			require.InDelta(t, yFace-groovePinShank, box.Min.Y, 1e-6)
			require.InDelta(t, yFace-groovePinShank-groovePinFloorGap, yFace-groovePinShank-0.2, 1e-9)
			require.InDelta(t, 0.40, r.dm.camDiscFace(seg)+GrooveDepth-box.Max.Y, 1e-6)
			for _, c := range []struct {
				at, want, tol float64
			}{{-1.0, 2.397, 0.04}, {-0.5, 2.397, 0.04}, {-0.2, 2.305, 0.04}, {-0.1, PinRadius, 0.01}, {0.1, PinRadius, 0.01},
				{0.6, PinRadius, 0.01}, {2.0, PinRadius, 0.01}} {
				got := largestRadius(pinSection(t, pin, yFace+c.at))
				require.InDelta(t, c.want, got, c.tol, "largest radius %+.1f from the stem's face", c.at)
			}
		})
	}
}

// TestSevenSegmentGroovePinFits measures the pin in its keyed hole on the built bodies: 0.2 of clearance round the round part
// (the 0.2 mm Tight minimum of "Part clearances"), 0.2 from the root to the hole's floor (the root chamfer 0.55 plus the floor round 0.75 make the 0.2 gap at the
// corner), at least 1.0 of stem material round the hole 1.0 inside the face, and at least 0.8 wherever the stem's 0.5 edge
// chamfers meet its face.
func TestSevenSegmentGroovePinFits(t *testing.T) {
	t.Parallel()
	for _, seg := range groovePinSegments() {
		t.Run(seg.String(), func(t *testing.T) {
			r, slider := sharedSlider(t, seg)
			_, pin := sharedGroovePin(t, seg)
			yFace := r.dm.Station(seg) + StemThickness/2
			for _, at := range []float64{-2.0, -0.3, -0.05} {
				gap := loopsGap(pinSection(t, pin, yFace+at), sectionAt(t, slider, yFace+at))
				require.InDelta(t, groovePinClearance, gap, fitBound, "pin to hole at %+.2f from the face", at)
			}
			// Near the root the chamfer opens the gap: 0.499 at 3.9 inside the face, 0.334 at 3.6.
			for _, c := range []struct{ at, want float64 }{{-3.9, 0.499}, {-3.6, 0.334}} {
				gap := loopsGap(pinSection(t, pin, yFace+c.at), sectionAt(t, slider, yFace+c.at))
				require.InDelta(t, c.want, gap, 0.05, "pin to hole at %+.2f from the face", c.at)
			}
			for _, c := range []struct{ at, least float64 }{{-3.9, 1.0}, {-3.6, 1.0}, {-2.0, 1.0}, {-0.3, 0.8}, {-0.05, 0.8}} {
				hole, other := holeLoops(sectionAt(t, slider, yFace+c.at))
				require.Len(t, hole, 1, "hole at %+.2f from the face", c.at)
				require.GreaterOrEqual(t, loopsGap(hole, other), c.least-fitBound, "wall round the hole at %+.2f", c.at)
			}
			// The hole's roof and floor: material 0.1 past the floor, void 0.1 before it.
			y := yFace - groovePinShank - groovePinFloorGap
			require.True(t, inBody(t, slider, r3.Vec{X: 0, Y: y - 0.1, Z: sliderPinZ}))
			require.False(t, inBody(t, slider, r3.Vec{X: 0, Y: y + 0.3, Z: sliderPinZ}))
			// The pin never reaches the stem's -Y face.
			require.True(t, inBody(t, slider, r3.Vec{X: 0, Y: r.dm.Station(seg) - StemThickness/2 + 0.1, Z: sliderPinZ}))
		})
	}
}

// TestSevenSegmentGroovePinSketches requires every sketch of the pin and of the slider with its keyed hole to report
// sketch.FullyConstrained with DOF 0, no redundant constraint and no conflict.
func TestSevenSegmentGroovePinSketches(t *testing.T) {
	t.Parallel()
	for _, seg := range groovePinSegments() {
		t.Run(seg.String(), func(t *testing.T) {
			pr, _ := sharedGroovePin(t, seg)
			require.NotEmpty(t, pr.sketches)
			for i, s := range pr.sketches {
				require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "pin sketch %d", i)
			}
			sr, _ := sharedSlider(t, seg)
			require.NotEmpty(t, sr.sketches)
			for i, s := range sr.sketches {
				require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "slider sketch %d", i)
			}
		})
	}
}

// TestSevenSegmentSliderPrintOverhang requires that, printed on its flat face (a, g, d on -X; b to f on -Y), no face of a slider
// leans more than 45 degrees from vertical: the keyed hole's roof is a 45 degree V, its floor is rounded, and nothing
// is cut by a downward face of more than 0.0 mm2 past 45 degrees.
func TestSevenSegmentSliderPrintOverhang(t *testing.T) {
	t.Parallel()
	for _, seg := range Segments() {
		t.Run(seg.String(), func(t *testing.T) {
			_, slider := sharedSlider(t, seg)
			up := r3.Vec{Y: 1}
			if _, _, horizontal := segmentCentre(seg); horizontal {
				up = r3.Vec{X: 1}
			}
			worst, steep := overhangOf(t, slider, up)
			t.Logf("slider %s: steepest downward face %.1f degrees, %.4f mm2 past 45", seg, worst, steep)
			require.LessOrEqual(t, worst, 45.1)
			require.Less(t, steep, 0.01)
		})
	}
}
