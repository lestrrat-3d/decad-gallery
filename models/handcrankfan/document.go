package handcrankfan

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
)

// chainTolerance is the coarsest mesh bound, in mm, that the Booleans may carry. The lofted blades hold their mesh to
// 0.0154 mm and the bevel gears to 0.09 mm, which a plain document refuses.
const chainTolerance = 0.1

// NewDocument returns a decad document whose Booleans admit the lofted blades. Build needs a document made with it;
// opts are applied after the chain tolerance.
func NewDocument(opts ...decad.DocumentOption) *decad.Document {
	return decad.New(append([]decad.DocumentOption{decad.WithChainTolerance(units.Millimeters(chainTolerance))}, opts...)...)
}
