package sevensegment_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestSevenSegmentTruthTable(t *testing.T) {
	// a b c d e f g per digit, as in instructions.md "Truth table and ramp table".
	want := [10]string{
		"1111110", "0110000", "1101101", "1111001", "0110011", "1011011", "1011111", "1110000", "1111111", "1111011",
	}
	for digit, row := range want {
		for i, s := range sevensegment.Segments() {
			lit, err := sevensegment.Lit(digit, s)
			require.NoError(t, err)
			require.Equal(t, row[i] == '1', lit, "digit %d segment %s", digit, s)
		}
	}
	_, err := sevensegment.Lit(10, sevensegment.SegA)
	require.Error(t, err)
	require.Equal(t, "a", sevensegment.SegA.String())
	require.Equal(t, "g", sevensegment.SegG.String())
}

func TestSevenSegmentValidate(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    sevensegment.Params
		want string
	}{
		{"low", sevensegment.Params{Digit: -1}, "sevensegment: digit -1 out of range 0..9"},
		{"high", sevensegment.Params{Digit: 10}, "sevensegment: digit 10 out of range 0..9"},
		{"placement", sevensegment.Params{Digit: 3, Placement: &r3.Frame{}}, "sevensegment: invalid placement frame"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.p.Validate()
			require.EqualError(t, err, tc.want)
			_, err = tc.p.Derive()
			require.EqualError(t, err, tc.want)
		})
	}
	require.NoError(t, sevensegment.DefaultParams().Validate())
}

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

func TestSevenSegmentCamProfile(t *testing.T) {
	dm, err := sevensegment.DefaultParams().Derive()
	require.NoError(t, err)
	maxPressure := 0.0
	activeLo, activeHi := 1e9, -1e9
	for _, s := range sevensegment.Segments() {
		// Each cam's ramp list matches the table.
		want := rampsPerCam[s]
		require.Len(t, dm.Ramps[s], len(want), "segment %s", s)
		for i, r := range dm.Ramps[s] {
			require.Equal(t, want[i].from, r.From, "segment %s ramp %d", s, i)
			require.Equal(t, want[i].rising, r.Rising, "segment %s ramp %d", s, i)
			require.InDelta(t, 90+36*float64(r.From)+8, r.Start, 1e-9)
			require.InDelta(t, r.Start+21.5, r.End, 1e-9)
		}
		for k := range 10 {
			lit, _ := sevensegment.Lit(k, s)
			level := 0.0
			if lit {
				level = sevensegment.Travel
			}
			for th := 36*float64(k) - 6.5; th <= 36*float64(k)+8.0+1e-9; th += 0.1 {
				require.InDelta(t, level, dm.Lift(s, th), 1e-9, "segment %s digit %d theta %.1f", s, k, th)
			}
		}
		// Ramps are monotonic and report the active zone and pressure angle.
		for _, r := range dm.Ramps[s] {
			prev := dm.GrooveRadius(s, r.Start)
			for psi := r.Start; psi <= r.End+1e-9; psi += 0.1 {
				cur := dm.GrooveRadius(s, psi)
				if r.Rising {
					require.GreaterOrEqual(t, cur, prev-1e-9)
				} else {
					require.LessOrEqual(t, cur, prev+1e-9)
				}
				prev = cur
				maxPressure = math.Max(maxPressure, dm.PressureAngle(s, psi))
				base := r.Start - 8 - 90
				lift := cur - sevensegment.OffRadius
				dwell := 0.0
				if !r.Rising {
					dwell = sevensegment.Travel
				}
				if math.Abs(lift-dwell) > 0.05 && math.Abs(lift-(sevensegment.Travel-dwell)) > 0.05 {
					off := math.Mod(psi-(90+base)+720, 36) // angle past psi_k
					activeLo, activeHi = math.Min(activeLo, off), math.Max(activeHi, off)
				}
			}
		}
	}
	require.LessOrEqual(t, maxPressure, 33.5)
	require.GreaterOrEqual(t, activeLo, 9.4)
	require.LessOrEqual(t, activeHi, 28.1)
	// Rising lift samples, "Truth table and ramp table".
	samples := map[float64]float64{8: 0, 9: 0.023, 10: 0.092, 12: 0.307, 14: 0.550, 16: 0.824, 18: 1.128, 20: 1.467,
		22: 1.843, 24: 2.258, 26: 2.696, 28: 2.946, 29: 2.994, 29.5: 3.0}
	for off, lift := range samples {
		got := dm.GrooveRadius(sevensegment.SegB, 90+36*4+off+0) // segment b, step 4->5 is falling
		_ = got
		rise := dm.GrooveRadius(sevensegment.SegB, 90+36*6+off) - sevensegment.OffRadius // step 6->7 rises
		require.InDelta(t, lift, rise, 0.005, "offset %g", off)
	}
}

func TestSevenSegmentRampGeometry(t *testing.T) {
	dm, err := sevensegment.DefaultParams().Derive()
	require.NoError(t, err)
	// Step 0->1 falls for cam a; step 1->2 rises (window starts at 134).
	rise := dm.Ramps[sevensegment.SegA][1].Geometry()
	require.InDelta(t, 20.5*math.Cos(134*math.Pi/180), rise.C1.X, 1e-9)
	require.InDelta(t, 18.5, math.Hypot(rise.C2.X, rise.C2.Y), 1e-9)
	require.InDelta(t, 5.6372, rise.LineLength, 1e-3)
	fall := dm.Ramps[sevensegment.SegA][0].Geometry()
	require.InDelta(t, 18.5, math.Hypot(fall.C1.X, fall.C1.Y), 1e-9)
	require.InDelta(t, 20.5, math.Hypot(fall.C2.X, fall.C2.Y), 1e-9)
}

