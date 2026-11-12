package escapement_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/escapement"
	"github.com/stretchr/testify/require"
)

func near(t *testing.T, want, got float64, name string) {
	t.Helper()
	require.InDelta(t, want, got, 1e-4, name)
}

func nearPt(t *testing.T, want [2]float64, got escapement.Pt, name string) {
	t.Helper()
	near(t, want[0], got.X, name+".X")
	near(t, want[1], got.Y, name+".Y")
}

func TestLayout(t *testing.T) {
	const deg = math.Pi / 180
	g, err := escapement.Layout(escapement.DefaultParams())
	require.NoError(t, err)

	near(t, 12*deg, g.Pitch, "pitch")
	near(t, 45*deg, g.HalfSpan, "half span")
	near(t, 56.5685, g.CentreDistance, "d")
	near(t, 40, g.TangentRadius, "rm")
	near(t, 38.6, g.RIn, "rin")
	near(t, 41.4, g.ROut, "rout")
	nearPt(t, [2]float64{0, 56.5685}, g.P, "P")
	nearPt(t, [2]float64{0, 70.5685}, g.S, "S")
	nearPt(t, [2]float64{22.8750, -39.6207}, g.G, "G")

	nearPt(t, [2]float64{-29.2569, 27.2770}, g.IL, "IL")
	nearPt(t, [2]float64{-27.2770, 29.2569}, g.FL, "FL")
	nearPt(t, [2]float64{27.2770, 29.2569}, g.IR, "IR")
	nearPt(t, [2]float64{29.2569, 27.2770}, g.FR, "FR")
	near(t, 137.0058, g.IL.Angle()/deg, "angle IL")
	near(t, 132.9942, g.FL.Angle()/deg, "angle FL")
	near(t, 47.0058, g.IR.Angle()/deg, "angle IR")
	near(t, 42.9942, g.FR.Angle()/deg, "angle FR")
	near(t, 90, (g.IL.Angle()-g.IR.Angle())/deg, "span")

	near(t, -0.5*deg, g.PsiEu, "psi Eu")
	near(t, 2*deg, g.PsiEf, "psi Ef")
	near(t, 0.5*deg, g.PsiXu, "psi Xu")
	near(t, -2*deg, g.PsiXf, "psi Xf")

	nearPt(t, [2]float64{-29.5114, -29.0351}, g.CE, "CE")
	nearPt(t, [2]float64{-26.3072, -28.2470}, g.FE, "FE")
	nearPt(t, [2]float64{27.5143, -27.0726}, g.CX, "CX")
	nearPt(t, [2]float64{28.2168, -30.2948}, g.FX, "FX")
	near(t, -135.4661, g.CE.Angle()/deg, "angle CE")
	near(t, -44.5364, g.CX.Angle()/deg, "angle CX")
	near(t, -144.4661, g.TopE/deg, "top E")
	near(t, -35.5364, g.TopX/deg, "top X")
	near(t, 5.2915, g.S0, "s0")

	nearPt(t, [2]float64{-31.4116, -22.4337}, g.GE, "GE")
	nearPt(t, [2]float64{-6.5102, -4.6495}, g.HE, "HE")
	nearPt(t, [2]float64{-33.6901, -24.0610}, g.KE, "KE")
	nearPt(t, [2]float64{-37.1773, -19.1784}, g.ME, "ME")
	nearPt(t, [2]float64{-7.7932, 1.8073}, g.NE, "NE")
	nearPt(t, [2]float64{31.4106, -22.4351}, g.GX, "GX")
	nearPt(t, [2]float64{6.5100, -4.6498}, g.HX, "HX")
	nearPt(t, [2]float64{33.6891, -24.0625}, g.KX, "KX")
	nearPt(t, [2]float64{37.1764, -19.1800}, g.MX, "MX")
	nearPt(t, [2]float64{7.7933, 1.8070}, g.NX, "NX")
	near(t, 13.0540, g.NX.Angle()/deg, "angle NX")
	near(t, 166.9435, g.NE.Angle()/deg, "angle NE")

	nearPt(t, [2]float64{-1.3272, 32.4729}, g.ToothL, "tooth L")
	nearPt(t, [2]float64{-4.5118, 32.1853}, g.ToothB, "tooth B")

	nearPt(t, [2]float64{2.5, -5.4544}, g.CrutchAR, "A_R")
	nearPt(t, [2]float64{-2.5, -41.8775}, g.CrutchBL, "B_L")
	nearPt(t, [2]float64{4, -9.1652}, g.PendER, "E_R")
	near(t, -180+24.6779, g.PendQL.Y, "Q_L")

	near(t, 45.75, g.GearCentre, "gear centre")
	near(t, 7.73, g.PinionRoot, "pinion root")
	near(t, 11.23, g.PinionTip, "pinion tip")
	near(t, 33.98, g.GreatRoot, "great root")
	near(t, 37.48, g.GreatTip, "great tip")
	near(t, 0.54, g.TipToRootA, "tip to root a")
	near(t, 0.54, g.TipToRootB, "tip to root b")
	near(t, 123.75, g.GreatPhase/deg, "great phase")
	near(t, 0.99425, g.WheelPhaseE/deg, "phi E")
	near(t, 6.99425, g.WheelPhaseX/deg, "phi X")
	near(t, 1.98848, g.Drop/deg, "drop")
	require.InDelta(t, 0.851, g.PendulumPeriod, 1e-3, "period 180")
}
