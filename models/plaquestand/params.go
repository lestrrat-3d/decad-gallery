package plaquestand

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
)

// Default plaque dimensions in mm: the measured size of the user's plaque. Changing them changes the geometry, so
// regenerate every plaque stand image and the gallery image with them.
const (
	// DefaultPlaqueWidth is the plaque width, along X.
	DefaultPlaqueWidth = 250.0
	// DefaultPlaqueHeight is the plaque height, along the slot axis.
	DefaultPlaqueHeight = 305.0
	// DefaultPlaqueThickness is the plaque thickness.
	DefaultPlaqueThickness = 12.8
)

// Other defaults, in mm and degrees.
const (
	// DefaultRecline is the slot's recline from vertical, in degrees.
	DefaultRecline = 15.0
	// DefaultLipHeight is the lip top's height above the seat, along the slot axis.
	DefaultLipHeight = 10.0
	// DefaultSlotClearance is the slot's clearance per side: LooseFit + FitMargin.
	DefaultSlotClearance = LooseFit + FitMargin
)

// Limits on the parameters, in mm and degrees.
const (
	MinPlaqueWidth, MaxPlaqueWidth         = 150.0, 400.0
	MinPlaqueHeight, MaxPlaqueHeight       = 120.0, 350.0
	MinPlaqueThickness, MaxPlaqueThickness = 2.0, 15.0
	MinRecline, MaxRecline                 = 10.0, 25.0
	MinLipHeight, MaxLipHeight             = 6.0, 20.0
	MinSlotClearance, MaxSlotClearance     = LooseFit + FitMargin, 1.5
)

// Fit constants in mm (docs/model-pipeline.md "Part clearances").
const (
	// LooseFit is the Loose class minimum surface-to-surface clearance.
	LooseFit = 0.5
	// FitMargin is the measurement margin added to the class minimum.
	FitMargin = 0.05
)

// Fixed dimensions in mm and degrees.
const (
	BaseThickness = 8.0 // T_b: base slab thickness
	LipThickness  = 5.0 // L_t: lip thickness normal to the slot axis
	BackThickness = 6.0 // B_t: back rest thickness normal to the slot axis
	RootRadius    = 4.0 // r_r: round at the lip root and back-rest root on the base top
	CornerRadius  = 2.0 // r_c: round at the base's four outer corners
	FrontFlat     = 5.0 // F: least flat run of the base top in front of the lip root
	BackFlat      = 5.0 // B: least flat run of the base top behind the back-rest root

	StandWidthRatio = 0.6 // stand width as a share of plaque width
	MinStandWidth   = 120.0
	MaxStandWidth   = 200.0
	BackHeightRatio = 0.2 // back rest height as a share of plaque height
	MinBackHeight   = 30.0
	MaxBackHeight   = 60.0

	// MinTipAngle is φ, the least tilt in degrees before the assembly tips.
	MinTipAngle = 15.0
	// ChamferSize is e, the chamfer of the side faces' outlines (edgefinish.MinChamfer).
	ChamferSize = 0.5

	// BedSize is the assumed build volume edge, and BedMargin the space kept free on each side of it.
	BedSize   = 256.0
	BedMargin = 5.0
)

// maxExtent is the largest extent the stand may have on any axis.
const maxExtent = BedSize - 2*BedMargin

// Params specifies a plaque stand.
type Params struct {
	// PlaqueWidth is W, the plaque's width along X.
	PlaqueWidth float64
	// PlaqueHeight is H, the plaque's height along the slot axis.
	PlaqueHeight float64
	// PlaqueThickness is t.
	PlaqueThickness float64
	// Recline is θ in degrees from vertical.
	Recline float64
	// LipHeight is the lip top's height above the seat along the slot axis.
	LipHeight float64
	// SlotClearance is c, the clearance per side of the plaque in the slot.
	SlotClearance float64
}

// DefaultParams returns the six defaults.
func DefaultParams() Params {
	return Params{
		PlaqueWidth:     DefaultPlaqueWidth,
		PlaqueHeight:    DefaultPlaqueHeight,
		PlaqueThickness: DefaultPlaqueThickness,
		Recline:         DefaultRecline,
		LipHeight:       DefaultLipHeight,
		SlotClearance:   DefaultSlotClearance,
	}
}

