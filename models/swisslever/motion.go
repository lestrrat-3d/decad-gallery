package swisslever

import (
	"fmt"
	"math"
)

// Lock names which pallet holds the wheel.
const (
	LockedEntry = "entry"
	LockedExit  = "exit"
	LockedNone  = ""
)

// Lever stages.
const (
	LeverBanking = "banking"
	LeverUnlock  = "unlock"
	LeverImpulse = "impulse"
)

// Wheel stages.
const (
	WheelLockEntry       = "lock-entry"
	WheelToothLiftEntry  = "tooth-lift-entry"
	WheelPalletLiftEntry = "pallet-lift-entry"
	WheelLockExit        = "lock-exit"
	WheelToothLiftExit   = "tooth-lift-exit"
	WheelPalletLiftExit  = "pallet-lift-exit"
)

// Tooth indices: kEntry works the entry pallet on swing +, kExit the exit pallet on swing -, and the entry landing of
// swing - is kEntry + 1.
const (
	toothEntry = 2
	toothExit  = 0
)

// State is the movement's pose at one phase. The wheel and lever angles are clockwise-positive seen from +Z; the
// balance angle is counter-clockwise-positive.
type State struct {
	Phase        float64
	BalanceAngle float64 // θ
	LeverAngle   float64 // ψ
	WheelAngle   float64 // φ
	LeverStage   string
	WheelStage   string
	Locked       string
}

// Motion returns the closed-form state at p.Phase.
func Motion(p Params) (State, error) {
	g, err := Layout(p)
	if err != nil {
		return State{}, err
	}
	return g.StateAt(p.Phase)
}

// StateAt returns the state at phase t of the geometry's amplitude.
func (g *Geometry) StateAt(t float64) (State, error) {
	theta := -g.Params.Amplitude * math.Cos(2*math.Pi*t)
	plus := t <= 0.5
	psi, lever, err := g.LeverAt(theta, plus)
	if err != nil {
		return State{}, err
	}
	phi, wheel, err := g.WheelAt(psi, plus)
	if err != nil {
		return State{}, err
	}
	locked := LockedNone
	switch wheel {
	case WheelLockEntry:
		locked = LockedEntry
	case WheelLockExit:
		locked = LockedExit
	}
	return State{Phase: t, BalanceAngle: theta, LeverAngle: psi, WheelAngle: phi, LeverStage: lever,
		WheelStage: wheel, Locked: locked}, nil
}

// LeverAt returns the lever angle and its stage for balance angle theta on swing + (theta increasing) or swing -.
func (g *Geometry) LeverAt(theta float64, plus bool) (float64, string, error) {
	if !plus {
		psi, stage, err := g.LeverAt(-theta, true)
		return -psi, stage, err
	}
	switch {
	case theta < g.Theta1:
		return -g.Banking, LeverBanking, nil
	case theta < g.Theta2:
		psi, err := g.psiKR(theta)
		return psi, LeverUnlock, err
	case theta < g.Theta3:
		psi, err := g.psiKL(theta)
		return psi, LeverImpulse, err
	}
	return g.Banking, LeverBanking, nil
}

// lockAngle returns the wheel angle at which the toe of tooth k, at wheel-frame angle 90° + kτ, lies on the locking
// face that starts at corner c (lever frame) along direction u, with the lever at psi.
func (g *Geometry) lockAngle(k int, c, u Pt, psi float64) (float64, error) {
	a := g.P.Add(c.Rot(-psi))
	v := u.Rot(-psi)
	// |a + s v|² = R_t² with |v| = 1; the root on the side s >= 0 (the larger one).
	b := a.Dot(v)
	disc := b*b - (a.Dot(a) - ToeRadius*ToeRadius)
	if disc < 0 {
		return 0, fmt.Errorf("swisslever: the locking face misses the toe circle at lever angle %v", psi)
	}
	s := -b + math.Sqrt(disc)
	if s < -1e-9 {
		return 0, fmt.Errorf("swisslever: the toe lies behind the locking corner at lever angle %v", psi)
	}
	return 90*deg + float64(k)*g.Pitch - a.Add(v.Scale(s)).Angle(), nil
}

