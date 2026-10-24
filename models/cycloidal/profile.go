package cycloidal

import "math"

// point is a position in a sketch plane, in millimeters.
type point struct{ x, y float64 }

// lobeTurnLimits are the direction changes, in radians, at which the adaptive sampler keeps a point, finest
// first. A fitted spline overshoots where a uniformly sampled lobe turns sharply, and the first bound keeps it
// smooth at any valid eccentricity with a few dozen points. Decad certifies a body by work proportional to its
// spline pieces times their points, so a drive with many lobes is sampled coarser when the finest sampling
// exceeds that work ceiling.
var lobeTurnLimits = []float64{5 * math.Pi / 180, 8 * math.Pi / 180, 12 * math.Pi / 180, 20 * math.Pi / 180}

// lobeFineSteps is the resolution of the trace the adaptive sampler thins.
const lobeFineSteps = 2000

// discPoint returns the rotor profile at curve parameter t for a disc centered at (cx, cy) and clocked by phi.
// The profile is the equidistant of a shortened epitrochoid, offset by the effective roller radius.
func discPoint(p Params, d dimensions, t, cx, cy, phi float64) point {
	n := float64(p.PinCount)
	r, e := d.pinCircleRadius, p.Eccentricity
	num := math.Sin((1 - n) * t)
	den := r/(e*n) - math.Cos((1-n)*t)
	psi := math.Atan2(num, den)
	x0 := r*math.Cos(t) - d.effectiveRadius*math.Cos(t+psi) - e*math.Cos(n*t)
	y0 := -r*math.Sin(t) + d.effectiveRadius*math.Sin(t+psi) + e*math.Sin(n*t)
	return point{
		x: cx + x0*math.Cos(phi) - y0*math.Sin(phi),
		y: cy + x0*math.Sin(phi) + y0*math.Cos(phi),
	}
}

// lobeParameters returns the curve parameters of one lobe, valley to valley over [0, 2π/L], thinned so that
// consecutive points turn by about lobeTurnLimit.
func lobeParameters(p Params, d dimensions, turnLimit float64) []float64 {
	lobes := float64(d.lobes)
	span := 2 * math.Pi / lobes
	fine := make([]point, lobeFineSteps+1)
	for i := range fine {
		fine[i] = discPoint(p, d, span*float64(i)/lobeFineSteps, 0, 0, 0)
	}
	direction := func(a, b point) float64 { return math.Atan2(b.y-a.y, b.x-a.x) }
	kept := []float64{0}
	turn := 0.0
	for i := 1; i < lobeFineSteps; i++ {
		delta := direction(fine[i], fine[i+1]) - direction(fine[i-1], fine[i])
		for delta > math.Pi {
			delta -= 2 * math.Pi
		}
		for delta < -math.Pi {
			delta += 2 * math.Pi
		}
		turn += math.Abs(delta)
		if turn >= turnLimit {
			kept = append(kept, span*float64(i)/lobeFineSteps)
			turn = 0
		}
	}
	return append(kept, span)
}

// profileCurvature returns the signed curvature numerator of the rotor profile at t; its sign is the sign of
// the curvature.
func profileCurvature(p Params, d dimensions, t float64) float64 {
	h := 1e-6
	a := discPoint(p, d, t-h, 0, 0, 0)
	b := discPoint(p, d, t, 0, 0, 0)
	c := discPoint(p, d, t+h, 0, 0, 0)
	x1, y1 := (c.x-a.x)/(2*h), (c.y-a.y)/(2*h)
	x2, y2 := (c.x-2*b.x+a.x)/(h*h), (c.y-2*b.y+a.y)/(h*h)
	return x1*y2 - y1*x2
}

// inflections returns the curve parameters inside one lobe where the profile changes from concave to convex.
func inflections(p Params, d dimensions) []float64 {
	span := 2 * math.Pi / float64(d.lobes)
	const steps = 4000
	inside := func(t float64) float64 { return math.Min(math.Max(t, span*1e-5), span*(1-1e-5)) }
	var out []float64
	prev := span * 1e-5
	for i := 1; i <= steps; i++ {
		cur := inside(span * float64(i) / steps)
		if (profileCurvature(p, d, prev) > 0) != (profileCurvature(p, d, cur) > 0) {
			lo, hi := prev, cur
			for range 60 {
				mid := (lo + hi) / 2
				if (profileCurvature(p, d, mid) > 0) == (profileCurvature(p, d, lo) > 0) {
					lo = mid
				} else {
					hi = mid
				}
			}
			out = append(out, (lo+hi)/2)
		}
		prev = cur
	}
	return out
}

// lobeSegments splits the sampled parameters of one lobe at its inflections. A spline that crosses an
// inflection has curvature of both signs, which decad cannot certify on one span, so each segment is a
// separate spline, and neighbors share the parameter at the cut.
func lobeSegments(p Params, d dimensions, turnLimit float64) [][]float64 {
	span := 2 * math.Pi / float64(d.lobes)
	cuts := inflections(p, d)
	var segments [][]float64
	var current []float64
	next := 0
	for _, t := range lobeParameters(p, d, turnLimit) {
		for next < len(cuts) && cuts[next] <= t {
			current = append(current, cuts[next])
			segments = append(segments, current)
			current = []float64{cuts[next]}
			next++
		}
		if len(current) > 0 && t-current[len(current)-1] < span*1e-4 {
			continue
		}
		current = append(current, t)
	}
	return append(segments, current)
}
