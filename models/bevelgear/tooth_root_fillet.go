package bevelgear

import (
	"math"

	"github.com/lestrrat-3d/sketch"
)

// The root fillet of a radial-flank tooth (docs/model-pipeline.md "Corners and roots").
//
// Every surface of a straight bevel tooth is a cone through the pitch apex, so a fillet drawn in the back-cone tooth
// sketch and lofted to the apex scales with the distance from the apex: each section is a similar copy of the heel
// section, and the radius at the toe is the heel radius times toe_scale.
//
// The fillet cannot end tangent to the blank's root cone. The tooth Union would then hold a tangent contact, which
// decad's sector join cannot certify. The fillet arc therefore stops at T3 short of tangency, and a straight run-out
// line tangent to the arc there crosses the root cone at rootRunoutAngle and ends on the tooth's own root arc, which
// lies rootArcDepthRatio of the module inside the blank's root. The material outline then follows the root cone,
// turns by the run-out angle where the run-out crosses it, follows the arc to the flank, and follows the flank.
const (
	// rootFilletRatio is the fillet radius over the module, the rack value the spur gears use.
	rootFilletRatio = 0.38
	// rootRunoutAngle is the angle in radians at which the run-out line crosses the blank's root cone. It is the
	// largest crease the corner check accepts at the toe of the fan pinion (docs/model-pipeline.md).
	rootRunoutAngle = 10 * math.Pi / 180
	// rootArcDepthRatio is the depth of the tooth's own root arc below the blank's root over the module: 0.05 mm at
	// module 2. A shallower arc leaves the sector join no certificate margin (0.02 mm at module 2 fails), and a deeper
	// one (0.2 mm at module 4) lets the neighbouring teeth's run-outs meet.
	rootArcDepthRatio = 0.025
	// rootFilletMinVirtualTeeth is the fewest virtual teeth that draw the fillet. decad certifies the join at the
	// fan pinion's 15.65 and above, and refuses it at 15.0 (module 4, 12 teeth against 16), 11.3 (8 against 8) and
	// 5.7 (4 against 4), where the neighbouring run-outs close the gap.
	rootFilletMinVirtualTeeth = 15.5
)

// rootFilletDims holds the heel section's fillet dimensions about the back-cone origin.
type rootFilletDims struct {
	rho    float64 // fillet radius
	hi     float64 // radius of the run-out's start T3 about the origin: blank root plus the run-out depth
	rc     float64 // distance from the origin to the fillet centre
	root   float64 // radius of the tooth's own root arc
	tangle float64 // distance from the origin to the flank tangent point T1
}

// radialRootFilletDims sizes the fillet of a radial-flank tooth, and reports false when it does not fit: the
// flank's tangent point must lie below the base circle, where the radial line ends.
func radialRootFilletDims(module, blankRoot, base float64) (rootFilletDims, bool) {
	rho := rootFilletRatio * module
	depth := rho * (1 - math.Cos(rootRunoutAngle)) / 2
	rc := blankRoot - depth + rho
	if rc <= rho {
		return rootFilletDims{}, false
	}
	foot := math.Sqrt(rc*rc - rho*rho)
	if foot >= base-0.1*module {
		return rootFilletDims{}, false
	}
	return rootFilletDims{rho: rho, hi: blankRoot + depth, rc: rc, root: blankRoot - rootArcDepthRatio*module, tangle: foot}, true
}

// rootFilletFor sizes the fillet of the tooth that p draws, and reports false when p has none: the member is not a
// straight one asking for it, its flanks start on an involute below the root, or the fillet does not fit.
func rootFilletFor(p Params) (rootFilletDims, bool) {
	if !p.rootFillet {
		return rootFilletDims{}, false
	}
	d, err := derive(p)
	if err != nil || d.virtualBase < d.virtualRoot || d.virtualTeeth < rootFilletMinVirtualTeeth {
		return rootFilletDims{}, false
	}
	return radialRootFilletDims(p.Module, d.virtualPitch-d.dedendum, d.virtualBase)
}

// drawRadialRootFillet draws one side of the root. flank is the first flank sample, on the base circle, and
// (fx, fy) are its coordinates. It returns the end D of the run-out on the tooth's root arc.
//
// Constraints: T1 lies on the flank's radial line through the origin; the fillet arc is tangent to that line at T1
// with radius rho and its centre rc from the origin; the run-out is tangent to the arc at T3, which lies hi from the
// origin. Dimensioning the centre's distance, not T3's collinearity, keeps the set free of redundancy.
func drawRadialRootFillet(s *sketch.Sketch, flank *sketch.Point, fx, fy float64, dims rootFilletDims) *sketch.Point {
	center := s.Origin()
	theta := math.Atan2(fy, fx)
	// The tooth is centred at pi. The fillet lies on the side of the flank away from the tooth's centre.
	sign := 1.0
	if math.Remainder(theta-math.Pi, 2*math.Pi) < 0 {
		sign = -1
	}
	rho, rc := dims.rho, dims.rc
	ca := theta + sign*math.Asin(rho/rc)
	cx, cy := rc*math.Cos(ca), rc*math.Sin(ca)
	k := (dims.hi*dims.hi - rc*rc - rho*rho) / (2 * rho * rc)
	t := ca + math.Pi + sign*math.Acos(-k)
	x3, y3 := cx+rho*math.Cos(t), cy+rho*math.Sin(t)
	// The run-out leaves T3 toward the gap's centre and inward, so its direction points down in radius.
	tx, ty := -math.Sin(t), math.Cos(t)
	if tx*x3+ty*y3 > 0 {
		tx, ty = -tx, -ty
	}
	bq := x3*tx + y3*ty
	cq := x3*x3 + y3*y3 - dims.root*dims.root
	sd := -bq - math.Sqrt(bq*bq-cq)

	t1 := s.CreatePoint(dims.tangle*math.Cos(theta), dims.tangle*math.Sin(theta))
	c := s.CreatePoint(cx, cy)
	t3 := s.CreatePoint(x3, y3)
	d := s.CreatePoint(x3+sd*tx, y3+sd*ty)
	line := s.CreateLine(t1, flank)
	// An arc runs counterclockwise from its start to its end.
	var arc *sketch.Arc
	if (t1.X()-cx)*(y3-cy)-(t1.Y()-cy)*(x3-cx) < 0 {
		arc = s.CreateArc(c, t3, t1)
	} else {
		arc = s.CreateArc(c, t1, t3)
	}
	runout := s.CreateLine(t3, d)
	s.AddConstraint(
		sketch.NewPointOnLine(center, line),
		sketch.NewTangent(line, arc),
		sketch.NewDistance(c, t1, rho),
		sketch.NewDistance(center, c, rc),
		sketch.NewTangent(runout, arc),
		sketch.NewDistance(center, t3, dims.hi),
	)
	return d
}

// radialToothOutlineEdges is the edge count of a filleted radial-flank tooth: two flanks, the tip arc, the root arc,
// and per side a flank line, a fillet arc and a run-out line.
const radialToothOutlineEdges = 10
