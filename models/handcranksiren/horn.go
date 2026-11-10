package handcranksiren

import (
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

// Horn dimensions in mm. The horn is a revolved wall whose bore flares at hornHalfAngle, with a flange that bolts to the
// collector's -X wall. Its axis runs along -X through (throatY, 84).
const (
	hornHalfAngle = 15 * math.Pi / 180
	hornLength    = 100.0
	hornWall      = 2.5
	lipSetback    = 0.5
	hornFlangeV   = 9.25 // the flange's far face and the cone's start, measured from the meridian origin along -X
	hornOriginX   = -50.0
)

// horn builds the horn by revolving its meridian about its axis, then cuts the hole and the nut cavity of each throat
// bolt. The meridian's U is the radius (+Y) and V runs along -X from O = (-50, throatY, 84), so its normal is +Z.
func (b *builder) horn() *decad.Body {
	origin := r3.Vec{X: hornOriginX, Y: throatY, Z: rotorHeight}
	end := hornFlangeV + hornLength
	mouth := throatRadius + hornLength*math.Tan(hornHalfAngle) // the bore's radius at the mouth
	sin, cos := math.Sin(hornHalfAngle), math.Cos(hornHalfAngle)
	s := b.section(frame(origin, r3.Vec{Y: 1}, r3.Vec{X: -1}))
	b.fail("draw horn meridian", meridianPolygon(s, []point2{
		{22.5, 1.25}, {29.5, 1.25}, {30, 1.75}, {30, 8.75}, {29.5, hornFlangeV}, {24.5, hornFlangeV},
		{mouth + hornWall - lipSetback*sin, end - lipSetback*cos}, {mouth + 2.0, end}, {mouth + 0.5, end},
		{mouth - lipSetback*sin, end - lipSetback*cos}, {throatRadius, hornFlangeV}, {throatRadius, 1.75},
	}))
	body := b.revolve(s, decad.ConstructionAxis{Origin: origin, Dir: r3.Vec{X: -1}})
	for _, site := range sitesNamed("throat-bolt") {
		body = b.fastenerCuts(body, site, false, true, true)
	}
	return body
}
