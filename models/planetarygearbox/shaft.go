package planetarygearbox

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// A printed shaft prints as two halves, each lying on its cut face. See "Split shafts" in docs/model-pipeline.md. Read
// along the seam, every square step is an inside corner, so a step that does not face a purchased bearing is a cone at
// 30° to the axis, and a flat that stops partway along a shaft ends in a 30° ramp. A step that faces a bearing keeps
// its square corner, and so does the start of a flat on which a sun's D bore stops.

const (
	// endFlatMargin is how far a flat on a shaft's outer end stays clear of the bearing boss that the shaft leaves.
	endFlatMargin = 1.0
	// rampOver is how far a flat tool's ramp reaches past the rod's surface, so the ramp leaves the rod cleanly.
	rampOver = collarMargin / 6
	// collarBand is the shortest cylinder at the collar's largest radius, on its bearing side, when the cone to the sun
	// section takes the rest of the collar.
	collarBand = 0.8
	// flatOver is how far a flat tool reaches past the rod's surface, and past the rod's end.
	flatOver = collarMargin
)

// flat is one flat of a shaft's rod: the plane FlatDepth inside the +X rim, at full depth from lo to hi. A flat with a
// ramp at one end leaves the rod there along a plane at 30° to the axis; one without ends square.
type flat struct {
	lo, hi         float64
	loRamp, hiRamp bool
	// section is the round section of the rod the flat cuts: the bearing section or, narrower on a 608ZZ shaft, the
	// sun section.
	section rod
}

// rampMargin is how far a flat's full-depth end stays clear of the bearing boss it faces, so its ramp stays clear too.
func (r *run) rampMargin() float64 {
	return math.Max(endFlatMargin, r.p.FlatDepth*coneRun+0.1)
}

// flatOutline is the meridian of the tool that cuts f: the half plane beyond the flat, which ends in a ramp or square.
func (r *run) flatOutline(f flat) []vertex {
	radius := f.section.diameter / 2
	x0, x1, xr := radius-r.p.FlatDepth, radius+flatOver, radius+rampOver
	cx0 := f.section.expr + " / 2 - flat_depth"
	cx1, cxr := f.section.expr+" / 2 + collar_margin", f.section.expr+" / 2 + collar_margin / 6"
	run := (xr - x0) * coneRun
	var v []vertex
	if f.loRamp {
		v = append(v, vertex{x1, f.lo - run, cx1}, vertex{xr, f.lo - run, cxr}, vertex{x0, f.lo, cx0})
	} else {
		v = append(v, vertex{x1, f.lo, cx1}, vertex{x0, f.lo, cx0})
	}
	if f.hiRamp {
		return append(v, vertex{x0, f.hi, ""}, vertex{xr, f.hi + run, cxr}, vertex{x1, f.hi + run, cx1})
	}
	return append(v, vertex{x0, f.hi, ""}, vertex{x1, f.hi, cx1})
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
// cover, where a coupling or a set screw grips, on the bearing section, and ends in a ramp on the bearing's side.
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
		flat{lo: start - flatOver, hi: start + p.InputShaftLength - r.rampMargin(), hiRamp: true, section: r.shaftRod()})
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
// below the step up to top, joined in one union, then a flat over the rod's free end that starts in a ramp, so no
// square corner is left.
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
		flat{lo: top - r.p.OutputShaftLength + r.rampMargin(), hi: top + flatOver, loRamp: true, section: r.shaftRod()})
}
