package escapement

import (
	"fmt"
	"math"
)

// Locked names which pallet holds the wheel.
const (
	LockedEntry = "entry"
	LockedExit  = "exit"
	LockedNone  = ""
)

// State is the movement's pose at one phase. The wheel, anchor and pendulum angles are clockwise-positive seen from
// +Z; GreatWheelAngle is counter-clockwise.
type State struct {
	Phase           float64
	AnchorAngle     float64 // ψ
	WheelAngle      float64 // φ
	PendulumAngle   float64 // β
	GreatWheelAngle float64 // φ·13/48, counter-clockwise
	Locked          string
}

// Pendulum returns the pendulum angle β for anchor angle psi.
func Pendulum(psi float64) float64 {
	return math.Atan2(CrutchLength*math.Sin(psi), SuspensionOffset+CrutchLength*math.Cos(psi))
}

// tipOnPallet returns the math angle about O of the point where the tip circle meets the pallet's impulse face from
// corner c to corner f (anchor frame, relative to P) when the anchor is at psi.
func (g *Geometry) tipOnPallet(c, f Pt, psi float64) (float64, error) {
	a := g.P.Add(c.Rot(-psi))
	b := g.P.Add(f.Rot(-psi))
	d := b.Sub(a)
	// |a + s d|^2 = R^2
	qa := d.X*d.X + d.Y*d.Y
	qb := 2 * (a.X*d.X + a.Y*d.Y)
	qc := a.X*a.X + a.Y*a.Y - TipRadius*TipRadius
	disc := qb*qb - 4*qa*qc
	if disc < 0 {
		return 0, fmt.Errorf("escapement: the impulse face misses the tip circle at anchor angle %v", psi)
	}
	root := math.Sqrt(disc)
	for _, s := range []float64{(-qb - root) / (2 * qa), (-qb + root) / (2 * qa)} {
		if s >= -1e-9 && s <= 1+1e-9 {
			return a.Add(Pt{d.X * s, d.Y * s}).Angle(), nil
		}
	}
	return 0, fmt.Errorf("escapement: the impulse face does not reach the tip circle at anchor angle %v", psi)
}

// wheelAt returns the wheel angle φ and the lock state for anchor angle psi on the swing that increases psi
// (increasing) or decreases it.
func (g *Geometry) wheelAt(psi float64, increasing bool) (float64, string, error) {
	const eps = 1e-12
	if increasing {
		switch {
		case psi < g.PsiEu:
			return g.WheelPhaseE, LockedEntry, nil
		case psi > g.PsiEf:
			return g.WheelPhaseX, LockedExit, nil
		}
		q, err := g.tipOnPallet(g.CE, g.FE, psi)
		if err != nil {
			return 0, "", err
		}
		if math.Abs(psi-g.PsiEu) < eps {
			return g.WheelPhaseE, LockedEntry, nil
		}
		return g.WheelPhaseE + g.IL.Angle() - q, LockedNone, nil
	}
	switch {
	case psi > g.PsiXu:
		return g.WheelPhaseX, LockedExit, nil
	case psi < g.PsiXf:
		return g.WheelPhaseE + g.Pitch, LockedEntry, nil
	}
	q, err := g.tipOnPallet(g.CX, g.FX, psi)
	if err != nil {
		return 0, "", err
	}
	if math.Abs(psi-g.PsiXu) < eps {
		return g.WheelPhaseX, LockedExit, nil
	}
	return g.WheelPhaseX + g.IR.Angle() - q, LockedNone, nil
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
	psi := -g.Params.Amplitude * math.Cos(2*math.Pi*t)
	phi, locked, err := g.wheelAt(psi, t <= 0.5)
	if err != nil {
		return State{}, err
	}
	return State{
		Phase:           t,
		AnchorAngle:     psi,
		WheelAngle:      phi,
		PendulumAngle:   Pendulum(psi),
		GreatWheelAngle: phi * PinionTeeth / GreatTeeth,
		Locked:          locked,
	}, nil
}

// WheelAt returns the wheel angle and lock state at anchor angle psi on the increasing or decreasing swing.
func (g *Geometry) WheelAt(psi float64, increasing bool) (float64, string, error) {
	return g.wheelAt(psi, increasing)
}
