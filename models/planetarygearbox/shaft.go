package planetarygearbox

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// A printed shaft prints as two halves, each lying on its cut face. See "Split shafts" in docs/model-pipeline.md. Read
// along the seam, every square step is an inside corner, so a step that does not face a purchased bearing is a cone at
// 30° to the axis. A step that faces a bearing keeps its square corner, and so does the end of a flat on which a D
// bore stops: a sun's on the input shaft, or a collar ring's on a free end.

const (
	// collarBand is the shortest cylinder at the collar's largest radius, on its bearing side, when the cone to the sun
	// section takes the rest of the collar.
	collarBand = 0.8
	// flatOver is how far a flat tool reaches past the rod's surface, and past the rod's end.
	flatOver = collarMargin
)

// A collar ring is a separate printed collar (see "Arbor collars" in docs/model-pipeline.md) pressed on the flat of the
// input or the output shaft's free end, outside the cover's bearing. A collar of the shaft's own body there would stop
// the shaft from passing the bearing during assembly, because the shaft already has a collar or a thick part on the
// bearing's other side. The ring keeps a part pressed on the free end off the bearing and the boss. Its bearing face
// stands collarBearingGap from the bearing's outer face, and the ring is located by the square end of the free end's
// flat, on which its D bore's tooth stops.
//
// The ring is collarRingLength long. Over its pressed length its wall is collarRingWall round the D bore, which is
// pressFitClearance outside the shaft: the 1.6 mm press-fit wall of Corners and roots with 0.1 mm to spare, so the
// measured wall meets it. At the bearing face its outside narrows to the shaft collar diameter, so it
// bears on the bearing's inner ring only, and its bore opens to collarRingReliefGap round the shaft, which keeps the
// 0.8 mm wall of Corners and roots there (collarMargin less collarRingReliefGap). A 30° cone joins each narrow part to
// the full one, and the bore's cone ends collarRingRelief from the face, where the press fit starts.
const (
	// collarRingLength is the collar ring's length along the shaft.
	collarRingLength = 4.0
	// collarRingRelief is how far the press fit starts from the ring's bearing face.
	collarRingRelief = 1.0
	// collarRingReliefGap is the radial gap between the ring's relieved bore and the round shaft, a Regular fit.
	collarRingReliefGap = 0.4
	// collarRingWall is the ring's wall round its D bore over the pressed length.
	collarRingWall = minPressFitWall + 0.1
	// minStub is the least length of a free end beyond its collar ring.
	minStub = 1.0
	// minShaftReach is the least InputShaftLength or OutputShaftLength: the gap to the bearing, the collar ring and
	// minStub of free end.
	minShaftReach = collarBearingGap + collarRingLength + minStub
)

// FrontCollarName and RearCollarName are the names of the collar rings on the input and the output shaft's free ends.
const (
	FrontCollarName = "Front Collar"
	RearCollarName  = "Rear Collar"
)

// flat is one flat of a shaft's rod: the plane FlatDepth inside the +X rim, at full depth from lo to hi. Both ends are
// square: each either runs past the end of the rod or is the end on which a D bore's tooth stops.
type flat struct {
	lo, hi float64
	// section is the round section of the rod the flat cuts: the bearing section or, narrower on a 608ZZ shaft, the
	// sun section.
	section rod
}

// flatOutline is the meridian of the tool that cuts f: the half plane beyond the flat, square at both ends.
func (r *run) flatOutline(f flat) []vertex {
	radius := f.section.diameter / 2
	x0, x1 := radius-r.p.FlatDepth, radius+flatOver
	return []vertex{
		{x1, f.lo, f.section.expr + " / 2 + collar_margin"}, {x0, f.lo, f.section.expr + " / 2 - flat_depth"},
		{x0, f.hi, ""}, {x1, f.hi, f.section.expr + " / 2 + collar_margin"},
	}
}

// freeFlatEnd is the distance from a cover's bearing boss face to the square end of the flat on the shaft's free end:
// the collar ring's bearing gap plus its relief, where the ring's D bore starts.
func freeFlatEnd() float64 { return collarBearingGap + collarRingRelief }

