package swisslever

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/spiralspring"
	"github.com/lestrrat-3d/r3"
)

// Fixed constants of instructions.md "Escapement geometry", in mm and radians.
const (
	// Teeth is the escape wheel tooth count N.
	Teeth = 15
	// ToeRadius is R_t, the radius of each tooth's locking corner.
	ToeRadius = 38.6
	// RootRadius is R_r, the escape wheel root circle.
	RootRadius = 32.0
	// HalfSpan is θ_s, half the angle the pallets span.
	HalfSpan = 30 * deg
	// Lock is κ, the lever angle from first lock to unlock at the drop.
	Lock = 2.5 * deg
	// ToothLift is λ_t, the lever angle while the pallet's locking corner rides the tooth's club face.
	ToothLift = 4.0 * deg
	// PalletLift is λ_p, the lever angle while the tooth's heel rides the pallet's impulse face.
	PalletLift = 4.5 * deg
	// Slide is σ, the run to the banking after the drop.
	Slide = 1.5 * deg
	// Draw is δ, the tilt of each locking face from the arc about the pallet arbor.
	Draw = 12 * deg
	// ClubWidth is a_c, the angle about the wheel centre from toe to heel.
	ClubWidth = 3 * deg
	// PalletWidth is w, the pallet width measured radially about the pallet arbor.
	PalletWidth = 4.6
	// Rake is ρ, the lean of the tooth's front face from the inward radial at the toe.
	Rake = 16 * deg
	// BackAngle is χ, the lean of the tooth's back face from the inward radial at the heel.
	BackAngle = 20 * deg
	// StoneLength is L_s, the pallet stone length along its locking face.
	StoneLength = 7.0
	// StoneKey is β_b, the turn of each stone's back face toward its locking face.
	StoneKey = 6 * deg
	// Overlap is ov, the overlap of every stacked or patterned union.
	Overlap = 0.5
	// SlotPress is how far each slot stands out from its stone.
	SlotPress = 0.05
	// SlotWall is the lever's wall round each slot.
	SlotWall = 1.95
)

// Fixed constants of instructions.md "Lever, fork and guard", in mm and radians.
const (
	// Distance is D, the pallet arbor to balance staff distance.
	Distance = 48.0
	// PinCentreRadius is r_j, the impulse pin's centre radius about the balance staff.
	PinCentreRadius = 8.5
	// PinRadius is r_p, the impulse pin radius.
	PinRadius = 1.5
	// SlotShake is c_s, the gap from the centred pin to each corner round.
	SlotShake = 0.25
	// CornerDirection is β_K, the direction of each corner round's centre from the centred pin, from -Y.
	CornerDirection = 75 * deg
	// ArmHalfWidth is b_a, the half width of the lever's main arm.
	ArmHalfWidth = 3.0
	// DartRadius is r_d, the dart tip radius.
	DartRadius = 0.6
	// DartReach is e, the dart tip's distance short of the balance staff.
	DartReach = 5.85
	// SafetyRadius is r_s, the safety roller radius.
	SafetyRadius = 6.444
	// CrescentRadius is r_cr, the safety roller crescent's radius.
	CrescentRadius = 4.03
	// CrescentOffset is the crescent centre's distance from the balance staff.
	CrescentOffset = 9.0
	// CrescentLip is the round at both crescent lips.
	CrescentLip = 0.4
	// BankingPostRadius is the banking posts' radius.
	BankingPostRadius = 2.0
	// BankingPostHeight is the height of a banking post above the back plate's inner face.
	BankingPostHeight = 9.80
	// BankingY is the lever-frame y of the banking contact.
	BankingY = 18.0
	// BuildBackoff is how far the wheel and the lever are backed off their last contact at the build pose.
	BuildBackoff = 0.05 * deg
)

