// Package planetarygearbox builds a compact planetary gearbox as a parametric decad model:
// a fixed internal ring gear that houses one or more stacked stages, each with a sun gear,
// planet gears and a carrier.
package planetarygearbox

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Stage holds the moving parts of one planetary stage.
type Stage struct {
	Sun     *decad.Body
	Planets []*decad.Body
	// Bearings holds one planet bearing envelope per planet, in the planet's pocket. A planet turns on its
	// bearing; the sun and the shafts turn with the carrier that drives them and need none.
	Bearings []*decad.Body
	// Pins holds one carrier pin per planet. Each is a separate part: a thin rod that carries the planet bearing
	// and a D-shaped thick part that presses through the plate's socket, level with the plate's upper face.
	Pins []*decad.Body
	// Carrier is the triangular plate with a D-shaped socket through it for each pin and a D hole through its middle
	// that takes the thick part of a shaft. It prints plate-down.
	Carrier *decad.Body
	// Shaft is a separate thin-thick shaft. Its D thick part is pressed through the carrier's hole, level with the
	// carrier's lower face. For every stage but the last it is the link shaft, whose thin rod also fills the next
	// stage's sun bore; for the last stage it is the output shaft, whose thin rod passes through the rear cover's
	// bearing, with the thick part's step short of it.
	Shaft *decad.Body
}

// Gearbox holds the bodies of a built gearbox.
type Gearbox struct {
	// Ring is the fixed internal gear. FrontCover and RearCover close its two ends, each with a shaft bearing in
	// a boss; the input shaft turns in the front one and the last carrier's shaft in the rear one.
	Ring                      *decad.Body
	FrontCover, RearCover     *decad.Body
	FrontBearing, RearBearing *decad.Body
	// InputShaft passes through the front bearing into the first sun's bore.
	InputShaft *decad.Body
	Stages     []Stage
	// Bolts holds the eight M2 bolts, four per cover, front cover's first. Each head is sunk flush in its cover's
	// outer face. Nuts holds the matching eight M2 nuts, each sunk flush in an end face of the ring.
	Bolts, Nuts []*decad.Body
}

// NamedBody is a body with its display name.
type NamedBody struct {
	Name string
	Body *decad.Body
}

// Names of the bodies that are not part of a stage.
const (
	RingName         = "Ring Gear"
	FrontCoverName   = "Front Cover"
	RearCoverName    = "Rear Cover"
	FrontBearingName = "Front Bearing"
	RearBearingName  = "Rear Bearing"
	InputShaftName   = "Input Shaft"
	OutputShaftName  = "Output Shaft"
)

// SunName is the name of the sun of stage index (0-based).
func SunName(stage int) string { return fmt.Sprintf("Sun Gear %d", stage+1) }

// PlanetName is the name of planet index of stage (both 0-based).
func PlanetName(stage, index int) string { return fmt.Sprintf("Planet Gear %d.%d", stage+1, index+1) }

// BearingName is the name of the bearing in planet index of stage (both 0-based).
func BearingName(stage, index int) string {
	return fmt.Sprintf("Planet Bearing %d.%d", stage+1, index+1)
}

// ShaftName is the name of the shaft of stage index (0-based) in a gearbox of stages stages: the link shaft to the
// next sun, or for the last stage the output shaft.
func ShaftName(stage, stages int) string {
	if stage == stages-1 {
		return OutputShaftName
	}
	return fmt.Sprintf("Link Shaft %d", stage+1)
}

// CarrierName is the name of the carrier of stage index (0-based).
func CarrierName(stage int) string { return fmt.Sprintf("Carrier %d", stage+1) }

// PinName is the name of the carrier pin index of stage (both 0-based).
func PinName(stage, index int) string { return fmt.Sprintf("Carrier Pin %d.%d", stage+1, index+1) }

// Named lists every body of the gearbox with its name: the ring, then each stage's sun,
// planets and carrier.
func (g Gearbox) Named() []NamedBody {
	out := []NamedBody{
		{RingName, g.Ring}, {FrontCoverName, g.FrontCover}, {RearCoverName, g.RearCover},
		{FrontBearingName, g.FrontBearing}, {RearBearingName, g.RearBearing}, {InputShaftName, g.InputShaft},
	}
	for i := range g.Bolts {
		bolt, nut := FrontBoltName(i), FrontNutName(i)
		if half := len(g.Bolts) / 2; i >= half {
			bolt, nut = RearBoltName(i-half), RearNutName(i-half)
		}
		out = append(out, NamedBody{bolt, g.Bolts[i]}, NamedBody{nut, g.Nuts[i]})
	}
	for k, stage := range g.Stages {
		out = append(out, NamedBody{SunName(k), stage.Sun})
		for i, planet := range stage.Planets {
			out = append(out, NamedBody{PlanetName(k, i), planet})
			out = append(out, NamedBody{BearingName(k, i), stage.Bearings[i]})
		}
		for i, pin := range stage.Pins {
			out = append(out, NamedBody{PinName(k, i), pin})
		}
		out = append(out, NamedBody{CarrierName(k), stage.Carrier})
		out = append(out, NamedBody{ShaftName(k, len(g.Stages)), stage.Shaft})
	}
	return out
}

