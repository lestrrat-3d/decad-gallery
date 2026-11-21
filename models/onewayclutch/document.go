package onewayclutch

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
)

// chainTolerance is the coarsest held-mesh bound, in mm, that the ring's Booleans may touch.
const chainTolerance = 0.02

// NewDocument returns a decad document whose Booleans admit the ring. Build and BuildParts need a document made with
// it, or with a chain tolerance of 0.02 mm or more; opts are applied after the chain tolerance.
func NewDocument(opts ...decad.DocumentOption) *decad.Document {
	return decad.New(append([]decad.DocumentOption{decad.WithChainTolerance(units.Millimeters(chainTolerance))}, opts...)...)
}