// Balance, hairspring and frame constants.
const (
	// PETGModulus is the hairspring's Young's modulus in N/mm².
	PETGModulus = 2100.0
	// PLADensity is the density of PLA in g/mm³.
	PLADensity = 1.24e-3
	// FrontPlateInner is the front plate's inner face height.
	FrontPlateInner = 27.0
	// HeadTop is the height of the bolts' head tops.
	HeadTop = 33.0
	// BoltLength is the M2 bolt length.
	BoltLength = 10.0
	// NutCavityAt is the nut cavity's near face, measured from the head top.
	NutCavityAt = 8.0
	// HubChordHalf is h, half the chord where a plate bar's hub ring of radius 10 meets the bar edge.
	HubChordHalf = 9.16515138991168
)

// StoneSection holds one pallet stone's section corners in the lever frame: the locking corner C, the letting-off
// corner F, the back corner G and the top corner Hs. The faces are the locking face C-Hs, the impulse face C-F, the
// back face F-G and the top G-Hs.
type StoneSection struct {
	C, F, G, Hs Pt
}

// Corners returns the corners in order.
func (s StoneSection) Corners() [4]Pt { return [4]Pt{s.C, s.F, s.G, s.Hs} }

// Geometry holds every derived value and point of the design. Angles are in radians. Points are in the model frame
// (origin on the escape arbor axis) unless a field says otherwise.
type Geometry struct {
	Params Params

	// Pitch is τ; Lift is λ; Banking is α; CentreDistance is d; LockRadius is r_m.
	Pitch, Lift, Banking, CentreDistance, LockRadius float64

	// O is the escape arbor, P the pallet arbor and B the balance staff.
	O, P, B Pt

	// Event angles of the lever: entry unlocks, entry tooth lift ends, entry releases, and the exit equivalents.
	PsiEu, PsiEt, PsiEf, PsiXu, PsiXt, PsiXf float64

	// IL and IR are the lock points on the toe circle.
	IL, IR Pt
	// CE and CX are the locking corners, FE and FX the letting-off corners, relative to P (lever frame).
	CE, CX, FE, FX Pt
	// HeelRadius is R_h.
	HeelRadius float64
	// XE and XX are the release points (world).
	XE, XX Pt
	// UE and UX are the locking face directions (unit vectors in the lever frame); UEAngle and UXAngle their angles.
	UE, UX           Pt
	UEAngle, UXAngle float64
	// PhiE and PhiX are the wheel angles at the entry and the exit lock.
	PhiE, PhiX float64

	// ToothT, ToothH, ToothL and ToothK are tooth 0's toe, heel, front foot and back foot (wheel frame).
	ToothT, ToothH, ToothL, ToothK Pt

	// Entry and Exit are the stone sections, EntrySlot and ExitSlot the lever's slots (corners C', F', G', H').
	Entry, Exit         StoneSection
	EntrySlot, ExitSlot [4]Pt

	// Lever outline (lever frame).
	JL, JR, VE1, VE2, VE3, VX1, VX2, VX3, VX4 Pt

	// Fork. CornerRadius is r_c, KR and KL the corner round centres, NotchRadius r_n, NotchCentre N_c.
	CornerRadius, NotchRadius float64
	KR, KL, NotchCentre       Pt
	// GapRadius is the dart root's gap round radius and GapRight, GapLeft their centres; DartTip is T_d.
	GapRadius         float64
	GapRight, GapLeft Pt
	DartTip           Pt
	// PinAngle is a_p, the local angle of the impulse pin and the crescent centre on the balance parts.
	PinAngle float64

	// Fork action: the events of swing +, the jump angle, the guard angle and the freedoms.
	Theta1, Theta2, ThetaDrop, Theta3 float64
	JumpPsi                           float64
	GuardAngle                        float64
	GuardFreedom                      float64
	GuardGap                          float64
	CornerFreedomBanking              float64
	CornerFreedomGuard                float64

	// Hairspring.
	Spring                    spiralspring.Dimensions
	GammaS, GammaF            float64
	EyeCentre                 Pt
	SpringRate                float64
	BalanceInertia, Period    float64
	WheelMass                 float64
	BankingLeft, BankingRight Pt
	Pillars                   [4]Pt
	Stud                      Pt
	Sites                     []m2fastener.Site
}

