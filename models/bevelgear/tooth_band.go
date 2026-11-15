package bevelgear

import (
	"fmt"
	"math"
)

// ToothBand is where a member's teeth lie along its shaft, for a check that cuts sections through them. Stations
// are distances from the pitch apex along the shaft, and a cone's radius at a station is its slope times the
// station, since every cone of the tooth passes through the apex.
type ToothBand struct {
	// ToeRoot and HeelRoot are the stations where the blank's root cone begins and ends. A section between them cuts
	// the teeth.
	ToeRoot, HeelRoot float64
	// HeelPitch is the station where the heel cone meets the pitch cone. A section behind it no longer cuts the pitch
	// circle through the teeth, though the root cone runs on to HeelRoot.
	HeelPitch float64
	// RootSlope, PitchSlope and TipSlope are the root, pitch and tip cones' radius per station.
	RootSlope, PitchSlope, TipSlope float64
	// Teeth is the member's tooth count.
	Teeth int
	// RootFillet is the root fillet's radius at the heel in mm, or zero when the member's roots keep a sharp corner.
	RootFillet float64
}

// ToothBand returns the band of the member that p describes.
func (p Params) ToothBand() (ToothBand, error) {
	d, err := derive(p)
	if err != nil {
		return ToothBand{}, err
	}
	section, err := blankSection(p, d)
	if err != nil {
		return ToothBand{}, fmt.Errorf("blank section: %w", err)
	}
	ded, toeRoot := section[3], section[4]
	band := ToothBand{
		ToeRoot:    toeRoot.station,
		HeelRoot:   ded.station,
		HeelPitch:  d.coneDist * math.Cos(d.coneAngle),
		RootSlope:  ded.radius / ded.station,
		PitchSlope: math.Tan(d.coneAngle),
		TipSlope:   math.Tan(d.coneAngle + math.Atan(p.Module/d.coneDist)),
		Teeth:      p.Teeth,
	}
	straight := p
	straight.rootFillet = p.SpiralAngle == 0
	if dims, ok := rootFilletFor(straight); ok {
		band.RootFillet = dims.rho
	}
	return band, nil
}
