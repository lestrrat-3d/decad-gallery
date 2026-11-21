package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentFits measures the fits of "Part fits" that do not need the assembled model: each is read on the part
// bodies built alone, in the world frame at digit 2 with the camshaft at theta 72 left unposed (theta 0).
func TestSevenSegmentFits(t *testing.T) {
	r := newTestRun(t)
	_, shaft := sharedBody(t, "shaft", (*run).shaft)

	t.Run("P1 cam bores on the shaft", func(t *testing.T) {
		for _, seg := range Segments() {
			_, cam := sharedCam(t, seg)
			y := r.dm.camDiscFace(seg) + 3.0 // mid-disc, away from the groove
			camLoops := sectionAt(t, cam, y)
			var bore [][]r3.Vec
			for _, l := range camLoops {
				if c, e := centroidExtent(l); c.Len() < 1 && e < 8 {
					bore = append(bore, l)
				}
			}
			require.Len(t, bore, 1, "cam %s", seg)
			gap := loopsGap(bore, sectionAt(t, shaft, y))
			// The bore is 0.05 wider than the shaft's round and its flat.
			require.InDelta(t, PressFit, gap, fitBound, "cam %s", seg)
			requireDoubleD(t, bore, camBoreFlat, "cam "+seg.String())
		}
		_, ratchet := sharedBody(t, "ratchet wheel", (*run).ratchetWheel)
		var bore [][]r3.Vec
		for _, l := range sectionAt(t, ratchet, -43.0) {
			if c, e := centroidExtent(l); c.Len() < 1 && e < 8 {
				bore = append(bore, l)
			}
		}
		require.Len(t, bore, 1, "ratchet wheel")
		require.InDelta(t, PressFit, loopsGap(bore, sectionAt(t, shaft, -43.0)), fitBound, "ratchet wheel")
		requireDoubleD(t, bore, ratchetBoreFlat, "ratchet wheel")
	})

	t.Run("L5 cam rims to the guide plate", func(t *testing.T) {
		_, cam := sharedCam(t, SegD)
		box, err := cam.Bounds()
		require.NoError(t, err)
		_, guide := sharedBody(t, "slotted guide plate", (*run).guidePlate)
		gbox, err := guide.Bounds()
		require.NoError(t, err)
		require.InDelta(t, 1.0, gbox.Min.Z-box.Max.Z, 0.01)
	})

	t.Run("R2 pin tip to the groove floor", func(t *testing.T) {
		_, cam := sharedCam(t, SegD)
		_, slider := sharedSlider(t, SegD)
		sbox, err := slider.Bounds()
		require.NoError(t, err)
		face := r.dm.camDiscFace(SegD)
		floor := face + GrooveDepth
		require.InDelta(t, 0.40, floor-sbox.Max.Y, 0.011)
		// The floor is where the section turns from groove to material at the groove's mid radius, digit 0 (d is lit).
		for _, d := range []struct {
			y     float64
			solid bool
		}{{floor - 0.1, false}, {floor + 0.1, true}} {
			loops := sectionAt(t, cam, d.y)
			require.Equal(t, d.solid, insideLoops(loops, polar(OnRadius, 90)), "y %.2f", d.y)
		}
	})

	t.Run("L1 stem to its cam face", func(t *testing.T) {
		_, slider := sharedSlider(t, SegD)
		face := r.dm.camDiscFace(SegD)
		loops := planSection(t, slider, 56)
		stemMax := math.Inf(-1)
		for _, l := range loops {
			for _, p := range l {
				stemMax = math.Max(stemMax, p.Y)
			}
		}
		require.InDelta(t, LooseFit, face-stemMax, fitBound)
	})

	t.Run("T3 stems in the guide slots, T9 blades in their tubes", func(t *testing.T) {
		_, guide := sharedBody(t, "slotted guide plate", (*run).guidePlate)
		_, top := sharedBody(t, "top plate", (*run).flagTopPlate)
		guideLoops := planSection(t, guide, 56)
		tubeLoops := planSection(t, top, 81)
		for _, seg := range Segments() {
			_, slider := sharedSlider(t, seg)
			stem := planSection(t, slider, 56)
			// Only the slot of this station is near the stem.
			near := nearLoops(guideLoops, r.dm.Station(seg))
			require.InDelta(t, TightFit, loopsGap(stem, near), fitBound, "stem %s", seg)
			blade := planSection(t, slider, 81)
			tube := nearCap(tubeLoops, blade)
			// The blade's straight walls keep 0.25 across the hole (the Tight fit; 0.35 along it). Its square corners keep at
			// least the Tight minimum 0.2 from the hole's corner rounds of 0.50: 0.21 (instructions.md "Blade corners").
			_, _, horizontal := segmentCentre(seg)
			walls := 1e9
			for _, p := range densify(blade, 0.05) {
				along, lo, hi := p.Y, r.dm.Station(seg)-1.2, r.dm.Station(seg)+2.4
				if horizontal {
					along, lo, hi = p.X, -1.5, 1.5
				}
				if p.Z != 0 || along < lo || along > hi {
					continue
				}
				if _, _, ok := bladePoint(seg, r.dm.Station(seg), p); ok {
					walls = math.Min(walls, nearest(p, tube))
				}
			}
			require.InDelta(t, bladeAcross, walls, fitBound, "blade walls %s", seg)
			require.GreaterOrEqual(t, loopsGap(blade, tube), 0.20, "blade corners %s", seg)
		}
	})

	t.Run("P3 registration pegs in their holes", func(t *testing.T) {
		_, guide := sharedBody(t, "guide plate", (*run).columnGuidePlate)
		_, top := sharedBody(t, "top plate", (*run).flagTopPlate)
		_, cradle := sharedBody(t, "cradle plate", (*run).cradlePlate)
		// The column pegs stand on the guide plate and the top pegs hang from the top plate. Each section is cut where the
		// peg is a plain cylinder of radius 1.5 and the hole in the cradle a plain cylinder of radius 1.55: below the top
		// pegs' collars and the mouth cones.
		for _, c := range []struct {
			name  string
			body  *decad.Body
			sites [2][2]float64
			z     float64
		}{{"column peg", guide, columnPegs, 81.0}, {"top peg", top, topPegs, 81.6}} {
			pegs, holes := planSection(t, c.body, c.z), planSection(t, cradle, c.z)
			for _, site := range c.sites {
				at := r3.Vec{X: site[0], Y: site[1]}
				peg, hole := loopAt(pegs, at, 3.0), loopAt(holes, at, 3.1)
				require.NotNil(t, peg, "%s at %v", c.name, site)
				require.NotNil(t, hole, "%s hole at %v", c.name, site)
				require.InDelta(t, PressFit, loopsGap([][]r3.Vec{peg}, [][]r3.Vec{hole}), fitBound, "%s at %v", c.name, site)
			}
		}
	})

	t.Run("L7 detent plate to the d stem", func(t *testing.T) {
		_, slider := sharedSlider(t, SegD)
		stem := planSection(t, slider, 56)
		low := math.Inf(1)
		for _, l := range stem {
			for _, p := range l {
				low = math.Min(low, p.Y)
			}
		}
		require.InDelta(t, 2.45, low-detentFace, 0.011)
	})
}

