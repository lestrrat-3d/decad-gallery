package handcrankfan

import (
	"fmt"
	"math"
	"strings"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Fastener sites. Every bolt is an m2bolt body and every nut an m2nut body, sized and cut by internal/m2fastener. A head
// sinks into a pocket of its part and ends flush with the part's face. A nut sits in a cavity its part encloses, apart
// from the bolt hole, so the part retains it; it drops in during a print pause.

// fastenerSite is one bolt and its captive nut, with the parts that hold them. The site's Head is on headHost's face.
type fastenerSite struct {
	m2fastener.Site
	bolt, nut         string
	headHost, nutHost string
	motion            string
}

// fastenerSites returns every site at crank angle 0. The blade site is blade 1's; the other blades' come from the
// pattern about the fan axis.
func fastenerSites() []fastenerSite {
	sites := []fastenerSite{{
		bolt: "fan-bolt-1", nut: "fan-nut-1", headHost: "fan-hub", nutHost: "fan-hub", motion: "fan",
		Site: m2fastener.Site{
			Head: r3.Vec{X: boltAt, Y: -(hubBack + hubLength), Z: axisHeight}, Dir: r3.Vec{Y: 1}, Corner: r3.Vec{X: 1},
			Length: 16, Cavity: 15.6,
		},
	}}
	for i, e := range gearboxEars {
		c := e.centre()
		sites = append(sites, fastenerSite{
			bolt: fmt.Sprintf("gearbox-bolt-%d", i+1), nut: fmt.Sprintf("gearbox-nut-%d", i+1),
			headHost: e.head, nutHost: e.nut, motion: "fixed",
			Site: m2fastener.Site{
				Head: c.Sub(e.dir.Scale(earHalf)), Dir: e.dir, Corner: r3.Vec{Z: 1}, Length: 10, Cavity: 9.6,
			},
		})
	}
	for i, w := range wallScrews {
		sites = append(sites, fastenerSite{
			bolt: fmt.Sprintf("gearbox-screw-%d", i+1), nut: fmt.Sprintf("gearbox-screw-nut-%d", i+1),
			headHost: w.head, nutHost: w.recv, motion: "fixed",
			Site: m2fastener.Site{Head: w.at(w.face), Dir: w.dir, Corner: r3.Vec{Z: 1}, Length: 16, Cavity: 15.6},
		})
	}
	for i, dx := range []float64{-gripScrewX, gripScrewX} {
		sites = append(sites, fastenerSite{
			bolt: fmt.Sprintf("handle-bolt-%d", i+1), nut: fmt.Sprintf("handle-nut-%d", i+1),
			headHost: "gearbox-base", nutHost: "handle", motion: "fixed",
			Site: m2fastener.Site{
				Head: r3.Vec{X: gripX + dx, Y: gripY, Z: cavityFloor}, Dir: r3.Vec{Z: -1}, Corner: r3.Vec{Y: 1},
				Length: 10, Cavity: 9.1,
			},
		})
	}
	return append(sites, fastenerSite{
		bolt: "crank-pin-bolt", nut: "crank-pin-nut", headHost: "crank-arm", nutHost: "crank-pin", motion: "crank",
		Site: m2fastener.Site{
			Head: r3.Vec{X: armStart, Z: crankHeight + crankThrow}, Dir: r3.Vec{X: 1}, Corner: r3.Vec{Y: 1},
			Length: 10, Cavity: 4.55,
		},
	})
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
	b.part(s.bolt, "hardware", s.motion, func() *decad.Body { return b.placeBolt(s) })
	b.part(s.nut, "hardware", s.motion, func() *decad.Body { return b.placeNut(s) })
}

// bladeSites returns the blade sites of an assembly with n blades, blade 1 first, at crank angle 0.
func bladeSites(n int) ([]fastenerSite, error) {
	first := fastenerSites()[0]
	sites := []fastenerSite{first}
	for k := 1; k < n; k++ {
		turned, err := first.Turned(r3.Vec{Z: axisHeight}, fanAxis, 2*math.Pi*float64(k)/float64(n))
		if err != nil {
			return nil, fmt.Errorf("turn blade %d's fastener: %w", k+1, err)
		}
		s := first
		s.Site = turned
		s.bolt, s.nut = fmt.Sprintf("fan-bolt-%d", k+1), fmt.Sprintf("fan-nut-%d", k+1)
		sites = append(sites, s)
	}
	return sites, nil
}

// fastenerSpec is the bought part a bolt or a nut is, which the bill of materials counts. It returns "" for any other
// name. Blade sites match by prefix, because their number depends on the parameters.
func fastenerSpec(name string) string {
	switch {
	case strings.HasPrefix(name, "fan-nut-"):
		return m2fastener.NutSpec
	case strings.HasPrefix(name, "fan-bolt-"):
		return m2fastener.BoltSpec(fastenerSites()[0].Length)
	}
	for _, s := range fastenerSites() {
		switch name {
		case s.bolt:
			return m2fastener.BoltSpec(s.Length)
		case s.nut:
			return m2fastener.NutSpec
		}
	}
	return ""
}

// placeBolt builds an m2bolt body whose head's top face is at s.Head and whose shank runs along s.Dir, and appends its
// sketches to the assembly.
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

// placeNut builds an m2nut body in the site's cavity and appends its sketches to the assembly.
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
