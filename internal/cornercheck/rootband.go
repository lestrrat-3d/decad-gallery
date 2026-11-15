package cornercheck

import "github.com/lestrrat-3d/r3"

// Crossings counts the places where the outline's loops cross the circle of the given radius about the axis through
// centre. A section through n teeth cuts the circle between the root and the tip radii 2n times.
func Crossings(outline SectionOutline, centre, axis r3.Vec, radius float64) int {
	a, ok := axis.Normalize()
	if !ok {
		return 0
	}
	side := func(p r3.Vec) bool {
		d := p.Sub(centre)
		return d.Sub(a.Scale(d.Dot(a))).Len() > radius
	}
	n := 0
	for _, loop := range outline.Loops {
		for i := range loop {
			if side(loop[i]) != side(loop[(i+1)%len(loop)]) {
				n++
			}
		}
	}
	return n
}

// Windows returns the stretches of the outline whose turn over the given window exceeds minTurn degrees: toward the
// material for Inside, away from it for Tip. Section reads the same stretches at the rule's own window and limit. A
// caller reads other windows to measure how a corner is spread, such as a fillet's quarter turn over 0.4 mm against
// a sharp corner's whole turn over 0.05 mm.
func Windows(outline SectionOutline, kind Kind, window, minTurn float64) []Finding {
	var out []Finding
	for _, loop := range outline.Loops {
		out = append(out, scanWindows(loop, outline.Normal, 0, kind, window, minTurn)...)
	}
	return out
}