// buildCollarRing builds a collar ring about the local origin: its bearing face on z = 0 and its outer face at
// collarRingLength, its D bore's flat on +X. The outside is one revolved meridian; the D bore and the relief at the
// bearing face are then cut, as a sun's bore and countersink are.
func (r *run) buildCollarRing(ctx context.Context) (*decad.Body, error) {
	const what = "collar ring"
	rod := r.d.shaft / 2
	face, outer := r.d.collar/2, rod+pressFitClearance+collarRingWall
	relief, press := rod+collarRingReliefGap, rod+pressFitClearance
	if r.p.FlatDepth == 0 {
		// A round bore is part of the revolved section: the ring is one annulus, its relief included. Decad refuses
		// the cut of a round bore along the axis of a revolved body, "a circle segment's CCW flag contradicts its
		// range order".
		ring, err := r.revolveMeridian(ctx, what, []vertex{
			{press, collarRingRelief, "shaft_diameter / 2 + shaft_bore_clearance"},
			{relief, collarRingRelief - (relief-press)*coneRun, "shaft_diameter / 2 + collar_ring_relief_gap"},
			{relief, 0, ""}, {face, 0, "shaft_collar_diameter / 2"},
			{outer, (outer - face) * coneRun, "shaft_diameter / 2 + shaft_bore_clearance + collar_ring_wall"},
			{outer, collarRingLength, ""}, {press, collarRingLength, ""},
		})
		if err != nil {
			return nil, err
		}
		if !ring.IsSolid() {
			return nil, fmt.Errorf("%s: result is not a solid", what)
		}
		return ring, nil
	}
	ring, err := r.revolveMeridian(ctx, what, []vertex{
		{0, 0, ""}, {face, 0, "shaft_collar_diameter / 2"},
		{outer, (outer - face) * coneRun, "shaft_diameter / 2 + shaft_bore_clearance + collar_ring_wall"},
		{outer, collarRingLength, ""}, {0, collarRingLength, ""},
	})
	if err != nil {
		return nil, err
	}
	bore, err := r.extrudeDSection(ctx, dSection{
		what: what + " bore", z: -1, length: collarRingLength + 2, radius: press, depth: r.p.FlatDepth,
	})
	if err != nil {
		return nil, err
	}
	if ring, err = decad.Cut(ctx, ring, bore); err != nil {
		return nil, fmt.Errorf("%s: cut D bore: %w", what, err)
	}
	tool, err := r.revolveMeridian(ctx, what+" relief", []vertex{
		{0, -1, ""}, {relief, -1, "shaft_diameter / 2 + collar_ring_relief_gap"},
		{relief, collarRingRelief - (relief-press)*coneRun, ""},
		{press, collarRingRelief, "shaft_diameter / 2 + shaft_bore_clearance"}, {0, collarRingRelief, ""},
	})
	if err != nil {
		return nil, err
	}
	if ring, err = decad.Cut(ctx, ring, tool); err != nil {
		return nil, fmt.Errorf("%s: cut relief: %w", what, err)
	}
	if !ring.IsSolid() {
		return nil, fmt.Errorf("%s: result is not a solid", what)
	}
	return ring, nil
}

// flatCut cuts each of the flats out of body, which is the shaft's rod. With FlatDepth zero or no flats it returns body
// unchanged. The tools of every flat join into one, so the body takes one cut however many flats there are.
func (r *run) flatCut(ctx context.Context, what string, body *decad.Body, flats ...flat) (*decad.Body, error) {
	if r.p.FlatDepth == 0 || len(flats) == 0 {
		return body, nil
	}
	var tools []*decad.Body
	for _, f := range flats {
		tool, err := r.extrudeMeridian(ctx, what+" flat", r.flatOutline(f), f.section.diameter/2+flatOver)
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	tool := tools[0]
	if len(tools) > 1 {
		var err error
		if tool, err = decad.Union(ctx, tools[0], tools[1:]...); err != nil {
			return nil, fmt.Errorf("%s: join flats: %w", what, err)
		}
	}
	return r.cutShaft(ctx, what, body, tool)
}

// cutShaft cuts tool out of the shaft body and requires a solid.
func (r *run) cutShaft(ctx context.Context, what string, body, tool *decad.Body) (*decad.Body, error) {
	cut, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("%s: cut flat: %w", what, err)
	}
	if !cut.IsSolid() {
		return nil, fmt.Errorf("%s: result is not a solid", what)
	}
	return cut, nil
}

