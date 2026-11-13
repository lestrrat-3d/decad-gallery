package ballbearing

// Params605ZZ returns the nominal 605ZZ dimensions in millimeters: bore 5,
// outside diameter 14, and width 5. Placement is nil.
func Params605ZZ() Params {
	return catalogEntry("605ZZ").Params()
}

// Params6082RS returns the nominal 608-2RS dimensions in millimeters: bore 8, outside
// diameter 22, and width 7. They equal DefaultParams. Placement is nil.
func Params6082RS() Params {
	return Params{
		BoreDiameter:  8,
		OuterDiameter: 22,
		Width:         7,
		Placement:     nil,
	}
}

// Params608ZZ returns the nominal 608ZZ dimensions in millimeters: bore 8, outside
// diameter 22, and width 7. They are the dimensions of the default 608-2RS
// envelope, which has no seal or shield. Placement is nil.
func Params608ZZ() Params {
	return catalogEntry("608ZZ").Params()
}

// catalogEntry returns a catalog entry the package itself declares.
func catalogEntry(name string) Spec {
	s, _ := Lookup(name)
	return s
}
