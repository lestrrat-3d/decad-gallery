package spiralspring

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
)

const (
	// ThicknessFloor is the least strip thickness, in mm.
	ThicknessFloor = 1.5
	// GapFloor is the least MinGap, in mm.
	GapFloor = 1.0
	// ClearanceFloor is the least bore and anchor clearance, in mm.
	ClearanceFloor = 0.2
	// MaxTurns is the most turns the sketch's numerical conditioning allows.
	MaxTurns = 40
)

// Params configures a spiral spring. All lengths are in millimeters.
type Params struct {
	// OuterDiameter is the relaxed coil diameter.
	OuterDiameter float64
	// HubDiameter is the outer diameter of the hub at the inner end.
	HubDiameter float64
	// Thickness is the strip thickness.
	Thickness float64
	// Height is the strip height along the axis.
	Height float64
	// MinGap is the least air gap between turns.
	MinGap float64
	// ArborDiameter is the arbor's diameter.
	ArborDiameter float64
	// ArborFlatDepth is the depth of the arbor's D flat; 0 gives a round bore.
	ArborFlatDepth float64
	// BoreClearance is the radial clearance between bore and arbor.
	BoreClearance float64
	// AnchorPinDiameter is the housing pin's diameter.
	AnchorPinDiameter float64
	// AnchorClearance is the radial clearance between eye hole and pin.
	AnchorClearance float64
	// Turns is the number of turns; 0 selects as many as fit.
	Turns int
	// Placement maps the local frame onto a frame; nil keeps the local frame.
	Placement *r3.Frame
}

// DefaultParams returns the default spring.
func DefaultParams() Params {
	return Params{
		OuterDiameter:     60,
		HubDiameter:       12,
		Thickness:         1.6,
		Height:            8,
		MinGap:            2.0,
		ArborDiameter:     5,
		ArborFlatDepth:    0.5,
		BoreClearance:     0.2,
		AnchorPinDiameter: 3,
		AnchorClearance:   0.35,
	}
}

// Dimensions holds the derived dimensions of a spring.
type Dimensions struct {
	Turns            int     // N
	Pitch            float64 // p
	Gap              float64 // g
	CenterOffset     float64 // d
	HubRadius        float64 // R
	CoilRadius       float64 // R_c
	FirstInnerRadius float64 // rho0
	SpokeWidth       float64 // w
	BoreRadius       float64 // r_b
	BoreFlatDistance float64 // f, 0 when the bore is round
	EyeCenterX       float64 // E.x
	EyeCenterY       float64 // E.y = -L
	EyeRadius        float64 // R_e
	EyeHoleDiameter  float64 // D_hole
	TailLength       float64 // L
	FilletRadius     float64 // r_f
	ActiveLength     float64
	WindStrain       float64
}

// Validate checks the parameters.
func (p Params) Validate() error {
	_, err := derive(p)
	return err
}

// Derive validates p and returns its derived dimensions.
func (p Params) Derive() (Dimensions, error) {
	return derive(p)
}

func errf(format string, args ...any) error {
	return fmt.Errorf("spiralspring: "+format, args...)
}

