package ballbearing

// Params605ZZ returns the nominal 605ZZ dimensions in millimeters: bore 5,
// outside diameter 14, and width 5. Placement is nil.
func Params605ZZ() Params {
	return Params{
		BoreDiameter:  5,
		OuterDiameter: 14,
		Width:         5,
		Placement:     nil,
	}
}