func TestSevenSegmentDetent(t *testing.T) {
	startDigit0() // the driven-back budget reads the built rotors
	t.Parallel()
	dm, err := sevensegment.DefaultParams().Derive()
	require.NoError(t, err)
	require.GreaterOrEqual(t, dm.SeatAngle, 4.5)
	require.LessOrEqual(t, dm.SeatAngle, 5.0)
	require.InDelta(t, sevensegment.PawlAngle-dm.SeatAngle, dm.LockPhase, 1e-12)
	require.InDelta(t, sevensegment.RatchetTip+sevensegment.LooseFit, dm.PawlArmInner, 1e-9)
	require.InDelta(t, dm.PawlArmInner+sevensegment.PawlArmRoot, dm.PawlArmOuter, 1e-9)
	require.InDelta(t, 11.876, sevensegment.RatchetTip, 0.001)
	require.InDelta(t, 50, sevensegment.FlankAngle(10), 1e-9)
	require.InDelta(t, 33.036, sevensegment.FlankAngle(31.25), 1e-3)
	// The seat: TightFit past the tip round from the lock face line, PawlFlankGap from the flank (pawl frame: lock face on
	// the ray at -SeatAngle).
	seat := r3.Vec{X: dm.SeatRadius}
	lock := r3.Vec{X: math.Cos(-dm.SeatAngle * math.Pi / 180), Y: math.Sin(-dm.SeatAngle * math.Pi / 180)}
	require.InDelta(t, sevensegment.PawlTipRound+sevensegment.TightFit, math.Abs(lock.X*seat.Y-lock.Y*seat.X), 0.005)
	best := math.Inf(1)
	for rel := 0.0; rel <= 36; rel += 0.005 {
		rr := dm.RatchetRadius(dm.LockPhase + rel)
		a := (rel - dm.SeatAngle) * math.Pi / 180
		best = math.Min(best, seat.Sub(r3.Vec{X: rr * math.Cos(a), Y: rr * math.Sin(a)}).Len())
	}
	require.InDelta(t, sevensegment.PawlTipRound+sevensegment.PawlFlankGap, best, 0.005)
	// Root round: the 0.6 round's centre lies 0.6 from the lock face and the flank; the seat is farther from the corner.
	require.Greater(t, seat.Len(), sevensegment.RatchetRoot+0.6)

	delta0, _ := dm.PawlDeflection(0)
	require.Zero(t, delta0)
	contact := math.NaN()
	prev := 0.0
	for psi := 0.0; psi <= dm.CrestAngle+1e-9; psi += 0.05 {
		d, _ := dm.PawlDeflection(psi)
		require.GreaterOrEqual(t, d, prev-2e-4, "psi %.2f", psi)
		prev = d
		if math.IsNaN(contact) && d > 1e-3 {
			contact = psi
		}
	}
	require.GreaterOrEqual(t, contact, 0.5)
	require.LessOrEqual(t, contact, 2.0)
	require.GreaterOrEqual(t, dm.CrestAngle-1.0, 28.05+sevensegment.TimingAllowance)
	require.InDelta(t, 4.35, dm.LiftMax, 0.03)
	// Nominal arm strain at the crest: the radial force that lifts the head by LiftMax (C_uu), times the lever to the root,
	// over the root section (E cancels). The FE of the real outline reads 1.66 % at the root round (1.5 % with large
	// deflection); instructions.md "Arm strain".
	c := pawlCompliance(dm, pawlE)
	force := dm.LiftMax / c[0][0]
	root := sevensegment.PawlBackV + sevensegment.PawlArmLength
	tr := sevensegment.PawlArmRoot
	strain := 6 * force * root / (sevensegment.DetentThickness * tr * tr) / pawlE
	t.Logf("crest force %.2f N, nominal root strain %.3f %%", force, 100*strain)
	require.LessOrEqual(t, strain, 0.0145)

	// Budget at friction 0.35 over [7.98, 29.55].
	worst := math.Inf(1)
	for psi := 7.98; psi <= 29.55; psi += 0.1 {
		worst = math.Min(worst, worstFlagBudget(t, dm, psi))
	}
	require.GreaterOrEqual(t, worst, 3.0, "detent budget at friction 0.35")
}

func TestSevenSegmentChannel(t *testing.T) {
	startDigit0()
	t.Parallel()
	dm, err := sevensegment.DefaultParams().Derive()
	require.NoError(t, err)
	half := math.Asin(1.9/22.9) * 180 / math.Pi
	// The entry channel is at the follower when psi = EntryAngle + 360*n; it lies inside digit 8's dwell [11.5, 26].
	for _, s := range sevensegment.Segments() {
		for _, psi := range []float64{sevensegment.EntryAngle - half, sevensegment.EntryAngle, sevensegment.EntryAngle + half} {
			require.InDelta(t, sevensegment.OnRadius, dm.GrooveRadius(s, psi), 1e-9, "segment %s psi %.2f", s, psi)
		}
	}
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
