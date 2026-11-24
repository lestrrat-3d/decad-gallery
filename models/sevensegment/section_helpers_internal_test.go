package sevensegment

import (
	"math"

	"github.com/lestrrat-3d/r3"
)

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
