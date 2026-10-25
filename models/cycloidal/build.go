package cycloidal

import (
	"context"
	"fmt"

	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
)

// Body names, as the Fusion add-in names them.
const (
	HousingName     = "Housing"
	CamName         = "Eccentric Cam"
	OutputPlateName = "Output Plate"
)

// DiscName is the name of rotor disc index (0-based): "Cycloidal Disk 1", "Cycloidal Disk 2".
func DiscName(index int) string { return fmt.Sprintf("Cycloidal Disk %d", index+1) }

// OutputPinName is the name of output pin index (0-based): "Output Pin 1" and so on.
func OutputPinName(index int) string { return fmt.Sprintf("Output Pin %d", index+1) }

// NamedBody is a finished body with the add-in's name for it.
type NamedBody struct {
	Name string
	Body *decad.Body
}

// Named lists every body of the drive with its name: the discs, the housing, the cam, the output plate and the
// output pins.
func (d Drive) Named() []NamedBody {
	var out []NamedBody
	for i, disc := range d.Discs {
		out = append(out, NamedBody{DiscName(i), disc})
	}
	out = append(out, NamedBody{HousingName, d.Housing}, NamedBody{CamName, d.Cam}, NamedBody{OutputPlateName, d.OutputPlate})
	for i, pin := range d.OutputPins {
		out = append(out, NamedBody{OutputPinName(i), pin})
	}
	return out
}

// Build constructs the drive in doc. ChamferSize chamfers the rotor discs, the housing, the output plate and
// the pin ends.
//
// Differences from the Fusion add-in:
//   - A disc rim's chamfer is not exact. Decad cannot offset a rim made only of fit splines exactly, so it
//     cuts the bevel with a lofted cutter whose offset curves are fit splines through points of the exact
//     offset. The chamfered disc is held as a mesh, and its bevel matches the ideal one within 0.2 % of
//     ChamferSize.
//   - With DiscGap = 0 the two cam sections are separated by a 0.25 mm relief and joined by a bridge
//     cylinder on the drive axis, because decad does not decide a union of bodies that share a face.
//   - Each lobe is drawn as three fit splines, one per curvature sign, because decad certifies a spline
//     only when its curvature has one sign. A drive with many lobes is sampled coarser when decad's work
//     ceiling refuses the finest sampling.
//   - The casing contour is an exact per-direction envelope instead of the add-in's binned radius, which
//     is jagged. The housing is one prism cut by the contour cavity and a base hole, not a base ring
//     joined to patterned casing sectors, because decad does not decide unions of bodies that share a face.
//   - The disc's bore and output holes are inner loops of its one sketch, not separate cuts.
//   - The add-in's four sub-components and its message for skipped chamfers are Fusion-only. Decad has no
//     component tree, and the gallery reports an unsupported chamfer as an error.
//
// doc must be made with a chain tolerance:
//
//	doc := decad.New(decad.WithChainTolerance(units.Millimeters(0.1)))
func Build(ctx context.Context, doc *decad.Document, p Params) (Drive, error) {
	r, err := begin(ctx, doc, p)
	if err != nil {
		return Drive{}, err
	}
	var drive Drive
	for index := range p.DiscCount {
		disc, err := r.chamferedDisc(ctx, index)
		if err != nil {
			return Drive{}, err
		}
		drive.Discs = append(drive.Discs, disc)
	}
	if drive.Cam, err = r.buildCam(ctx); err != nil {
		return Drive{}, err
	}
	housing, err := r.buildHousing(ctx)
	if err != nil {
		return Drive{}, err
	}
	if drive.Housing, err = r.chamferedBy(ctx, housing, "housing",
		r.housingOuterCircles()); err != nil {
		return Drive{}, err
	}
	plate, pin, err := r.buildOutput(ctx)
	if err != nil {
		return Drive{}, err
	}
	if drive.OutputPlate, err = r.chamfered(ctx, plate, "output plate"); err != nil {
		return Drive{}, err
	}
	// One pin is chamfered and repeated, so the M pins come from one chamfer and one pattern.
	if pin, err = r.chamfered(ctx, pin, "output pin"); err != nil {
		return Drive{}, err
	}
	if drive.OutputPins, err = r.patternPins(ctx, pin); err != nil {
		return Drive{}, err
	}
	return drive, nil
}

// chamfered chamfers the outer loops of both end caps of body by Params.ChamferSize; it returns body unchanged
// when ChamferSize is zero.
func (r *run) chamfered(ctx context.Context, body *decad.Body, what string) (*decad.Body, error) {
	return r.chamferedBy(ctx, body, what,
		decad.Edges(decad.OuterLoopOf(decad.CapStart(body))).Or(decad.OuterLoopOf(decad.CapEnd(body))))
}

// chamferedBy chamfers the edges sel selects by Params.ChamferSize, or returns body unchanged when it is zero.
// The housing comes out of booleans, which leave no cap provenance, so it names its two outer circles instead.
func (r *run) chamferedBy(ctx context.Context, body *decad.Body, what string, sel decad.EdgeSelector) (*decad.Body, error) {
	if r.p.ChamferSize == 0 {
		return body, nil
	}
	size, err := r.doc.Param("chamfer_size")
	if err != nil {
		return nil, fmt.Errorf("cycloidal: read chamfer size: %w", err)
	}
	out, err := decad.Chamfer(ctx, body, sel, size)
	if err != nil {
		return nil, fmt.Errorf("cycloidal: chamfer %s: %w", what, err)
	}
	return out, nil
}

// housingOuterCircles selects the two circles where the housing's outer wall meets its end faces. The base's
// center hole has circular rims too, so the query keeps only circles longer than the circle between the
// housing's inner and outer diameters.
func (r *run) housingOuterCircles() decad.EdgeSelector {
	longer := math.Pi * (r.d.housingOuter + r.d.housingInner) / 2
	return decad.Edges(decad.Circular(), decad.LongerThan(units.Millimeters(longer))).Exactly(2)
}

// chamferedDisc builds rotor disc index and chamfers both outer rims of it.
func (r *run) chamferedDisc(ctx context.Context, index int) (*decad.Body, error) {
	disc, err := r.buildDisc(ctx, index)
	if err != nil {
		return nil, err
	}
	return r.chamfered(ctx, disc, fmt.Sprintf("disc %d", index+1))
}
