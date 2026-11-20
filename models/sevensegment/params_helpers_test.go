package sevensegment_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
)

// rampsPerCam is instructions.md "Ramps per cam": From and Rising per segment.
var rampsPerCam = map[sevensegment.Segment][]struct {
	from   int
	rising bool
}{
	sevensegment.SegA: {{0, false}, {1, true}, {3, false}, {4, true}},
	sevensegment.SegB: {{4, false}, {6, true}},
	sevensegment.SegC: {{1, false}, {2, true}},
	sevensegment.SegD: {{0, false}, {1, true}, {3, false}, {4, true}, {6, false}, {7, true}},
	sevensegment.SegE: {{0, false}, {1, true}, {2, false}, {5, true}, {6, false}, {7, true}, {8, false}, {9, true}},
	sevensegment.SegF: {{0, false}, {3, true}, {6, false}, {7, true}},
	sevensegment.SegG: {{1, true}, {6, false}, {7, true}, {9, false}},
}

// worstBudget is the least flagBudgetRatio over the ten steps and the angles 7.98 to 29.55 at friction mu, with the slider
// weight and the extra cam force that the audit of the built rotors gives (the larger of the two rotor types).
func worstBudget(t *testing.T, dm sevensegment.Dimensions, mu float64) float64 {
	t.Helper()
	l := builtLoads(t)
	fx := budgetFx(t, mu)
	worst := math.Inf(1)
	for psi := 7.98; psi <= 29.55+1e-9; psi += 0.05 {
		for k := range 10 {
			r, _ := flagBudgetRatio(dm, k, psi, mu, fx, l.ws)
			worst = math.Min(worst, r)
		}
	}
	return worst
}

// pawlE is the printed PETG modulus the budget uses, MPa: the low end of the 1.8-2.0 GPa range. knobE is the high end the
// knob torque uses.
const (
	pawlE = 1800.0
	knobE = 2100.0
)

// pawlCompliance is one pawl's compliance at its seat in the pawl frame (seat on +u, arm along +v), rows and columns
// (Fx, Fy, M about the seat): the straight arm from v = PawlBackV over PawlArmLength with its inner edge on PawlArmInner,
// thickness linear from PawlArmTip to PawlArmRoot, width DetentThickness, modulus e; bending, axial and shear (1.2)
// energy, the head rigid. The plane-stress FE of the real outline is about 9 % stiffer (instructions.md, the budget table notes).
func pawlCompliance(dm sevensegment.Dimensions, e float64) [3][3]float64 {
	const n = 2000
	g := e / (2 * 1.38)
	px := dm.SeatRadius
	var c [3][3]float64
	for i := range n {
		s := (float64(i) + 0.5) / n * sevensegment.PawlArmLength
		ds := sevensegment.PawlArmLength / n
		t := sevensegment.PawlArmTip + (sevensegment.PawlArmRoot-sevensegment.PawlArmTip)*s/sevensegment.PawlArmLength
		x, y := dm.PawlArmInner+t/2, sevensegment.PawlBackV+s
		b := sevensegment.DetentThickness
		inertia, area := b*t*t*t/12, b*t
		var res [3][3]float64 // moment, axial, shear for unit Fx, Fy, M
		for k, f := range [3][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}} {
			res[k] = [3]float64{(px-x)*f[1] + y*f[0] + f[2], f[1], f[0]}
		}
		for a := range 3 {
			for bb := range 3 {
				c[a][bb] += (res[a][0]*res[bb][0]/(e*inertia) + res[a][1]*res[bb][1]/(e*area) +
					1.2*res[a][2]*res[bb][2]/(g*area)) * ds
			}
		}
	}
	return c
}

// detentTorque is the torque of both pawls on the wheel at psi past a rest, in N.mm: release drives the wheel back to the
// rest against friction mu, climb resists the knob. The contact force N (n + -mu t) on the flank of normal angle beta
// closes the head's normal interference delta cos(beta) through the compliance c (the closed form of instructions.md).
func detentTorque(dm sevensegment.Dimensions, c [3][3]float64, psi, mu float64, release bool) float64 {
	delta, rc, beta := dm.PawlContact(psi)
	if delta <= 0 {
		return 0
	}
	sb, cb := math.Sincos(beta * math.Pi / 180)
	sgn := 1.0
	if !release {
		sgn = -1
	}
	f := [2]float64{cb + sgn*mu*sb, -sb + sgn*mu*cb}
	m := -sgn * sevensegment.PawlTipRound * mu
	g0 := c[0][0]*f[0] + c[0][1]*f[1] + c[0][2]*m
	g1 := c[1][0]*f[0] + c[1][1]*f[1] + c[1][2]*m
	force := delta * cb / (cb*g0 - sb*g1)
	return 2 * force * rc * (sb - sgn*mu*cb)
}

// flagLoad is the opposing torque at step k (theta = 36k + psi) for the flag mechanism, friction mu, slider weight ws, extra
// cam force fx of every segment that changes within 1.5 degrees of psi, and its count n: T_opp = 7 mu W 19.5 + 0.2 +
// n (W + F_x) h' + n mu F_x 19.5 + n mu 1.77 tan(a) (W + F_x) h', with h' the greatest lift slope in mm/rad and a the
// greatest pressure angle over those segments. The last term is the guide friction: the cam pushes each slider sideways
// by (W + F_x) tan(a) and its stem slot and blade tube react that with a lever of 1.77.
func flagLoad(dm sevensegment.Dimensions, k int, psi, mu, fx, ws float64) (float64, int) {
	slope, alpha, n := 0.0, 0.0, 0
	for _, s := range sevensegment.Segments() {
		changing := false
		for th := psi - 1.5; th <= psi+1.5+1e-9; th += 0.25 {
			t := 36*float64(k) + th
			sl := math.Abs(dm.Lift(s, t+0.05)-dm.Lift(s, t-0.05)) / (0.1 * math.Pi / 180)
			slope = math.Max(slope, sl)
			if sl > 1e-6 {
				changing = true
				alpha = math.Max(alpha, dm.PressureAngle(s, 90+t))
			}
		}
		if changing {
			n++
		}
	}
	fn := float64(n)
	guide := fn * mu * 1.77 * math.Tan(alpha*math.Pi/180) * (ws + fx) * slope
	return 7*mu*ws*19.5 + 2*0.1 + fn*(ws+fx)*slope + fn*mu*fx*19.5 + guide, n
}

// flagBudgetRatio is the detent's release torque over the opposing torque at step k and angle psi.
func flagBudgetRatio(dm sevensegment.Dimensions, k int, psi, mu, fx, ws float64) (float64, int) {
	load, n := flagLoad(dm, k, psi, mu, fx, ws)
	return detentTorque(dm, pawlCompliance(dm, pawlE), psi, mu, true) / load, n
}
