package escapement

import (
	"fmt"
	"math"
)

// Fixed constants of instructions.md "Escapement geometry", in mm and radians.
const (
	// Teeth is the escape wheel's tooth count N.
	Teeth = 30
	// TipRadius is the tooth tip radius R.
	TipRadius = 40.0
	// ToothHeight is the tooth height H.
	ToothHeight = 7.0
	// RootRadius is the escape wheel's root radius R - H.
	RootRadius = TipRadius - ToothHeight
	// Rake is ρ, the angle from the radial line at the tip to the tooth's front face.
	Rake = 10 * deg
	// BackAngle is χ, the angle from the radial line at the tip to the tooth's back face.
	BackAngle = 30 * deg
	// PalletSpan is the number of tooth pitches the pallets span.
	PalletSpan = 7.5
	// PalletWidth is the pallet width w, measured radially about the pallet arbor.
	PalletWidth = 2.8
	// PalletLength is Lpal, the pallet length about the pallet arbor beyond the locking corner.
	PalletLength = 9 * deg
	// ArmWidth is the anchor arm width b.
	ArmWidth = 6.0
	// HubRadius is the anchor hub radius rh.
	HubRadius = 8.0
	// Overlap is the overlap of every stacked or patterned union.
	Overlap = 0.5
	// SuspensionOffset is the distance from the pallet arbor to the pendulum suspension.
	SuspensionOffset = 14.0
	// CrutchLength is the distance from the pallet arbor to the crutch pin.
	CrutchLength = 45.0

	// Module and the tooth counts of the gear pair.
	Module         = 1.5
	PinionTeeth    = 13
	GreatTeeth     = 48
	PressureAngle  = 20 * deg
	GearTipMargin  = 0.02
	GearRootMargin = 0.02
	// Drive pose: the drum's local frame sits at this great-wheel-local angle.
	DrumOffset = 20.8 * deg
	// BuildBackoff is how far the wheel is backed off at the build pose.
	BuildBackoff = 0.05 * deg
)

// Pt is a point in a plane, in mm.
type Pt struct{ X, Y float64 }

// Angle returns the math angle of p about the origin.
func (p Pt) Angle() float64 { return math.Atan2(p.Y, p.X) }

// Len returns the distance of p from the origin.
func (p Pt) Len() float64 { return math.Hypot(p.X, p.Y) }

// Add returns p + q.
func (p Pt) Add(q Pt) Pt { return Pt{p.X + q.X, p.Y + q.Y} }

// Sub returns p - q.
func (p Pt) Sub(q Pt) Pt { return Pt{p.X - q.X, p.Y - q.Y} }

// Rot turns p by the math angle a about the origin.
func (p Pt) Rot(a float64) Pt {
	s, c := math.Sincos(a)
	return Pt{c*p.X - s*p.Y, s*p.X + c*p.Y}
}

// Polar returns the point (r at math angle a).
func Polar(r, a float64) Pt {
	s, c := math.Sincos(a)
	return Pt{r * c, r * s}
}

// Geometry holds every derived value and point of the design. Points named in "world" are in the model frame; the
// anchor and crutch points are relative to the pallet arbor, the pendulum points relative to the suspension.
type Geometry struct {
	Params Params

	// Pitch is τ; HalfSpan is θ; CentreDistance is d; TangentRadius is rm; RIn and ROut bound the pallet band.
	Pitch, HalfSpan, CentreDistance, TangentRadius, RIn, ROut float64

	// Axes: O escape arbor, P pallet arbor, S suspension, G great wheel arbor.
	O, P, S, G Pt

	// IL, FL, IR, FR are the lock and release points on the tip circle (world).
	IL, FL, IR, FR Pt

	// Event angles of the anchor.
	PsiEu, PsiEf, PsiXu, PsiXf float64

	// Pallet corners (relative to P), and the pallet top angles.
	CE, FE, CX, FX Pt
	TopE, TopX     float64

	// Arm points (relative to P).
	GE, HE, KE, ME, NE Pt
	GX, HX, KX, MX, NX Pt
	S0                 float64

	// Tooth feet on radius Rr - Overlap, relative to O, tooth 0 with its tip at (0, R).
	ToothTip, ToothL, ToothB Pt

	// Crutch outline points relative to P.
	CrutchAR, CrutchAL, CrutchBL, CrutchBR Pt
	// Pendulum outline points relative to S.
	PendER, PendEL, PendQL, PendQR Pt

	// Gear pair.
	PinionPitchR, GreatPitchR      float64
	PinionRoot, PinionTip          float64
	GreatRoot, GreatTip            float64
	GearCentre                     float64
	PinionPhase, GreatPhase        float64
	WheelPhaseE, WheelPhaseX, Drop float64
	TipToRootA, TipToRootB         float64
	PendulumPeriod                 float64
}

// tip returns the point where the tip circle meets the circle of radius r about P, on the left (entry) or right side.
func tip(g *Geometry, r float64, left bool) Pt {
	d := g.CentreDistance
	y := (TipRadius*TipRadius - r*r + d*d) / (2 * d)
	x := math.Sqrt(TipRadius*TipRadius - y*y)
	if left {
		x = -x
	}
	return Pt{x, y}
}

