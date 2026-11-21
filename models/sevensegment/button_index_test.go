package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/fastenertool"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestButtonIndexInvalid(t *testing.T) {
	for _, phase := range []float64{-1, 361, math.NaN(), math.Inf(1)} {
		_, err := buildButtonIndex(t.Context(), decad.New(), phase)
		require.Error(t, err)
	}
	_, err := buildButtonIndex(nil, decad.New(), 0) //nolint:staticcheck // Check nil-context rejection.
	require.Error(t, err)
	_, err = buildButtonIndex(t.Context(), nil, 0)
	require.Error(t, err)
	require.Equal(t, 0.0, indexAdvance(108))
	require.Equal(t, 36.0, indexAdvance(252))
	require.InDelta(t, 18, indexAdvance(180), 1e-10)
}

func TestButtonIndexMotion(t *testing.T) {
	doc := decad.New()
	wheel, err := indexWheel(t.Context(), doc)
	require.NoError(t, err)
	initial, err := r3.RotationAround(r3.Vec{Z: 28}, r3.Vec{Y: 1}, units.Degrees(324))
	require.NoError(t, err)
	wheel, err = decad.Transform(t.Context(), wheel, initial)
	require.NoError(t, err)
	frame, err := indexFrame(0)
	require.NoError(t, err)
	rivet, err := indexRivet(t.Context(), doc, frame)
	require.NoError(t, err)
	a := math.Atan2(24, 10)
	A := a + 3*math.Pi/5
	B := math.Atan2(20*math.Sin(A)-24, 20*math.Cos(A)-10)
	throw := 26 * math.Sin(math.Pi/10)
	center := frame.Origin().Add(frame.U().Scale(throw * math.Cos(a-B))).Add(frame.V().Scale(throw * math.Sin(a-B)))
	gaugeFrame, err := r3.NewFrame(center.Add(r3.Vec{Y: 4.4}), frame.U(), frame.V())
	require.NoError(t, err)
	gauge, _, err := fastenertool.Cylinder(t.Context(), doc, gaugeFrame, 1.725, 4)
	require.NoError(t, err)
	wm, rm, gm := buttonIndexData(t, wheel), buttonIndexData(t, rivet), buttonIndexData(t, gauge)
	for _, phi := range []float64{0, 104, 108, 115, 122, 150, 180, 210, 238, 245, 252, 256, 354} {
		input, err := r3.RotationAround(frame.Origin(), r3.Vec{Y: -1}, units.Degrees(phi))
		require.NoError(t, err)
		r, g := rm.moved(t, input), gm.moved(t, input)
		for _, offset := range []float64{-.286, 0, .286} {
			output, err := r3.RotationAround(r3.Vec{Z: 28}, r3.Vec{Y: 1}, units.Degrees(indexAdvance(phi)+offset))
			require.NoError(t, err)
			w := wm.moved(t, output)
			noseCenter := input.Apply(center.Add(r3.Vec{Y: 2.4}))
			require.False(t, buttonIndexInside(wm, output, noseCenter), "nose center inside wheel phi=%g offset=%g", phi, offset)
			gap, clear, err := w.Exceeds(r, 0)
			require.NoError(t, err)
			require.True(t, clear, "rivet phi=%g offset=%g lower gap=%g", phi, offset, gap)
			gap, clear, err = w.Exceeds(g, 0)
			require.NoError(t, err)
			require.True(t, clear, "gauge phi=%g offset=%g lower gap=%g", phi, offset, gap)
		}
	}
}

func TestButtonIndexContact(t *testing.T) {
	doc := decad.New()
	wheel, err := indexWheel(t.Context(), doc)
	require.NoError(t, err)
	frame, err := indexFrame(180)
	require.NoError(t, err)
	rivet, err := indexRivet(t.Context(), doc, frame)
	require.NoError(t, err)
	wheelData := buttonIndexData(t, wheel)
	rivetData := buttonIndexData(t, rivet)
	for _, offset := range []float64{-3, 3} {
		turn, err := r3.RotationAround(r3.Vec{Z: 28}, r3.Vec{Y: 1}, units.Degrees(324+indexAdvance(180)+offset))
		require.NoError(t, err)
		wheelMesh := wheelData.moved(t, turn)
		inside := false
		for _, face := range rivetData.triangles {
			p := rivetData.vertices[face[0]].Add(rivetData.vertices[face[1]]).Add(rivetData.vertices[face[2]]).Scale(1.0 / 3)
			if p.Y < -69.2 || p.Y > -65.5 {
				continue
			}
			distance, e := wheelMesh.PointDistance(p)
			require.NoError(t, e)
			if distance <= wheelData.bound+rivetData.bound+1e-4 {
				continue
			}
			if buttonIndexInside(wheelData, turn, p) {
				inside = true
				break
			}
		}
		require.True(t, inside, "no rivet surface point is inside wheel at offset %g", offset)
	}
}

func TestButtonIndexRivetStops(t *testing.T) {
	doc := decad.New()
	frame, err := indexFrame(0)
	require.NoError(t, err)
	crank, err := indexCrank(t.Context(), doc, frame)
	require.NoError(t, err)
	rivet, err := indexRivet(t.Context(), doc, frame)
	require.NoError(t, err)
	cm, rm := buttonIndexData(t, crank), buttonIndexData(t, rivet)
	identity, err := r3.Translation(r3.Vec{})
	require.NoError(t, err)
	crankMesh := cm.moved(t, identity)
	rivetMesh := rm.moved(t, identity)
	gap, clear, err := crankMesh.Exceeds(rivetMesh, 0)
	require.NoError(t, err)
	require.True(t, clear, "rivet working clearance lower bound %g", gap)
	for _, shift := range []float64{-.2, .2} {
		inside := false
		for _, face := range rm.triangles {
			p := rm.vertices[face[0]].Add(rm.vertices[face[1]]).Add(rm.vertices[face[2]]).Scale(1.0 / 3).Add(r3.Vec{Y: shift})
			distance, e := crankMesh.PointDistance(p)
			require.NoError(t, e)
			if distance <= cm.bound+rm.bound+1e-4 {
				continue
			}
			if buttonIndexInside(cm, identity, p) {
				inside = true
				break
			}
		}
		require.True(t, inside, "rivet shift %g passes its stop", shift)
	}
}

