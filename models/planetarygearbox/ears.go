package planetarygearbox

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"
)

const (
	// earWall is the material around an ear's bore: the ear is as wide as the bore plus this much on each side.
	earWall = 3
	// earGap is the smallest gap kept between two neighbouring ears.
	earGap = 1
)

// earHalfWidth is half the width of an ear: the bore's radius plus the wall.
func earHalfWidth(bore float64) float64 { return bore/2 + earWall }

// earCenterRadius is the distance from the axis to the center of an ear's bore. The bore sits clear of the covers'
// outer wall, which has the ring's outer radius, by its own radius plus the wall.
func earCenterRadius(ringOuter, bore float64) float64 { return ringOuter + earHalfWidth(bore) }

// earAngle is the angle of ear i of count. The ears are spread evenly, the first on +Y, which keeps them off the
// bolt angles for the usual counts.
func earAngle(i, count int) float64 { return math.Pi/2 + 2*math.Pi*float64(i)/float64(count) }

// earRootRound is the radius of the rounds where an ear meets the ring: half the thinner of the ring wall and the
// ear's width. The ears carry the gearbox's reaction torque, so their roots are stressed roots.
func earRootRound(p Params) float64 {
	return math.Min(p.RingWall, 2*earHalfWidth(p.EarBoreDiameter)) / 2
}

// earPoints are the corners of one ear's outline.
type earPoints struct {
	center *sketch.Point
	// root is where the ear's two straight edges meet the ring's outer circle. rootUp is on the counter-clockwise
	// side of the ear's axis and rootDown on the clockwise side.
	rootUp, rootDown *sketch.Point
	// tipUp and tipDown are the ends of the half circle round the bore, on the same two sides.
	tipUp, tipDown *sketch.Point
}

// addRingOutline draws the ring's outer boundary on s: a circle of the ring's outer radius, or with ears a single
// closed loop of circular arcs of that radius between the ears and, for each ear, two straight edges and a half
// circle round its bore. The bore of every ear is a hole in the loop.
//
// An ear is laid out by relations, not by fixed corner coordinates. Its axis is a construction line from the sketch
// origin to the bore's center, at the center radius; the first axis is vertical and each later axis turns by an equal
// step from the one before. The straight edges run parallel to the axis, the half circle has the ear's half width as
// radius and is centered on the bore, and a construction line through the bore's center, square to the axis, holds
// the half circle's ends. The edges end on the ring's outer circle.
func (r *run) addRingOutline(s *sketch.Sketch) error {
	p, d := r.p, r.d
	if p.EarCount == 0 {
		return r.addCircle(s, 2*d.ringOuter, "2 * ring_outer_radius")
	}
	half := earHalfWidth(p.EarBoreDiameter)
	radius := earCenterRadius(d.ringOuter, p.EarBoreDiameter)
	// root is the distance along an ear's axis at which its straight edges reach the outer circle.
	root := math.Sqrt(d.ringOuter*d.ringOuter - half*half)

	ears := make([]earPoints, p.EarCount)
	var axes []*sketch.Line
	ups := make([]*sketch.Line, p.EarCount)
	downs := make([]*sketch.Line, p.EarCount)
	for i := range ears {
		angle := earAngle(i, p.EarCount)
		sin, cos := math.Sincos(angle)
		at := func(along, side float64) *sketch.Point {
			return s.CreatePoint(along*cos-side*sin, along*sin+side*cos)
		}
		e := earPoints{
			center: at(radius, 0), rootUp: at(root, half), rootDown: at(root, -half),
			tipUp: at(radius, half), tipDown: at(radius, -half),
		}
		ears[i] = e

		axis := s.CreateLine(s.Origin(), e.center)
		axis.SetConstruction(true)
		axes = append(axes, axis)
		distance := sketch.NewDistance(s.Origin(), e.center, radius)
		s.AddConstraint(distance)
		if err := s.Bind(distance, r.world.Params(),
			"ring_outer_radius + ear_bore_diameter / 2 + ear_wall"); err != nil {
			return fmt.Errorf("ear %d: bind center radius: %w", i+1, err)
		}
		switch i {
		case 0:
			s.AddConstraint(sketch.NewVertical(axis))
		default:
			s.AddConstraint(sketch.NewAngle(axes[i-1], axis, 360/float64(p.EarCount)))
		}

		bore := s.CreateCircle(e.center, p.EarBoreDiameter/2)
		boreDia := sketch.NewDiameter(bore, p.EarBoreDiameter)
		s.AddConstraint(boreDia)
		if err := s.Bind(boreDia, r.world.Params(), "ear_bore_diameter"); err != nil {
			return fmt.Errorf("ear %d: bind bore diameter: %w", i+1, err)
		}

		across := s.CreateLine(e.tipDown, e.tipUp)
		across.SetConstruction(true)
		up, down := s.CreateLine(e.tipUp, e.rootUp), s.CreateLine(e.tipDown, e.rootDown)
		ups[i], downs[i] = up, down
		end := s.CreateArc(e.center, e.tipDown, e.tipUp)
		endRadius := sketch.NewRadius(end, half)
		s.AddConstraint(
			endRadius,
			sketch.NewPerpendicular(axis, across),
			sketch.NewPointOnLine(e.center, across),
			sketch.NewParallel(axis, up),
			sketch.NewParallel(axis, down),
		)
		if err := s.Bind(endRadius, r.world.Params(), "ear_bore_diameter / 2 + ear_wall"); err != nil {
			return fmt.Errorf("ear %d: bind width: %w", i+1, err)
		}
	}

	// Between two ears the outline follows the outer circle, counter-clockwise from one ear's rootUp to the next
	// ear's rootDown. With one ear it runs the long way round to that ear's own rootDown.
	arcs := make([]*sketch.Arc, len(ears))
	for i, e := range ears {
		next := ears[(i+1)%len(ears)]
		arc := s.CreateArc(s.Origin(), e.rootUp, next.rootDown)
		arcs[i] = arc
		outer := sketch.NewRadius(arc, d.ringOuter)
		s.AddConstraint(outer)
		if err := s.Bind(outer, r.world.Params(), "ring_outer_radius"); err != nil {
			return fmt.Errorf("ear %d: bind outer radius: %w", i+1, err)
		}
	}
	// Each ear meets the ring in two rounds. With one ear, the arc before and after it is the same arc.
	round := earRootRound(p)
	for i, e := range ears {
		prev := arcs[(i+len(ears)-1)%len(ears)]
		if _, err := sketch.CreateFillet(s, ups[i], arcs[i], e.rootUp, round); err != nil {
			return fmt.Errorf("ear %d: round counter-clockwise root: %w", i+1, err)
		}
		if _, err := sketch.CreateFillet(s, prev, downs[i], e.rootDown, round); err != nil {
			return fmt.Errorf("ear %d: round clockwise root: %w", i+1, err)
		}
	}
	return nil
}