// TestSevenSegmentPressWalls requires 1.6 mm or more of wall round each press-fit bore: the cam hubs, ratchet wheel,
// collar ring and knob, read at sections through the bore.
func TestSevenSegmentPressWalls(t *testing.T) {
	r := newTestRun(t)
	check := func(name string, body *decad.Body, y float64) {
		loops := sectionAt(t, body, y)
		var bore, others [][]r3.Vec
		for _, l := range loops {
			if c, e := centroidExtent(l); c.Len() < 1 && e < 8 {
				bore = append(bore, l)
			} else {
				others = append(others, l)
			}
		}
		require.Len(t, bore, 1, name)
		require.GreaterOrEqual(t, loopsGap(bore, others), 1.6-fitBound, name)
	}
	for _, seg := range Segments() {
		_, cam := sharedCam(t, seg)
		// The hub, away from the disc: its wall is the hub radius less the bore (the marks lie in the disc).
		check("cam hub "+seg.String(), cam, r.dm.camDiscFace(seg)-3)
	}
	_, ratchet := sharedBody(t, "ratchet wheel", (*run).ratchetWheel)
	check("ratchet wheel", ratchet, -43.0)
	_, ring := sharedBody(t, "collar ring", (*run).collarRing)
	check("collar ring", ring, -59.5)
	_, knob := sharedBody(t, "knob", (*run).knob)
	check("knob", knob, -65)
}

