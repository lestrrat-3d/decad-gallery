package handcrankfan

import (
	"fmt"
	"math"
	"strings"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// Fastener dimensions in mm. Every bolt is an m2bolt body and every nut an m2nut body. A head sinks into a pocket of its
// part and ends flush with the part's face. A nut sits in a cavity its part encloses, apart from the bolt hole, so the
// part retains it; it drops in during a print pause. The pockets and the cavity keep the Fasteners rule's 0.1 radial
// clearance, not a fit class.
const (
	fastenerClearance = 0.1                                       // radial gap around a head or a nut
	boltHole          = 1.3                                       // clearance hole radius, 2.6 across
	headPocketRadius  = m2bolt.HeadDiameter/2 + fastenerClearance // 2.0
	headPocketDepth   = m2bolt.HeadHeight                         // 2.0: the head ends flush
	nutCavityFlats    = m2nut.AcrossFlats + 2*fastenerClearance   // 4.2
	nutCavityHeight   = m2nut.Thickness + 2*fastenerClearance     // 1.8: the nut floats 0.1 from each face
	fastenerStock     = 1.0                                       // a tool's reach beyond a face it opens
)

// fastenerSite is one bolt and its captive nut. head is the centre of the head's top face, on headHost's face; dir
// runs from the head toward the tip; corner is the unit vector across dir on which the nut's and the cavity's corners
// lie. cavity is the nut cavity's near face, measured from head along dir; the cavity runs nutCavityHeight beyond it.
type fastenerSite struct {
	bolt, nut         string
	headHost, nutHost string
	motion            string
	head, dir, corner r3.Vec
	length, cavity    float64
}

// at is the point t along the site's line from the head's top face.
func (s fastenerSite) at(t float64) r3.Vec { return s.head.Add(s.dir.Scale(t)) }

// across is a frame at distance t from the head whose normal runs along dir and whose U axis is the corner direction.
func (s fastenerSite) across(t float64) r3.Frame {
	return frame(s.at(t), s.corner, s.dir.Cross(s.corner))
}

// turned returns the site turned by angle about the line through center along axis.
func (s fastenerSite) turned(center, axis r3.Vec, angle float64) (fastenerSite, error) {
	xf, err := r3.RotationAround(center, axis, units.Radians(angle))
	if err != nil {
		return s, err
	}
	s.head, s.dir, s.corner = xf.Apply(s.head), xf.ApplyDir(s.dir), xf.ApplyDir(s.corner)
	return s, nil
}

// fastenerSites returns every site at crank angle 0. The blade site is blade 1's; the other blades' come from the
// pattern about the fan axis.
func fastenerSites() []fastenerSite {
	sites := []fastenerSite{{
		bolt: "fan-bolt-1", nut: "fan-nut-1", headHost: "fan-hub", nutHost: "fan-hub", motion: "fan",
		head: r3.Vec{X: boltAt, Y: -(hubBack + hubLength), Z: axisHeight}, dir: r3.Vec{Y: 1}, corner: r3.Vec{X: 1},
		length: 16, cavity: 15.8,
	}}
	for i, e := range gearboxEars {
		c := e.centre()
		sites = append(sites, fastenerSite{
			bolt: fmt.Sprintf("gearbox-bolt-%d", i+1), nut: fmt.Sprintf("gearbox-nut-%d", i+1),
			headHost: e.head, nutHost: e.nut, motion: "fixed",
			head: c.Sub(e.dir.Scale(earHalf)), dir: e.dir, corner: r3.Vec{Z: 1}, length: 10, cavity: 9.8,
		})
	}
	for i, w := range wallScrews {
		sites = append(sites, fastenerSite{
			bolt: fmt.Sprintf("gearbox-screw-%d", i+1), nut: fmt.Sprintf("gearbox-screw-nut-%d", i+1),
			headHost: w.head, nutHost: w.recv, motion: "fixed",
			head: w.at(w.face), dir: w.dir, corner: r3.Vec{Z: 1}, length: 16, cavity: 15.8,
		})
	}
	for i, dx := range []float64{-gripScrewX, gripScrewX} {
		sites = append(sites, fastenerSite{
			bolt: fmt.Sprintf("handle-bolt-%d", i+1), nut: fmt.Sprintf("handle-nut-%d", i+1),
			headHost: "gearbox-base", nutHost: "handle", motion: "fixed",
			head: r3.Vec{X: gripX + dx, Y: gripY, Z: cavityFloor}, dir: r3.Vec{Z: -1}, corner: r3.Vec{Y: 1},
			length: 10, cavity: 9.3,
		})
	}
	return append(sites, fastenerSite{
		bolt: "crank-pin-bolt", nut: "crank-pin-nut", headHost: "crank-arm", nutHost: "crank-pin", motion: "crank",
		head: r3.Vec{X: armStart, Z: crankHeight + crankThrow}, dir: r3.Vec{X: 1}, corner: r3.Vec{Y: 1},
		length: 10, cavity: 4.75,
	})
}

// bladeSites returns the blade sites of an assembly with n blades, blade 1 first, at crank angle 0.
func bladeSites(n int) ([]fastenerSite, error) {
	first := fastenerSites()[0]
	sites := []fastenerSite{first}
	for k := 1; k < n; k++ {
		s, err := first.turned(r3.Vec{Z: axisHeight}, fanAxis, 2*math.Pi*float64(k)/float64(n))
		if err != nil {
			return nil, fmt.Errorf("turn blade %d's fastener: %w", k+1, err)
		}
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
		return "M2 hex nut"
	case strings.HasPrefix(name, "fan-bolt-"):
		return boltSpec(fastenerSites()[0].length)
	}
	for _, s := range fastenerSites() {
		switch name {
		case s.bolt:
			return boltSpec(s.length)
		case s.nut:
			return "M2 hex nut"
		}
	}
	return ""
}

func boltSpec(length float64) string { return fmt.Sprintf("M2 x %g socket head cap screw", length) }

// placeBolt builds an m2bolt body of s.length with the head's top face at s.head and the shank along s.dir, and
// appends its sketches to the assembly.
func (b *builder) placeBolt(s fastenerSite) *decad.Body {
	if b.err != nil {
		return nil
	}
	f := frame(s.at(headPocketDepth), s.corner, s.dir.Scale(-1).Cross(s.corner))
	m, err := m2bolt.Build(b.ctx, b.doc, m2bolt.Params{Length: s.length, Placement: &f})
	b.fail("build bolt", err)
	if err != nil {
		return nil
	}
	b.a.Sketches = append(b.a.Sketches, m.Profile, m.Socket)
	return m.Body
}

// placeNut builds an m2nut body whose lower face lies fastenerClearance beyond the cavity's near face, corners on
// s.corner, and appends its sketches to the assembly.
func (b *builder) placeNut(s fastenerSite) *decad.Body {
	if b.err != nil {
		return nil
	}
	f := s.across(s.cavity + fastenerClearance)
	m, err := m2nut.Build(b.ctx, b.doc, m2nut.Params{Placement: &f})
	b.fail("build nut", err)
	if err != nil {
		return nil
	}
	b.a.Sketches = append(b.a.Sketches, m.Hexagon, m.LowerChamfer, m.UpperChamfer)
	return m.Body
}

// fastenerCuts cuts from host the features whose flag is true: pocket is s's head pocket (radius headPocketRadius,
// from fastenerStock outside the face to headPocketDepth inside), hole its clearance hole (radius boltHole, from
// fastenerStock outside the face to fastenerStock beyond the tip) and cavity its nut cavity (a hexagon of nutCavityFlats
// across, nutCavityHeight long, starting at s.cavity, corners on s.corner). The caller passes the flags for the body it
// holds; the flags alone decide which cuts happen.
func (b *builder) fastenerCuts(host *decad.Body, s fastenerSite, pocket, hole, cavity bool) *decad.Body {
	if pocket {
		host = b.cut(host, b.cylinder(s.across(-fastenerStock), headPocketRadius, fastenerStock+headPocketDepth))
	}
	if hole {
		tip := headPocketDepth + s.length
		host = b.cut(host, b.cylinder(s.across(-fastenerStock), boltHole, fastenerStock+tip+fastenerStock))
	}
	if cavity {
		host = b.cut(host, b.hexagon(s.across(s.cavity), nutCavityFlats, nutCavityHeight))
	}
	return host
}