// SpringParams returns the hairspring's spiralspring parameters, without a placement.
func SpringParams() spiralspring.Params {
	sp := spiralspring.DefaultParams()
	sp.OuterDiameter, sp.HubDiameter, sp.Thickness, sp.Height, sp.MinGap = 60, 12, 1.5, 3, 1.0
	sp.Turns = 0
	sp.ArborDiameter, sp.ArborFlatDepth, sp.BoreClearance = 6.6, 0.75, 0.25
	sp.AnchorPinDiameter, sp.AnchorClearance = 3.0, 0.4
	return sp
}

// Layout computes the geometry of the movement for p.
func Layout(p Params) (Geometry, error) {
	if err := p.Validate(); err != nil {
		return Geometry{}, err
	}
	g := Geometry{Params: p}
	if err := g.layoutEscapement(); err != nil {
		return Geometry{}, err
	}
	g.layoutLever()
	if err := g.layoutFork(); err != nil {
		return Geometry{}, err
	}
	if err := g.layoutBalance(); err != nil {
		return Geometry{}, err
	}
	g.layoutFrame()
	return g, nil
}

// layoutEscapement computes the event angles, the lock and release points, the tooth and the stones.
func (g *Geometry) layoutEscapement() error {
	g.Pitch = 2 * math.Pi / Teeth
	g.Lift = ToothLift + PalletLift
	g.Banking = (g.Lift+Lock)/2 + Slide
	g.CentreDistance = ToeRadius / math.Cos(HalfSpan)
	g.LockRadius = ToeRadius * math.Tan(HalfSpan)
	g.P = Pt{0, g.CentreDistance}
	g.B = Pt{0, g.CentreDistance + Distance}

	g.PsiEu = (Lock - g.Lift) / 2
	g.PsiEt = g.PsiEu + ToothLift
	g.PsiEf = (g.Lift + Lock) / 2
	g.PsiXu, g.PsiXt, g.PsiXf = -g.PsiEu, -g.PsiEt, -g.PsiEf

	g.IL = Polar(ToeRadius, 90*deg+HalfSpan)
	g.IR = Polar(ToeRadius, 90*deg-HalfSpan)
	g.CE = g.IL.Sub(g.P).Rot(g.PsiEu)
	g.CX = g.IR.Sub(g.P).Rot(g.PsiXu)
	g.HeelRadius = g.P.Add(g.CE.Rot(-g.PsiEt)).Len()

	var err error
	if g.XE, err = g.releasePoint(g.LockRadius-PalletWidth, true); err != nil {
		return err
	}
	if g.XX, err = g.releasePoint(g.LockRadius+PalletWidth, false); err != nil {
		return err
	}
	g.FE = g.XE.Sub(g.P).Rot(g.PsiEf)
	g.FX = g.XX.Sub(g.P).Rot(g.PsiXf)

	g.UE = g.CE.Rot(90 * deg).Scale(-1 / g.LockRadius).Rot(-Draw)
	g.UX = g.CX.Rot(90 * deg).Scale(1 / g.LockRadius).Rot(-Draw)
	g.UEAngle, g.UXAngle = g.UE.Angle(), g.UX.Angle()

	mod := func(a, m float64) float64 { return a - m*math.Floor(a/m) }
	g.PhiE = mod(90*deg-g.IL.Angle(), g.Pitch)
	g.PhiX = g.PhiE + g.Pitch/2

	g.ToothT = Pt{0, ToeRadius}
	g.ToothH = Polar(g.HeelRadius, 90*deg+ClubWidth)
	if g.ToothL, err = toothFoot(g.ToothT, Pt{-math.Sin(Rake), -math.Cos(Rake)}); err != nil {
		return err
	}
	inward := g.ToothH.Scale(-1 / g.ToothH.Len()).Rot(-BackAngle)
	if g.ToothK, err = toothFoot(g.ToothH, inward); err != nil {
		return err
	}

	g.Entry, err = g.stone(g.CE, g.FE, g.UE)
	if err != nil {
		return err
	}
	g.Exit, err = g.stone(g.CX, g.FX, g.UX)
	return err
}