// run is the state one build shares.
type run struct {
	doc   *decad.Document
	world *sketch.World
	p     Params
	d     dims
	frame r3.Transform
}

// Build constructs the gearbox in doc.
//
// Each distinct part (sun, planet, ring, carrier) is built once about the local origin and
// then copied and moved into place, because gear unions fail on bodies built at an offset.
// The parameter table holds the gearbox entries, and the spur gear builds overwrite the
// spur names (teeth, thickness and others) with the last gear built.
func Build(ctx context.Context, doc *decad.Document, p Params) (Gearbox, error) {
	if ctx == nil {
		return Gearbox{}, errors.New("planetarygearbox: nil context")
	}
	if doc == nil {
		return Gearbox{}, errors.New("planetarygearbox: nil document")
	}
	if err := p.Validate(); err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
	}
	world := sketch.NewWorld()
	if err := defineParameters(doc.Params(), p); err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: define document parameters: %w", err)
	}
	if err := defineParameters(world.Params(), p); err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: define sketch parameters: %w", err)
	}
	d, err := readDims(doc, p)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
	}
	frame, err := frameTransform(p)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
	}
	r := &run{doc: doc, world: world, p: p, d: d, frame: frame}
	return r.build(ctx)
}

func (r *run) build(ctx context.Context) (Gearbox, error) {
	p, d := r.p, r.d
	var box Gearbox

	ring, err := r.buildRing(ctx)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
	}
	move, err := pose(r.frame, 0, 0, 0, r.ringShift())
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: place ring: %w", err)
	}
	if box.Ring, err = decad.Transform(ctx, ring, move); err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: place ring: %w", err)
	}

	if err := r.buildHousing(ctx, &box); err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
	}
	if err := r.buildFasteners(ctx, &box); err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
	}

	sun, err := r.buildSun(ctx)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
	}
	planet, err := r.buildPlanet(ctx)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
	}
	bearing, err := r.buildBearing(ctx, "planet bearing", planetBearing(p))
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
	}
	pin, err := r.buildPin(ctx)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
	}
	carrier, err := r.buildCarrier(ctx)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
	}
	// A link shaft and the output shaft are thin-thick. Each has a D thick part pressed through a carrier's hole,
	// level with the carrier's lower face. The link shaft's thick part ends short of the next sun and its thin rod
	// runs through that sun's bore, ending level with the sun's top. The output shaft's thick part ends short of the
	// rear bearing's inner race and its thin rod rises through the bearing, ending OutputShaftLength beyond the
	// bearing boss.
	var link *decad.Body
	if p.Stages > 1 {
		linkTop := p.GearThickness + p.CarrierThickness + p.StageGap
		if link, err = r.buildSteppedShaft(ctx, "link shaft", p.CarrierThickness+p.StageGap-fitClearance, linkTop,
			span{p.CarrierThickness + p.StageGap, linkTop + 1}); err != nil {
			return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
		}
	}
	outputTop := p.CarrierThickness + p.StageGap + p.CoverThickness + shaftBearing(p).Width + p.OutputShaftLength
	output, err := r.buildSteppedShaft(ctx, "output shaft",
		p.CarrierThickness+p.StageGap+p.CoverThickness-fitClearance, outputTop)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
	}

	// Collect every pose first: each master's last instance retires the master.
	var sunMoves, planetMoves, bearingMoves, pinMoves, carrierMoves, linkMoves, outputMoves []r3.Transform
	for k := range p.Stages {
		z := float64(k) * d.stagePitch
		m, err := pose(r.frame, 0, 0, 0, z)
		if err != nil {
			return Gearbox{}, fmt.Errorf("planetarygearbox: sun pose: %w", err)
		}
		sunMoves = append(sunMoves, m)
		for i := range p.PlanetCount {
			phi := PlanetOrbitAngle(p, i)
			m, err := pose(r.frame, PlanetToothAngle(p, i), d.orbit*math.Cos(phi), d.orbit*math.Sin(phi), z)
			if err != nil {
				return Gearbox{}, fmt.Errorf("planetarygearbox: planet pose: %w", err)
			}
			planetMoves = append(planetMoves, m)
			m, err = pose(r.frame, 0, d.orbit*math.Cos(phi), d.orbit*math.Sin(phi),
				z+p.GearThickness-planetBearing(p).Width+planetBearingLift)
			if err != nil {
				return Gearbox{}, fmt.Errorf("planetarygearbox: bearing pose: %w", err)
			}
			bearingMoves = append(bearingMoves, m)
			// The pin turns with its orbit angle, so its flat faces the same way as its socket.
			m, err = pose(r.frame, phi, d.orbit*math.Cos(phi), d.orbit*math.Sin(phi), z+p.GearThickness+p.StageGap)
			if err != nil {
				return Gearbox{}, fmt.Errorf("planetarygearbox: pin pose: %w", err)
			}
			pinMoves = append(pinMoves, m)
		}
		m, err = pose(r.frame, 0, 0, 0, z+p.GearThickness+p.StageGap)
		if err != nil {
			return Gearbox{}, fmt.Errorf("planetarygearbox: carrier pose: %w", err)
		}
		carrierMoves = append(carrierMoves, m)
		if k == p.Stages-1 {
			outputMoves = append(outputMoves, m)
		} else {
			linkMoves = append(linkMoves, m)
		}
	}
	suns, err := instances(ctx, sun, sunMoves)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: suns: %w", err)
	}
	planets, err := instances(ctx, planet, planetMoves)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: planets: %w", err)
	}
	bearings, err := instances(ctx, bearing, bearingMoves)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: bearings: %w", err)
	}
	pins, err := instances(ctx, pin, pinMoves)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: pins: %w", err)
	}
	carriers, err := instances(ctx, carrier, carrierMoves)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: carriers: %w", err)
	}
	var links []*decad.Body
	if link != nil {
		if links, err = instances(ctx, link, linkMoves); err != nil {
			return Gearbox{}, fmt.Errorf("planetarygearbox: link shafts: %w", err)
		}
	}
	outputs, err := instances(ctx, output, outputMoves)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: output shaft: %w", err)
	}
	shafts := append(links, outputs...)
	for k := range p.Stages {
		box.Stages = append(box.Stages, Stage{
			Sun:      suns[k],
			Planets:  planets[k*p.PlanetCount : (k+1)*p.PlanetCount],
			Bearings: bearings[k*p.PlanetCount : (k+1)*p.PlanetCount],
			Pins:     pins[k*p.PlanetCount : (k+1)*p.PlanetCount],
			Carrier:  carriers[k],
			Shaft:    shafts[k],
		})
	}
	if err := moveBodies(r.doc, box); err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
	}
	return box, nil
}

