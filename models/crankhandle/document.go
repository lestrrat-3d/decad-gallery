package crankhandle

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
)

// chainTolerance is the coarsest mesh bound, in mm, that the Booleans may carry. The pin's cuts after its union refuse a
// plain document with a chain-depth error.
const chainTolerance = 0.01

// NewDocument returns a decad document whose Booleans admit the pin. Build and BuildParts need a document made with it,
// or with a chain tolerance of 0.005 mm or more; opts are applied after the chain tolerance.
func NewDocument(opts ...decad.DocumentOption) *decad.Document {
	return decad.New(append([]decad.DocumentOption{decad.WithChainTolerance(units.Millimeters(chainTolerance))}, opts...)...)
}