func TestButtonIndexCrankKeyFit(t *testing.T) {
	doc := decad.New()
	f, err := indexFrame(0)
	require.NoError(t, err)
	crank, err := indexCrank(t.Context(), doc, f)
	require.NoError(t, err)
	_, key, err := indexShaft(t.Context(), doc, f)
	require.NoError(t, err)
	a, b := buttonIndexMesh(t, crank), buttonIndexMesh(t, key)
	gap, clear, err := a.Exceeds(b, 0)
	require.NoError(t, err)
	require.True(t, clear, "crank/key lower gap %g", gap)
}

func TestButtonIndexKeyShaft(t *testing.T) {
	doc := decad.New()
	frame, err := indexFrame(0)
	require.NoError(t, err)
	shaft, key, err := indexShaft(t.Context(), doc, frame)
	require.NoError(t, err)
	sm, km := buttonIndexDataAt(t, shaft, .005), buttonIndexDataAt(t, key, .005)
	identity, err := r3.Translation(r3.Vec{})
	require.NoError(t, err)
	shaftMesh, keyMesh := sm.moved(t, identity), km.moved(t, identity)
	for _, u := range []float64{-2, 0, 2} {
		p := frame.Origin().Add(r3.Vec{Y: -2.6}).Add(frame.U().Scale(u))
		sd, err := shaftMesh.PointDistance(p)
		require.NoError(t, err)
		kd, err := keyMesh.PointDistance(p)
		require.NoError(t, err)
		require.Greater(t, sd, sm.bound+1e-4)
		require.Greater(t, kd, km.bound+1e-4)
		require.False(t, buttonIndexInside(sm, identity, p), "key axis inside shaft material at %g", u)
		require.True(t, buttonIndexInside(km, identity, p), "key axis outside key at %g", u)
	}
}

func TestButtonIndexDisplay(t *testing.T) {
	doc := decad.New()
	display, err := Build(t.Context(), doc, Params{Digit: 9})
	require.NoError(t, err)
	require.NotNil(t, display)
	boundary, err := buildButtonIndex(t.Context(), doc, 0)
	require.NoError(t, err)
	require.NotNil(t, boundary.Wheel)
	require.NotNil(t, boundary.Crank)
	require.NotNil(t, boundary.Rivet)
	require.NotNil(t, boundary.Shaft)
	require.NotNil(t, boundary.Key)
	require.NoError(t, doc.VerifyContainers().Err())
	// The manual knob, detent plate and ratchet are excluded from these fits.
	wheel := buttonIndexMesh(t, boundary.Wheel)
	outputShaft := buttonIndexMesh(t, display.Shaft)
	gap, separated, err := wheel.Exceeds(outputShaft, 0)
	require.NoError(t, err)
	require.True(t, separated, "wheel/output shaft clearance lower bound %g", gap)
	crank := buttonIndexMesh(t, boundary.Crank)
	inputShaft := buttonIndexMesh(t, boundary.Shaft)
	gap, separated, err = crank.Exceeds(inputShaft, 0)
	require.NoError(t, err)
	require.True(t, separated, "crank/input shaft clearance lower bound %g", gap)
	key := buttonIndexMesh(t, boundary.Key)
	gap, separated, err = crank.Exceeds(key, 0)
	require.NoError(t, err)
	require.True(t, separated, "crank/key clearance lower bound %g", gap)
	for i := range 7 {
		seg := Segment(i)
		cam := buttonIndexDataAt(t, display.Cams[i], .05)
		slider := buttonIndexDataAt(t, display.Sliders[i], .05)
		lit9, err := Lit(9, seg)
		require.NoError(t, err)
		lit0, err := Lit(0, seg)
		require.NoError(t, err)
		baseLift := 0.0
		if lit9 {
			baseLift = 3
		}
		for _, phi := range []float64{108, 132, 156, 180, 204, 228, 252} {
			rotation, err := r3.RotationAround(r3.Vec{Z: 28}, r3.Vec{Y: 1}, units.Degrees(indexAdvance(phi)))
			require.NoError(t, err)
			lo, hi := buttonIndexGroove(t, cam, rotation, display.Dimensions.Station(seg)+3)
			lift := (lo+hi)/2 - 46
			t.Logf("segment %s phase %.0f lift %.4f", seg, phi, lift)
			if phi == 108 {
				expected := 0.0
				if lit9 {
					expected = 3
				}
				require.InDelta(t, expected, lift, .1)
			}
			if phi == 252 {
				expected := 0.0
				if lit0 {
					expected = 3
				}
				require.InDelta(t, expected, lift, .1)
			}
			translation, err := r3.Translation(r3.Vec{Z: lift - baseLift})
			require.NoError(t, err)
			camMesh, sliderMesh := cam.moved(t, rotation), slider.moved(t, translation)
			gap, clear, err := camMesh.Exceeds(sliderMesh, 0)
			require.NoError(t, err)
			require.True(t, clear, "segment %s phi %g clearance lower bound %g", seg, phi, gap)
		}
	}
}
