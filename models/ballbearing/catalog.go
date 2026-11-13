package ballbearing

import "slices"

// SeatClearance is the radial clearance between a bearing's outer race and a bore that holds it fixed.
const SeatClearance = 0.05

// PressFitClearance is the radial clearance between a shaft and a printed bore it is pressed into.
const PressFitClearance = 0.05

// Spec is one purchased bearing of the catalog. Lengths are millimeters.
type Spec struct {
	Name              string // catalog name, e.g. "605ZZ"
	BoreDiameter      float64
	OuterDiameter     float64
	Width             float64
	InnerRaceDiameter float64 // outside diameter of the inner race's face; a shoulder or collar must be narrower
}

// catalog lists the purchasable bearings. InnerRaceDiameter is a repository assumption, not a catalogue value.
var catalog = []Spec{
	{Name: "605ZZ", BoreDiameter: 5, OuterDiameter: 14, Width: 5, InnerRaceDiameter: 8},
	{Name: "608ZZ", BoreDiameter: 8, OuterDiameter: 22, Width: 7, InnerRaceDiameter: 12},
}

// Catalog returns a copy of the bearing catalog, in catalog order.
func Catalog() []Spec {
	return slices.Clone(catalog)
}

// Lookup returns the catalog bearing whose Name equals name exactly, or the zero Spec and false.
func Lookup(name string) (Spec, bool) {
	for _, s := range catalog {
		if s.Name == name {
			return s, true
		}
	}
	return Spec{}, false
}

// Params returns the bearing's bore, outside diameter and width, with nil Placement and Lift.
func (s Spec) Params() Params {
	return Params{BoreDiameter: s.BoreDiameter, OuterDiameter: s.OuterDiameter, Width: s.Width}
}

// ShaftDiameter is the diameter of a shaft or pin through the bore: the bore itself, with no clearance.
func (s Spec) ShaftDiameter() float64 {
	return s.BoreDiameter
}

// SeatDiameter is the diameter of a bore or pocket that holds the outer race fixed.
func (s Spec) SeatDiameter() float64 {
	return s.OuterDiameter + 2*SeatClearance
}

// PressBoreDiameter is the diameter of a printed bore that a ShaftDiameter shaft presses into.
func (s Spec) PressBoreDiameter() float64 {
	return s.BoreDiameter + 2*PressFitClearance
}
