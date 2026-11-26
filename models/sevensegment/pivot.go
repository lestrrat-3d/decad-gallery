package sevensegment

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/r3"
)

// The printed pivot block: one per rotor end, 14 in all. A block is a square 4.8 x 5.9 (s by z) and 1.807 long along u,
// pressed into a square pocket split by the seam between the cradle plate and the top plate. A stepped cone stands on its
// rotor-side face: a 45 degree frustum from a Ø4.0 base to Ø1.49, then an 8 degree cone to a ball tip of R 0.45. The cone
// turns in the rotor's end bore, which keeps its r 0.9 cylinder and a 45 degree mouth; the cone bears on the edge where the
// cylinder meets the mouth. The ball tips rest 0.2 from the bore floors and stop the rotor along its axis. The Ø4.0 base
// and the thinning section follow docs/model-pipeline.md "Thin pins". All lengths are in mm and measured along u from the
// rotor's middle.
const (
	pivotWall    = RotorHalfLength + 0.25 // the plates' end wall (u 18.625); the pocket opens through it
	pivotOuter   = pivotWall + 3.0        // the block's outer face (u 21.625); the pocket ends PressFit beyond
	pivotEndPlay = 0.2                    // the ball tip's distance from the bore floor at the modeled pose; the rotor's axial play each way (T6)
	pivotBaseR   = 2.0                    // the frustum's base on the block's face: Ø4.0

	// The pin's cone and ball tip. The 8 degree half angle is the largest that keeps every drive and visibility limit
	// (instructions.md "Pivot pin"); pivotConeSin, pivotConeCos and pivotConeTan are sin, cos and tan of it
	// (TestSevenSegmentPivotConstants).
	pivotConeSin = 0.13917310096006544
	pivotConeCos = 0.9902680687415704
	pivotConeTan = 0.14054083470239145
	pivotTipR    = 0.45 // the ball tip: over the 0.4 that "Tips" asks for

	pivotBoreFloor = 15.85                         // 0.85 beyond type B's second pocket floor (u 15.0)
	pivotPinTip    = pivotBoreFloor + pivotEndPlay // the ball tip's apex
	pivotTangentR  = pivotTipR * pivotConeCos      // where the cone meets the ball: r 0.4456
	pivotTangentU  = pivotPinTip + pivotTipR*(1-pivotConeSin)

	// The cone's radial clearance in the bore's edge. The tight end keeps 0.2 normal to the cone, which is 0.2/cos of the half
	// angle radially. With the tip at pivotEndPlay from the floor the cone is pivotEndPlay*tan higher in the bore, and the
	// clearance at the modeled pose is larger by that much.
	pivotTight    = 0.2 / pivotConeCos
	pivotClear    = pivotTight + pivotEndPlay*pivotConeTan // 0.2301
	pivotContactR = JournalRadius - pivotClear             // the pin's radius at the bore's edge, modeled: 0.6699
	pivotContactU = pivotTangentU + (pivotContactR-pivotTangentR)/pivotConeTan
	pivotMouthR   = JournalRadius + (RotorHalfLength - pivotContactU) // the 45 degree mouth's radius at the end face

	// The frustum is the line r = u - pivotFrustumC. It keeps 0.2 normal to the mouth with the rotor pushed pivotEndPlay
	// toward the block and dropped by pivotTight in the worst direction: the mouth's radius at u is
	// JournalRadius + u - pivotEndPlay - pivotContactU there, and a 45 degree gap of 0.2 normal is 0.2*sqrt2 radially, plus
	// the drop. The block's face is where the frustum reaches the base radius; the junction is where it meets the cone.
	pivotFrustumC  = pivotContactU + pivotEndPlay - JournalRadius + 0.2*math.Sqrt2 + pivotTight
	pivotBlockFrom = pivotFrustumC + pivotBaseR // 19.818
	pivotBlockLong = pivotOuter - pivotBlockFrom
	pivotJunctionU = (pivotFrustumC + pivotTangentR - pivotTangentU*pivotConeTan) / (1 - pivotConeTan)
	pivotJunctionR = pivotJunctionU - pivotFrustumC

	pivotBlockSide  = 4.8                                // the block's width, along s
	pivotBlockHigh  = 2 * (cradleThick - 1.0 - PressFit) // the block's height, along z: 1.0 of cradle stays under its pocket (5.9)
	pivotCornerR    = 0.675                              // the block's rounds; the pocket's are 0.5, PressFit larger
	pivotPocketR    = pivotCornerR + PressFit            //
	pivotPocketHalf = pivotBlockSide/2 + PressFit        // the pocket's half width along s
	pivotPocketLow  = pivotBlockHigh/2 + PressFit        // the pocket's floor, below the axis: 1.0 of cradle under it
	pivotPocketHigh = Seam + pivotBlockHigh/2 + PressFit // its roof, above the axis: 1.6 into the top plate, over the seam

	// The rotor's end bore: radius JournalRadius from the floor at pivotBoreFloor to the bore's edge at pivotContactU, a
	// 0.5 round at the floor, a teardrop roof truncated at pivotRoofTop from the axis, and a 45 degree mouth from the edge
	// to pivotMouthR at the end face.
	pivotFloorRond = 0.5
	pivotRoofTop   = 1.00

	// pivotJournalPlay is the radial play the drive sees at type B's second pocket mouth (|u| 10.5): the modeled clearance
	// plus 10.5/pivotContactU of the change the end play makes at the bore's edge.
	pivotJournalPlay = pivotClear + 10.5/pivotContactU*pivotEndPlay*pivotConeTan
	// pivotWedge is the friction radius of the cone's contact against a straight journal at friction 0.5:
	// sqrt(1.25)/sqrt(cos^2(8 degrees) + 0.25) (TestSevenSegmentPivotConstants).
	pivotWedge = 1.0078388
)

