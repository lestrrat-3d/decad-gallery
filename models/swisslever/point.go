package swisslever

import "math"

// Pt is a point or vector in a plane, in mm.
type Pt struct{ X, Y float64 }

// Angle returns the math angle of p about the origin.
func (p Pt) Angle() float64 { return math.Atan2(p.Y, p.X) }

// Len returns the distance of p from the origin.
func (p Pt) Len() float64 { return math.Hypot(p.X, p.Y) }

// Add returns p + q.
func (p Pt) Add(q Pt) Pt { return Pt{p.X + q.X, p.Y + q.Y} }

// Sub returns p - q.
func (p Pt) Sub(q Pt) Pt { return Pt{p.X - q.X, p.Y - q.Y} }

// Scale returns p scaled by k.
func (p Pt) Scale(k float64) Pt { return Pt{p.X * k, p.Y * k} }

// Dot returns the dot product of p and q.
func (p Pt) Dot(q Pt) float64 { return p.X*q.X + p.Y*q.Y }

// Cross returns the z component of the cross product of p and q.
func (p Pt) Cross(q Pt) float64 { return p.X*q.Y - p.Y*q.X }

// Rot turns p by the math angle a about the origin.
func (p Pt) Rot(a float64) Pt {
	s, c := math.Sincos(a)
	return Pt{c*p.X - s*p.Y, s*p.X + c*p.Y}
}

// Polar returns the point (r at math angle a).
func Polar(r, a float64) Pt {
	s, c := math.Sincos(a)
	return Pt{r * c, r * s}
}

// normAngle returns a turned by whole turns into (-π, π].
func normAngle(a float64) float64 {
	a = math.Mod(a, 2*math.Pi)
	switch {
	case a > math.Pi:
		a -= 2 * math.Pi
	case a <= -math.Pi:
		a += 2 * math.Pi
	}
	return a
}

// lineIntersect returns the point where the line through p along d meets the line through q along e.
func lineIntersect(p, d, q, e Pt) (Pt, bool) {
	den := d.Cross(e)
	if math.Abs(den) < 1e-14 {
		return Pt{}, false
	}
	s := q.Sub(p).Cross(e) / den
	return p.Add(d.Scale(s)), true
}

// circleCircle returns the two intersections of the circle of radius a about the origin with the circle of radius r
// about c, the one on the left of the direction origin to c first.
func circleCircle(a float64, c Pt, r float64) (Pt, Pt, bool) {
	d := c.Len()
	if d == 0 || d > a+r || d < math.Abs(a-r) {
		return Pt{}, Pt{}, false
	}
	ell := (a*a - r*r + d*d) / (2 * d)
	h2 := a*a - ell*ell
	if h2 < 0 {
		return Pt{}, Pt{}, false
	}
	h := math.Sqrt(h2)
	base := c.Scale(ell / d)
	perp := Pt{-c.Y, c.X}.Scale(h / d)
	return base.Add(perp), base.Sub(perp), true
}
