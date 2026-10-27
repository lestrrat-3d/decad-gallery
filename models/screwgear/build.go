package screwgear

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/units"
)

// Mechanism is the three bodies Build adds to a document.
type Mechanism struct {
	GearA, GearB, Cage *decad.Body
}

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
	var m Mechanism
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
	for _, body := range []**decad.Body{&m.GearA, &m.GearB, &m.Cage} {
		placed, err := placement.Place(ctx, *body, &pf.frame)
		if err != nil {
			return nil, fmt.Errorf("screwgear: %w", err)
		}
		*body = placed
	}
	return &m, nil
}
