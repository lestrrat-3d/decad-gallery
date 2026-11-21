package turbinefan

import (
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
)

// Rotor size limits and the scale constants, in mm.
const (
	// BaseRotorDiameter is the size the blade station table is written for.
	BaseRotorDiameter = 128.0
	// DefaultRotorDiameter is the rotor size of every preset.
	DefaultRotorDiameter = 192.0
	// MinRotorDiameter and MaxRotorDiameter bound Params.RotorDiameter.
	MinRotorDiameter = 128.0
	MaxRotorDiameter = 224.0
	// BaseHubRadius is the hub radius at BaseRotorDiameter.
	BaseHubRadius = 14.0
	// MaxHubRadius keeps the hub rim HeadPocketRadius + 0.5 inside the 605ZZ clutch's ear circle (21.6), so the ear
	// bolts' heads and a 1.5 mm hex key pass the hub to the adapter's ear pockets. One hub serves both mounts.
	MaxHubRadius = 18.9
	// BladeBoltTipFloor is the least height of a blade bolt's tip above the back face: the nut slot keeps 2.4 of hub
	// under it.
	BladeBoltTipFloor = 2.0
)

// Size holds every dimension that follows the rotor diameter, in mm, in the local frame. Build it with SizeFor.
//
// The blade scale is s = RotorDiameter / BaseRotorDiameter. Every airfoil section, the hub length, the pitch reference
// radius and the hold radius scale by s. The hub radius is 14 s, capped at MaxHubRadius, and the round blade root, the
// stud, the socket and the blade bolt circle scale by HubScale j = HubRadius / BaseHubRadius.
type Size struct {
	RotorDiameter float64
	Scale         float64 // s: RotorDiameter / BaseRotorDiameter
	HubScale      float64 // j: HubRadius / BaseHubRadius
	TipRadius     float64 // RotorDiameter / 2
	RootRadius    float64 // HubRadius + RegularFit: the blade's root station
	RefRadius     float64 // 48 s: BladePitch is the blade angle here, 75 % of the tip radius
	HoldRadius    float64 // 21.5 s: inboard of this the blade angle stays constant
	HubRadius     float64 // min(14 s, MaxHubRadius)
	HubLength     float64 // 20 s
	HubMid        float64 // HubLength / 2: the sockets' and blade bolts' plane

	// BladeStudRadius is the circumradius of the blade's hexagonal root stud, which runs from radius BladeStudStart to
	// BladeStudEnd. Two of its corners lie on the blade's local X axis, and its flats face local Y.
	BladeStudRadius float64 // 3 j
	BladeStudStart  float64 // BladeSocketStart + RegularFit
	BladeStudEnd    float64 // 16 j
	// BladeSocketRadius is the circumradius of the hub's hexagonal socket, whose flats stand a Tight fit from the stud's.
	// The socket's floor is at radius BladeSocketStart.
	BladeSocketRadius float64 // BladeStudRadius + TightFit / cos 30 degrees
	BladeSocketStart  float64 // 5.7 j
	// BladeBoltAt is the blade bolt's distance from the fan axis. BladeBoltLength is its M2 length and BladeBoltCavity
	// the distance from its head's top face to the near face of the nut cavity.
	BladeBoltAt     float64 // 9.5 j
	BladeBoltLength float64 // the longest standard length that keeps its tip BladeBoltTipFloor above the back face
	BladeBoltCavity float64 // BladeBoltLength - 0.4
}

// SizeFor derives the sizes of a rotor of the given diameter, which must be finite and lie in
// [MinRotorDiameter, MaxRotorDiameter].
func SizeFor(rotorDiameter float64) (Size, error) {
	if math.IsNaN(rotorDiameter) || math.IsInf(rotorDiameter, 0) ||
		rotorDiameter < MinRotorDiameter || rotorDiameter > MaxRotorDiameter {
		return Size{}, fmt.Errorf("turbinefan: RotorDiameter must lie in [%g, %g] mm", MinRotorDiameter, MaxRotorDiameter)
	}
	s := rotorDiameter / BaseRotorDiameter
	sz := Size{RotorDiameter: rotorDiameter, Scale: s}
	sz.HubRadius = math.Min(BaseHubRadius*s, MaxHubRadius)
	sz.HubScale = sz.HubRadius / BaseHubRadius
	j := sz.HubScale
	sz.TipRadius = rotorDiameter / 2
	sz.RootRadius = sz.HubRadius + RegularFit
	sz.RefRadius = 48 * s
	sz.HoldRadius = 21.5 * s
	sz.HubLength = 20 * s
	sz.HubMid = sz.HubLength / 2
	sz.BladeStudRadius = 3 * j
	sz.BladeSocketStart = 5.7 * j
	sz.BladeStudStart = sz.BladeSocketStart + RegularFit
	sz.BladeStudEnd = 16 * j
	sz.BladeSocketRadius = sz.BladeStudRadius + TightFit/math.Cos(math.Pi/6)
	sz.BladeBoltAt = 9.5 * j
	for _, l := range m2fastener.BoltSize.StandardLengths {
		if sz.HubLength-m2fastener.HeadPocketDepth-l >= BladeBoltTipFloor-1e-9 {
			sz.BladeBoltLength = l
		}
	}
	if sz.BladeBoltLength == 0 {
		return Size{}, errors.New("turbinefan: no standard blade bolt length fits the hub")
	}
	sz.BladeBoltCavity = sz.BladeBoltLength - 0.4
	if sz.HubLength-sz.BladeBoltCavity > sz.HubMid-sz.BladeSocketRadius-1 {
		return Size{}, errors.New("turbinefan: blade nut slot leaves less than 1 mm of hub above the socket")
	}
	return sz, nil
}

// Beta is the blade angle at radius r, measured from the plane of rotation, for a blade angle of pitch at RefRadius.
// The shape follows the angle of a uniform axial inflow plus designAttack, held constant across the root. The pitch
// turns the whole blade, as on a turbine with a pitch bearing.
func (s Size) Beta(r, pitch float64) float64 {
	return designAttack + math.Atan(math.Tan(refPitch-designAttack)*s.RefRadius/math.Max(r, s.HoldRadius)) +
		(pitch - refPitch)
}

// valid reports whether s is what SizeFor returns for its RotorDiameter.
func (s Size) valid() bool {
	want, err := SizeFor(s.RotorDiameter)
	return err == nil && s == want
}
