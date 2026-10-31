package spinningwheel_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestSpinningWheelMotion(t *testing.T) {
	for _, scale := range []float64{.8, 1, 1.2} {
		for _, ratio := range []int{7, 9, 12, 17} {
			p := spinningwheel.DefaultParams()
			p.Scale, p.Ratio = scale, ratio
			last := math.NaN()
			for i := 0; i <= 360; i++ {
				p.WheelAngle = float64(i) * math.Pi / 180
				s, err := spinningwheel.Motion(p)
				require.NoError(t, err)
				// The heel rotates about X; the rear crank rotates about Y.
				require.InDelta(t, 25*scale, math.Hypot(s.CrankPin.X-s.Wheel.X, s.CrankPin.Z-s.Wheel.Z), 1e-8)
				wantToe := s.PedalFrame.ToWorld(r3.Vec{X: 220 * scale, Y: 450 * scale, Z: 20 * scale})
				require.InDelta(t, 0, wantToe.Sub(s.Toe).Len(), 1e-8)
				require.InDelta(t, 486*scale, s.Toe.Sub(s.RodHead).Len(), 1e-8)
				require.InDelta(t, 32*scale, s.RodHead.Sub(s.CrankPin).Len(), 1e-8)
				require.InDelta(t, 0, s.RodFrame.V().Sub(s.GimbalFrame.V()).Len(), 1e-10, "cross bearings stay aligned")
				require.InDelta(t, 1, s.PedalFrame.U().X, 1e-10, "heel axis stays along X")
				require.InDelta(t, -1, s.GimbalFrame.N().Y, 1e-10, "upper crank bearing stays along Y")
				require.Greater(t, s.Toe.Z, 39*scale)
				require.Less(t, s.Toe.Z, 140*scale)
				if !math.IsNaN(last) {
					require.Less(t, math.Abs(s.PedalAngle-last), .01, "no branch jumps")
				}
				last = s.PedalAngle
				require.InDelta(t, float64(ratio), s.WheelPitch/s.FlyerPitch, 1e-10)
				require.InDelta(t, float64(ratio)*p.WheelAngle, s.FlyerAngle, 1e-10)
				wound := (s.FlyerAngle - s.BobbinAngle) * 31 * scale
				require.InDelta(t, p.YarnFeedPerTurn*p.WheelAngle/(2*math.Pi), wound, 1e-8)
			}
		}
	}
}

func TestSpinningWheelAdjustments(t *testing.T) {
	for _, scale := range []float64{.8, 1, 1.2} {
		p := spinningwheel.DefaultParams()
		p.Scale = scale
		var originalBand float64
		for i := 0; i <= 6; i++ {
			p.BrakeTravel = float64(i)
			s, err := spinningwheel.Motion(p)
			require.NoError(t, err)
			if i == 0 {
				originalBand = s.BrakeLength
			}
			require.InDelta(t, originalBand, s.BrakeLength, 1e-9, "adjuster trades tail lengths")
			require.InDelta(t, 25*scale+float64(i)/2, s.SpringLength, 1e-9)
		}
		p.DriveTension = 0
		lo, err := spinningwheel.Motion(p)
		require.NoError(t, err)
		p.DriveTension = 12
		hi, err := spinningwheel.Motion(p)
		require.NoError(t, err)
		require.Greater(t, hi.BeltLength, lo.BeltLength, "hinged head must tension the drive band")
		for i, ratio := range []int{7, 9, 12, 17} {
			p.Ratio = ratio
			s, err := spinningwheel.Motion(p)
			require.NoError(t, err)
			require.InDelta(t, float64(278-8*i)*scale, s.HeadShift, 1e-9, "selected groove aligns with Y0")
		}
	}
}

func TestSpinningWheelRejectsInvalidParams(t *testing.T) {
	valid := spinningwheel.DefaultParams()
	for _, change := range []func(*spinningwheel.Params){
		func(p *spinningwheel.Params) { p.Scale = .79 }, func(p *spinningwheel.Params) { p.Scale = 1.21 },
		func(p *spinningwheel.Params) { p.Scale = math.NaN() }, func(p *spinningwheel.Params) { p.WheelAngle = math.Inf(1) },
		func(p *spinningwheel.Params) { p.DriveTension = -1 }, func(p *spinningwheel.Params) { p.DriveTension = 13 },
		func(p *spinningwheel.Params) { p.BrakeTravel = -1 }, func(p *spinningwheel.Params) { p.BrakeTravel = 7 },
		func(p *spinningwheel.Params) { p.YarnFeedPerTurn = -1 }, func(p *spinningwheel.Params) { p.YarnFeedPerTurn = 201 },
		func(p *spinningwheel.Params) { p.Ratio = 8 },
	} {
		p := valid
		change(&p)
		d := decad.New()
		_, err := spinningwheel.Build(t.Context(), d, p)
		require.Error(t, err)
		require.Zero(t, len(d.Bodies()))
	}
}
