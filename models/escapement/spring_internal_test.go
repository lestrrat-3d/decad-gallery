package escapement

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

func TestSpringLayout(t *testing.T) {
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	require.Equal(t, 9, g.Spring.Turns)
	require.InDelta(t, 2.6, g.Spring.Pitch, 1e-4)
	require.InDelta(t, 1.1, g.Spring.Gap, 1e-4)
	require.InDelta(t, 8.4, g.Spring.FirstInnerRadius, 1e-4)
	require.InDelta(t, 1142.3, g.Spring.ActiveLength, 0.1)
	require.InDelta(t, 34.0015, math.Hypot(g.Spring.EyeCenterX, g.Spring.EyeCenterY), 1e-4)
	require.InDelta(t, 3.4, g.Spring.EyeRadius, 1e-4)
	require.InDelta(t, 3.8, g.Spring.EyeHoleDiameter, 1e-4)
	require.InDelta(t, 66.7916, g.SpringTurn/deg, 1e-4)
	require.InDelta(t, 41.7639, g.CarrierOffset/deg, 1e-4)
	require.InDelta(t, 0, g.SpringAnchor.X, 1e-4)
	require.InDelta(t, -79.7515, g.SpringAnchor.Y, 1e-4)
	require.InDelta(t, 3.10, g.SpringRate, 0.01)
	require.InDelta(t, 43.9, g.FullWindTorque, 0.01*43.9+0.01)
	require.InDelta(t, 0.0093, g.FullWindStrain, 0.0001)
	require.InDelta(t, 1.84, g.RunTurns, 0.01)
	// The offset is the same for every pose: it comes from the default parameters.
	p := DefaultParams()
	p.Phase = 0.6
	g2, err := Layout(p)
	require.NoError(t, err)
	require.InDelta(t, g.CarrierOffset, g2.CarrierOffset, 1e-12)
}

// TestSpringRoot checks the mainspring's root: fillets of 0.55 mm, over the 0.5 mm floor of "Corners and roots", and
// a taper offset of 0.0692 mm that thickens the strip to 1.638 mm at the spoke.
func TestSpringRoot(t *testing.T) {
	dm, err := SpringParams().Derive()
	require.NoError(t, err)
	require.GreaterOrEqual(t, dm.FilletRadius, 0.5)
	require.InDelta(t, 0.55, dm.FilletRadius, 1e-4)
	require.InDelta(t, 0.0692, dm.TaperOffset, 1e-4)
	require.InDelta(t, 1.638, SpringParams().Thickness+2*dm.TaperOffset, 1e-3)
	require.InDelta(t, 0.962, dm.Gap-2*dm.TaperOffset, 1e-3)
}

func TestMainspring(t *testing.T) {
	b := testBuilder(t)
	spring, err := b.buildMainspring()
	require.NoError(t, err)
	require.True(t, spring.IsSolid())
	bb, err := spring.Bounds()
	require.NoError(t, err)
	const slack = 3e-3
	require.GreaterOrEqual(t, bb.Min.X, -32.0-slack)
	require.LessOrEqual(t, bb.Max.X, 31.22+slack)
	require.GreaterOrEqual(t, bb.Min.Y, -83.16-slack)
	require.LessOrEqual(t, bb.Max.Y, -13.75+slack)
	require.InDelta(t, 11.25, bb.Min.Z, slack)
	require.InDelta(t, 17.25, bb.Max.Z, slack)

	// The eye's hole is a loop of the section at mid-height whose points lie 1.9 mm from the anchor pin's centre.
	o := sectionAt(t, spring, 14.25)
	found := false
	for _, l := range loopsBy(o) {
		var lo, hi = math.Inf(1), 0.0
		for _, v := range l {
			d := v.Sub(b.g.SpringAnchor).Len()
			lo, hi = math.Min(lo, d), math.Max(hi, d)
		}
		if hi < 3 {
			require.InDelta(t, 1.9, lo, 0.02+o.OffSurface.Base())
			require.InDelta(t, 1.9, hi, 0.02+o.OffSurface.Base())
			found = true
		}
	}
	require.True(t, found, "the eye's hole")
	require.NotEmpty(t, b.sketches, "the item's profile sketch is recorded")
}

func TestPawlCarrier(t *testing.T) {
	b := testBuilder(t)
	carrier, err := b.buildPawlCarrier()
	require.NoError(t, err)
	require.True(t, carrier.IsSolid())
	// The pawl tips: the farthest vertices of the pawl layer's outline, turned by DrumOffset + CarrierOffset into the great
	// wheel's frame, lie where the old drum's did: (12.5 at g - 116°) and (12.5 at g - 112°) with g = 90° and 270°.
	outline := loopsBy(sectionAt(t, carrier, 7))[0]
	want := []float64{-116 + 90, -112 + 90, -116 + 270, -112 + 270}
	for _, w := range want {
		best := math.Inf(1)
		for _, v := range outline {
			if v.Len() < 12.4 {
				continue
			}
			a := normAng(v.Rot(DrumOffset+b.g.CarrierOffset).Angle() - w*deg - DrumOffset)
			best = math.Min(best, math.Abs(a))
		}
		require.Less(t, best, 3e-3, "pawl tip at %v°", w)
	}
	// Each slot ends in a full round of radius 0.75 about (8.25 at g - 5.209°): its centre lies in the slot's air, and
	// a point 0.7 mm from it toward the larger angles is air while a point 0.8 mm from it is material.
	for _, g := range []float64{90*deg - b.g.CarrierOffset, 270*deg - b.g.CarrierOffset} {
		c := g - slotEndRadius/(carrierCoreRadius+slotEndRadius)
		centre := Polar(carrierCoreRadius+slotEndRadius, c)
		along := Pt{-math.Sin(c), math.Cos(c)}
		air, wall := centre.Add(Pt{0.7 * along.X, 0.7 * along.Y}), centre.Add(Pt{0.8 * along.X, 0.8 * along.Y})
		require.False(t, inside(t, carrier, air.X, air.Y, 7.25), "slot end round, air side")
		require.True(t, inside(t, carrier, wall.X, wall.Y, 7.25), "slot end round, wall")
	}
	require.True(t, inside(t, carrier, -4.2, 0, 10), "the boss exists at z 10")
	require.True(t, inside(t, carrier, -4.9, 0, 10), "the boss reaches r 5.0")
	require.False(t, inside(t, carrier, -5.1, 0, 10), "the boss ends at r 5.0")
	require.False(t, inside(t, carrier, -4.2, 0, 11.1), "the boss ends at z 11")
}

