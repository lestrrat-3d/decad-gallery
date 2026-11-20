package swisslever_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/swisslever"
	"github.com/stretchr/testify/require"
)

func TestLayout(t *testing.T) {
	g, err := swisslever.Layout(swisslever.DefaultParams())
	require.NoError(t, err)

	// Derived values.
	near(t, 24*deg, g.Pitch, "tau")
	near(t, 8.5*deg, g.Lift, "lambda")
	near(t, 7*deg, g.Banking, "alpha")
	near(t, 44.5714, g.CentreDistance, "d")
	near(t, 22.2857, g.LockRadius, "r_m")
	nearPt(t, [2]float64{0, 44.5714}, g.P, "P")
	nearPt(t, [2]float64{0, 92.5714}, g.B, "B")
	near(t, -3*deg, g.PsiEu, "psi Eu")
	near(t, 1*deg, g.PsiEt, "psi Et")
	near(t, 5.5*deg, g.PsiEf, "psi Ef")
	near(t, 3*deg, g.PsiXu, "psi Xu")
	near(t, -1*deg, g.PsiXt, "psi Xt")
	near(t, -5.5*deg, g.PsiXf, "psi Xf")
	nearPt(t, [2]float64{-19.3000, 33.4286}, g.IL, "I_L")
	nearPt(t, [2]float64{19.3000, 33.4286}, g.IR, "I_R")
	nearPt(t, [2]float64{-19.8567, -10.1175}, g.CE, "C_E")
	nearPt(t, [2]float64{19.8567, -10.1175}, g.CX, "C_X")
	near(t, 22.2857, g.CE.Len(), "|C_E|")
	near(t, -153, g.CE.Angle()/deg, "angle C_E")
	near(t, -27, g.CX.Angle()/deg, "angle C_X")
	near(t, 40.1546, g.HeelRadius, "R_h")
	nearPt(t, [2]float64{-15.9182, 36.8647}, g.XE, "X_E")
	nearPt(t, [2]float64{23.9037, 32.2647}, g.XX, "X_X")
	nearPt(t, [2]float64{-15.1063, -9.1970}, g.FE, "F_E")
	nearPt(t, [2]float64{22.6141, -14.5412}, g.FX, "F_X")
	near(t, 17.6857, g.FE.Len(), "|F_E|")
	near(t, -148.6661, g.FE.Angle()/deg, "angle F_E")
	near(t, 26.8857, g.FX.Len(), "|F_X|")
	near(t, -32.7417, g.FX.Angle()/deg, "angle F_X")
	near(t, 105, g.UEAngle/deg, "u_E")
	near(t, 51, g.UXAngle/deg, "u_X")
	near(t, 18, g.PhiE/deg, "phi_E")
	near(t, 30, g.PhiX/deg, "phi_X")

	// Escape wheel tooth 0.
	nearPt(t, [2]float64{0, 38.6}, g.ToothT, "T")
	nearPt(t, [2]float64{-2.1015, 40.0996}, g.ToothH, "H")
	nearPt(t, [2]float64{-2.0551, 31.4329}, g.ToothL, "L")
	nearPt(t, [2]float64{-4.8453, 31.1251}, g.ToothK, "K")

	// Pallet stones.
	for _, c := range []struct {
		name string
		got  swisslever.StoneSection
		want [4][2]float64
	}{
		{"entry", g.Entry, [4][2]float64{{-19.8567, -10.1175}, {-15.1063, -9.1970}, {-17.6475, -2.5769}, {-21.6685, -3.3560}}},
		{"exit", g.Exit, [4][2]float64{{19.8567, -10.1175}, {22.6141, -14.5412}, {26.5920, -8.4157}, {24.2620, -4.6775}}},
	} {
		got := c.got.Corners()
		for i, n := range []string{"C", "F", "G", "Hs"} {
			nearPt(t, c.want[i], got[i], c.name+" "+n)
		}
	}

	// Slots and the lever outline.
	for i, w := range [][2]float64{{-19.8930, -10.1755}, {-15.0382, -9.2347}, {-17.6159, -2.5198}, {-21.7306, -3.3171}} {
		nearPt(t, w, g.EntrySlot[i], "entry slot")
	}
	for i, w := range [][2]float64{{19.7954, -10.1137}, {22.6132, -14.6343}, {26.6513, -8.4162}, {24.2673, -4.5915}} {
		nearPt(t, w, g.ExitSlot[i], "exit slot")
	}
	nearPt(t, [2]float64{-24.1557, -1.8008}, g.VE1, "VE1")
	nearPt(t, [2]float64{-21.3062, -12.4356}, g.VE2, "VE2")
	nearPt(t, [2]float64{-12.3844, -10.7068}, g.VE3, "VE3")
	nearPt(t, [2]float64{17.4057, -9.9662}, g.VX1, "VX1")
	nearPt(t, [2]float64{22.5795, -18.2666}, g.VX2, "VX2")
	nearPt(t, [2]float64{28.9624, -8.4377}, g.VX3, "VX3")
	nearPt(t, [2]float64{24.4743, -1.2372}, g.VX4, "VX4")
	nearPt(t, [2]float64{-3, 12}, g.JL, "JL")
	nearPt(t, [2]float64{3, 12}, g.JR, "JR")

	// Fork and guard.
	near(t, 0.66616, g.CornerRadius, "r_c")
	nearPt(t, [2]float64{2.33384, 38.87465}, g.KR, "K_R")
	nearPt(t, [2]float64{-2.33384, 38.87465}, g.KL, "K_L")
	near(t, 1.66767, g.NotchRadius, "r_n")
	near(t, 37.20698, g.NotchCentre.Y-g.NotchRadius, "notch bottom")
	near(t, 0.53384, g.GapRadius, "gap round")
	nearPt(t, [2]float64{1.13384, 38.87465}, g.GapRight, "gap centre")
	nearPt(t, [2]float64{0, 42.15}, g.DartTip, "T_d")
	near(t, -27.33541*deg, g.Theta1, "theta1")
	near(t, -11.81885*deg, g.Theta2, "theta2")
	near(t, 22.12829*deg, g.ThetaDrop, "theta_d")
	near(t, 27.33541*deg, g.Theta3, "theta3")
	near(t, -2.02444*deg, g.JumpPsi, "jump")
	require.InDelta(t, 5.0*deg, g.GuardAngle, 0.01*deg, "psi_g")
	near(t, 0.9799, g.GuardGap, "guard gap")
	near(t, 1.32721, g.CornerFreedomBanking, "corner freedom at the banking")
	near(t, 0.3993, g.CornerFreedomGuard, "corner freedom with the dart on the roller")

	// Hairspring and balance.
	near(t, -23.9697, g.GammaS/deg, "gamma_s")
	near(t, 66.0303, g.GammaF/deg, "gamma_f")
	nearPt(t, [2]float64{-32.0106, 92.5714}, g.EyeCentre, "eye")
	near(t, 1.8312, g.SpringRate, "k")
	require.InDelta(t, 13.1, g.WheelMass, 13.1*0.02, "balance wheel mass")
	require.InDelta(t, 15617, g.BalanceInertia, 15617*0.005, "inertia")
	require.InDelta(t, 0.580, g.Period, 0.002, "period")
	near(t, 8, float64(g.Spring.Turns), "turns")
	require.InDelta(t, 967.611, g.Spring.ActiveLength, 1e-3, "active length")

	// Frame.
	nearPt(t, [2]float64{-7.1564, 61.8279}, g.BankingLeft, "left banking post")
	nearPt(t, [2]float64{7.1564, 61.8279}, g.BankingRight, "right banking post")
	nearPt(t, [2]float64{-32.0106, 92.5714}, g.Stud, "stud")
	require.Len(t, g.Sites, 4)
	for i, s := range g.Sites {
		require.NoError(t, s.Validate(), "site %d", i+1)
		near(t, 33, s.Head.Z, "head top")
		near(t, g.Pillars[i].X, s.Head.X, "site x")
	}
}
