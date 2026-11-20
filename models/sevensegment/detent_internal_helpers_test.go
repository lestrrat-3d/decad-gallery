package sevensegment

import (
	"math"

	"github.com/lestrrat-3d/r3"
)

// sectionBound is the allowance for the meshes of a section, in mm.
const sectionBound = 0.03

// densify adds points along every segment of the loops, at most step apart.
func densify(loops [][]r3.Vec, step float64) []r3.Vec {
	var out []r3.Vec
	for _, l := range loops {
		for i, a := range l {
			b := l[(i+1)%len(l)]
			n := int(math.Ceil(b.Sub(a).Len() / step))
			for k := 0; k < max(n, 1); k++ {
				out = append(out, a.Add(b.Sub(a).Scale(float64(k)/float64(max(n, 1)))))
			}
		}
	}
	return out
}

func segDist(p, a, b r3.Vec) float64 {
	e := b.Sub(a)
	l := e.Dot(e)
	f := 0.0
	if l > 0 {
		f = math.Max(0, math.Min(1, p.Sub(a).Dot(e)/l))
	}
	return p.Sub(a.Add(e.Scale(f))).Len()
}

func nearest(p r3.Vec, loops [][]r3.Vec) float64 {
	best := math.Inf(1)
	for _, l := range loops {
		for i, a := range l {
			best = math.Min(best, segDist(p, a, l[(i+1)%len(l)]))
		}
	}
	return best
}

func turnLoops(loops [][]r3.Vec, deg float64) [][]r3.Vec {
	out := make([][]r3.Vec, len(loops))
	for i, l := range loops {
		for _, p := range l {
			out[i] = append(out[i], rotate(p, deg))
		}
	}
	return out
}

// overlaps reports whether a boundary point of either section lies in the other's material, by more than inset.
func overlaps(a, b [][]r3.Vec) bool {
	for _, p := range densify(a, 0.05) {
		if insideLoops(b, p) && nearest(p, b) > 0.01 {
			return true
		}
	}
	for _, p := range densify(b, 0.05) {
		if insideLoops(a, p) && nearest(p, a) > 0.01 {
			return true
		}
	}
	return false
}