// Layout computes the geometry of the movement for p.
func Layout(p Params) (Geometry, error) {
	if err := p.Validate(); err != nil {
		return Geometry{}, err
	}
	g := Geometry{Params: p}
	g.Pitch = 2 * math.Pi / Teeth
	g.HalfSpan = PalletSpan * g.Pitch / 2
	g.CentreDistance = TipRadius / math.Cos(g.HalfSpan)
	g.TangentRadius = TipRadius * math.Tan(g.HalfSpan)
	g.RIn = g.TangentRadius - PalletWidth/2
	g.ROut = g.TangentRadius + PalletWidth/2
	g.O = Pt{}
	g.P = Pt{0, g.CentreDistance}
	g.S = Pt{0, g.CentreDistance + SuspensionOffset}
	g.G = Polar(m2c(), -90*deg)

	g.IL, g.FL = tip(&g, g.ROut, true), tip(&g, g.RIn, true)
	g.IR, g.FR = tip(&g, g.RIn, false), tip(&g, g.ROut, false)

	lam, kap := p.LiftAngle, p.LockAngle
	g.PsiEu, g.PsiEf = -(lam-kap)/2, (lam+kap)/2
	g.PsiXu, g.PsiXf = (lam-kap)/2, -(lam+kap)/2

	g.CE = g.IL.Sub(g.P).Rot(g.PsiEu)
	g.FE = g.FL.Sub(g.P).Rot(g.PsiEf)
	g.CX = g.IR.Sub(g.P).Rot(g.PsiXu)
	g.FX = g.FR.Sub(g.P).Rot(g.PsiXf)
	g.TopE = g.CE.Angle() - PalletLength
	g.TopX = g.CX.Angle() + PalletLength

	g.S0 = math.Sqrt(HubRadius*HubRadius - ArmWidth*ArmWidth)
	uE, nE := Polar(1, g.TopE), Polar(1, g.TopE-90*deg)
	uX, nX := Polar(1, g.TopX), Polar(1, g.TopX+90*deg)
	scale := func(u Pt, k float64) Pt { return Pt{u.X * k, u.Y * k} }
	g.GE, g.HE, g.KE = scale(uE, g.RIn), scale(uE, HubRadius), scale(uE, g.ROut)
	g.ME = g.KE.Add(scale(nE, ArmWidth))
	g.NE = scale(uE, g.S0).Add(scale(nE, ArmWidth))
	g.GX, g.HX, g.KX = scale(uX, g.RIn), scale(uX, HubRadius), scale(uX, g.ROut)
	g.MX = g.KX.Add(scale(nX, ArmWidth))
	g.NX = scale(uX, g.S0).Add(scale(nX, ArmWidth))

	// Tooth 0: tip (0, R); feet on r = Rr - Overlap, on the lines at Rake and BackAngle from the inward radial,
	// leaning toward -X.
	g.ToothTip = Pt{0, TipRadius}
	g.ToothL = toothFoot(Rake)
	g.ToothB = toothFoot(BackAngle)

	// Crutch (origin P): hub r 6, arm half width 2.5, end circle r 4 at (0, -45).
	g.CrutchAR = Pt{2.5, -math.Sqrt(36 - 2.5*2.5)}
	g.CrutchAL = Pt{-2.5, g.CrutchAR.Y}
	g.CrutchBL = Pt{-2.5, -CrutchLength + math.Sqrt(16-2.5*2.5)}
	g.CrutchBR = Pt{2.5, g.CrutchBL.Y}
	// Pendulum (origin S): eye r 10, rod half width 4, bob r 25 at (0, -Lp).
	g.PendER = Pt{4, -math.Sqrt(100 - 16)}
	g.PendEL = Pt{-4, g.PendER.Y}
	g.PendQL = Pt{-4, -p.PendulumLength + math.Sqrt(625-16)}
	g.PendQR = Pt{4, g.PendQL.Y}

	// Gear pair.
	g.PinionPitchR = Module * PinionTeeth / 2
	g.GreatPitchR = Module * GreatTeeth / 2
	g.GearCentre = g.PinionPitchR + g.GreatPitchR
	clearance := math.Max(0.25*Module, 0.5)
	g.PinionRoot = g.PinionPitchR - Module - clearance - GearRootMargin
	g.PinionTip = g.PinionPitchR + Module - GearTipMargin
	g.GreatRoot = g.GreatPitchR - Module - clearance - GearRootMargin
	g.GreatTip = g.GreatPitchR + Module - GearTipMargin
	g.TipToRootA = g.GearCentre - g.PinionTip - g.GreatRoot
	g.TipToRootB = g.GearCentre - g.GreatTip - g.PinionRoot
	g.PinionPhase = -90 * deg
	g.GreatPhase = 90*deg + math.Pi/GreatTeeth

	g.WheelPhaseE = math.Mod(math.Mod(90*deg-g.IL.Angle(), g.Pitch)+g.Pitch, g.Pitch)
	g.WheelPhaseX = g.WheelPhaseE + g.Pitch/2
	g.Drop = g.Pitch/2 - (g.IL.Angle() - g.FL.Angle())
	g.PendulumPeriod = 2 * math.Pi * math.Sqrt(p.PendulumLength/9810)
	if math.IsNaN(g.CE.X) {
		return Geometry{}, fmt.Errorf("escapement: layout produced a nonfinite corner")
	}
	return g, nil
}

// m2c is the gear centre distance, module times half the tooth sum.
func m2c() float64 { return Module * (PinionTeeth + GreatTeeth) / 2 }

// toothFoot returns the foot of the tooth face that leans by angle from the inward radial at the tip (0, R), on the
// circle of radius RootRadius - Overlap, toward -X.
func toothFoot(angle float64) Pt {
	dir := Pt{-math.Sin(angle), -math.Cos(angle)}
	rf := RootRadius - Overlap
	// |T + s dir| = rf with T = (0, R): s² + 2 s R dir.Y + R² - rf² = 0, nearer root.
	b := TipRadius * dir.Y
	s := -b - math.Sqrt(b*b-(TipRadius*TipRadius-rf*rf))
	return Pt{s * dir.X, TipRadius + s*dir.Y}
}
