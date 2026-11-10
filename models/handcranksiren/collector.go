package handcranksiren

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/r3"
)

// Collector dimensions in mm. The collector is a rounded-square box coaxial with R. The ports blow into it through its
// rear wall, and it turns the air 90 degrees out through a round throat in its -X wall, where the horn bolts on.
const (
	collectorBack         = -31.90 // rear face, on the stator
	collectorCeiling      = -39.90 // the rear wall's inner face
	collectorFloor        = -91.90 // the end wall's inner face
	collectorEnd          = -96.90 // end face
	collectorHalf         = 51.0   // outer half-width in X and in Z about R
	collectorCorner       = 10.0
	collectorCavityHalf   = 46.0
	collectorCavityCorner = 7.0
	collectorBore         = 44.0 // radius of the air opening through the rear wall
	throatY               = -65.90
	throatRadius          = 22.0
	throatWallOut         = -51.0 // the -X wall's outer face
	throatWallIn          = -46.0 // the -X wall's inner face
	throatToolFrom        = -45.0 // the throat tool starts here, inside the cavity, and runs 7 toward -X
)

// corners returns four equal corner radii.
func corners(r float64) []float64 { return []float64{r, r, r, r} }

// chamferAlong chamfers by size the kept loops of the large faces whose normal lies along normal at the offsets.
func (b *builder) chamferAlong(body *decad.Body, normal r3.Vec, keep edgefinish.LoopFilter, size float64,
	offsets ...float64) *decad.Body {
	if b.err != nil || body == nil {
		return nil
	}
	out, _, err := edgefinish.Chamfer(b.ctx, body, normal, offsets, keep, size)
	b.fail("finish edges", err)
	return out
}

// collector builds the box with its bore and cavity, finishes its edges, cuts the throat in the -X wall, and cuts the
// hole and nut cavity of each collector bolt and the pocket and hole of each throat bolt.
func (b *builder) collector() *decad.Body {
	rotor, _, _ := b.axes()
	body := b.roundedPrism(rotor.across(collectorEnd),
		outline(-collectorHalf, collectorHalf, rotorHeight-collectorHalf, rotorHeight+collectorHalf),
		corners(collectorCorner), collectorBack-collectorEnd)
	body = b.cut(body, b.cylinder(rotor.across(collectorCeiling-1), collectorBore, 10))
	// The bore tool reaches 1 into the cavity, so the cavity cut joins a void that exists.
	body = b.cut(body, b.roundedPrism(rotor.across(collectorFloor),
		outline(-collectorCavityHalf, collectorCavityHalf, rotorHeight-collectorCavityHalf, rotorHeight+collectorCavityHalf),
		corners(collectorCavityCorner), collectorCeiling-collectorFloor))
	body = b.finish(body, collectorBack, collectorEnd)
	body = b.chamferAlong(body, r3.Vec{Y: 1}, edgefinish.Long(edgefinish.Inner), edgefinish.MinChamfer, collectorCeiling)
	// The throat's frame has U = +Z and V = +Y, so its normal points to -X.
	body = b.cut(body, b.cylinder(frame(r3.Vec{X: throatToolFrom, Y: throatY, Z: rotorHeight}, r3.Vec{Z: 1}, r3.Vec{Y: 1}),
		throatRadius, 7))
	body = b.chamferAlong(body, r3.Vec{X: 1}, edgefinish.Long(edgefinish.Inner), edgefinish.MinChamfer, throatWallOut, throatWallIn)
	for _, site := range sitesNamed("collector-bolt") {
		body = b.fastenerCuts(body, site, false, true, true)
	}
	for _, site := range sitesNamed("throat-bolt") {
		body = b.fastenerCuts(body, site, true, true, false)
	}
	return body
}