// TestFrontPlateSpringAnchor checks the blind socket for the spring anchor pin: open from the plate's inner face, with
// the spine solid beside it and plate above it.
func TestFrontPlateSpringAnchor(t *testing.T) {
	b := testBuilder(t)
	front, err := b.buildFrontPlate()
	require.NoError(t, err)
	q := b.g.SpringAnchor
	require.False(t, inside(t, front, q.X+1.0, q.Y, 29), "the socket is open")
	require.True(t, inside(t, front, q.X+2.5, q.Y, 29), "the spine stands beside the socket")
	require.True(t, inside(t, front, q.X, q.Y, 33), "the plate closes the socket's top")
}

// TestSpringFits checks the fits of the mainspring and its parts on the default build.
func TestSpringFits(t *testing.T) {
	_, m, err := sharedDefault(t)
	require.NoError(t, err)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	const loose = 0.5

	// S1: the hub's D bore on the arbor's flat, Tight: 0.30 at the build pose (TestPoseNoOverlap keeps 0.2 at every phase).
	// The spring's outline is one long loop whose Clearance bound is wide, so the bore is measured by its own vertices.
	so, ao := sectionAt(t, m.Mainspring, 14.25), sectionAt(t, m.GreatWheelArbor, 14.25)
	arbor := loopsBy(ao)[0]
	for i := range arbor {
		arbor[i] = arbor[i].Sub(g.G)
	}
	bore := math.Inf(1)
	for _, l := range loopsBy(so) {
		for _, v := range l {
			v = v.Sub(g.G)
			if v.Len() > 4 {
				continue
			}
			for j := range arbor {
				bore = math.Min(bore, distPointSeg(v, arbor[j], arbor[(j+1)%len(arbor)]))
			}
		}
	}
	require.InDelta(t, 0.30, bore, 0.03+so.OffSurface.Base()+ao.OffSurface.Base(), "S1 hub bore on the arbor")
	// S2: the eye on the anchor pin, Regular 0.40.
	pinOutline := loopsBy(sectionAt(t, m.SpringAnchorPin, 14.25))
	require.InDelta(t, 0.4, springGap(so, g.SpringAnchor, 0, 4, pinOutline), 0.03+so.OffSurface.Base(), "S2 eye on the anchor pin")
	// S3: the hub's faces stand 0.25 from the carrier boss and the retainer.
	axial(t, m.Mainspring, 11.25, m.PawlCarrier, 11.0, 0.25, "S3 hub to the carrier boss")
	axial(t, m.Mainspring, 17.25, m.SpringRetainer, 17.5, 0.25, "S3 hub to the retainer")
	// S4: the anchor pin in its socket, press 0.05.
	radial(t, m.FrontPlate, m.SpringAnchorPin, 29.5, 0.05, 0, "S4 anchor pin in the front plate")
	// S6: the relaxed coils clear the pinion tips by 2.52. The spring and the pinion meet at z 11.25 to 17.25.
	wheel := loopsBy(sectionAt(t, m.EscapeWheel, 14.25))
	require.GreaterOrEqual(t, springGap(so, g.G, 4, 100, wheel), 2.52-0.1, "S6 coils to the pinion tips")
	// L1: the spring to the escape wheel, 4.75 axially; L7: to the great wheel face 2.75, to the pawl layer 2.0 and the
	// anchor pin's tip to the great wheel face 1.75.
	axial(t, m.EscapeWheel, 22, m.Mainspring, 17.25, 4.75, "L1 spring to the escape wheel")
	axial(t, m.Mainspring, 11.25, m.GreatWheel, 8.5, 2.75, "L7 spring to the great wheel face")
	axial(t, m.Mainspring, 11.25, m.PawlCarrier, 9.25, 2.0, "L7 spring to the pawl layer")
	axial(t, m.SpringAnchorPin, 10.25, m.GreatWheel, 8.5, 1.75, "L7 anchor pin to the great wheel face")
	// L8: the coils other than the eye keep 0.5 or more from the anchor pin at every height.
	require.GreaterOrEqual(t, springGap(so, g.SpringAnchor, 4, 100, pinOutline), loose, "L8 coils to the anchor pin")
}

// springGap is the least distance from the spring's outline vertices that lie between rMin and rMax from centre to the
// polygons other. A mainspring's outline is one long loop whose Clearance bound is wide, so fits are measured from its
// vertices; the outline's own bound is added by the callers' tolerances.
func springGap(so decad.SectionOutline, centre Pt, rMin, rMax float64, other [][]Pt) float64 {
	best := math.Inf(1)
	for _, l := range loopsBy(so) {
		for _, v := range l {
			if d := v.Sub(centre).Len(); d < rMin || d > rMax {
				continue
			}
			for _, o := range other {
				for j := range o {
					best = math.Min(best, distPointSeg(v, o[j], o[(j+1)%len(o)]))
				}
			}
		}
	}
	return best
}
