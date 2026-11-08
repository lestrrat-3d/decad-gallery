package screwgear

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// Mechanism is the three bodies Build adds to a document.
type Mechanism struct {
	GearA, GearB, Cage *decad.Body

	// pair is the placement: its origin is the centre C, U is ê, V is k̂ and the normal is n̂.
	pair r3.Frame
}

// Container names for the bodies Build adds, one per kind.
const (
	gearsContainerName  = "Gears"
	sleeveContainerName = "Sleeve"
)

// chainTolerance is the coarsest bound decad may carry where a bore or window cut meets a
// face an earlier cut left held to 0.052 mm. Six chained cuts on one curved tube need it, and
// every bound the cage carries stays proven.
const chainTolerance = 0.1

// NewDocument returns a decad document that admits the sleeve's chain of six cuts. Build needs a
// document made with this tolerance; opts are applied after it.
func NewDocument(opts ...decad.DocumentOption) *decad.Document {
	return decad.New(append([]decad.DocumentOption{decad.WithChainTolerance(units.Millimeters(chainTolerance))}, opts...)...)
}

// Build adds the two gears and the sleeve to doc and returns them. ctx and doc must not be nil.
// The mechanism is built in the pair frame, where the centre is the origin, ê is X, k̂ is Y and
// n̂ is Z, because booleans fail on bodies built at an offset; every body is moved onto
// Params.Placement after the last boolean. A nil Placement leaves the mechanism in the world frame.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Mechanism, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("screwgear: context and document must not be nil")
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	pf, err := newPairFrame(p)
	if err != nil {
		return nil, err
	}
	m := Mechanism{pair: pf.frame}
	for i, dst := range []**decad.Body{&m.GearA, &m.GearB} {
		body, err := pf.gears[i].ribbonBody(ctx, doc)
		if err != nil {
			return nil, fmt.Errorf("screwgear: gear %c: %w", 'A'+i, err)
		}
		*dst = body
	}
	if m.Cage, err = buildSleeve(ctx, doc, newSleeve(pf)); err != nil {
		return nil, fmt.Errorf("screwgear: sleeve: %w", err)
	}
	gears, err := doc.Root().ChildNamed(gearsContainerName)
	if err != nil {
		return nil, fmt.Errorf("screwgear: %w", err)
	}
	sleeve, err := doc.Root().ChildNamed(sleeveContainerName)
	if err != nil {
		return nil, fmt.Errorf("screwgear: %w", err)
	}
	for _, part := range []struct {
		body **decad.Body
		in   *decad.Container
	}{{&m.GearA, gears}, {&m.GearB, gears}, {&m.Cage, sleeve}} {
		placed, err := placement.Place(ctx, *part.body, &pf.frame)
		if err != nil {
			return nil, fmt.Errorf("screwgear: %w", err)
		}
		if err := part.in.AddBody(placed); err != nil {
			return nil, fmt.Errorf("screwgear: place body in container %q: %w", part.in.Name(), err)
		}
		*part.body = placed
	}
	return &m, nil
}
