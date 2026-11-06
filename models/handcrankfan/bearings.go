package handcrankfan

import (
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// Shaft, bearing and spline dimensions in mm. Every shaft is 7.8 across, which is an 8 mm shaft with the model's usual 0.1
// clearance, so it turns in a 608-2RS bearing (8 bore, 22 outside diameter, 7 wide). Each bearing sits in a round seat
// 0.1 larger than its outside, 7.4 deep, behind a relief bore that frees the inner ring. Three grooves along the shaft
// take three teeth in a gear's bore, so six faces carry the torque and the round shaft still slides through every bore.
const (
	shaftRadius  = 3.9
	seatRadius   = 11.1
	seatDepth    = 7.4
	reliefRadius = 9.5
	reliefDepth  = 2.1
	shaftHole    = 4.2 // the plain hole between two bearings

	splineBore = 4.0 // a spline bore is 0.1 larger than the shaft
	grooveFlor = 2.9 // radius of a groove's floor
	grooveHalf = 1.0 // half the width of a groove
	toothTip   = 3.0 // radius of a tooth's tip, 0.1 above the groove floor
	toothHalf  = 0.9 // half the width of a tooth, 0.1 less than the groove's
)

// shaftAxis is one shaft's line. at(t) is the point t along it, and e1 and e2 span the plane across it.
type shaftAxis struct{ point, dir, e1, e2 r3.Vec }

func (a shaftAxis) at(t float64) r3.Vec { return a.point.Add(a.dir.Scale(t)) }

// The crank shaft and the drive shaft are measured by X, the fan shaft by the distance behind the apex (-Y).
var (
	crankSpline = shaftAxis{r3.Vec{Z: crankHeight}, r3.Vec{X: 1}, r3.Vec{Y: 1}, r3.Vec{Z: 1}}
	driveSpline = shaftAxis{r3.Vec{Z: axisHeight}, r3.Vec{X: 1}, r3.Vec{Y: 1}, r3.Vec{Z: 1}}
	fanSpline   = shaftAxis{r3.Vec{Z: axisHeight}, r3.Vec{Y: -1}, r3.Vec{X: 1}, r3.Vec{Z: 1}}
)

// spokeFrame is the plane at t whose U axis points out from the shaft's axis at three equal steps from the first spoke
// (angle k * 120 degrees plus 90) and whose normal is the shaft's direction.
func (a shaftAxis) spokeFrame(t float64, k int) r3.Frame {
	phi := math.Pi/2 + 2*math.Pi*float64(k)/3
	u := a.e1.Scale(math.Cos(phi)).Add(a.e2.Scale(math.Sin(phi)))
	return frame(a.at(t), u, a.dir.Cross(u))
}

// grooves cuts the three grooves into shaft from t = from to t = to, which must run past the shaft's end or sit inside it.
func (b *builder) grooves(shaft *decad.Body, a shaftAxis, from, to float64) *decad.Body {
	for k := range 3 {
		shaft = b.cut(shaft, b.panel(a.spokeFrame(from, k), grooveFlor, -grooveHalf, shaftRadius+1.5, grooveHalf, to-from))
	}
	return shaft
}

// splineCut cuts a spline bore, a round bore with three teeth, from host between t = from and t = from + length.
func (b *builder) splineCut(host *decad.Body, a shaftAxis, from, length float64) *decad.Body {
	tool := b.cylinder(a.spokeFrame(from, 0), splineBore, length)
	for k := range 3 {
		tool = b.cut(tool, b.panel(a.spokeFrame(from-1, k), toothTip, -toothHalf, splineBore+1, toothHalf, length+2))
	}
	return b.cut(host, tool)
}

// hole cuts a round hole of radius r about the shaft's axis from host between t0 and t1.
func (b *builder) hole(host *decad.Body, a shaftAxis, r, t0, t1 float64) *decad.Body {
	return b.cut(host, b.cylinder(a.spokeFrame(t0, 0), r, t1-t0))
}

// seat cuts a bearing seat into host: the round pocket between pocket0 and pocket1 and a relief bore between relief0
// and relief1 that frees the inner ring where the pocket's floor is.
func (b *builder) seat(host *decad.Body, a shaftAxis, pocket0, pocket1, relief0, relief1 float64) *decad.Body {
	host = b.hole(host, a, seatRadius, pocket0, pocket1)
	return b.hole(host, a, reliefRadius, relief0, relief1)
}

// funnelSeat cuts a bearing seat whose relief is a 45° cone that narrows toward smaller t, from the seat's floor at
// floor to the plain hole. The seat opens toward larger t, at mouth. Printed with the mouth down, the pocket's walls
// are vertical, the ledge for the bearing is 1.6 wide and the cone is no flatter than 45°, so the seat needs no
// support.
func (b *builder) funnelSeat(host *decad.Body, a shaftAxis, floor, mouth float64) *decad.Body {
	host = b.hole(host, a, seatRadius, floor, mouth)
	cone := b.cylinder(a.spokeFrame(floor-(reliefRadius-shaftHole)-0.1, 0), reliefRadius, reliefRadius-shaftHole+0.2)
	if b.err != nil || cone == nil {
		return nil
	}
	chamfered, err := decad.Chamfer(b.ctx, cone, decad.Edges(decad.OuterLoopOf(decad.CapStart(cone))),
		units.Millimeters(reliefRadius-shaftHole))
	b.fail("chamfer relief cone", err)
	return b.cut(host, chamfered)
}

// bearing builds the 608-2RS envelope that sits on the shaft from t to t + 7.
func (b *builder) bearing(a shaftAxis, t float64) *decad.Body {
	if b.err != nil {
		return nil
	}
	f := a.spokeFrame(t, 0)
	m, err := ballbearing.Build(b.ctx, b.doc, ballbearing.Params{
		BoreDiameter: 8, OuterDiameter: 22, Width: 7, Placement: &f})
	b.fail("build bearing", err)
	if err != nil {
		return nil
	}
	b.a.Sketches = append(b.a.Sketches, m.Sketch)
	return m.Body
}

// bearingSite is one bearing: its part name, the shaft it carries and where it starts along that shaft.
type bearingSite struct {
	name string
	axis shaftAxis
	at   float64
}

// bearingSites are the seven bearings: one in the post for each of the crank and drive shafts, one in the crank-side
// wall for each of them, one in the crank block, and two for the fan shaft.
var bearingSites = [...]bearingSite{
	{"bearing-crank-post", crankSpline, boxLeftX + 0.3},
	{"bearing-drive-post", driveSpline, boxLeftX + 0.3},
	{"bearing-crank-wall", crankSpline, crankWallX + 0.3},
	{"bearing-drive-wall", driveSpline, crankWallX + 0.3},
	{"bearing-crank-block", crankSpline, crankFace - 7.3},
	{"bearing-fan-wall", fanSpline, -fanWallY + 0.3},
	{"bearing-fan-block", fanSpline, fanBlockFace - 7.3},
}

// bearings builds the purchased bearings.
func (b *builder) bearings() {
	for _, site := range bearingSites {
		b.part(site.name, "bearing", "fixed", func() *decad.Body { return b.bearing(site.axis, site.at) })
	}
}
