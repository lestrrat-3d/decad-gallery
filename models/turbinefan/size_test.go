package turbinefan_test

// CI: isolated-tests

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/stretchr/testify/require"
)

// TestSizeFor checks the derived sizes of the compact (128), default (192) and large (224) rotors, and that 128 equals
// the constants the fan had before it was sized by its rotor.
func TestSizeFor(t *testing.T) {
	type want struct {
		diameter, scale, hubScale, tip float64
		hubRadius, hubLength, hubMid   float64
		rootRadius, st2R, rootChord    float64
		st3R, st3Chord, maxChord       float64
		ref, hold                      float64
		studR, studStart, studEnd      float64
		socketR, socketStart           float64
		boltAt, boltLen, cavity, tipZ  float64
		slotLow, slotHigh, hubBetween  float64
		headPocketLow, studTravel      float64
	}
	for _, w := range []want{
		{128, 1, 1, 64, 14, 20, 10, 14.4, 16.3, 8, 19, 11.5, 17.8, 48, 21.5, 3, 6.105, 16, 3.405, 5.7, 9.5, 16, 15.6, 2.0,
			2.4, 4.4, 2.195, 18, 7.895},
		{192, 1.5, 1.35, 96, 18.9, 30, 15, 19.3, 21.865, 10.8, 28.5, 17.25, 26.7, 72, 32.25, 4.05, 8.1, 21.6, 4.455,
			7.695, 12.825, 25, 24.6, 3.0, 3.4, 5.4, 5.145, 28, 10.8},
		{224, 1.75, 1.35, 112, 18.9, 35, 17.5, 19.3, 21.865, 10.8, 33.25, 20.125, 31.15, 84, 37.625, 4.05, 8.1, 21.6,
			4.455, 7.695, 12.825, 25, 24.6, 8.0, 8.4, 10.4, 2.645, 33, 10.8},
	} {
		sz, err := turbinefan.SizeFor(w.diameter)
		require.NoError(t, err, "rotor %g", w.diameter)
		const tol = 1e-9
		require.Equal(t, w.diameter, sz.RotorDiameter)
		require.InDelta(t, w.scale, sz.Scale, tol)
		require.InDelta(t, w.hubScale, sz.HubScale, tol)
		require.InDelta(t, w.tip, sz.TipRadius, tol)
		require.InDelta(t, w.hubRadius, sz.HubRadius, tol)
		require.InDelta(t, w.hubLength, sz.HubLength, tol)
		require.InDelta(t, w.hubMid, sz.HubMid, tol)
		require.InDelta(t, w.hubMid+2.5, sz.BladeSocketZ(), tol)
		require.InDelta(t, w.rootRadius, sz.RootRadius, tol)
		r, chord := turbinefan.StationRadiusChord(sz, 1)
		require.InDelta(t, w.rootRadius, r, tol)
		require.InDelta(t, w.rootChord, chord, tol)
		r, _ = turbinefan.StationRadiusChord(sz, 2)
		require.InDelta(t, w.st2R, r, tol)
		r, chord = turbinefan.StationRadiusChord(sz, 3)
		require.InDelta(t, w.st3R, r, tol)
		require.InDelta(t, w.st3Chord, chord, tol)
		_, chord = turbinefan.StationRadiusChord(sz, 5)
		require.InDelta(t, w.maxChord, chord, tol)
		r, _ = turbinefan.StationRadiusChord(sz, 10)
		require.InDelta(t, w.tip, r, tol)
		require.InDelta(t, w.ref, sz.RefRadius, tol)
		require.InDelta(t, w.hold, sz.HoldRadius, tol)
		require.InDelta(t, w.studR, sz.BladeStudRadius, tol)
		require.InDelta(t, w.studStart, sz.BladeStudStart, tol)
		require.InDelta(t, w.studEnd, sz.BladeStudEnd, tol)
		require.InDelta(t, w.socketR, sz.BladeSocketRadius, tol)
		require.InDelta(t, w.socketStart, sz.BladeSocketStart, tol)
		require.InDelta(t, w.boltAt, sz.BladeBoltAt, tol)
		require.InDelta(t, w.boltLen, sz.BladeBoltLength, tol)
		require.InDelta(t, w.cavity, sz.BladeBoltCavity, tol)
		require.InDelta(t, w.tipZ, sz.HubLength-m2fastener.HeadPocketDepth-sz.BladeBoltLength, tol, "bolt tip z")
		slotHigh := sz.HubLength - sz.BladeBoltCavity
		require.InDelta(t, w.slotLow, slotHigh-m2fastener.NutCavityHeight, tol, "nut slot low z")
		require.InDelta(t, w.slotHigh, slotHigh, tol, "nut slot high z")
		require.InDelta(t, w.hubBetween+2.5, sz.BladeSocketZ()-sz.BladeSocketRadius-slotHigh, tol, "hub between slot and socket")
		require.InDelta(t, w.headPocketLow, sz.HubLength-m2fastener.HeadPocketDepth, tol, "head pocket low z")
		require.InDelta(t, w.studTravel, sz.HubRadius-sz.BladeStudStart, tol, "stud travel out of the socket")
	}

	for _, d := range []float64{127.9, 224.1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err := turbinefan.SizeFor(d)
		require.Error(t, err, "rotor %g", d)
	}
}

// TestSizeForBeta requires the blade angle at s r on a rotor of scale s to equal the 128 rotor's angle at r.
func TestSizeForBeta(t *testing.T) {
	base := mustSize(t, 128)
	for _, d := range []float64{192, 224} {
		sz := mustSize(t, d)
		for _, r := range []float64{14.4, 20, 30, 48, 64} {
			require.InDelta(t, base.Beta(r, turbinefan.DefaultBladePitch), sz.Beta(r*sz.Scale, turbinefan.DefaultBladePitch), 1e-12)
		}
	}
}

// TestMaxHubRadius requires MaxHubRadius to keep the hub rim HeadPocketRadius + 0.5 inside the smallest clutch's ear
// circle.
func TestMaxHubRadius(t *testing.T) {
	least := math.Inf(1)
	for _, shaft := range onewayclutch.Shafts() {
		dm, err := onewayclutch.Params{
			Shaft: shaft.Name, Freewheel: onewayclutch.Clockwise, BoreClearance: ballbearing.PressFitClearance,
		}.Derive()
		require.NoError(t, err)
		least = math.Min(least, dm.EarCircle)
	}
	require.InDelta(t, least-m2fastener.HeadPocketRadius-0.5, turbinefan.MaxHubRadius, 1e-9)
}

func TestSizeForBoltLengthAtRoundingBoundaries(t *testing.T) {
	for _, tc := range []struct {
		diameter float64
		length   float64
	}{
		{140.8, 18},
		{153.6, 20},
		{185.6, 25},
	} {
		t.Run(fmt.Sprintf("%.1f", tc.diameter), func(t *testing.T) {
			diameter := math.Nextafter(tc.diameter, math.Inf(-1))
			size, err := turbinefan.SizeFor(diameter)
			require.NoError(t, err, "diameter %.17g", diameter)
			require.Equal(t, tc.length, size.BladeBoltLength, "diameter %.17g", diameter)
		})
	}
}
