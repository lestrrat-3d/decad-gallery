// Package fastener builds bolts and nuts from a catalog of size presets.
package fastener

import (
	"fmt"
	"math"
	"slices"

	"github.com/lestrrat-3d/decad-gallery/internal/hexnut"
)

// shankOverlap is how far a hex head bolt's shank reaches into the head.
const shankOverlap = 1.0

// HeadStyle selects a bolt's head construction.
type HeadStyle int

const (
	// HeadSocketCap is an ISO 4762 socket head cap screw: round head, hex socket, drill point.
	HeadSocketCap HeadStyle = iota + 1
	// HeadHex is an ISO 4017 hex head bolt: hexagonal head chamfered on both faces.
	HeadHex
)

// String returns "socket head cap screw", "hex head bolt", or "HeadStyle(<n>)" for any other value.
func (h HeadStyle) String() string {
	switch h {
	case HeadSocketCap:
		return "socket head cap screw"
	case HeadHex:
		return "hex head bolt"
	default:
		return fmt.Sprintf("HeadStyle(%d)", int(h))
	}
}

// HexChamferAngle is the corner chamfer angle from the face of a hex head and of a nut, in degrees.
const HexChamferAngle = hexnut.ChamferAngle

// SocketCapHead holds the HeadSocketCap dimensions in millimeters. Zero for any other head style.
type SocketCapHead struct {
	HeadDiameter         float64 // dk
	SocketAcrossFlats    float64 // s
	SocketDepth          float64 // t, top face to the start of the pilot well
	HeadChamferWidth     float64 // v, radial width of the top-edge chamfer
	HeadChamferAngle     float64 // degrees from the top face
	UnderheadRadius      float64 // r, fillet where the shank meets the head
	DrillPointAngle      float64 // included angle of the cone below the socket, degrees
	DrillPointBaseRadius float64 // radius of the pilot well and the cone's base
	PilotDepth           float64 // pilot well depth below the socket floor
}

// HexHead holds the HeadHex dimensions in millimeters. Zero for any other head style.
type HexHead struct {
	AcrossFlats     float64 // s
	BearingDiameter float64 // dw, flat bearing circle on each head face
}

// AcrossCorners returns AcrossFlats / cos(30°).
func (h HexHead) AcrossCorners() float64 {
	return h.AcrossFlats / math.Cos(math.Pi/6)
}

// BoltSize holds a bolt's nominal dimensions in millimeters. The thread is not modeled.
type BoltSize struct {
	Head            HeadStyle
	ShankDiameter   float64   // ds
	HeadHeight      float64   // k
	TipChamfer      float64   // radial and axial size u of the 45° tip chamfer
	StandardLengths []float64 // underside of head to tip, strictly ascending; treat as read-only
	SocketCap       SocketCapHead
	Hex             HexHead
}

// NutSize holds an ISO 4032 hex nut's nominal dimensions in millimeters.
type NutSize struct {
	BoreDiameter    float64 // d, nominal thread diameter; the thread is not modeled
	AcrossFlats     float64 // s
	Thickness       float64 // m
	BearingDiameter float64 // dw
}

// AcrossCorners returns AcrossFlats / cos(30°).
func (n NutSize) AcrossCorners() float64 {
	return n.AcrossFlats / math.Cos(math.Pi/6)
}

// Preset is one catalog size: a bolt and the nut that pairs with it.
type Preset struct {
	Name string // catalog name, e.g. "M2"
	Bolt BoltSize
	Nut  NutSize
}

// catalog lists the sizes. Add a size only with a cited standard; never invent dimensions.
var catalog = []Preset{
	{
		// M2: socket head cap screw per ISO 4762 / JIS B 1176, hex nut per ISO 4032 / JIS B 1181.
		Name: "M2",
		Bolt: BoltSize{
			Head:            HeadSocketCap,
			ShankDiameter:   2.0,
			HeadHeight:      2.0,
			TipChamfer:      0.25,
			StandardLengths: []float64{6, 10, 16, 18, 20, 25},
			SocketCap: SocketCapHead{
				HeadDiameter:         3.8,
				SocketAcrossFlats:    1.5,
				SocketDepth:          1.0,
				HeadChamferWidth:     0.2,
				HeadChamferAngle:     30.0,
				UnderheadRadius:      0.1,
				DrillPointAngle:      118.0,
				DrillPointBaseRadius: 0.7,
				PilotDepth:           0.1,
			},
		},
		Nut: NutSize{BoreDiameter: 2.0, AcrossFlats: 4.0, Thickness: 1.6, BearingDiameter: 3.48},
	},
	{
		// M5: hex head bolt per ISO 4017 / JIS B 1180, hex nut per ISO 4032 style 1 / JIS B 1181.
		Name: "M5",
		Bolt: BoltSize{
			Head:            HeadHex,
			ShankDiameter:   5.0,
			HeadHeight:      3.5,
			TipChamfer:      0.5,
			StandardLengths: []float64{16, 20, 25, 30, 35, 40, 45, 50, 60, 70, 80},
			Hex:             HexHead{AcrossFlats: 8.0, BearingDiameter: 6.88},
		},
		Nut: NutSize{BoreDiameter: 5.0, AcrossFlats: 8.0, Thickness: 4.7, BearingDiameter: 6.9},
	},
}

