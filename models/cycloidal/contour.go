package cycloidal

import "math"

const (
	// contourPoints is the number of points one pin pitch of the contour is sampled at, ends included.
	contourPoints = 81
	// contourCamSteps is the number of cam angles the envelope sweep samples over one turn of the cam. The sweep
	// samples one pin pitch's share of them, ceil(contourCamSteps / N), because the cam turning by 2π/N only
	// turns the whole picture by one pin pitch.
	contourCamSteps = 360
	// contourCurveSteps is the number of points of the rotor profile the sweep traces at each cam angle.
	contourCurveSteps = 2000
)

// housingContour returns one pin pitch of the casing's inner wall, from -π/N to +π/N about a pin centered on
// +X: the disc's swept envelope pushed out by the clearance. As the cam turns by θ the disc center orbits the
// drive axis at the eccentricity and the disc turns by -θ/L. For each direction the contour takes the largest
// distance from the drive axis at which any traced disc profile crosses that direction, measured on the
// segment between neighboring profile points, so the result is smooth. Its first and last points sit exactly
// on the pitch boundaries and neighboring pitches share an endpoint.
func housingContour(p Params, d dimensions) []point {
	n := float64(p.PinCount)
	half := math.Pi / n
	pitch := 2 * half
	reach := make([]float64, contourPoints)
	hit := make([]bool, contourPoints)
	trace := make([]point, contourCurveSteps+1)
	// The cam at angle θ+2π/N is the cam at θ turned by one pin pitch about the drive axis, as the disc turns by
	// -θ/L = -θ/(N-1) and L+1 = N. So the sweep over a whole turn of the cam is the sweep over one pitch's worth
	// of cam angles, with every profile segment folded into the pitch it lands in.
	camSteps := (contourCamSteps + p.PinCount - 1) / p.PinCount
	for i := range camSteps {
		theta := pitch * float64(i) / float64(camSteps)
		cx, cy := p.Eccentricity*math.Cos(theta), p.Eccentricity*math.Sin(theta)
		phi := -theta / float64(d.lobes)
		for j := range trace {
			trace[j] = discPoint(p, d, 2*math.Pi*float64(j)/contourCurveSteps, cx, cy, phi)
		}
		for j := range contourCurveSteps {
			a, b := trace[j], trace[j+1]
			alpha, beta := math.Atan2(a.y, a.x), math.Atan2(b.y, b.x)
			if beta-alpha > math.Pi {
				beta -= 2 * math.Pi
			} else if alpha-beta > math.Pi {
				beta += 2 * math.Pi
			}
			// The pitch the segment lies in, and its two neighbors for a segment across a pitch boundary.
			home := math.Round((alpha + beta) / 2 / pitch)
			for m := home - 1; m <= home+1; m++ {
				shift := m * pitch
				lo, hi := math.Min(alpha, beta)-shift, math.Max(alpha, beta)-shift
				if hi < -half || lo > half || hi == lo {
					continue
				}
				sn, cs := math.Sincos(-shift)
				ra := point{a.x*cs - a.y*sn, a.x*sn + a.y*cs}
				rb := point{b.x*cs - b.y*sn, b.x*sn + b.y*cs}
				first := max(int(math.Ceil((lo+half)/pitch*(contourPoints-1))), 0)
				last := min(int(math.Floor((hi+half)/pitch*(contourPoints-1))), contourPoints-1)
				for k := first; k <= last; k++ {
					target := -half + pitch*float64(k)/(contourPoints-1)
					// The point of segment ab on the ray at angle target.
					s, c := math.Sincos(target)
					den := (rb.x-ra.x)*s - (rb.y-ra.y)*c
					if den == 0 {
						continue
					}
					u := (ra.y*c - ra.x*s) / den
					if u < 0 || u > 1 {
						continue
					}
					r := (ra.x+u*(rb.x-ra.x))*c + (ra.y+u*(rb.y-ra.y))*s
					if !hit[k] || r > reach[k] {
						reach[k], hit[k] = r, true
					}
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
