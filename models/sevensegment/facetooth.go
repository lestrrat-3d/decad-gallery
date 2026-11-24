package sevensegment

import (
	"fmt"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// Face teeth: a ring of ten sawtooth teeth whose ramps are helicoids about the shaft axis. Decad builds a ramp as a Coil
// and cannot join a Coil result to a ring, so each tooth is cut from a blank and then copied (see steps.md, "Face
// teeth"). The teeth are built in a local frame whose Z is the shaft axis and whose angle runs counter-clockwise about
// +Z; the part's own frame places them.

// ChainTolerance is the chain tolerance of the display's Decad document, millimetres.
const ChainTolerance = 0.02

// NewDocument returns the Decad document the display is built in. The face teeth need its chain tolerance.
func NewDocument() *decad.Document {
	return decad.New(decad.WithChainTolerance(units.Millimeters(ChainTolerance)))
}

// faceTeeth describes ten teeth. Tooth k rises from its sharp valley at 36k (height 0) over Rise degrees to its sharp
// peak H, then falls over Fall degrees to the next valley. Rise + Fall is 36.
type faceTeeth struct {
	Name       string
	R0, R1     float64 // radial extent of the teeth
	H          float64
	Rise, Fall float64
	Tip        float64 // the peak is cut back by this much: the tip's land
	Bot        float64 // the blank's bottom above the sharp valley; the plate's top lies above it (the root land)
	RampSplit  int     // pieces of each radial profile edge of the gentler ramp's tool
	SteepSplit int     // the same for the steeper ramp's tool, when not 0
	Overhang   float64 // how far a tool's profile reaches past R0 and R1
	Plate      float64 // height of the plate's top above the sharp valley under these teeth
	_          struct{}
}

func (f faceTeeth) top() float64 { return f.H - f.Tip }

// rectangle sketches the rectangle r0..r1 by z0..z1 on a meridian plane, fully constrained: a corner dimensioned from the
// sketch origin on the shaft axis, each side held horizontal or vertical, and the width and height.
func rectangle(s *sketch.Sketch, r0, r1, z0, z1 float64) {
	a := sketchcheck.DimensionPoint(s, r0, z0)
	b, c, d := s.CreatePoint(r1, z0), s.CreatePoint(r1, z1), s.CreatePoint(r0, z1)
	ab, bc, cd, da := s.CreateLine(a, b), s.CreateLine(b, c), s.CreateLine(c, d), s.CreateLine(d, a)
	s.AddConstraint(sketch.NewHorizontal(ab), sketch.NewDistance(a, b, r1-r0), sketch.NewVertical(bc),
		sketch.NewDistance(b, c, z1-z0), sketch.NewHorizontal(cd), sketch.NewVertical(da))
}

// meridianFrame is the sketch plane through the axis: U along +X (radius), V along +Z (height).
func meridianFrame() r3.Frame {
	f, _ := r3.NewFrame(r3.Vec{}, r3.Vec{X: 1}, r3.Vec{Z: 1})
	return f
}

var localAxis = decad.SketchLine{Start: decad.Point2{U: 0, V: 0}, End: decad.Point2{U: 0, V: 1}}

// turnZ turns about the local axis by deg degrees, counter-clockwise about +Z.
func turnZ(deg float64) (r3.Transform, error) {
	return r3.RotationAround(r3.Vec{}, r3.Vec{Z: 1}, units.Degrees(deg))
}

// helixTool builds the tool for one ramp: the rectangle ri..ro by z0..z0+h swept helically over span degrees, its
// bottom rising by rise (falling when falling is true), then turned to start at angle start.
func (r *run) helixTool(ri, ro, z0, h, start, span, rise float64, falling bool) (*decad.Body, error) {
	s, err := r.sketchOn(meridianFrame())
	if err != nil {
		return nil, err
	}
	rectangle(s, ri, ro, z0, z0+h)
	pr, err := r.profile(s, 0)
	if err != nil {
		return nil, fmt.Errorf("tool profile: %w", err)
	}
	axis := localAxis
	var opts []decad.CoilOption
	if falling {
		axis = decad.SketchLine{Start: decad.Point2{U: 0, V: 1}, End: decad.Point2{U: 0, V: 0}}
		opts = append(opts, decad.WithLeftHand())
	}
	body, err := decad.Coil(r.ctx, r.doc, s, pr, axis, units.Millimeters(rise*360/span), units.Scalar(span/360), opts...)
	if err != nil {
		return nil, fmt.Errorf("coil: %w", err)
	}
	if start == 0 {
		return body, nil
	}
	t, err := turnZ(start)
	if err != nil {
		return nil, err
	}
	return decad.Transform(r.ctx, body, t)
}

// tooth builds tooth 0: a partial revolve of the blank cut by one rising and one falling tool. The blank's end planes lie
// inside the region each tool removes, and each tool's caps lie one degree outside the blank, so no cap touches material.
func (r *run) faceTooth(f faceTeeth) (*decad.Body, error) {
	b, top := f.Bot, f.top()
	upLead, dnLead := f.H/f.Rise, f.H/f.Fall // mm of height per degree
	rf := b / upLead                         // rising foot at z = b
	ff := 36 - b/dnLead                      // falling foot at z = b
	th0 := rf - 0.25/upLead                  // the rising helicoid 0.25 below b at the blank's start plane
	th1 := ff + 0.25/dnLead                  // the falling helicoid 0.25 below b at the blank's end plane
	s, err := r.sketchOn(meridianFrame())
	if err != nil {
		return nil, err
	}
	rectangle(s, f.R0, f.R1, b, top)
	pr, err := r.profile(s, 0)
	if err != nil {
		return nil, fmt.Errorf("blank profile: %w", err)
	}
	blank, err := decad.Revolve(r.ctx, r.doc, s, pr, localAxis, decad.AngleExtent{A: units.Degrees(th1 - th0), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("blank: %w", err)
	}
	t0, err := turnZ(th0)
	if err != nil {
		return nil, err
	}
	if blank, err = decad.Transform(r.ctx, blank, t0); err != nil {
		return nil, err
	}
	ri, ro := f.R0-f.Overhang, f.R1+f.Overhang
	us, ue := th0-1, f.Rise+2
	z0 := us * upLead
	up, err := r.helixTool(ri, ro, z0, top+1.5-z0, us, ue-us, (ue-us)*upLead, false)
	if err != nil {
		return nil, fmt.Errorf("rising tool: %w", err)
	}
	if blank, err = decad.Cut(r.ctx, blank, up); err != nil {
		return nil, fmt.Errorf("rising cut: %w", err)
	}
	ds, de := f.Rise-2, th1+1
	zdn := func(th float64) float64 { return f.H - (th-f.Rise)*dnLead }
	dn, err := r.helixTool(ri, ro, zdn(ds), top+1.5-zdn(th1), ds, de-ds, (de-ds)*dnLead, true)
	if err != nil {
		return nil, fmt.Errorf("falling tool: %w", err)
	}
	if blank, err = decad.Cut(r.ctx, blank, dn); err != nil {
		return nil, fmt.Errorf("falling cut: %w", err)
	}
	return blank, nil
}

// faceTeethBodies returns tooth 0 and its nine copies, each turned 36 degrees more, all turned by shift degrees.
func (r *run) faceTeethBodies(f faceTeeth, shift float64) ([]*decad.Body, error) {
	tooth, err := r.faceTooth(f)
	if err != nil {
		return nil, err
	}
	// The first tooth is the original turned by shift, so no loose tooth is left in the document.
	first, err := turnZ(shift)
	if err != nil {
		return nil, err
	}
	head, err := decad.Transform(r.ctx, tooth, first)
	if err != nil {
		return nil, err
	}
	out := []*decad.Body{head}
	for k := 1; k < ClickerSteps; k++ {
		t, err := turnZ(ClickerStep * float64(k))
		if err != nil {
			return nil, err
		}
		c, err := decad.Copy(r.ctx, head, decad.WithCopyPlacement(t))
		if err != nil {
			return nil, fmt.Errorf("copy %d: %w", k, err)
		}
		out = append(out, c)
	}
	return out, nil
}

// joinTeeth unions the teeth into body one at a time.
func (r *run) joinTeeth(body *decad.Body, teeth []*decad.Body) (*decad.Body, error) {
	for k, t := range teeth {
		var err error
		if body, err = decad.Union(r.ctx, body, t); err != nil {
			return nil, fmt.Errorf("union tooth %d: %w", k+1, err)
		}
	}
	return body, nil
}

// localBox is a box on the radius from r0 to r1, w wide, over local Z from z0 to z1, turned by angle degrees.
func (r *run) localBox(r0, r1, w, z0, z1, angle float64) (*decad.Body, error) {
	s, err := r.sketchOn(across(z0))
	if err != nil {
		return nil, err
	}
	polygon(s, [][2]float64{{r0, -w / 2}, {r1, -w / 2}, {r1, w / 2}, {r0, w / 2}})
	body, err := r.extrude(s, 0, z1-z0, decad.Along)
	if err != nil {
		return nil, err
	}
	if angle == 0 {
		return body, nil
	}
	t, err := turnZ(angle)
	if err != nil {
		return nil, err
	}
	return decad.Transform(r.ctx, body, t)
}