// cutPivotBore cuts the revolved part of one rotor end's pivot bore in the rotor's local frame: end sgn (+1 for +u).
func (r *run) cutPivotBore(body *decad.Body, sgn float64) (*decad.Body, error) {
	rr := pivotFloorRond
	pts := [][2]float64{{0, pivotBoreFloor}, {JournalRadius - rr, pivotBoreFloor}}
	pts = append(pts, quarterRound(JournalRadius-rr, pivotBoreFloor+rr, rr, -90, 0, 6)[1:]...)
	pts = append(pts, [2]float64{JournalRadius, pivotContactU}, [2]float64{pivotMouthR + 0.2, RotorHalfLength + 0.2},
		[2]float64{0, RotorHalfLength + 0.2})
	if sgn < 0 {
		for i := range pts {
			pts[i][1] = -pts[i][1]
		}
		for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
			pts[i], pts[j] = pts[j], pts[i]
		}
	}
	return r.cutMeridian(body, pts, nil)
}

// cutPivotRoof cuts the teardrop roof of one end's bore: a prism along u from the bore floor to past the end face. Its
// section is the bore's tangent gable: 45 degree sides from the tangent points of the circle up to pivotRoofTop from the
// axis, cut flat there (0.30 wide). The point faces up in print (the rotor prints yellow face down, so up is rotor-local
// (-1, -1)). The pentagon's lower corner sits on the axis, so no side lies at the bore's own height.
func (r *run) cutPivotRoof(body *decad.Body, sgn float64) (*decad.Body, error) {
	k := 1 / math.Sqrt2
	z0 := pivotBoreFloor
	if sgn < 0 {
		z0 = -(RotorHalfLength + 0.2)
	}
	length := RotorHalfLength + 0.2 - pivotBoreFloor
	// U = (-1, 1)/sqrt2 and V = (-1, -1)/sqrt2 (the point's direction) give the normal +Z.
	s, err := r.sketchOn(local(r3.Vec{Z: z0}, r3.Vec{X: -k, Y: k}, r3.Vec{X: -k, Y: -k}))
	if err != nil {
		return nil, err
	}
	t := JournalRadius * k // the tangent point, on the 45 degree line
	h := JournalRadius*math.Sqrt2 - pivotRoofTop
	polygon(s, [][2]float64{{0, 0}, {t, t}, {h, pivotRoofTop}, {-h, pivotRoofTop}, {-t, t}})
	return r.cutExtrude(body, s, 0, 1, length, decad.Along)
}

// pivotPinMeridian is the pin's (r, u) meridian at the +u end, for one revolve. It starts 0.1 inside the block (a cone that
// starts on the face is refused), runs the 45 degree frustum to the junction and the 8 degree cone to the ball, and closes
// the ball tip in 8 chords. The ball is tangent to the cone, so its arc starts 8 degrees from the equator.
func pivotPinMeridian() [][2]float64 {
	pts := [][2]float64{{0, pivotBlockFrom + 0.1}, {pivotBaseR + 0.1, pivotBlockFrom + 0.1},
		{pivotJunctionR, pivotJunctionU}, {pivotTangentR, pivotTangentU}}
	arc := quarterRound(0, pivotPinTip+pivotTipR, pivotTipR, -8, -90, 8)
	return append(pts, arc[1:]...)
}

// pivotBlock builds one block in its own frame: local z is u, local x is s and local y is up. It stands on the rotor's
// middle plane and reaches from u = pivotPinTip to pivotOuter at the +u end.
func (r *run) pivotBlock() (*decad.Body, error) {
	wrap := func(what string, err error) error { return fmt.Errorf("pivot block %s: %w", what, err) }
	body, err := r.roundedPrism(pivotBlockSide/2, pivotBlockHigh/2, pivotBlockHigh/2, pivotCornerR, pivotBlockFrom, pivotOuter)
	if err != nil {
		return nil, wrap("body", err)
	}
	pin, err := r.meridian(pivotPinMeridian())
	if err != nil {
		return nil, wrap("pin", err)
	}
	if body, err = decad.Union(r.ctx, body, pin); err != nil {
		return nil, wrap("union pin", err)
	}
	return body, nil
}