// buildInputShaft builds the input shaft about the local origin: a rod that starts InputShaftLength below the front
// cover's bearing boss and ends flush with the top of the first sun, revolved from one meridian. The rod is as wide as
// the bearing's bore where the bearing sits and no wider than the sun section (6 mm at most, see maxSunShaft) where
// the sun sits. A collar between the front bearing's inner race and the sun's lower face keeps the sun off the cover,
// with collarBearingGap to the bearing and collarClearance to the sun. The collar's bearing side is a square step that
// faces the bearing's inner ring. Its sun side is a 30° cone from the collar to the sun section that takes the
// collar's end, so the collar's whole length, measured to where the cone meets the sun section, stays what it was.
// A cone down a 1.2 mm step is shorter than the collar. Where the sun section is narrower than the bearing section by
// more than that (a 608ZZ shaft), the cone is longer than the collar's cylinder, so the collar keeps a cylinder of only
// collarBand at its bearing-side face, and the cone starts above it at the radius that makes it meet the sun section at
// collarTop (4.41 mm instead of 5.2). The band keeps the face's outer corner at 90°; a cone that started at the face
// would leave a tip of exactly 60°, and a band under 0.8 mm adds its 30° to the corner's 90° inside the tip window.
//
// One flat runs from the sun's lower face to the top, through the sun, on the sun section, and ends square at the
// sun's lower face, where the sun's D bore stops on it. A second flat runs along the end that reaches out of the
// cover, on the bearing section, where the front collar ring and a part pressed on outside it grip. It ends square
// freeFlatEnd outside the bearing boss, where the collar ring's D bore stops on it.
func (r *run) buildInputShaft(ctx context.Context) (*decad.Body, error) {
	p, d := r.p, r.d
	start := -(p.StageGap + p.CoverThickness + shaftBearing(p).Width + p.InputShaftLength)
	collarBottom := -(p.StageGap + p.CoverThickness) + collarBearingGap
	collarTop := -collarClearance
	bearing, sun, collar := d.shaft/2, d.sunShaft/2, d.collar/2
	// The collar's largest radius is the shaft collar radius, or less when the cone to the sun section would not fit.
	collarRadius := math.Min(collar, sun+(collarTop-collarBottom-collarBand)/coneRun)
	if collarRadius <= bearing {
		return nil, fmt.Errorf("input shaft: the collar from %v to %v is too short for a cone down to the sun section",
			collarBottom, collarTop)
	}
	coneBottom := collarTop - (collarRadius-sun)*coneRun
	const bearingExpr, sunExpr, collarExpr = "shaft_diameter / 2", "sun_shaft_diameter / 2", "shaft_collar_diameter / 2"
	collarVertex := vertex{collarRadius, collarBottom, collarExpr}
	if collarRadius != collar {
		collarVertex.expr = ""
	}
	outline := []vertex{
		{0, start, ""}, {bearing, start, bearingExpr}, {bearing, collarBottom, ""}, collarVertex,
	}
	if coneBottom > collarBottom+1e-9 {
		outline = append(outline, vertex{collarRadius, coneBottom, ""})
	}
	outline = append(outline, vertex{sun, collarTop, sunExpr}, vertex{sun, p.GearThickness, ""}, vertex{0, p.GearThickness, ""})
	shaft, err := r.revolveMeridian(ctx, "input shaft", outline)
	if err != nil {
		return nil, err
	}
	return r.flatCut(ctx, "input shaft", shaft,
		flat{lo: 0, hi: p.GearThickness + flatOver, section: r.sunRod()},
		flat{lo: start - flatOver, hi: start + p.InputShaftLength - freeFlatEnd(), section: r.shaftRod()})
}