// releasePoint returns the intersection of the heel circle with the circle of radius r about P, on the left or right.
func (g *Geometry) releasePoint(r float64, left bool) (Pt, error) {
	d := g.CentreDistance
	y := (g.HeelRadius*g.HeelRadius - r*r + d*d) / (2 * d)
	x2 := g.HeelRadius*g.HeelRadius - y*y
	if x2 < 0 {
		return Pt{}, fmt.Errorf("swisslever: the heel circle misses the circle of radius %v about the pallet arbor", r)
	}
	x := math.Sqrt(x2)
	if left {
		x = -x
	}
	return Pt{x, y}, nil
}

// toothFoot returns where the line from start along dir meets the circle of radius RootRadius - Overlap.
func toothFoot(start, dir Pt) (Pt, error) {
	rf := RootRadius - Overlap
	// |start + s dir|² = rf² with |dir| = 1: s² + 2 s b + c = 0, nearer root.
	b := start.Dot(dir)
	disc := b*b - (start.Dot(start) - rf*rf)
	if disc < 0 {
		return Pt{}, fmt.Errorf("swisslever: a tooth face misses the root circle")
	}
	s := -b - math.Sqrt(disc)
	return start.Add(dir.Scale(s)), nil
}

// stone returns the stone section for the locking corner c, letting-off corner f and locking face direction u.
func (g *Geometry) stone(c, f, u Pt) (StoneSection, error) {
	hs := c.Add(u.Scale(StoneLength))
	back := u.Rot(StoneKey)
	gg, ok := lineIntersect(f, back, hs, f.Sub(c))
	if !ok {
		return StoneSection{}, fmt.Errorf("swisslever: a stone's back face is parallel to its top")
	}
	return StoneSection{C: c, F: f, G: gg, Hs: hs}, nil
}

// side is one stone face moved outward: a point on it and its direction.
type side struct{ at, dir Pt }

// sides returns the four faces of s moved out by dist away from the section's centroid, in the order impulse C-F, back
// F-G, top G-Hs, locking Hs-C.
func (s StoneSection) sides(dist float64) [4]side {
	c := s.Corners()
	var cen Pt
	for _, v := range c {
		cen = cen.Add(v)
	}
	cen = cen.Scale(0.25)
	var out [4]side
	for i := range c {
		a, b := c[i], c[(i+1)%4]
		e := b.Sub(a)
		e = e.Scale(1 / e.Len())
		n := Pt{-e.Y, e.X}
		if n.Dot(a.Sub(cen)) < 0 {
			n = n.Scale(-1)
		}
		out[i] = side{at: a.Add(n.Scale(dist)), dir: e}
	}
	return out
}

const (
	sideImpulse = iota
	sideBack
	sideTop
	sideLocking
)

// corner returns the intersection of faces i and j.
func corner(s [4]side, i, j int) Pt {
	p, _ := lineIntersect(s[i].at, s[i].dir, s[j].at, s[j].dir)
	return p
}

// slot returns the slot of s: its sides moved out by SlotPress, as the corners C', F', G', H'.
func slot(s StoneSection) [4]Pt {
	sd := s.sides(SlotPress)
	return [4]Pt{corner(sd, sideLocking, sideImpulse), corner(sd, sideImpulse, sideBack),
		corner(sd, sideBack, sideTop), corner(sd, sideTop, sideLocking)}
}

// layoutLever computes the slots and the lever outline.
func (g *Geometry) layoutLever() {
	g.EntrySlot, g.ExitSlot = slot(g.Entry), slot(g.Exit)
	out := SlotPress + SlotWall
	e, x := g.Entry.sides(out), g.Exit.sides(out)
	g.VE1 = corner(e, sideLocking, sideTop)
	g.VE2 = corner(e, sideLocking, sideImpulse)
	g.VE3 = corner(e, sideImpulse, sideBack)
	g.VX1 = corner(x, sideLocking, sideImpulse)
	g.VX2 = corner(x, sideImpulse, sideBack)
	g.VX3 = corner(x, sideBack, sideTop)
	g.VX4 = corner(x, sideTop, sideLocking)
	g.JL, g.JR = Pt{-ArmHalfWidth, 12}, Pt{ArmHalfWidth, 12}
}

