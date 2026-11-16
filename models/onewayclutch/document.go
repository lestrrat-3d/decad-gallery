package onewayclutch

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
)

// chainTolerance is the coarsest held-mesh bound, in mm, that the ring's Booleans may touch. The mount nut pockets at
// the ears' rear ends meet held facets just past a plain document's gate: the third pocket's held bound is 0.001328 mm
// against the gate's 0.001324 mm. The ring's held bound is 0.0285 mm.
const chainTolerance = 0.02

// NewDocument returns a decad document whose Booleans admit the ring. Build and BuildParts need a document made with
// it, or with a chain tolerance of 0.02 mm or more; opts are applied after the chain tolerance.
func NewDocument(opts ...decad.DocumentOption) *decad.Document {
	return decad.New(append([]decad.DocumentOption{decad.WithChainTolerance(units.Millimeters(chainTolerance))}, opts...)...)
}