// buildHousing builds the front and rear covers, their bearings and the input shaft, and places them in box.
// The covers sit against the ring's two ends; each is built once and the front one is turned over so its
// bearing boss faces away from the ring.
func (r *run) buildHousing(ctx context.Context, box *Gearbox) error {
	p, d := r.p, r.d
	width := shaftBearing(p).Width
	cover, err := r.buildCover(ctx)
	if err != nil {
		return err
	}
	front, err := poseFlipped(r.frame, -p.StageGap)
	if err != nil {
		return fmt.Errorf("front cover pose: %w", err)
	}
	rear, err := pose(r.frame, 0, 0, 0, d.ringLength-p.StageGap)
	if err != nil {
		return fmt.Errorf("rear cover pose: %w", err)
	}
	covers, err := instances(ctx, cover, []r3.Transform{front, rear})
	if err != nil {
		return fmt.Errorf("covers: %w", err)
	}
	box.FrontCover, box.RearCover = covers[0], covers[1]

	bearing, err := r.buildBearing(ctx, "cover bearing", shaftBearing(p))
	if err != nil {
		return err
	}
	frontBearing, err := pose(r.frame, 0, 0, 0, -(p.StageGap + p.CoverThickness + width))
	if err != nil {
		return fmt.Errorf("front bearing pose: %w", err)
	}
	rearBearing, err := pose(r.frame, 0, 0, 0, d.ringLength-p.StageGap+p.CoverThickness)
	if err != nil {
		return fmt.Errorf("rear bearing pose: %w", err)
	}
	bearings, err := instances(ctx, bearing, []r3.Transform{frontBearing, rearBearing})
	if err != nil {
		return fmt.Errorf("cover bearings: %w", err)
	}
	box.FrontBearing, box.RearBearing = bearings[0], bearings[1]

	shaft, err := r.buildInputShaft(ctx)
	if err != nil {
		return err
	}
	place, err := pose(r.frame, 0, 0, 0, 0)
	if err != nil {
		return fmt.Errorf("input shaft pose: %w", err)
	}
	if box.InputShaft, err = decad.Transform(ctx, shaft, place); err != nil {
		return fmt.Errorf("place input shaft: %w", err)
	}
	return nil
}
