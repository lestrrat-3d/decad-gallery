package sevensegment

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/r3"
)

// The printed pivot block: one per rotor end, 14 in all. A block is a square 3.2 x 3.2 (s by z) and 2.0 long along u,
// pressed into a square pocket split by the seam between the cradle plate and the top plate. A Ø0.8 pin stands on its
// rotor-side face and turns in the rotor's end bore. All lengths are in mm and measured along u from the rotor's middle.
const (
	pivotPinRadius  = 0.7                                // option B: Ø1.4; the rotor's end bore (JournalRadius) leaves 0.2, the Tight minimum
	pivotBlockSide  = 4.8                                // the block's width, along s
	pivotBlockHigh  = 2 * (cradleThick - 1.0 - PressFit) // the block's height, along z: 1.0 of cradle stays under its pocket (5.9)
	pivotBlockFrom  = RotorHalfLength + 0.25             // the end wall: the block's rotor-side face (T6 0.25 from the rotor's end face)
	pivotBlockLong  = 3.0                                //
	pivotPinTip     = pivotBlockFrom - 1.875             // the pin stands 1.25 from the block's face
	pivotRootLeg    = 0.25                               // the pin's 45 degree root cone, inside the end gap
	pivotTipLeg     = 0.1                                // the pin's tip chamfer
	pivotCornerR    = 0.675                              // the block's rounds; the pocket's are 0.5, PressFit larger
	pivotPocketR    = pivotCornerR + PressFit            //
	pivotPocketHalf = pivotBlockSide/2 + PressFit        // the pocket's half width along s
	pivotPocketLow  = pivotBlockHigh/2 + PressFit        // the pocket's floor, below the axis: 1.0 of cradle under it
	pivotPocketHigh = Seam + pivotBlockHigh/2 + PressFit // its roof, above the axis: 1.6 into the top plate, over the seam

	// The rotor's end bore: radius JournalRadius from the floor at pivotBoreFloor to the end face, a 0.5 round at the floor,
	// a teardrop roof truncated at pivotRoofTop from the axis, and a 45 degree countersink to pivotMouthR at the end face.
	pivotBoreFloor = 16.2
	pivotFloorRond = 0.75
	pivotRoofTop   = 1.00
	pivotMouthR    = 1.395
)

// cutPivotBore cuts the revolved part of one rotor end's pivot bore in the rotor's local frame: end sgn (+1 for +u).
func (r *run) cutPivotBore(body *decad.Body, sgn float64) (*decad.Body, error) {
	rr := pivotFloorRond
	mouth := RotorHalfLength - (pivotMouthR - JournalRadius) // where the 45 degree countersink starts
	pts := [][2]float64{{0, pivotBoreFloor}, {JournalRadius - rr, pivotBoreFloor}}
	pts = append(pts, quarterRound(JournalRadius-rr, pivotBoreFloor+rr, rr, -90, 0, 6)[1:]...)
	pts = append(pts, [2]float64{JournalRadius, mouth}, [2]float64{pivotMouthR + 0.2, RotorHalfLength + 0.2},
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

// pivotBlock builds one block in its own frame: local z is u, local x is s and local y is up. It stands on the rotor's
// middle plane and reaches from u = pivotPinTip to pivotBlockFrom+pivotBlockLong at the +u end.
func (r *run) pivotBlock() (*decad.Body, error) {
	wrap := func(what string, err error) error { return fmt.Errorf("pivot block %s: %w", what, err) }
	body, err := r.roundedPrism(pivotBlockSide/2, pivotBlockHigh/2, pivotBlockHigh/2, pivotCornerR, pivotBlockFrom, pivotBlockFrom+pivotBlockLong)
	if err != nil {
		return nil, wrap("body", err)
	}
	// The pin, with its 45 degree root cone starting inside the block (a cone that starts on the face is refused).
	leg := pivotRootLeg + 0.1
	pin, err := r.meridian([][2]float64{{0, pivotBlockFrom + 0.1}, {pivotPinRadius + leg, pivotBlockFrom + 0.1},
		{pivotPinRadius, pivotBlockFrom - pivotRootLeg}, {pivotPinRadius, pivotPinTip + pivotTipLeg},
		{pivotPinRadius - pivotTipLeg, pivotPinTip}, {0, pivotPinTip}})
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
	b, err := r.roundedPrism(pivotPocketHalf, pivotPocketLow, pivotPocketHigh, pivotPocketR, pivotBlockFrom-1, pivotBlockFrom+pivotBlockLong+PressFit)
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
