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

// Build constructs the drive in doc. ChamferSize chamfers the housing, the output plate and the pin ends, but
// not the rotor discs: the outer loop of a disc cap is made only of fit splines, and decad cannot offset a
// chain of free-form edges exactly, so it refuses that chamfer.
func Build(ctx context.Context, doc *decad.Document, p Params) (Drive, error) {
	r, err := begin(ctx, doc, p)
	if err != nil {
		return Drive{}, err
	}
	var drive Drive
	for index := range p.DiscCount {
		disc, err := r.buildDisc(ctx, index)
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
	plate, pins, err := r.buildOutput(ctx)
	if err != nil {
		return Drive{}, err
	}
	if drive.OutputPlate, err = r.chamfered(ctx, plate, "output plate"); err != nil {
		return Drive{}, err
	}
	for i := range pins {
		if pins[i], err = r.chamfered(ctx, pins[i], "output pin"); err != nil {
			return Drive{}, err
		}
	}
	drive.OutputPins = pins
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
