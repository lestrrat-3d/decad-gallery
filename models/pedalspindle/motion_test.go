package pedalspindle_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/pedalspindle"
	"github.com/stretchr/testify/require"
)

func TestPedalSpindleMotion(t *testing.T) {
	for _, scale := range []float64{0.8, 1, 1.2} {
		for i := range 721 {
			angle := 2 * math.Pi * float64(i) / 720
			p := pedalspindle.Params{Scale: scale, CrankAngle: angle}
			s, err := pedalspindle.Motion(p)
			require.NoError(t, err)
			require.InDelta(t, 200*scale, s.Toe.Sub(s.Heel).Len(), 1e-8, "phase %d", i)
			require.InDelta(t, 320*scale, s.CrankPin.Sub(s.Toe).Len(), 1e-8, "phase %d", i)
			dx, dz := s.CrankPin.X-s.Input.X, s.CrankPin.Z-s.Input.Z
			require.InDelta(t, 25*scale, math.Hypot(dx, dz), 1e-8)
			require.InDelta(t, 102.3*scale, s.Intermediate.Sub(s.Input).Len(), 1e-8)
			require.InDelta(t, 102.3*scale, s.Output.Sub(s.Intermediate).Len(), 1e-8)
			require.Equal(t, 9.0, s.Ratio)
			// Evaluate all four treadle corners against the top of the real base.
			for _, x := range []float64{-12, 212} {
				for _, z := range []float64{-8, 8} {
					height := s.Heel.Z + scale*(x*math.Sin(s.PedalAngle)+z*math.Cos(s.PedalAngle))
					require.Greater(t, height-18*scale, 12.9*scale, "phase %d clearance", i)
				}
			}
			// Phase changes away from the wrap point must carry -3 and +9 rotations.
			p.CrankAngle += 1e-5
			next, err := pedalspindle.Motion(p)
			require.NoError(t, err)
			require.InDelta(t, 0, math.Remainder(next.IntermediateAngle-s.IntermediateAngle+3e-5, 2*math.Pi), 1e-10)
			require.InDelta(t, 0, math.Remainder(next.SpindleAngle-s.SpindleAngle-9e-5, 2*math.Pi), 1e-10)
		}
	}
	for _, angle := range []float64{math.MaxFloat64, -math.MaxFloat64} {
		p := pedalspindle.DefaultParams()
		p.CrankAngle = angle
		s, err := pedalspindle.Motion(p)
		require.NoError(t, err)
		require.False(t, math.IsInf(s.SpindleAngle, 0) || math.IsNaN(s.SpindleAngle))
	}
}

func TestPedalSpindleRejectsInvalidParams(t *testing.T) {
	for _, item := range []struct {
		name         string
		scale, angle float64
	}{
		{"zero", 0, 0}, {"below", .799, 0}, {"above", 1.201, 0},
		{"nan-scale", math.NaN(), 0}, {"infinite-scale", math.Inf(1), 0},
		{"nan-angle", 1, math.NaN()}, {"infinite-angle", 1, math.Inf(1)},
	} {
		t.Run(item.name, func(t *testing.T) {
			doc := decad.New()
			_, err := pedalspindle.Build(t.Context(), doc, pedalspindle.Params{Scale: item.scale, CrankAngle: item.angle})
			require.Error(t, err)
			require.Zero(t, len(doc.Bodies()))
		})
	}
	_, err := pedalspindle.Build(t.Context(), nil, pedalspindle.DefaultParams())
	require.ErrorContains(t, err, "document")
	// The public boundary explicitly rejects nil contexts; exercise that error path.
	_, err = pedalspindle.Build(nil, decad.New(), pedalspindle.DefaultParams()) //nolint:staticcheck
	require.ErrorContains(t, err, "context")
}
