package handcranksiren

import (
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Siren head dimensions in mm. The drum, rotor, stator and collector are coaxial with R. The horn's axis runs along -X.
const (
	drumBackFace   = -11.50 // the back plate's outer face
	drumPlateFace  = -15.50 // the back plate's inner face, where the drum bolt heads end
	drumFlangeBack = -21.40
	windowRIn      = 14.0
	windowROut     = 24.0
	windowHalf     = 40 * math.Pi / 180
	windowAngle    = math.Pi / 3 // the first window is centred at 60 degrees
	windowY        = -16.50
	windowLength   = 6.0
	postRadius     = 5.0
	postY          = -12.50
	postLength     = 12.25
	holeChamfered  = shaftHoleRadius + 0.5 // the shaft hole's rim, after its 0.5 meridian chamfers
	windows        = 3
	posts          = 3
)

// meridian starts a sketch on the plane z = 84 through R whose U is the radius (+X) and V is Y (+Y), so a full revolve
// about R builds a body of revolution.
func (b *builder) meridian() *sketch.Sketch {
	return b.section(frame(r3.Vec{Z: rotorHeight}, r3.Vec{X: 1}, r3.Vec{Y: 1}))
}

// revolveAboutR turns the one valid profile of s a full turn about the rotor axis.
func (b *builder) revolveAboutR(s *sketch.Sketch) *decad.Body {
	return b.revolve(s, decad.ConstructionAxis{Origin: r3.Vec{Z: rotorHeight}, Dir: r3.Vec{Y: 1}})
}

// ring returns body patterned count times about R. It consumes body.
func (b *builder) ring(body *decad.Body, count int) *decad.Body {
	if b.err != nil || body == nil {
		return nil
	}
	out, err := decad.Patterned(b.ctx, body, decad.CircularPattern{
		Center: r3.Vec{Z: rotorHeight}, Axis: r3.Vec{Y: 1}, Count: count})
	b.fail("pattern about the rotor axis", err)
	return out
}

// drum builds the drum: a revolved cup, three windows through its back plate, three posts that reach the gearbox, and the
// pocket and hole of each drum bolt and each collector bolt.
func (b *builder) drum() *decad.Body {
	s := b.meridian()
	b.fail("draw drum meridian", meridianPolygon(s, []point2{
		{holeChamfered, drumBackFace}, {48.5, drumBackFace}, {49, -12.00}, {49, drumFlangeBack}, {57.5, drumFlangeBack},
		{58, -21.90}, {58, -27.90}, {57.5, -28.40}, {46.5, -28.40}, {46, -27.90}, {46, drumPlateFace},
		{holeChamfered, drumPlateFace}, {shaftHoleRadius, drumPlateFace + 0.5}, {shaftHoleRadius, drumBackFace - 0.5},
	}))
	body := b.revolveAboutR(s)

	// The windows are cut through the back plate, one tool patterned about R.
	f := frame(r3.Vec{Z: rotorHeight, Y: windowY},
		r3.Vec{X: math.Cos(windowAngle), Z: math.Sin(windowAngle)}, r3.Vec{Y: 1}.Cross(r3.Vec{X: math.Cos(windowAngle), Z: math.Sin(windowAngle)}))
	ws := b.section(f)
	port(ws, windowRIn, windowROut, windowHalf)
	window := b.ring(b.extrude(ws, 0, windowLength), windows)
	body = b.cut(body, window)
	body = b.finishInner(body, drumBackFace, drumPlateFace)

	// The posts start inside the back plate so the union has no shared face.
	post := b.cylinder(frame(r3.Vec{X: drumBoltR, Y: postY, Z: rotorHeight}, r3.Vec{X: 1}, r3.Vec{Z: -1}), postRadius, postLength)
	body = b.union(body, b.ring(post, posts))

	for _, site := range sitesNamed("drum-bolt") {
		body = b.fastenerCuts(body, site, true, true, false)
	}
	for _, site := range sitesNamed("collector-bolt") {
		body = b.fastenerCuts(body, site, true, true, false)
	}
	return body
}

// windowLoops keeps the long hole loops that are not a single circle: the windows. The shaft hole's rim is chamfered in
// the meridian, because the posts' union is refused once the cut shaft-hole rim also carries a chamfer.
func windowLoops(loop *decad.Loop, edges []*decad.Edge) bool {
	return len(edges) > 1 && edgefinish.Long(edgefinish.Inner)(loop, edges)
}

// finishInner chamfers the window loops of the large faces at the Y offsets.
func (b *builder) finishInner(body *decad.Body, offsets ...float64) *decad.Body {
	if b.err != nil || body == nil {
		return nil
	}
	out, _, err := edgefinish.Chamfer(b.ctx, body, r3.Vec{Y: 1}, offsets, windowLoops, edgefinish.MinChamfer)
	b.fail("finish edges", err)
	return out
}
