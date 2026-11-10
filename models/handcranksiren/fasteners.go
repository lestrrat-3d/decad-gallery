package handcranksiren

import (
	"fmt"
	"math"
	"strings"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Fastener sites in mm. Every bolt is an m2bolt body and every nut an m2nut body, sized and cut by internal/m2fastener. A
// head sinks into a pocket of its part and ends flush with the part's face. A nut sits in a cavity its part encloses
// apart from the bolt hole, so the part retains it; it drops in during a print pause.
const (
	gearboxBoltLength   = 16
	gearboxCavity       = 11.75
	drumBoltLength      = 18
	drumCavity          = 17.0
	collectorBoltLength = 16
	collectorNutCavity  = 12.0
	throatBoltLength    = 10
	throatCavity        = 6.75
	gripBoltLength      = 20
	gripCavity          = 17.75
	pinBoltLength       = 10
	pinCavity           = 4.75

	frontFace         = 0.0  // gearbox-front's outer (front) face, where its bolt heads end
	rearFace          = 37.6 // gearbox-rear's outer face
	boltPitch         = 3    // drum posts and drum bolts are 120 degrees apart
	drumBoltR         = 30.0 // radius of the drum bolts and the posts about R
	collectorBoltR    = 53.5 // radius of the collector bolts about R
	throatBoltR       = 26.0 // radius of the throat bolts about the throat axis
	drumHead          = -15.50
	collectorBoltHead = -21.40
	gripHeadY         = -16.25 // the grip's window face, where its bolt heads end
	throatHead        = -46.0  // the throat bolt heads' X, flush with the collector wall's inner face
)

// The grip bolts' X and Z: four in a rectangle, X varying fastest.
var (
	gripBoltX = [2]float64{85, 99}
	gripBoltZ = [2]float64{30, 100}
)

// fastenerSite is one bolt and its captive nut, with the parts that hold them.
type fastenerSite struct {
	m2fastener.Site
	bolt, nut         string
	headHost, nutHost string
	motion            string
}

// fastenerSites returns every site at crank angle 0, in table order.
func fastenerSites() []fastenerSite {
	var sites []fastenerSite
	add := func(bolt, nut, headHost, nutHost, motion string, site m2fastener.Site) {
		sites = append(sites, fastenerSite{Site: site, bolt: bolt, nut: nut, headHost: headHost, nutHost: nutHost, motion: motion})
	}
	x, y, z := r3.Vec{X: 1}, r3.Vec{Y: 1}, r3.Vec{Z: 1}
	front := [4]r3.Vec{{X: -17.5, Y: frontFace, Z: 74}, {X: 103.5, Y: frontFace, Z: 74},
		{X: 30, Y: frontFace, Z: 127.5}, {X: 30, Y: frontFace, Z: 3.5}}
	rear := [4]r3.Vec{{X: -17.5, Y: rearFace, Z: 50}, {X: 103.5, Y: rearFace, Z: 50},
		{X: 75, Y: rearFace, Z: 127.5}, {X: 75, Y: rearFace, Z: 3.5}}
	corners := [4]r3.Vec{z, z, x, x}
	for i := range front {
		add(fmt.Sprintf("gearbox-front-bolt-%d", i+1), fmt.Sprintf("gearbox-front-nut-%d", i+1),
			"gearbox-front", "gearbox-frame", "fixed",
			m2fastener.Site{Head: front[i], Dir: y, Corner: corners[i], Length: gearboxBoltLength, Cavity: gearboxCavity})
	}
	for i := range rear {
		add(fmt.Sprintf("gearbox-rear-bolt-%d", i+1), fmt.Sprintf("gearbox-rear-nut-%d", i+1),
			"gearbox-rear", "gearbox-frame", "fixed",
			m2fastener.Site{Head: rear[i], Dir: y.Scale(-1), Corner: corners[i], Length: gearboxBoltLength, Cavity: gearboxCavity})
	}
	for k := range boltPitch {
		phi := 2 * math.Pi * float64(k) / boltPitch
		radial := r3.Vec{X: math.Cos(phi), Z: math.Sin(phi)}
		add(fmt.Sprintf("drum-bolt-%d", k+1), fmt.Sprintf("drum-nut-%d", k+1), "drum", "gearbox-front", "fixed",
			m2fastener.Site{Head: r3.Vec{Z: rotorHeight, Y: drumHead}.Add(radial.Scale(drumBoltR)), Dir: y, Corner: radial,
				Length: drumBoltLength, Cavity: drumCavity})
	}
	for k := range 4 {
		psi := math.Pi/4 + math.Pi/2*float64(k)
		radial := r3.Vec{X: math.Cos(psi), Z: math.Sin(psi)}
		add(fmt.Sprintf("collector-bolt-%d", k+1), fmt.Sprintf("collector-nut-%d", k+1), "drum", "collector", "fixed",
			m2fastener.Site{Head: r3.Vec{Z: rotorHeight, Y: collectorBoltHead}.Add(radial.Scale(collectorBoltR)), Dir: y.Scale(-1),
				Corner: r3.Vec{X: -math.Sin(psi), Z: math.Cos(psi)}, Length: collectorBoltLength, Cavity: collectorNutCavity})
	}
	for k := range 4 {
		a := math.Pi/4 + math.Pi/2*float64(k)
		add(fmt.Sprintf("throat-bolt-%d", k+1), fmt.Sprintf("throat-nut-%d", k+1), "collector", "horn", "fixed",
			m2fastener.Site{Head: r3.Vec{X: throatHead, Y: throatY + throatBoltR*math.Cos(a), Z: rotorHeight + throatBoltR*math.Sin(a)},
				Dir: x.Scale(-1), Corner: r3.Vec{Y: math.Cos(a), Z: math.Sin(a)}, Length: throatBoltLength, Cavity: throatCavity})
	}
	for k := range 4 {
		add(fmt.Sprintf("grip-bolt-%d", k+1), fmt.Sprintf("grip-nut-%d", k+1), "grip", "gearbox-front", "fixed",
			m2fastener.Site{Head: r3.Vec{X: gripBoltX[k%2], Y: gripHeadY, Z: gripBoltZ[k/2]}, Dir: y, Corner: x,
				Length: gripBoltLength, Cavity: gripCavity})
	}
	add("crank-pin-bolt", "crank-pin-nut", "crank-arm", "crank-pin", "crank",
		m2fastener.Site{Head: r3.Vec{X: crankOffset, Y: 38.6, Z: rotorHeight + 55}, Dir: y, Corner: x,
			Length: pinBoltLength, Cavity: pinCavity})
	return sites
}

// sitesNamed returns the sites whose bolt name starts with prefix, in table order.
func sitesNamed(prefix string) []fastenerSite {
	var out []fastenerSite
	for _, s := range fastenerSites() {
		if strings.HasPrefix(s.bolt, prefix) {
			out = append(out, s)
		}
	}
	return out
}

// hardware adds s's bolt and then its nut to the assembly, each posed by the site's motion.
func (b *builder) hardware(s fastenerSite) {
	b.part(s.bolt, BoltsContainerName, s.motion, func() *decad.Body { return b.placeBolt(s) })
	b.part(s.nut, NutsContainerName, s.motion, func() *decad.Body { return b.placeNut(s) })
}

// placeBolt builds an m2bolt body whose head's top face is at s.Head and whose shank runs along s.Dir.
func (b *builder) placeBolt(s fastenerSite) *decad.Body {
	if b.err != nil {
		return nil
	}
	m, err := m2fastener.PlaceBolt(b.ctx, b.doc, s.Site)
	b.fail("build bolt", err)
	if err != nil {
		return nil
	}
	b.a.Sketches = append(b.a.Sketches, m.Profile, m.Socket)
	return m.Body
}

// placeNut builds an m2nut body in the site's cavity.
func (b *builder) placeNut(s fastenerSite) *decad.Body {
	if b.err != nil {
		return nil
	}
	m, err := m2fastener.PlaceNut(b.ctx, b.doc, s.Site)
	b.fail("build nut", err)
	if err != nil {
		return nil
	}
	b.a.Sketches = append(b.a.Sketches, m.Hexagon, m.LowerChamfer, m.UpperChamfer)
	return m.Body
}

// tool records a fastener tool's sketch and returns its body.
func (b *builder) tool(body *decad.Body, sk *sketch.Sketch, err error) *decad.Body {
	b.fail("build fastener tool", err)
	if err != nil {
		return nil
	}
	b.a.Sketches = append(b.a.Sketches, sk)
	return body
}

// fastenerCuts cuts from host the features whose flag is true: the head pocket, the clearance hole and the nut cavity of
// s. The caller passes the flags for the body it holds; the flags alone decide which cuts happen.
func (b *builder) fastenerCuts(host *decad.Body, s fastenerSite, pocket, hole, cavity bool) *decad.Body {
	if pocket && b.err == nil {
		tool, sk, err := m2fastener.PocketTool(b.ctx, b.doc, s.Site)
		host = b.cut(host, b.tool(tool, sk, err))
	}
	if hole && b.err == nil {
		tool, sk, err := m2fastener.HoleTool(b.ctx, b.doc, s.Site)
		host = b.cut(host, b.tool(tool, sk, err))
	}
	if cavity && b.err == nil {
		tool, sk, err := m2fastener.CavityTool(b.ctx, b.doc, s.Site)
		host = b.cut(host, b.tool(tool, sk, err))
	}
	return host
}
