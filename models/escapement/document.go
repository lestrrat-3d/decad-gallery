package escapement

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
)

// chainTolerance is the coarsest mesh bound, in mm, that the movement's chains of Booleans may carry. The plates take
// many cuts and then a chamfer, which a plain document refuses.
const chainTolerance = 0.1

// NewDocument returns a decad document that admits the movement's chains of Booleans and chamfers. Build needs a
// document made with it; opts are applied after the chain tolerance.
func NewDocument(opts ...decad.DocumentOption) *decad.Document {
	return decad.New(append([]decad.DocumentOption{decad.WithChainTolerance(units.Millimeters(chainTolerance))}, opts...)...)
}