func derive(p Params) (Dimensions, error) {
	fields := []struct {
		name string
		v    float64
	}{
		{"OuterDiameter", p.OuterDiameter}, {"HubDiameter", p.HubDiameter}, {"Thickness", p.Thickness},
		{"Height", p.Height}, {"MinGap", p.MinGap}, {"ArborDiameter", p.ArborDiameter},
		{"ArborFlatDepth", p.ArborFlatDepth}, {"BoreClearance", p.BoreClearance},
		{"AnchorPinDiameter", p.AnchorPinDiameter}, {"AnchorClearance", p.AnchorClearance},
	}
	for _, f := range fields {
		if math.IsNaN(f.v) || math.IsInf(f.v, 0) {
			return Dimensions{}, errf("%s must be finite", f.name)
		}
	}
	if p.Thickness < ThicknessFloor {
		return Dimensions{}, errf("Thickness must be at least %.1f mm", ThicknessFloor)
	}
	if p.MinGap < GapFloor {
		return Dimensions{}, errf("MinGap must be at least %.1f mm", GapFloor)
	}
	if p.Height < 2*p.Thickness {
		return Dimensions{}, errf("Height must be at least 2*Thickness (%.2f mm)", 2*p.Thickness)
	}
	if p.ArborDiameter <= 0 {
		return Dimensions{}, errf("ArborDiameter must be above 0")
	}
	if p.AnchorPinDiameter <= 0 {
		return Dimensions{}, errf("AnchorPinDiameter must be above 0")
	}
	if p.ArborFlatDepth < 0 || p.ArborFlatDepth > p.ArborDiameter/4 {
		return Dimensions{}, errf("ArborFlatDepth must be between 0 and ArborDiameter/4 (%.3f mm)",
			p.ArborDiameter/4)
	}
	if p.BoreClearance < ClearanceFloor {
		return Dimensions{}, errf("BoreClearance must be at least %.1f mm", ClearanceFloor)
	}
	if p.AnchorClearance < ClearanceFloor {
		return Dimensions{}, errf("AnchorClearance must be at least %.1f mm", ClearanceFloor)
	}

	R := p.HubDiameter / 2
	Rc := p.OuterDiameter / 2
	t := p.Thickness
	rb := p.ArborDiameter/2 + p.BoreClearance
	f := 0.0
	if p.ArborFlatDepth > 0 {
		f = p.ArborDiameter/2 - p.ArborFlatDepth + p.BoreClearance
		// Not in the numbered rules: a flat at or beyond the bore radius cuts nothing.
		if f >= rb {
			return Dimensions{}, errf("ArborFlatDepth must exceed BoreClearance so the bore flat cuts the bore")
		}
	}
	if R-rb < ThicknessFloor {
		return Dimensions{}, errf("hub wall HubDiameter/2 - bore radius must be at least %.1f mm", ThicknessFloor)
	}
	if R < 2.5*t {
		return Dimensions{}, errf("HubDiameter must be at least 5*Thickness (%.2f mm)", 5*t)
	}
	if p.OuterDiameter <= p.HubDiameter {
		return Dimensions{}, errf("OuterDiameter must exceed HubDiameter")
	}
	if p.Turns < 0 {
		return Dimensions{}, errf("Turns must not be negative")
	}
	nMax := int(math.Floor((Rc-R)/(t+p.MinGap)+1e-9)) - 1
	if nMax < 1 {
		return Dimensions{}, errf("OuterDiameter leaves no room for one turn")
	}
	if p.Turns > 0 && p.Turns > nMax {
		return Dimensions{}, errf("Turns %d needs a gap below MinGap; at most %d turns fit", p.Turns, nMax)
	}
	n := p.Turns
	if n == 0 {
		n = nMax
	}
	if n > MaxTurns {
		return Dimensions{}, errf("at most %d turns are supported, got %d", MaxTurns, n)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return Dimensions{}, errf("Placement must be a valid frame")
	}

	pitch := (Rc - R) / float64(n+1)
	g := pitch - t
	d := pitch / 2
	rho0 := R + g + d
	rk := func(k int) float64 { return rho0 + float64(k)*d }
	hole := p.AnchorPinDiameter + 2*p.AnchorClearance
	re := hole/2 + t
	rhoPrev := rk(2*n-2) + t
	lateral := rk(2*n-1) + t/2 + d
	rootTerm := rhoPrev + g + re
	l := math.Max(2*re, math.Sqrt(math.Max(0, rootTerm*rootTerm-lateral*lateral)))
	nf := float64(n)
	return Dimensions{
		Turns:            n,
		Pitch:            pitch,
		Gap:              g,
		CenterOffset:     d,
		HubRadius:        R,
		CoilRadius:       Rc,
		FirstInnerRadius: rho0,
		SpokeWidth:       2 * t,
		BoreRadius:       rb,
		BoreFlatDistance: f,
		EyeCenterX:       -(rk(2*n-1) + t/2),
		EyeCenterY:       -l,
		EyeRadius:        re,
		EyeHoleDiameter:  hole,
		TailLength:       l,
		FilletRadius:     math.Min(t/2, g/4),
		ActiveLength:     math.Pi * (2*nf*(rho0+t/2) + d*nf*(2*nf-1)),
		WindStrain:       (t / 2) * (1/(R+t/2) - 1/(R+g+t/2)),
	}, nil
}