// buildLinkShaft builds a link shaft about the local origin, its bottom on the carrier's lower face, revolved from one
// meridian: a thick part of the shaft collar diameter that fills the carrier's hole and ends level with the plate's
// upper face, a 30° cone from there to the rod, and the rod up to the next sun's top. The cone runs along the rod into
// the next sun, whose rear bore has a matching countersink (see countersinkSun), because the 0.75 mm of thick part that
// would stand above the plate cannot hold a cone. One plane at the flat's depth follows the shaft: it takes the thick
// part's D, runs up the cone at the same 30°, FlatDepth inside its surface, and goes on along the rod, so the flat has
// no end to square off.
func (r *run) buildLinkShaft(ctx context.Context) (*decad.Body, error) {
	p, d := r.p, r.d
	thickTop := p.CarrierThickness
	top := p.GearThickness + p.CarrierThickness + p.StageGap
	rod, collar := d.sunShaft/2, d.collar/2
	coneTop := thickTop + (collar-rod)*coneRun
	const rodExpr, collarExpr = "sun_shaft_diameter / 2", "shaft_collar_diameter / 2"
	shaft, err := r.revolveMeridian(ctx, "link shaft", []vertex{
		{0, 0, ""}, {collar, 0, collarExpr}, {collar, thickTop, ""}, {rod, coneTop, rodExpr}, {rod, top, ""},
		{0, top, ""},
	})
	if err != nil {
		return nil, err
	}
	if p.FlatDepth == 0 {
		return shaft, nil
	}
	f := p.FlatDepth
	tool, err := r.extrudeMeridian(ctx, "link shaft flat", []vertex{
		{collar + flatOver, -flatOver, "shaft_collar_diameter / 2 + collar_margin"},
		{collar - f, -flatOver, "shaft_collar_diameter / 2 - flat_depth"},
		{collar - f, thickTop, ""},
		{rod - f, coneTop, "sun_shaft_diameter / 2 - flat_depth"},
		{rod - f, top + flatOver, ""},
		{collar + flatOver, top + flatOver, "shaft_collar_diameter / 2 + collar_margin"},
	}, collar+flatOver)
	if err != nil {
		return nil, err
	}
	return r.cutShaft(ctx, "link shaft", shaft, tool)
}

// buildOutputShaft builds the output shaft about the local origin, its bottom on the last carrier's lower face: a D
// thick part of the shaft collar diameter, pressed through the carrier's hole, that ends fitClearance short of the rear
// bearing's inner race, with a square step that faces the race, and a round rod of the shaft diameter from stepOverlap
// below the step up to top, joined in one union, then a flat over the rod's free end that starts square freeFlatEnd
// outside the bearing boss, where the rear collar ring's D bore stops on it.
func (r *run) buildOutputShaft(ctx context.Context, thickTop, top float64) (*decad.Body, error) {
	const what = "output shaft"
	thick, err := r.extrudeDSection(ctx, dSection{
		what: what + " thick part", z: 0, length: thickTop,
		radius: r.d.collar / 2, depth: r.p.FlatDepth, expr: "shaft_collar_diameter",
	})
	if err != nil {
		return nil, err
	}
	rod, err := r.extrudeShaft(ctx, what, r.shaftRod(), thickTop-stepOverlap, top-thickTop+stepOverlap)
	if err != nil {
		return nil, err
	}
	shaft, err := decad.Union(ctx, thick, rod)
	if err != nil {
		return nil, fmt.Errorf("%s: join thick part and rod: %w", what, err)
	}
	if !shaft.IsSolid() {
		return nil, fmt.Errorf("%s: result is not a solid", what)
	}
	return r.flatCut(ctx, what, shaft,
		flat{lo: top - r.p.OutputShaftLength + freeFlatEnd(), hi: top + flatOver, section: r.shaftRod()})
}
