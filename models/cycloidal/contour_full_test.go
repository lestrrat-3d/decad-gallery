package cycloidal

import "math"

// fullCamSteps is the cam angles the reference sweep samples over one whole turn of the cam.
const fullCamSteps = 720

// housingContourFull is the sweep over a whole turn of the cam, with every profile segment tested against the
// one pin pitch only. housingContour must agree with it.
func housingContourFull(p Params, d dimensions) []point {
	return housingContourSweep(p, d, fullCamSteps, contourCurveSteps)
}

// housingContourSweep is housingContourFull at the given sample counts.
func housingContourSweep(p Params, d dimensions, camSteps, curveSteps int) []point {
	n := float64(p.PinCount)
	half := math.Pi / n
	reach := make([]float64, contourPoints)
	hit := make([]bool, contourPoints)
	trace := make([]point, curveSteps+1)
	for i := range camSteps {
		theta := 2 * math.Pi * float64(i) / float64(camSteps)
		cx, cy := p.Eccentricity*math.Cos(theta), p.Eccentricity*math.Sin(theta)
		phi := -theta / float64(d.lobes)
		for j := range trace {
			trace[j] = discPoint(p, d, 2*math.Pi*float64(j)/float64(curveSteps), cx, cy, phi)
		}
		for j := range curveSteps {
			a, b := trace[j], trace[j+1]
			alpha, beta := math.Atan2(a.y, a.x), math.Atan2(b.y, b.x)
			if beta-alpha > math.Pi {
				beta -= 2 * math.Pi
			} else if alpha-beta > math.Pi {
				beta += 2 * math.Pi
			}
			lo, hi := math.Min(alpha, beta), math.Max(alpha, beta)
			if hi < -half || lo > half || hi == lo {
				continue
			}
			first := max(int(math.Ceil((lo+half)/(2*half)*(contourPoints-1))), 0)
			last := min(int(math.Floor((hi+half)/(2*half)*(contourPoints-1))), contourPoints-1)
			for k := first; k <= last; k++ {
				target := -half + 2*half*float64(k)/(contourPoints-1)
				// The point of segment ab on the ray at angle target.
				s, c := math.Sincos(target)
				den := (b.x-a.x)*s - (b.y-a.y)*c
				if den == 0 {
					continue
				}
				u := (a.y*c - a.x*s) / den
				if u < 0 || u > 1 {
					continue
				}
				r := (a.x+u*(b.x-a.x))*c + (a.y+u*(b.y-a.y))*s
				if !hit[k] || r > reach[k] {
					reach[k], hit[k] = r, true
				}
			}
		}
	}
	// The envelope is symmetric about the pin, so each point takes the mean of itself and its mirror image.
	out := make([]point, contourPoints)
	for k := range out {
		angle := -half + 2*half*float64(k)/(contourPoints-1)
		r := p.DiskClearance + (reach[k]+reach[contourPoints-1-k])/2
		out[k] = point{r * math.Cos(angle), r * math.Sin(angle)}
	}
	return out
}