// pinCentre returns the impulse pin's centre in the P frame (world axes) at balance angle theta.
func pinCentre(theta float64) Pt {
	return Pt{0, Distance}.Add(Pt{0, -PinCentreRadius}.Rot(theta))
}

// contactAngle returns the lever angle at which the corner round centred at kc touches the pin whose centre is q (P
// frame, world axes), choosing the root that leaves the round on its own side of the pin.
func (g *Geometry) contactAngle(q, kc Pt) (float64, error) {
	r := PinRadius + g.CornerRadius
	x1, x2, ok := circleCircle(kc.Len(), q, r)
	if !ok {
		return 0, fmt.Errorf("swisslever: the pin at %v, %v does not touch the corner round", q.X, q.Y)
	}
	sign := 1.0
	if kc.X < 0 {
		sign = -1
	}
	for _, x := range []Pt{x1, x2} {
		psi := normAngle(kc.Angle() - x.Angle())
		if (kc.X-q.Rot(psi).X)*sign > 0 {
			return psi, nil
		}
	}
	return 0, fmt.Errorf("swisslever: no contact root leaves the corner round on its own side")
}

// psiKR returns the lever angle at which the right corner round touches the pin at balance angle theta; psiKL is the
// left round's.
func (g *Geometry) psiKR(theta float64) (float64, error) {
	return g.contactAngle(pinCentre(theta), g.KR)
}
func (g *Geometry) psiKL(theta float64) (float64, error) {
	return g.contactAngle(pinCentre(theta), g.KL)
}

// scanRoot finds the first sign change of f from the negative to the positive side (or the reverse) on [from, to] in
// steps of step, then bisects it to 1e-12 rad. An error from f or no sign change is returned.
func scanRoot(f func(float64) (float64, error), from, to, step float64) (float64, error) {
	prev, err := f(from)
	if err != nil {
		return 0, err
	}
	for a := from + step; a <= to+1e-12; a += step {
		v, err := f(a)
		if err != nil {
			return 0, err
		}
		if (prev < 0) == (v < 0) {
			prev = v
			continue
		}
		lo, hi := a-step, a
		for hi-lo > 1e-12 {
			m := (lo + hi) / 2
			mv, err := f(m)
			if err != nil {
				return 0, err
			}
			if (mv < 0) == (prev < 0) {
				lo = m
			} else {
				hi = m
			}
		}
		return (lo + hi) / 2, nil
	}
	return 0, fmt.Errorf("swisslever: no sign change between %v and %v", from, to)
}

