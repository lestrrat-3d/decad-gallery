package bevelgear_test

// rootCone is the default gear's root cone about the shaft: the radius at axial position x, in mm. Both members' roots
// lie on it, from 10.14 at x 11.36 to 14.60 at x 16.36.
func rootCone(x float64) float64 { return 10.14 + 0.892*(x-11.36) }
