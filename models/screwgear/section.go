package screwgear

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// PairSection is the two ribbons cut by one plane parallel to the mid-plane between their axes.
type PairSection struct {
	// Plane is the cutting plane. Its U axis is ê, its V axis is k̂ and its normal is n̂, the common
	// perpendicular of the two axes, so plane coordinates read the same for every placement.
	Plane r3.Frame
	// Height is the plane's distance from the mid-plane along n̂, in millimeters.
	Height       float64
	GearA, GearB decad.SectionOutline
	// Clearance is the least distance between the two outlines, with the closest pair of points and
	// whether the outlines overlap. It is nil when either ribbon misses the plane.
	Clearance *decad.SectionClearance
}

// MidPlane returns the plane Height millimeters from the mid-plane between the gear axes along n̂.
// A positive Height moves toward gear B. The mid-plane holds the centre C and is spanned by ê and k̂.
func (m *Mechanism) MidPlane(height float64) (r3.Frame, error) {
	plane, err := r3.NewFrame(m.pair.Origin().Add(m.pair.N().Scale(height)), m.pair.U(), m.pair.V())
	if err != nil {
		return r3.Frame{}, fmt.Errorf("screwgear: mid-plane frame: %w", err)
	}
	return plane, nil
}

// SectionPair cuts both ribbons with the plane MidPlane(height) and returns their outlines, each
// read from the body's verified mesh at chord tolerance tol (millimeters). The ribbons mesh where
// the engagement overlaps their crests, 1.15 mm deep at the defaults, so heights of a few tenths of
// a millimeter either side of zero show the teeth interleaving. ctx and m must not be nil.
func (m *Mechanism) SectionPair(ctx context.Context, height, tol float64) (PairSection, error) {
	plane, err := m.MidPlane(height)
	if err != nil {
		return PairSection{}, err
	}
	out := PairSection{Plane: plane, Height: height}
	for i, dst := range []*decad.SectionOutline{&out.GearA, &out.GearB} {
		body := []*decad.Body{m.GearA, m.GearB}[i]
		outline, err := body.SectionOutline(ctx, plane, units.Millimeters(tol))
		if err != nil {
			return PairSection{}, fmt.Errorf("screwgear: section gear %c: %w", 'A'+i, err)
		}
		*dst = outline
	}
	if len(out.GearA.Loops) == 0 || len(out.GearB.Loops) == 0 {
		return out, nil
	}
	clearance, err := out.GearA.Clearance(out.GearB)
	if err != nil {
		return PairSection{}, fmt.Errorf("screwgear: section clearance: %w", err)
	}
	out.Clearance = &clearance
	return out, nil
}