// Names returns the catalog names in catalog order. Each call returns a fresh slice.
func Names() []string {
	names := make([]string, 0, len(catalog))
	for _, p := range catalog {
		names = append(names, p.Name)
	}
	return names
}

// Lookup returns a deep copy (StandardLengths cloned) of the preset whose Name equals name exactly, or the zero Preset
// and false.
func Lookup(name string) (Preset, bool) {
	for _, p := range catalog {
		if p.Name == name {
			p.Bolt.StandardLengths = slices.Clone(p.Bolt.StandardLengths)
			return p, true
		}
	}
	return Preset{}, false
}

func positive(v float64) bool {
	return v > 0 && !math.IsInf(v, 0)
}

// Validate rejects a preset that cannot build its bolt or nut.
func (p Preset) Validate() error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("fastener: preset %q: %s", p.Name, fmt.Sprintf(format, args...))
	}
	if p.Name == "" {
		return fail("Name must not be empty")
	}
	b := p.Bolt
	if !positive(b.ShankDiameter) || !positive(b.HeadHeight) || !positive(b.TipChamfer) {
		return fail("ShankDiameter, HeadHeight and TipChamfer must be finite and positive")
	}
	if b.TipChamfer >= b.ShankDiameter/2 {
		return fail("TipChamfer must be below half the ShankDiameter")
	}
	if len(b.StandardLengths) == 0 {
		return fail("StandardLengths must not be empty")
	}
	for i, l := range b.StandardLengths {
		if math.IsNaN(l) || math.IsInf(l, 0) || !(l > b.TipChamfer) {
			return fail("StandardLengths[%d] must be finite and above TipChamfer", i)
		}
		if i > 0 && !(l > b.StandardLengths[i-1]) {
			return fail("StandardLengths must be strictly ascending")
		}
	}
	switch b.Head {
	case HeadSocketCap:
		if err := p.validateSocketCap(fail); err != nil {
			return err
		}
	case HeadHex:
		if err := p.validateHex(fail); err != nil {
			return err
		}
	default:
		return fail("unknown head style %d", int(b.Head))
	}
	n := p.Nut
	if err := (hexnut.Size{
		Thread: n.BoreDiameter, AcrossFlats: n.AcrossFlats, Thickness: n.Thickness, BearingDiameter: n.BearingDiameter,
	}).Validate(); err != nil {
		return fmt.Errorf("fastener: preset %q: nut: %w", p.Name, err)
	}
	if n.BoreDiameter != b.ShankDiameter {
		return fail("nut BoreDiameter must equal the bolt ShankDiameter")
	}
	return nil
}

func (p Preset) validateSocketCap(fail func(string, ...any) error) error {
	b, s := p.Bolt, p.Bolt.SocketCap
	if b.Hex != (HexHead{}) {
		return fail("Hex must be zero for a socket head cap screw")
	}
	for _, v := range []float64{s.HeadDiameter, s.SocketAcrossFlats, s.SocketDepth, s.HeadChamferWidth,
		s.HeadChamferAngle, s.UnderheadRadius, s.DrillPointAngle, s.DrillPointBaseRadius, s.PilotDepth} {
		if !positive(v) {
			return fail("every SocketCap dimension must be finite and positive")
		}
	}
	if !(s.HeadDiameter > b.ShankDiameter+2*s.UnderheadRadius) {
		return fail("SocketCap.HeadDiameter must exceed ShankDiameter plus twice UnderheadRadius")
	}
	if !(s.HeadChamferAngle < 90) {
		return fail("SocketCap.HeadChamferAngle must be below 90 degrees")
	}
	if !(s.HeadChamferWidth*math.Tan(s.HeadChamferAngle*math.Pi/180) < b.HeadHeight) {
		return fail("SocketCap head chamfer must be lower than HeadHeight")
	}
	if !(s.SocketAcrossFlats/math.Sqrt(3) < s.HeadDiameter/2-s.HeadChamferWidth) {
		return fail("SocketCap socket corners must lie inside the head's top face")
	}
	if !(s.DrillPointBaseRadius < s.SocketAcrossFlats/2) {
		return fail("SocketCap.DrillPointBaseRadius must be below half SocketAcrossFlats")
	}
	if !(s.DrillPointAngle < 180) {
		return fail("SocketCap.DrillPointAngle must be below 180 degrees")
	}
	apex := b.HeadHeight - s.SocketDepth - s.PilotDepth - s.DrillPointBaseRadius/math.Tan(s.DrillPointAngle/2*math.Pi/180)
	if !(apex > 0) {
		return fail("SocketCap drill point apex must lie above the head's underside")
	}
	return nil
}

func (p Preset) validateHex(fail func(string, ...any) error) error {
	b, h := p.Bolt, p.Bolt.Hex
	if b.SocketCap != (SocketCapHead{}) {
		return fail("SocketCap must be zero for a hex head bolt")
	}
	if !positive(h.AcrossFlats) || !positive(h.BearingDiameter) {
		return fail("Hex dimensions must be finite and positive")
	}
	if !(b.ShankDiameter < h.BearingDiameter && h.BearingDiameter < h.AcrossFlats) {
		return fail("Hex needs ShankDiameter < BearingDiameter < AcrossFlats")
	}
	if !(b.HeadHeight > shankOverlap) {
		return fail("HeadHeight must exceed the shank overlap %g", shankOverlap)
	}
	return nil
}
