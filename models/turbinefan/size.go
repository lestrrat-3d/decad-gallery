package turbinefan

import (
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
)

const (
	// BaseRotorDiameter is the reference rotor diameter for the blade station table, in mm.
	BaseRotorDiameter = 128.0
	// DefaultRotorDiameter is the rotor diameter used by the presets, in mm.
	DefaultRotorDiameter = 192.0
	// MinRotorDiameter is the inclusive minimum supported rotor diameter, in mm.
	MinRotorDiameter = 128.0
	// MaxRotorDiameter is the inclusive maximum supported rotor diameter, in mm.
	MaxRotorDiameter = 224.0
	// BaseHubRadius is the hub radius at BaseRotorDiameter, in mm.
	BaseHubRadius = 14.0
	// MaxHubRadius leaves room for 605ZZ clutch ear-bolt heads and a 1.5 mm hex key at the 21.6 mm ear circle.
	MaxHubRadius = 18.9
	// BladeBoltTipFloor is the minimum blade-bolt tip height above the hub's back face, in mm.
	BladeBoltTipFloor = 2.0
)

// Size holds derived dimensions in mm in the fan's local frame. Scale and HubScale are dimensionless.
// Obtain a Size with SizeFor, which requires a finite rotor diameter within the inclusive supported limits.
type Size struct {
	// RotorDiameter is the selected rotor diameter.
	RotorDiameter float64
	// Scale is the rotor diameter divided by BaseRotorDiameter; airfoil sections, hub length, RefRadius and HoldRadius scale by it.
	Scale float64
	// HubScale is HubRadius divided by BaseHubRadius; the hub, stud, socket and blade-bolt circle dimensions scale by it.
	HubScale float64
	// TipRadius is RotorDiameter / 2, the outer radius of the blade.
	TipRadius float64
	// RootRadius is HubRadius + RegularFit, the first airfoil section radius.
	RootRadius float64
	// RefRadius is 48 * Scale, the rated-pitch station.
	RefRadius float64
	// HoldRadius is 21.5 * Scale, the inboard limit where blade twist is held constant.
	HoldRadius float64
	// HubRadius is min(BaseHubRadius * Scale, MaxHubRadius).
	HubRadius float64
	// HubLength is 20 * Scale, the physical hub height from back to front face.
	HubLength float64
	// HubMid is HubLength / 2, the geometric midpoint of the hub.
	HubMid float64

	// BladeStudRadius is 3 * HubScale, the radius of the circular stud.
	BladeStudRadius float64
	// BladeStudStart is BladeSocketStart + RegularFit + 0.005, the radial start of the stud.
	BladeStudStart float64
	// BladeStudEnd is 16 * HubScale, a legacy outer station; the full circular stud ends at 14.6 * HubScale.
	BladeStudEnd float64
	// BladeSocketRadius is BladeStudRadius + RegularFit + 0.005, the radius of the socket.
	BladeSocketRadius float64
	// BladeSocketStart is 5.7 * HubScale, the radial socket floor.
	BladeSocketStart float64
	// BladeBoltAt is 9.5 * HubScale, the blade bolt's radial position from the fan axis.
	BladeBoltAt float64
	// BladeBoltLength is the longest standard M2 bolt length that preserves the minimum tip height above the hub back face.
	BladeBoltLength float64
	// BladeBoltCavity measures from the bolt head's top face to the nut slot's near face.
	BladeBoltCavity float64
}

// SizeFor derives the dimensions of a rotor with the given diameter in mm. The diameter must be finite and lie
// within [MinRotorDiameter, MaxRotorDiameter].
func SizeFor(rotorDiameter float64) (Size, error) {
	if math.IsNaN(rotorDiameter) || math.IsInf(rotorDiameter, 0) ||
		rotorDiameter < MinRotorDiameter || rotorDiameter > MaxRotorDiameter {
		return Size{}, fmt.Errorf("turbinefan: RotorDiameter must lie in [%g, %g] mm", MinRotorDiameter, MaxRotorDiameter)
	}

	scale := rotorDiameter / BaseRotorDiameter
	size := Size{RotorDiameter: rotorDiameter, Scale: scale}
	size.HubRadius = math.Min(BaseHubRadius*scale, MaxHubRadius)
	size.HubScale = size.HubRadius / BaseHubRadius
	size.TipRadius = rotorDiameter / 2
	size.RootRadius = size.HubRadius + RegularFit
	size.RefRadius = 48 * scale
	size.HoldRadius = 21.5 * scale
	size.HubLength = 20 * scale
	size.HubMid = size.HubLength / 2

	j := size.HubScale
	size.BladeStudRadius = 3 * j
	size.BladeSocketStart = 5.7 * j
	size.BladeStudStart = size.BladeSocketStart + RegularFit + 0.005
	size.BladeStudEnd = 16 * j
	size.BladeSocketRadius = size.BladeStudRadius + RegularFit + 0.005
	size.BladeBoltAt = 9.5 * j
	for _, length := range m2bolt.StandardLengths() {
		if size.HubLength-m2fastener.HeadPocketDepth-length >= BladeBoltTipFloor-1e-9 {
			size.BladeBoltLength = length
		}
	}
	if size.BladeBoltLength == 0 {
		return Size{}, errors.New("turbinefan: no standard blade bolt length fits the hub")
	}
	size.BladeBoltCavity = size.BladeBoltLength - 0.4
	if size.HubLength-size.BladeBoltCavity > size.BladeSocketZ()-size.BladeSocketRadius-1 {
		return Size{}, errors.New("turbinefan: blade nut slot leaves less than 1 mm of hub above the socket")
	}
	return size, nil
}

// BladeSocketZ returns the assembled socket and stud axis height in mm from the hub's back face.
// It is HubMid + 2.5 mm and is independent of blade pitch, hand and mount.
func (size Size) BladeSocketZ() float64 { return size.HubMid + 2.5 }

func (size Size) valid() bool {
	want, err := SizeFor(size.RotorDiameter)
	return err == nil && size == want
}

// Beta returns the blade angle in radians from the plane of rotation at radius, in mm.
// Pitch is in radians at RefRadius. Below HoldRadius, the angle remains at the HoldRadius value;
// pitch adjusts the sections by pitch - refPitch.
func (size Size) Beta(radius, pitch float64) float64 {
	return designAttack + math.Atan(math.Tan(refPitch-designAttack)*size.RefRadius/math.Max(radius, size.HoldRadius)) +
		(pitch - refPitch)
}
