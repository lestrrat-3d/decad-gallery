package cycloidal

import (
	"math"
)

// outlinePerimeter returns the length of the rotor outline of d, summed over a fine sampling of its lobes.
func outlinePerimeter(p Params, d dimensions) float64 {
	const steps = 200000
	span := 2 * math.Pi / float64(d.lobes)
	var length float64
	prev := discPoint(p, d, 0, 0, 0, 0)
	for i := 1; i <= steps; i++ {
		cur := discPoint(p, d, span*float64(i)/steps, 0, 0, 0)
		length += math.Hypot(cur.x-prev.x, cur.y-prev.y)
		prev = cur
	}
	return length * float64(d.lobes)
}