// layoutFork computes the corner rounds, the dart, the pin-in-fork events and the guard.
func (g *Geometry) layoutFork() error {
	sb, cb := math.Sincos(CornerDirection)
	g.CornerRadius = (ArmHalfWidth - (PinRadius+SlotShake)*sb) / (1 + sb)
	k := Pt{0, Distance - PinCentreRadius}.Add(Pt{sb, -cb}.Scale(PinRadius + g.CornerRadius + SlotShake))
	g.KR, g.KL = k, Pt{-k.X, k.Y}
	g.NotchRadius = g.KR.X - g.CornerRadius
	g.NotchCentre = Pt{0, g.KR.Y}
	g.GapRadius = (g.NotchRadius - DartRadius) / 2
	g.GapRight = Pt{DartRadius + g.GapRadius, g.KR.Y}
	g.GapLeft = Pt{-g.GapRight.X, g.GapRight.Y}
	g.DartTip = Pt{0, Distance - DartReach}

	alpha := g.Banking
	const step = 0.05 * deg
	var err error
	g.Theta1, err = scanRoot(func(th float64) (float64, error) {
		return pinCentre(th).Rot(-alpha).Sub(g.KR).Len() - (PinRadius + g.CornerRadius), nil
	}, -60*deg, 0, step)
	if err != nil {
		return fmt.Errorf("swisslever: theta1: %w", err)
	}
	g.Theta2, err = scanRoot(func(th float64) (float64, error) {
		psi, err := g.psiKR(th)
		return psi - g.PsiEu, err
	}, g.Theta1+1e-9, 0, step)
	if err != nil {
		return fmt.Errorf("swisslever: theta2: %w", err)
	}
	if g.JumpPsi, err = g.psiKL(g.Theta2); err != nil {
		return fmt.Errorf("swisslever: jump angle: %w", err)
	}
	g.ThetaDrop, err = scanRoot(func(th float64) (float64, error) {
		psi, err := g.psiKL(th)
		return psi - g.PsiEf, err
	}, g.Theta2, 60*deg, step)
	if err != nil {
		return fmt.Errorf("swisslever: drop angle: %w", err)
	}
	g.Theta3, err = scanRoot(func(th float64) (float64, error) {
		psi, err := g.psiKL(th)
		return psi - alpha, err
	}, g.ThetaDrop, 60*deg, step)
	if err != nil {
		return fmt.Errorf("swisslever: theta3: %w", err)
	}

	// Guard: the dart tip against the safety roller, with the roller's centre seen at lever angle psi.
	gap := func(psi float64) float64 {
		return g.DartTip.Sub(Pt{0, Distance}.Rot(psi)).Len() - DartRadius - SafetyRadius
	}
	g.GuardGap = gap(alpha)
	lo, hi := 0.0, alpha
	if gap(lo) >= 0 || gap(hi) <= 0 {
		return fmt.Errorf("swisslever: the dart does not meet the safety roller between 0 and the banking")
	}
	for hi-lo > 1e-14 {
		m := (lo + hi) / 2
		if gap(m) < 0 {
			lo = m
		} else {
			hi = m
		}
	}
	g.GuardAngle = (lo + hi) / 2
	g.GuardFreedom = alpha - g.GuardAngle
	free := func(psi float64) float64 {
		return Pt{0, Distance}.Rot(psi).Sub(g.KR).Len() - (g.CornerRadius + PinCentreRadius + PinRadius)
	}
	g.CornerFreedomBanking = free(alpha)
	g.CornerFreedomGuard = free(g.GuardAngle)
	return nil
}

// layoutBalance computes the hairspring placement and the balance's inertia and period.
func (g *Geometry) layoutBalance() error {
	dims, err := SpringParams().Derive()
	if err != nil {
		return fmt.Errorf("swisslever: hairspring: %w", err)
	}
	g.Spring = dims
	eye := Pt{dims.EyeCenterX, dims.EyeCenterY}
	g.GammaS = normAngle(math.Pi - eye.Angle())
	g.GammaF = g.GammaS + 90*deg
	g.EyeCentre = g.B.Add(Pt{-eye.Len(), 0})
	sp := SpringParams()
	g.SpringRate = PETGModulus * sp.Height * math.Pow(sp.Thickness, 3) / (12 * dims.ActiveLength)
	g.WheelMass, g.BalanceInertia = balanceInertia()
	g.PinAngle = -90*deg - g.GammaF
	g.Period = 2 * math.Pi * math.Sqrt(g.BalanceInertia*1e-9/(g.SpringRate*1e-3))
	return nil
}

// layoutFrame computes the banking posts, the pillars, the hairspring stud and the fastener sites.
func (g *Geometry) layoutFrame() {
	post := func(sign float64) Pt {
		return g.P.Add(Pt{sign * (ArmHalfWidth + BankingPostRadius), BankingY}.Rot(-sign * g.Banking))
	}
	g.BankingLeft, g.BankingRight = post(-1), post(1)
	g.Pillars = [4]Pt{{-48, -40}, {48, -40}, {-48, 136}, {48, 136}}
	g.Stud = g.EyeCentre
	for _, c := range g.Pillars {
		g.Sites = append(g.Sites, m2fastener.Site{Head: r3.Vec{X: c.X, Y: c.Y, Z: HeadTop}, Dir: r3.Vec{Z: -1},
			Corner: r3.Vec{X: 1}, Length: BoltLength, Cavity: NutCavityAt})
	}
}
