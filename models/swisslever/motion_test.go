package swisslever_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/swisslever"
	"github.com/stretchr/testify/require"
)

func TestMotion(t *testing.T) {
	st, err := swisslever.Motion(swisslever.DefaultParams())
	require.NoError(t, err)
	require.InDelta(t, 0, st.BalanceAngle, 1e-12)
	require.InDelta(t, 0.37662*deg, st.LeverAngle, 1e-5*deg*10)
	require.InDelta(t, 20.61688*deg, st.WheelAngle, 1e-5*deg*10)
	require.Equal(t, swisslever.LeverImpulse, st.LeverStage)
	require.Equal(t, swisslever.WheelToothLiftEntry, st.WheelStage)
	require.Equal(t, swisslever.LockedNone, st.Locked)

	for _, amp := range []float64{180 * deg, 45 * deg, 270 * deg} {
		p := swisslever.DefaultParams()
		p.Amplitude = amp
		g, err := swisslever.Layout(p)
		require.NoError(t, err)
		const n = 4000
		var states []swisslever.State
		for i := 0; i <= n; i++ {
			s, err := g.StateAt(float64(i) / n)
			require.NoError(t, err)
			require.LessOrEqual(t, math.Abs(s.LeverAngle), g.Banking+1e-12, "|psi| <= alpha")
			states = append(states, s)
		}
		require.InDelta(t, g.Pitch, states[n].WheelAngle-states[0].WheelAngle, 1e-9, "one tooth per period")

		recoil := 0.0
		for i := 1; i <= n; i++ {
			a, b := states[i-1], states[i]
			dphi := b.WheelAngle - a.WheelAngle
			switch {
			case dphi < -1e-12:
				// phi falls only while a toe slides on a drawn locking face.
				require.Contains(t, []string{swisslever.WheelLockEntry, swisslever.WheelLockExit}, a.WheelStage, "step %d", i)
				require.Equal(t, a.WheelStage, b.WheelStage, "step %d", i)
				recoil -= dphi
				require.LessOrEqual(t, recoil, 0.5803*deg+1e-6, "recoil run at step %d", i)
			case dphi > 1*deg:
				// Only the drops jump: across psi_Ef on swing + and psi_Xf on swing -.
				recoil = 0
				plus := b.Phase <= 0.5
				if plus {
					require.Less(t, a.LeverAngle, g.PsiEf+1e-9, "drop step %d", i)
					require.Greater(t, b.LeverAngle, g.PsiEf-1e-9, "drop step %d", i)
				} else {
					require.Greater(t, a.LeverAngle, g.PsiXf-1e-9, "drop step %d", i)
					require.Less(t, b.LeverAngle, g.PsiXf+1e-9, "drop step %d", i)
				}
				require.Greater(t, dphi, 2.5*deg)
			default:
				recoil = 0
			}
			dpsi := math.Abs(b.LeverAngle - a.LeverAngle)
			if dpsi > 0.6*deg {
				// The lever jumps only where the wheel takes over from the pin, at theta_2 and -theta_2.
				lo, hi := math.Min(a.BalanceAngle, b.BalanceAngle), math.Max(a.BalanceAngle, b.BalanceAngle)
				at := g.Theta2
				if b.Phase > 0.5 {
					at = -g.Theta2
				}
				require.True(t, lo <= at && at <= hi, "psi jumps at step %d (theta %v)", i, a.BalanceAngle/deg)
			}
		}
	}
}

func TestWheelAt(t *testing.T) {
	g, err := swisslever.Layout(swisslever.DefaultParams())
	require.NoError(t, err)
	for _, c := range []struct {
		psi  float64
		plus bool
		phi  float64
		name string
	}{
		{-g.Banking, true, 18.4047, swisslever.WheelLockEntry},
		{g.PsiEu, true, 18.0, swisslever.WheelLockEntry},
		{g.PsiEt, true, 21.0775, swisslever.WheelToothLiftEntry},
		{g.PsiEf, true, 27.6452, swisslever.WheelPalletLiftEntry},
		{g.PsiEf + 1e-9, true, 30.3414, swisslever.WheelLockExit},
		{g.Banking, true, 30.5803, swisslever.WheelLockExit},
		{g.PsiXu, false, 30.0, swisslever.WheelLockExit},
		{g.PsiXt, false, 32.9225, swisslever.WheelToothLiftExit},
		{g.PsiXf, false, 39.5334, swisslever.WheelPalletLiftExit},
		{g.PsiXf - 1e-9, false, 42.2728, swisslever.WheelLockEntry},
		{-g.Banking, false, 42.4047, swisslever.WheelLockEntry},
	} {
		phi, stage, err := g.WheelAt(c.psi, c.plus)
		require.NoError(t, err)
		require.InDelta(t, c.phi, phi/deg, 1e-4, "phi at psi %v", c.psi/deg)
		require.Equal(t, c.name, stage, "stage at psi %v", c.psi/deg)
	}
}