// TestSevenSegmentPinClearance sweeps the camshaft through a whole turn in 3 degree steps and checks, for every cam, that
// the pin (radius 1.5 on the groove's centreline at the follower's polar angle, 90 + theta) keeps RegularFit from the
// groove's walls. The cam is built unrotated; turning it by theta moves the follower to cam-local angle 90 + theta.
func TestSevenSegmentPinClearance(t *testing.T) {
	r := newTestRun(t)
	for _, seg := range Segments() {
		_, cam := sharedCam(t, seg)
		// The pin's tip end lies in the groove: read at 1.0 mm below the groove face.
		loops := sectionAt(t, cam, r.dm.camDiscFace(seg)+1.0)
		for theta := 0.0; theta < 360; theta += 3 {
			psi := 90 + theta
			centre := polar(r.dm.GrooveRadius(seg, psi), psi)
			var ring [][]r3.Vec
			var pts []r3.Vec
			for a := 0.0; a < 360; a += 10 {
				pts = append(pts, centre.Add(polar(sliderPinR, a)))
			}
			ring = append(ring, pts)
			require.GreaterOrEqual(t, loopsGap(ring, loops), RegularFit-fitBound-0.02, "cam %s theta %.0f", seg, theta)
		}
	}
}

// TestSevenSegmentBladeRoof reads the gap between a slider's blade ridge, lifted to its lit height, and the roof of its
// tube's hole in the top plate, both from the built bodies: 0.90 by design (modeled; 0.80 in the physical geometry).
// A blade built taller than designed closes it.
func TestSevenSegmentBladeRoof(t *testing.T) {
	r := newTestRun(t)
	_, top := sharedBody(t, "top plate", (*run).flagTopPlate)
	for _, seg := range Segments() {
		_, slider := sharedSlider(t, seg)
		box, err := slider.Bounds()
		require.NoError(t, err)
		h := bladeHole(seg, r.dm.Station(seg))
		cx, cy := (h[0]+h[1])/2, (h[2]+h[3])/2
		lo, hi := 86.0, 88.0 // void at lo (the hole), material at hi (the roof)
		require.False(t, inBody(t, top, r3.Vec{X: cx, Y: cy, Z: lo}), "blade hole %s", seg)
		require.True(t, inBody(t, top, r3.Vec{X: cx, Y: cy, Z: hi}), "blade hole roof %s", seg)
		for hi-lo > 0.004 {
			m := (lo + hi) / 2
			if inBody(t, top, r3.Vec{X: cx, Y: cy, Z: m}) {
				hi = m
			} else {
				lo = m
			}
		}
		gap := (lo+hi)/2 - (box.Max.Z + Travel)
		require.InDelta(t, 0.90, gap, 0.03, "blade ridge to the tube roof, slider %s", seg)
	}
}

