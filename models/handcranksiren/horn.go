package handcranksiren

import (
	"math"

	"github.com/lestrrat-3d/decad"
)

// Horn dimensions in mm. The horn is a revolved wall whose bore flares at hornHalfAngle, with a flange that bolts to the drum.
const (
	hornHalfAngle = 15 * math.Pi / 180
	hornLength    = 100.0
	hornWall      = 2.5
	hornBack      = -31.90
	hornFlange    = -39.90
	hornEnd       = hornFlange - hornLength
	lipSetback    = 0.5
)

// horn builds the horn by revolving its meridian about R, then cuts the hole and the nut cavity of each horn bolt.
func (b *builder) horn() *decad.Body {
	mouth := 44 + hornLength*math.Tan(hornHalfAngle) // the bore's radius at the mouth
	sin, cos := math.Sin(hornHalfAngle), math.Cos(hornHalfAngle)
	s := b.meridian()
	b.fail("draw horn meridian", meridianPolygon(s, []point2{
		{44.5, hornBack}, {57.5, hornBack}, {58, -32.40}, {58, -39.40}, {57.5, hornFlange}, {46.5, hornFlange},
		{mouth + hornWall - lipSetback*sin, hornEnd + lipSetback*cos}, {mouth + 2.0, hornEnd}, {mouth + 0.5, hornEnd},
		{mouth - lipSetback*sin, hornEnd + lipSetback*cos}, {44, hornFlange}, {44, -32.40},
	}))
	body := b.revolveAboutR(s)
	for _, site := range sitesNamed("horn-bolt") {
		body = b.fastenerCuts(body, site, false, true, true)
	}
	return body
}
