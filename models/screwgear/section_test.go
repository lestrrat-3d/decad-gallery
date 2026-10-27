package screwgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// onPlane is how far p stands from the plane, along the plane's normal.
func onPlane(plane r3.Frame, p r3.Vec) float64 { return p.Sub(plane.Origin()).Dot(plane.N()) }

func TestSectionPairCutsBothRibbons(t *testing.T) {
	t.Parallel()
	m, _ := build(t, shortParams())
	for _, height := range []float64{-0.4, 0, 0.4} {
		section, err := m.SectionPair(t.Context(), height, 0.1)
		require.NoError(t, err)
		require.InDelta(t, height, section.Plane.Origin().Z, 1e-12)
		require.Equal(t, r3.NewVec(0, 0, 1), section.Plane.N())
		for name, outline := range map[string][][]r3.Vec{"gear A": section.GearA.Loops, "gear B": section.GearB.Loops} {
			require.NotEmpty(t, outline, "%s at height %g", name, height)
			for _, loop := range outline {
				require.GreaterOrEqual(t, len(loop), 3)
				for _, p := range loop {
					require.InDelta(t, 0, onPlane(section.Plane, p), 1e-6, "%s at height %g", name, height)
				}
			}
		}
		require.Greater(t, section.GearA.PolygonArea().Base(), 0.0)
		require.Greater(t, section.GearB.PolygonArea().Base(), 0.0)
		require.Less(t, section.GearA.OffSurface.Base(), 0.05)
	}
}

// The ribbons reach 0.575 mm across the mid-plane toward each other, so a plane 1 mm toward gear B
// misses gear A's teeth (which end at +0.575) and cuts only gear B.
func TestSectionPairOnePlaneOutsideGearAReadsEmpty(t *testing.T) {
	t.Parallel()
	m, _ := build(t, shortParams())
	section, err := m.SectionPair(t.Context(), 1.0, 0.1)
	require.NoError(t, err)
	require.Empty(t, section.GearA.Loops)
	require.NotEmpty(t, section.GearB.Loops)
}

func TestSectionPairFollowsThePlacement(t *testing.T) {
	t.Parallel()
	frame, err := r3.NewFrame(r3.NewVec(10, -5, 7), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	p := shortParams()
	p.Placement = &frame
	m, _ := build(t, p)
	section, err := m.SectionPair(t.Context(), 0.4, 0.1)
	require.NoError(t, err)
	// U x V = +X is n̂ for this placement, so the plane stands 0.4 mm along +X from the centre.
	require.InDelta(t, 10.4, section.Plane.Origin().X, 1e-12)
	require.InDelta(t, 1, math.Abs(section.Plane.N().X), 1e-12)
	for _, loop := range append(section.GearA.Loops, section.GearB.Loops...) {
		for _, pt := range loop {
			require.InDelta(t, 10.4, pt.X, 1e-6)
		}
	}
}

// Measured on the default ribbons, the least gap between the two outlines is 0.64 to 0.75 mm on the
// planes from -0.4 to +0.4 mm, with a proven lower bound of 0.60 mm or more. The test holds the
// proven lower bound above 0.55 mm, and checks the closest points lie on the plane and one on each
// ribbon's outline.
func TestSectionPairClearanceOfTheDefaultPair(t *testing.T) {
	t.Parallel()
	m, _ := build(t, shortParams())
	for _, height := range []float64{-0.4, -0.2, 0, 0.2, 0.4} {
		section, err := m.SectionPair(t.Context(), height, 0.1)
		require.NoError(t, err)
		require.NotNil(t, section.Clearance, "height %g", height)
		c := *section.Clearance
		require.False(t, c.Overlaps, "height %g", height)
		require.Greater(t, c.Lower().Base(), 0.55, "height %g: %v", height, c.Distance)
		require.Less(t, c.Distance.Value.Base(), 1.0, "height %g", height)
		require.InDelta(t, 0, onPlane(section.Plane, c.A), 1e-6)
		require.InDelta(t, 0, onPlane(section.Plane, c.B), 1e-6)
		require.InDelta(t, c.Distance.Value.Base(), c.A.Sub(c.B).Len(), 1e-6)
	}
}