// Dimensions are the values Derive computes from Params, in mm and degrees. Every point is (y, z) on the plane x = 0,
// stored as r3.Vec{Y, Z}. Symbols: W, H, t, c, θ from Params; T_b, L_t, B_t, r_r, r_c, F, B, φ, e from the constants.
type Dimensions struct {
	// StandWidth is W_s: clamp(StandWidthRatio·W, MinStandWidth, MaxStandWidth).
	StandWidth float64
	// BackHeight is H_b: clamp(BackHeightRatio·H, MinBackHeight, MaxBackHeight).
	BackHeight float64
	// SlotWidth is s: t + 2c.
	SlotWidth float64
	// SlotRadius is R: s/2.
	SlotRadius float64
	// BackWall is h_b: H_b − B_t/2, the straight back wall's length.
	BackWall float64
	// LipWall is h_l: LipHeight − L_t/2, the straight front wall's length.
	LipWall float64
	// Axis is u: (0, sin θ, cos θ), along the slot.
	Axis r3.Vec
	// Normal is n: (0, cos θ, −sin θ), pointing back and down.
	Normal r3.Vec
	// Seat is C = (C_y, C_z), with C_y = max(C_y,geom, C_y,stab) and C_z = T_b + R: the slot round's centre, through which the plaque's bottom face centre line passes.
	Seat r3.Vec
	// Depth is D: ceil(max(D_geom, D_stab)), the stand's extent along Y.
	Depth float64
	// Height is max(C4.z + B_t/2, C6.z + L_t/2), the stand's extent along Z: the higher of the back rest's top and the lip's top.
	Height float64
	// PlayAngle is α in degrees: atan(2c / h_b), the plaque's lean inside the slot clearance.
	PlayAngle float64
	// TipFront = atan((G_f.y - r_c)/G_f.z), TipBack = atan((D - r_c - G_b.y)/G_b.z) and
	// TipSide = atan((W_s/2 - e)/G_f.z) are the tilt angles in degrees at which the assembly tips forward, backward and
	// sideways.
	TipFront, TipBack, TipSide float64
	// Outline is the stand's side profile.
	Outline Outline
}

// Validate returns the error Derive returns.
func (p Params) Validate() error {
	_, err := p.Derive()
	return err
}

func outside(v, lo, hi float64) bool { return !(v >= lo && v <= hi) }

func clamp(v, lo, hi float64) float64 { return math.Min(math.Max(v, lo), hi) }

func rad(deg float64) float64 { return deg * math.Pi / 180 }

func deg(r float64) float64 { return r * 180 / math.Pi }

