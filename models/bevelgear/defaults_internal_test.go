package bevelgear

// spiralDefaults is DefaultParams with the verified 35° spiral teeth, the pair the spiral tests build.
func spiralDefaults() Params {
	p := DefaultParams()
	p.SpiralAngle = verifiedSpiralAngle
	return p
}
