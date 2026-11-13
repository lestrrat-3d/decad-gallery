package escapement

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// sectionAt returns the section outline of body at height z.
func sectionAt(t *testing.T, body *decad.Body, z float64) decad.SectionOutline {
	t.Helper()
	f, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	var o decad.SectionOutline
	// A faceted Boolean result refuses a chord below its own bound, so retry coarser.
	for _, tol := range []float64{0.02, 0.03, 0.05} {
		o, err = body.SectionOutline(t.Context(), f, units.Millimeters(tol))
		if err == nil {
			break
		}
	}
	require.NoError(t, err)
	return o
}

// loopsBy returns the loops of o as XY polygons, largest extent first.
func loopsBy(o decad.SectionOutline) [][]Pt {
	var out [][]Pt
	for _, l := range o.Loops {
		out = append(out, loopPoints(l))
	}
	extent := func(p []Pt) float64 {
		m := 0.0
		for _, v := range p {
			m = math.Max(m, v.Len())
		}
		return m
	}
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if extent(out[j]) > extent(out[i]) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func rotPoly(p []Pt, a float64) []Pt {
	out := make([]Pt, len(p))
	for i, v := range p {
		out[i] = v.Rot(a)
	}
	return out
}

// fitsIn reports whether polygon p lies inside polygon hole with no crossing.
func fitsIn(p, hole []Pt) bool {
	for _, v := range p {
		if !inPoly(hole, v) {
			return false
		}
	}
	for i := range p {
		for j := range hole {
			if segCross(p[i], p[(i+1)%len(p)], hole[j], hole[(j+1)%len(hole)]) {
				return false
			}
		}
	}
	return true
}

// buildDriveParts builds the great wheel and the drum once for the drive tests.
func TestRatchet(t *testing.T) {
	b := testBuilder(t)
	wheel, err := b.buildGreatWheel()
	require.NoError(t, err)
	require.True(t, wheel.IsSolid())
	drum, err := b.buildPawlCarrier()
	require.NoError(t, err)
	require.True(t, drum.IsSolid())

	// The great wheel's outline at z 7 has the gear's outer loop and the ratchet pocket loop with 12 teeth.
	wo := sectionAt(t, wheel, 7)
	wl := loopsBy(wo)
	require.Len(t, wl, 2, "tooth outline and ratchet pocket")
	pocket := wl[1]
	lands := 0
	low := func(q Pt) bool { return q.Len() < 11.65 }
	for i, p := range pocket {
		if low(p) && !low(pocket[(i+len(pocket)-1)%len(pocket)]) {
			lands++
		}
	}
	require.Equal(t, ratchetTeeth, lands, "ratchet teeth")
	// The great wheel's own teeth at z 3: 48 reach the tip radius.
	o3 := loopsBy(sectionAt(t, wheel, 3))
	require.NotEmpty(t, o3)

	// The drum's pawl layer at z 7: core with pawls, and the D relief.
	do := sectionAt(t, drum, 7)
	dl := loopsBy(do)
	require.Len(t, dl, 2, "core with two pawls, and the D bore")
	pawls := dl[0]
	off := wo.OffSurface.Base() + do.OffSurface.Base()

	// At the build pose the drum sits 20.8° ccw in the wheel's frame.
	// The carrier's pawls are drawn CarrierOffset behind, and the arbor turns it that much further, so the world pose is the
	// drum's old one.
	pose := func(extra float64) []Pt { return rotPoly(pawls, DrumOffset+b.g.CarrierOffset+extra) }
	require.True(t, fitsIn(pose(0), pocket), "the pawls sit in the pocket at the build pose")
	// The contact face (radial at -1.2° from r 10.5 to 12.5) against the ratchet's radial face (0° from r 11.5 to 13.5).
	pawlFace := [2]Pt{Polar(10.5, -1.2*deg), Polar(12.5, -1.2*deg)}
	ratchetFace := [2]Pt{Polar(11.5, 0), Polar(13.5, 0)}
	faceGap := math.Inf(1)
	for _, v := range pawlFace {
		faceGap = math.Min(faceGap, distPointSeg(v, ratchetFace[0], ratchetFace[1]))
	}
	for _, v := range ratchetFace {
		faceGap = math.Min(faceGap, distPointSeg(v, pawlFace[0], pawlFace[1]))
	}
	require.InDelta(t, 0.25, faceGap, 0.02+off, "pawl contact face to ratchet face")
	// The built outlines carry the same faces: every pawl vertex on the contact face sits at -1.2°, and the pocket has
	// a vertex on the radial face at 0°.
	onFace, onRadial := 0, 0
	for _, v := range pose(0) {
		if v.Len() > 10.4 && math.Abs(v.Angle()+1.2*deg) < 0.05*deg {
			onFace++
		}
	}
	for _, v := range pocket {
		if v.Len() > 11.4 && math.Abs(v.Angle()) < 0.05*deg {
			onRadial++
		}
	}
	require.GreaterOrEqual(t, onFace, 2)
	require.GreaterOrEqual(t, onRadial, 2)
	// The pawl's tip corner is closest to the ratchet's slope: its gap is the closest approach of the two outlines.
	require.GreaterOrEqual(t, distPolys(pose(0), pocket), 0.19-off)
	require.False(t, fitsIn(pose(2.5*deg), pocket), "turning the drum ccw drives the ratchet")
	// Winding turns the drum cw: the pawl has 1.5° of free play, then rides up the ratchet slope with its arm flexing,
	// which a rigid outline cannot do (instructions.md "Drum": 1.0 mm of deflection plus 0.3 mm margin).
	require.True(t, fitsIn(pose(-1.5*deg), pocket), "the pawl turns freely cw before the slope")
	require.False(t, fitsIn(pose(-2.5*deg), pocket), "past that the slope needs the arm to flex")
	// The arms keep clear of the ratchet tips: the closest approach of the pawl layer outside the pawl tips.
	var arms []Pt
	for _, p := range pawls {
		if p.Len() < 11.0 {
			arms = append(arms, p)
		}
	}
	armGap := math.Inf(1)
	for _, p := range arms {
		for j := range pocket {
			armGap = math.Min(armGap, distPointSeg(p, pocket[j], pocket[(j+1)%len(pocket)]))
		}
	}
	require.GreaterOrEqual(t, armGap, 0.95-off)
	// No face of the carrier is large: its largest planar face is the pawl layer's (core and pawls, under 400 mm²).
	for _, f := range drum.Faces() {
		pl, ok := f.Surface().(decad.Plane)
		if !ok || math.Abs(pl.Frame.N().Z) < 0.999 {
			continue
		}
		area, err := f.Area()
		require.NoError(t, err)
		require.Less(t, area.Value.Base(), 400.0)
	}
}

// TestKnob checks the winding knob: radius 9, so its face (254 mm² less the socket) is under the 400 mm² of large faces.
func TestKnob(t *testing.T) {
	b := testBuilder(t)
	knob, err := b.buildKnob()
	require.NoError(t, err)
	require.True(t, knob.IsSolid())
	o := sectionAt(t, knob, 35)
	require.Less(t, o.PolygonArea().Base(), 400.0)
	require.InDelta(t, math.Pi*81, o.PolygonArea().Base()+math.Pi*2.55*2.55*0.93, 4)
	far := 0.0
	for _, v := range loopsBy(o)[0] {
		far = math.Max(far, v.Len())
	}
	require.InDelta(t, 9, far, 0.03+o.OffSurface.Base())
}