func TestGuard(t *testing.T) {
	g, err := swisslever.Layout(swisslever.DefaultParams())
	require.NoError(t, err)
	require.InDelta(t, 2.0*deg, g.GuardFreedom, 0.01*deg)
	require.Less(t, g.GuardFreedom, swisslever.Lock+swisslever.Slide)

	// The pin path stays clear of the outer corner round with the lever at the guard angle.
	least := math.Inf(1)
	for th := -math.Pi; th <= math.Pi; th += 0.01 * deg {
		q := swisslever.Pt{X: 0, Y: swisslever.Distance}.Add(swisslever.Pt{X: 0, Y: -swisslever.PinCentreRadius}.Rot(th))
		d := q.Rot(g.GuardAngle).Sub(g.KR).Len() - (g.CornerRadius + swisslever.PinRadius)
		least = math.Min(least, d)
	}
	require.GreaterOrEqual(t, least, 0.39)

	// Every contact lands on its round's upper half: directions 126.2 to 163.2 degrees from K_R.
	lo, hi := math.Inf(1), math.Inf(-1)
	for th := g.Theta1; th <= g.Theta2; th += 0.05 * deg {
		psi, stage, err := g.LeverAt(th, true)
		require.NoError(t, err)
		require.Equal(t, swisslever.LeverUnlock, stage)
		q := swisslever.Pt{X: 0, Y: swisslever.Distance}.Add(swisslever.Pt{X: 0, Y: -swisslever.PinCentreRadius}.Rot(th))
		dir := q.Rot(psi).Sub(g.KR)
		require.InDelta(t, g.CornerRadius+swisslever.PinRadius, dir.Len(), 1e-9)
		a := dir.Angle() / deg
		lo, hi = math.Min(lo, a), math.Max(hi, a)
	}
	// On swing - the same round drives the pin through the impulse.
	for th := -g.Theta3 + 1e-9; th <= -g.Theta2; th += 0.05 * deg {
		psi, stage, err := g.LeverAt(th, false)
		require.NoError(t, err)
		require.Equal(t, swisslever.LeverImpulse, stage)
		q := swisslever.Pt{X: 0, Y: swisslever.Distance}.Add(swisslever.Pt{X: 0, Y: -swisslever.PinCentreRadius}.Rot(th))
		a := q.Rot(psi).Sub(g.KR).Angle() / deg
		lo, hi = math.Min(lo, a), math.Max(hi, a)
	}
	require.InDelta(t, 126.2, lo, 0.1)
	require.InDelta(t, 163.2, hi, 0.1)
	require.True(t, lo > 0 && hi < 180)
	for th := g.Theta2; th <= g.Theta3; th += 0.05 * deg {
		psi, stage, err := g.LeverAt(th, true)
		require.NoError(t, err)
		require.Equal(t, swisslever.LeverImpulse, stage)
		q := swisslever.Pt{X: 0, Y: swisslever.Distance}.Add(swisslever.Pt{X: 0, Y: -swisslever.PinCentreRadius}.Rot(th))
		a := q.Rot(psi).Sub(g.KL).Angle() / deg
		require.True(t, a > 0 && a < 180, "left round contact at %v", a)
	}

	// The lever passes the same events on the mirrored swing.
	psi, stage, err := g.LeverAt(-g.Theta2+1e-9, false)
	require.NoError(t, err)
	require.Equal(t, swisslever.LeverUnlock, stage)
	require.InDelta(t, g.PsiXu, psi, 1e-6)
}
