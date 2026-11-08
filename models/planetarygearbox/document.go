package planetarygearbox

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
)

// chainTolerance is the coarsest mesh bound, in mm, that the ring's Booleans may carry. The ring takes its nut
// cavities and its tooth spaces and then has its end faces chamfered round the mounting ears, and the chamfer meets
// tooth-space facets held to about 0.0034 mm, which a plain document refuses.
const chainTolerance = 0.1

// NewDocument returns a decad document that admits the ring's chain of cuts and chamfers. Build needs a document
// made with it when the gearbox has mounting ears; opts are applied after the chain tolerance.
func NewDocument(opts ...decad.DocumentOption) *decad.Document {
	return decad.New(append([]decad.DocumentOption{decad.WithChainTolerance(units.Millimeters(chainTolerance))}, opts...)...)
}