// roundedPrism is the block's or the pocket's body between u0 and u1: the intersection of the section prism (a rectangle: half width half
// along s, below and above the axis along z, its four long edges rounded rad) and the plan prism (u by s, its far corners at u1 rounded rad, taken
// through the whole height). The section prism starts at u0 and the plan prism 1 mm before it, so the near corners
// of the plan prism's rounds lie outside the body and u0 is a plain face.
func (r *run) roundedPrism(half, below, above, rad, u0, u1 float64) (*decad.Body, error) {
	secS, err := r.sketchOn(local(r3.Vec{Z: u0}, r3.Vec{X: 1}, r3.Vec{Y: 1}))
	if err != nil {
		return nil, err
	}
	if err := rrect(secS, -half, half, -below, above, rad); err != nil {
		return nil, fmt.Errorf("section: %w", err)
	}
	sec, err := r.extrude(secS, 0, u1-u0+1, decad.Along)
	if err != nil {
		return nil, fmt.Errorf("section prism: %w", err)
	}
	// The plan sketch lies in the plane y = 0 with U = +Z (u) and V = +X (s): its normal is Z x X = +Y.
	planS, err := r.sketchOn(local(r3.Vec{Y: -below - 1}, r3.Vec{Z: 1}, r3.Vec{X: 1}))
	if err != nil {
		return nil, err
	}
	if err := rrect(planS, u0-1, u1, -half, half, rad); err != nil {
		return nil, fmt.Errorf("plan: %w", err)
	}
	plan, err := r.extrude(planS, 0, below+above+2, decad.Along)
	if err != nil {
		return nil, fmt.Errorf("plan prism: %w", err)
	}
	return decad.Intersect(r.ctx, sec, plan)
}

// pivotPocketTool builds the pocket of segment seg's end sgn in the world: the block's 3.3 square pocket from the bore's
// end wall to 0.05 past the block's outer face, centred on the rotor axis at RotorAxisZ. It is cut from the cradle plate
// and from the top plate, and each keeps its half.
func (r *run) pivotPocketTool(seg Segment, sgn float64) (*decad.Body, error) {
	b, err := r.roundedPrism(pivotPocketHalf, pivotPocketLow, pivotPocketHigh, pivotPocketR, pivotWall-1, pivotOuter+PressFit)
	if err != nil {
		return nil, fmt.Errorf("pocket %s: %w", seg, err)
	}
	return r.toSegment(seg, sgn, b)
}

// cutPivotPockets cuts both of segment seg's pockets from a plate.
func (r *run) cutPivotPockets(body *decad.Body, seg Segment) (*decad.Body, error) {
	for _, sgn := range []float64{-1, 1} {
		tool, err := r.pivotPocketTool(seg, sgn)
		if err != nil {
			return nil, err
		}
		if body, err = decad.Cut(r.ctx, body, tool); err != nil {
			return nil, fmt.Errorf("cut pivot pocket %s: %w", seg, err)
		}
	}
	return body, nil
}

// toSegment places a body built in the rotor-local frame (z along u) at segment seg's world position at turn 0 with one
// placement of the rotor frame; the frame is turned 180 degrees about the local up axis when sgn is -1, so the body
// lands at the -u end.
func (r *run) toSegment(seg Segment, sgn float64, b *decad.Body) (*decad.Body, error) {
	frame := rotorFrame(seg)
	if sgn < 0 {
		// The -u end is the +u block turned 180 degrees about the local up axis (v): u and n reverse.
		var err error
		if frame, err = r3.NewFrame(frame.Origin(), frame.U().Scale(-1), frame.V()); err != nil {
			return nil, fmt.Errorf("segment %s: frame: %w", seg, err)
		}
	}
	b, err := placement.Place(r.ctx, b, &frame)
	if err != nil {
		return nil, fmt.Errorf("segment %s: place: %w", seg, err)
	}
	return b, nil
}

// pivotBlocks builds the 14 blocks, in the world at the model's Placement, indexed [segment][0 for the -u end, 1 for +u].
func (r *run) pivotBlocks() ([7][2]*decad.Body, error) {
	var out [7][2]*decad.Body
	proto, err := r.pivotBlock()
	if err != nil {
		return out, err
	}
	for _, seg := range Segments() {
		for i, sgn := range []float64{-1, 1} {
			// The last block is the prototype itself, so no body is left over in the document.
			b := proto
			if seg != SegG || sgn < 0 {
				var err error
				if b, err = decad.Copy(r.ctx, proto); err != nil {
					return out, fmt.Errorf("pivot block %s: copy: %w", seg, err)
				}
			}
			if b, err = r.toSegment(seg, sgn, b); err != nil {
				return out, err
			}
			if out[seg][i], err = placement.Place(r.ctx, b, r.p.Placement); err != nil {
				return out, fmt.Errorf("pivot block %s: place: %w", seg, err)
			}
		}
	}
	return out, nil
}
