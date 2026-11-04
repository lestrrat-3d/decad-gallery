package spinningwheel

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
)

// chainTolerance is the coarsest bound decad may carry where a Boolean meets a face an earlier
// Boolean left held, as in the yarn guide and leader, the brake springs and the takeup screw.
// Every bound the parts carry stays proven; Bound grows to cover what was admitted.
const chainTolerance = 0.1

// NewDocument returns a decad document that admits the wheel's chains of Booleans over held
// meshes. Build, BuildPreview and BuildConnector need a document made with this tolerance;
// opts are applied after it.
func NewDocument(opts ...decad.DocumentOption) *decad.Document {
	return decad.New(append([]decad.DocumentOption{decad.WithChainTolerance(units.Millimeters(chainTolerance))}, opts...)...)
}
