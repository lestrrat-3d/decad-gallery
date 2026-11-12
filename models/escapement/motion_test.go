package escapement_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/escapement"
	"github.com/stretchr/testify/require"
)

func TestMotion(t *testing.T) {
	const deg = math.Pi / 180
	cases := map[string]func(*escapement.Params){
		"default": func(*escapement.Params) {},
		"wide": func(p *escapement.Params) {
			p.LockAngle, p.LiftAngle, p.Amplitude = 1.25*deg, 3*deg, 6*deg
		},
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			p := escapement.DefaultParams()
			edit(&p)
			g, err := escapement.Layout(p)
			require.NoError(t, err)
			const n = 4000
			prev, err := g.StateAt(0)
			require.NoError(t, err)
			first := prev
			jumps := 0
			for i := 1; i <= n; i++ {
				s, err := g.StateAt(float64(i) / n)
				require.NoError(t, err)
				require.GreaterOrEqual(t, s.WheelAngle, prev.WheelAngle-1e-12, "phase %v", s.Phase)
				if s.Locked != "" && s.Locked == prev.Locked {
					require.InDelta(t, prev.WheelAngle, s.WheelAngle, 1e-12, "locked wheel moved at %v", s.Phase)
				}
				if d := s.WheelAngle - prev.WheelAngle; d > g.Drop/2 && prev.Locked == "" && s.Locked != "" {
					require.InDelta(t, g.Drop, d, 0.2*deg, "jump size at %v", s.Phase)
					jumps++
				}
				require.InDelta(t, escapement.Pendulum(s.AnchorAngle), s.PendulumAngle, 1e-12)
				require.InDelta(t, s.WheelAngle*13.0/48.0, s.GreatWheelAngle, 1e-12)
				prev = s
			}
			require.InDelta(t, g.Pitch, prev.WheelAngle-first.WheelAngle, 1e-9)
			require.Equal(t, 2, jumps)

			// The two drops equal the closed-form drop, evaluated exactly at the release angles.
			for _, rel := range []struct {
				psi        float64
				increasing bool
				to         float64
			}{
				{g.PsiEf, true, g.WheelPhaseX},
				{g.PsiXf, false, g.WheelPhaseE + g.Pitch},
			} {
				var before float64
				var err error
				if rel.increasing {
					before, _, err = g.WheelAt(rel.psi-1e-12, true)
				} else {
					before, _, err = g.WheelAt(rel.psi+1e-12, false)
				}
				require.NoError(t, err)
				require.InDelta(t, g.Drop, rel.to-before, 1e-6)
			}
		})
	}
	// Pendulum angle at the default amplitude.
	s, err := escapement.Motion(func() escapement.Params { p := escapement.DefaultParams(); p.Phase = 0.5; return p }())
	require.NoError(t, err)
	require.InDelta(t, 3.814*deg, s.PendulumAngle, 1e-3*deg)
}