// TestSevenSegmentJournalFit (T5) measures the journal fit on the built bodies: the radius of segment g's stub in the black
// body and the radius of its half-hole in the cradle plate, each found by bisection along a radius at u = 13 (the middle
// of the stub). Their difference is the Tight minimum 0.20, to the tessellation bound.
func TestSevenSegmentJournalFit(t *testing.T) {
	f := sharedFlag(t, SegG)
	_, cradle := sharedBody(t, "cradle plate", (*run).cradlePlate)
	// The stub: material inside the radius, along the a axis of the rotor at u = 13.
	lo, hi := 0.3, 0.9
	for hi-lo > 0.002 {
		m := (lo + hi) / 2
		if inBody(t, f.Black, rotorPoint(SegG, m, 0, 13.0)) {
			lo = m
		} else {
			hi = m
		}
	}
	stub := (lo + hi) / 2
	// The half-hole: void inside the radius below the axis, material outside it.
	lo, hi = 0.5, 1.3
	for hi-lo > 0.002 {
		m := (lo + hi) / 2
		if inBody(t, cradle, r3.Vec{X: 13.0, Y: 0, Z: RotorAxisZ - m}) {
			hi = m
		} else {
			lo = m
		}
	}
	hole := (lo + hi) / 2
	t.Logf("stub radius %.3f, half-hole radius %.3f, clearance %.3f", stub, hole, hole-stub)
	require.InDelta(t, JournalRadius, stub, 0.02, "the stub radius")
	require.InDelta(t, journalHoleRadius, hole, 0.02, "the half-hole radius")
	require.InDelta(t, 0.20, hole-stub, 0.03, "the journal clearance (T5)")
}

// TestSevenSegmentStackPlay bounds the cam stack's play along the shaft (instructions.md "Cam stack, axial play"), read on
// the built bodies. Rearward the stack stops where cam a's D chord meets the square end of the shaft's flats, each part
// PressFit behind the next, so a cam can move back by the gaps behind it (cam d 0.35). Forward nothing but the pin tips
// stops it. Pushed fully back, with its slider leaned forward as far as the stem slot and the blade hole allow, every groove
// pin keeps 0.75 or more of engagement in its groove (computed 0.77, cam d unlit).
func TestSevenSegmentStackPlay(t *testing.T) {
	r := newTestRun(t)
	_, guide := sharedBody(t, "slotted guide plate", (*run).guidePlate)
	gbox, err := guide.Bounds()
	require.NoError(t, err)
	_, ratchet := sharedBody(t, "ratchet wheel", (*run).ratchetWheel)
	rbox, err := ratchet.Bounds()
	require.NoError(t, err)
	// The flats' square end: the cam section is a double D just inside it and round just past it.
	_, shaft := sharedBody(t, "shaft", (*run).shaft)
	_, maxZ, _ := extent(sectionAt(t, shaft, shaftCamFlatTo-0.02))
	require.InDelta(t, shaftCamFlat, maxZ, 0.02)
	_, maxZ, _ = extent(sectionAt(t, shaft, shaftCamFlatTo+0.02))
	require.InDelta(t, shaftCamRadius, maxZ, 0.02)
	back, rear := shaftCamFlatTo, 0.0
	worst := math.Inf(1)
	for i := len(stationOrder) - 1; i >= 0; i-- {
		seg := stationOrder[i]
		_, cam := sharedCam(t, seg)
		box, err := cam.Bounds()
		require.NoError(t, err)
		require.InDelta(t, PressFit, back-box.Max.Y, 0.01, "gap behind cam %s", seg)
		rear += back - box.Max.Y
		back = box.Min.Y
		_, slider := sharedSlider(t, seg)
		sbox, err := slider.Bounds()
		require.NoError(t, err)
		_, _, horizontal := segmentCentre(seg)
		blade := bladeAlong
		if horizontal {
			blade = bladeAcross
		}
		for _, lift := range []float64{0, Travel} {
			pin, ridge := sliderPinZ+lift, bladeRidgeZ+lift
			lean := TightFit + (TightFit+blade)*(gbox.Min.Z-pin)/(ridge-gbox.Min.Z)
			engage := sbox.Max.Y - (r.dm.camDiscFace(seg) + rear) - lean
			t.Logf("cam %s lift %.0f: back %.2f, slider lean %.3f, engagement %.3f", seg, lift, rear, lean, engage)
			worst = math.Min(worst, engage)
		}
	}
	require.InDelta(t, PressFit, back-rbox.Max.Y, 0.01, "gap between cam d and the ratchet wheel")
	require.InDelta(t, 0.35, rear, 0.02, "cam d's rearward travel")
	require.GreaterOrEqual(t, worst, 0.75, "the least groove pin engagement")
}