// Derive checks p and computes its dimensions. It makes no geometry calls.
func (p Params) Derive() (Dimensions, error) {
	switch {
	case outside(p.PlaqueWidth, MinPlaqueWidth, MaxPlaqueWidth):
		return Dimensions{}, fmt.Errorf("plaquestand: PlaqueWidth must be from %g to %g mm, got %g",
			MinPlaqueWidth, MaxPlaqueWidth, p.PlaqueWidth)
	case outside(p.PlaqueHeight, MinPlaqueHeight, MaxPlaqueHeight):
		return Dimensions{}, fmt.Errorf("plaquestand: PlaqueHeight must be from %g to %g mm, got %g",
			MinPlaqueHeight, MaxPlaqueHeight, p.PlaqueHeight)
	case outside(p.PlaqueThickness, MinPlaqueThickness, MaxPlaqueThickness):
		return Dimensions{}, fmt.Errorf("plaquestand: PlaqueThickness must be from %g to %g mm, got %g",
			MinPlaqueThickness, MaxPlaqueThickness, p.PlaqueThickness)
	case outside(p.Recline, MinRecline, MaxRecline):
		return Dimensions{}, fmt.Errorf("plaquestand: Recline must be from %g to %g degrees, got %g",
			MinRecline, MaxRecline, p.Recline)
	case outside(p.LipHeight, MinLipHeight, MaxLipHeight):
		return Dimensions{}, fmt.Errorf("plaquestand: LipHeight must be from %g to %g mm, got %g",
			MinLipHeight, MaxLipHeight, p.LipHeight)
	case outside(p.SlotClearance, MinSlotClearance, MaxSlotClearance):
		return Dimensions{}, fmt.Errorf("plaquestand: SlotClearance must be from %g to %g mm, got %g",
			MinSlotClearance, MaxSlotClearance, p.SlotClearance)
	}

	W, H, t, c := p.PlaqueWidth, p.PlaqueHeight, p.PlaqueThickness, p.SlotClearance
	theta := rad(p.Recline)
	sT, cT := math.Sin(theta), math.Cos(theta)
	const (
		Tb, Lt, Bt = BaseThickness, LipThickness, BackThickness
		rr, rc     = RootRadius, CornerRadius
		e          = ChamferSize
	)
	tanPhi := math.Tan(rad(MinTipAngle))

	var dm Dimensions
	dm.StandWidth = clamp(StandWidthRatio*W, MinStandWidth, MaxStandWidth)
	dm.BackHeight = clamp(BackHeightRatio*H, MinBackHeight, MaxBackHeight)
	dm.SlotWidth = t + 2*c
	dm.SlotRadius = dm.SlotWidth / 2
	R := dm.SlotRadius
	dm.BackWall = dm.BackHeight - Bt/2
	dm.LipWall = p.LipHeight - Lt/2
	dm.Axis = r3.Vec{Y: sT, Z: cT}
	dm.Normal = r3.Vec{Y: cT, Z: -sT}
	Cz := Tb + R
	alpha := math.Atan(2 * c / dm.BackWall)
	dm.PlayAngle = deg(alpha)
	kf := 1 / math.Tan(rad(45)+theta/2)
	kb := 1 / math.Tan(rad(45)-theta/2)
	l7 := -(R + (R+Lt)*sT) / cT
	l3 := (-R + (R+Bt)*sT) / cT
	thetaF, thetaB := theta-alpha, theta+alpha

	cyGeom := rc + FrontFlat + (R+Lt)*cT - l7*sT + rr*kf
	cyStab := rc + (Cz+(H/2)*math.Cos(thetaF))*tanPhi - (H/2)*math.Sin(thetaF)
	Cy := math.Max(cyGeom, cyStab)
	dm.Seat = r3.Vec{Y: Cy, Z: Cz}

	q7y := Cy - (R+Lt)*cT + l7*sT
	q3y := Cy + (R+Bt)*cT + l3*sT
	gfy, gfz := Cy+(H/2)*math.Sin(thetaF), Cz+(H/2)*math.Cos(thetaF)
	gby, gbz := Cy+(H/2)*math.Sin(thetaB), Cz+(H/2)*math.Cos(thetaB)
	dGeom := q3y + rr*kb + BackFlat + rc
	dStab := rc + gby + gbz*tanPhi
	dm.Depth = math.Ceil(math.Max(dGeom, dStab))

	dm.TipFront = deg(math.Atan((gfy - rc) / gfz))
	dm.TipBack = deg(math.Atan((dm.Depth - rc - gby) / gbz))
	dm.TipSide = deg(math.Atan((dm.StandWidth/2 - e) / gfz))

	n, u := dm.Normal, dm.Axis
	c4 := r3.Vec{Y: Cy, Z: Cz}.Add(n.Scale(R + Bt/2)).Add(u.Scale(dm.BackWall))
	c6 := r3.Vec{Y: Cy, Z: Cz}.Sub(n.Scale(R + Lt/2)).Add(u.Scale(dm.LipWall))
	dm.Height = math.Max(c4.Z+Bt/2, c6.Z+Lt/2)

	switch {
	case dm.TipSide < MinTipAngle:
		return Dimensions{}, fmt.Errorf("plaquestand: side tip angle %.2f degrees is under %g", dm.TipSide, MinTipAngle)
	case dm.StandWidth > maxExtent:
		return Dimensions{}, fmt.Errorf("plaquestand: stand width extent %.1f mm exceeds the %g mm print limit",
			dm.StandWidth, maxExtent)
	case dm.Depth > maxExtent:
		return Dimensions{}, fmt.Errorf("plaquestand: stand depth extent %.1f mm exceeds the %g mm print limit",
			dm.Depth, maxExtent)
	case dm.Height > maxExtent:
		return Dimensions{}, fmt.Errorf("plaquestand: stand height extent %.1f mm exceeds the %g mm print limit",
			dm.Height, maxExtent)
	}

	dm.Outline = buildOutline(outlineInput{
		depth: dm.Depth, seat: dm.Seat, radius: R, backWall: dm.BackWall, lipWall: dm.LipWall,
		axis: u, normal: n, q3y: q3y, q7y: q7y, kf: kf, kb: kb,
	})
	return dm, nil
}
