package plaquestand

import (
	"math"

	"github.com/lestrrat-3d/r3"
)

// Segment is one line or arc of the profile, on the plane x = 0, in traversal order.
type Segment struct {
	// Start and End have X = 0.
	Start, End r3.Vec
	// Center is the arc's centre. It is unused for a line.
	Center r3.Vec
	// Radius is 0 for a line.
	Radius float64
	// Convex is set for an arc along which the outline turns counter-clockwise.
	Convex bool
}

// IsArc reports whether the segment is an arc.
func (s Segment) IsArc() bool { return s.Radius > 0 }

// Sweep returns an arc's angle in radians: 2·asin(min(1, |End−Start| / (2·Radius))). It is 0 for a line. Near π the
// asin form is only accurate to about 1e-7.
func (s Segment) Sweep() float64 {
	if !s.IsArc() {
		return 0
	}
	return 2 * math.Asin(math.Min(1, s.End.Sub(s.Start).Len()/(2*s.Radius)))
}

// Outline is a bracket's closed side profile: 13 lines alternating with 13 tangent arcs, starting with a line.
type Outline []Segment

// Area returns the outline's area in mm²: the shoelace sum over every segment's chord, plus (r²/2)(φ − sin φ) for each
// convex arc and minus it for each concave arc, with φ the arc's sweep. Every sweep is at most π.
func (o Outline) Area() float64 {
	sum := 0.0
	for _, s := range o {
		sum += (s.Start.Y*s.End.Z - s.End.Y*s.Start.Z) / 2
		if !s.IsArc() {
			continue
		}
		phi := s.Sweep()
		segment := s.Radius * s.Radius / 2 * (phi - math.Sin(phi))
		if s.Convex {
			sum += segment
			continue
		}
		sum -= segment
	}
	return sum
}

// Length returns the outline's length in mm: each line's length plus each arc's Radius·Sweep.
func (o Outline) Length() float64 {
	sum := 0.0
	for _, s := range o {
		if s.IsArc() {
			sum += s.Radius * s.Sweep()
			continue
		}
		sum += s.End.Sub(s.Start).Len()
	}
	return sum
}

// outlineInput carries what buildOutline needs from Derive.
type outlineInput struct {
	depth, radius, backWall, lipWall float64
	seat, axis, normal               r3.Vec
	// channelBack, channelFront and channelFloor are the rail channel's back wall y, front wall y and floor z.
	channelBack, channelFront, channelFloor float64
	// q3y and q7y are where the back rest's back line and the lip's front line meet the base top (z = BaseThickness).
	q3y, q7y float64
	// kf and kb are the tangent lengths per unit root radius at the lip root and the back-rest root.
	kf, kb float64
}

// buildOutline returns the 26 segments of a bracket's profile, in order, as instructions.md lists them.
func buildOutline(in outlineInput) Outline {
	const (
		tb, lt, bt = BaseThickness, LipThickness, BackThickness
		rr, rc     = RootRadius, CornerRadius
	)
	D, R := in.depth, in.radius
	const rm, rf = ChannelMouthRound, RailRound + JointFit
	ycb, ycf, zf := in.channelBack, in.channelFront, in.channelFloor
	// at returns the point P(a, b) = C + a·n + b·u.
	at := func(a, b float64) r3.Vec { return in.seat.Add(in.normal.Scale(a)).Add(in.axis.Scale(b)) }
	v := func(y, z float64) r3.Vec { return r3.Vec{Y: y, Z: z} }
	line := func(a, b r3.Vec) Segment { return Segment{Start: a, End: b} }
	arc := func(a, b, c r3.Vec, r float64, convex bool) Segment {
		return Segment{Start: a, End: b, Center: c, Radius: r, Convex: convex}
	}

	backFoot := v(in.q3y+rr*in.kb, tb)
	backTan := v(in.q3y, tb).Add(in.axis.Scale(rr * in.kb))
	lipTan := v(in.q7y, tb).Add(in.axis.Scale(rr * in.kf))
	lipFoot := v(in.q7y-rr*in.kf, tb)

	return Outline{
		line(v(rc, 0), v(D-rc, 0)),
		arc(v(D-rc, 0), v(D, rc), v(D-rc, rc), rc, true),
		line(v(D, rc), v(D, tb-rc)),
		arc(v(D, tb-rc), v(D-rc, tb), v(D-rc, tb-rc), rc, true),
		line(v(D-rc, tb), v(ycb+rm, tb)),
		arc(v(ycb+rm, tb), v(ycb, tb-rm), v(ycb+rm, tb-rm), rm, true),
		line(v(ycb, tb-rm), v(ycb, zf+rf)),
		arc(v(ycb, zf+rf), v(ycb-rf, zf), v(ycb-rf, zf+rf), rf, false),
		line(v(ycb-rf, zf), v(ycf+rf, zf)),
		arc(v(ycf+rf, zf), v(ycf, zf+rf), v(ycf+rf, zf+rf), rf, false),
		line(v(ycf, zf+rf), v(ycf, tb-rm)),
		arc(v(ycf, tb-rm), v(ycf-rm, tb), v(ycf-rm, tb-rm), rm, true),
		line(v(ycf-rm, tb), backFoot),
		arc(backFoot, backTan, v(in.q3y+rr*in.kb, tb+rr), rr, false),
		line(backTan, at(R+bt, in.backWall)),
		arc(at(R+bt, in.backWall), at(R, in.backWall), at(R+bt/2, in.backWall), bt/2, true),
		line(at(R, in.backWall), at(R, 0)),
		arc(at(R, 0), at(-R, 0), in.seat, R, false),
		line(at(-R, 0), at(-R, in.lipWall)),
		arc(at(-R, in.lipWall), at(-R-lt, in.lipWall), at(-R-lt/2, in.lipWall), lt/2, true),
		line(at(-R-lt, in.lipWall), lipTan),
		arc(lipTan, lipFoot, v(in.q7y-rr*in.kf, tb+rr), rr, false),
		line(lipFoot, v(rc, tb)),
		arc(v(rc, tb), v(0, tb-rc), v(rc, tb-rc), rc, true),
		line(v(0, tb-rc), v(0, rc)),
		arc(v(0, rc), v(rc, 0), v(rc, rc), rc, true),
	}
}