// toothPoint returns tooth k's point w (tooth 0's wheel-frame point) turned to tooth k.
func (g *Geometry) toothPoint(k int, w Pt) Pt { return w.Rot(float64(k) * g.Pitch) }

// clubAngle returns the wheel angle at which the corner c (lever frame) at lever angle psi lies on tooth k's club
// line, taking the root nearest ref.
func (g *Geometry) clubAngle(k int, c Pt, psi, ref float64) (float64, error) {
	q := g.P.Add(c.Rot(-psi))
	t, h := g.toothPoint(k, g.ToothT), g.toothPoint(k, g.ToothH)
	e := h.Sub(t)
	e = e.Scale(1 / e.Len())
	n := Pt{-e.Y, e.X}
	off := n.Dot(t)
	ratio := off / q.Len()
	if math.Abs(ratio) > 1 {
		return 0, fmt.Errorf("swisslever: the corner is off the club line's reach at lever angle %v", psi)
	}
	best, found := 0.0, false
	for _, sgn := range []float64{1, -1} {
		phi := n.Angle() - q.Angle() + sgn*math.Acos(ratio)
		phi = ref + normAngle(phi-ref)
		if !found || math.Abs(phi-ref) < math.Abs(best-ref) {
			best, found = phi, true
		}
	}
	return best, nil
}

// heelAngle returns the wheel angle at which tooth k's heel lies on the impulse face from corner c to corner f (lever
// frame) at lever angle psi.
func (g *Geometry) heelAngle(k int, c, f Pt, psi float64) (float64, error) {
	a := g.P.Add(c.Rot(-psi))
	b := g.P.Add(f.Rot(-psi))
	d := b.Sub(a)
	qa := d.Dot(d)
	qb := 2 * a.Dot(d)
	qc := a.Dot(a) - g.HeelRadius*g.HeelRadius
	disc := qb*qb - 4*qa*qc
	if disc < 0 {
		return 0, fmt.Errorf("swisslever: the impulse face misses the heel circle at lever angle %v", psi)
	}
	root := math.Sqrt(disc)
	for _, s := range []float64{(-qb - root) / (2 * qa), (-qb + root) / (2 * qa)} {
		if s >= -1e-9 && s <= 1+1e-9 {
			return g.toothPoint(k, g.ToothH).Angle() - a.Add(d.Scale(s)).Angle(), nil
		}
	}
	return 0, fmt.Errorf("swisslever: the impulse face does not reach the heel circle at lever angle %v", psi)
}

// WheelAt returns the wheel angle and its stage for lever angle psi on swing + (psi increasing) or swing -.
func (g *Geometry) WheelAt(psi float64, plus bool) (float64, string, error) {
	if plus {
		switch {
		case psi <= g.PsiEu:
			phi, err := g.lockAngle(toothEntry, g.CE, g.UE, psi)
			return phi, WheelLockEntry, err
		case psi <= g.PsiEt:
			phi, err := g.clubAngle(toothEntry, g.CE, psi, g.PhiE)
			return phi, WheelToothLiftEntry, err
		case psi <= g.PsiEf:
			phi, err := g.heelAngle(toothEntry, g.CE, g.FE, psi)
			return phi, WheelPalletLiftEntry, err
		}
		phi, err := g.lockAngle(toothExit, g.CX, g.UX, psi)
		return phi, WheelLockExit, err
	}
	switch {
	case psi >= g.PsiXu:
		phi, err := g.lockAngle(toothExit, g.CX, g.UX, psi)
		return phi, WheelLockExit, err
	case psi >= g.PsiXt:
		phi, err := g.clubAngle(toothExit, g.CX, psi, g.PhiX)
		return phi, WheelToothLiftExit, err
	case psi >= g.PsiXf:
		phi, err := g.heelAngle(toothExit, g.CX, g.FX, psi)
		return phi, WheelPalletLiftExit, err
	}
	phi, err := g.lockAngle(toothEntry+1, g.CE, g.UE, psi)
	return phi, WheelLockEntry, err
}
