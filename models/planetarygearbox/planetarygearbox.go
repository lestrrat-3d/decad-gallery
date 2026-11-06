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
	// Carrier is the plate, the planet pins and the stub shaft in one solid. The stub of a
	// carrier before the last passes through the next stage's sun; the last carrier's stub is
	// the output shaft.
	Carrier *decad.Body
}

// Gearbox holds the bodies of a built gearbox.
type Gearbox struct {
	Ring   *decad.Body
	Stages []Stage
}

// NamedBody is a body with its display name.
type NamedBody struct {
	Name string
	Body *decad.Body
}

// Names of the bodies.
const RingName = "Ring Gear"

// SunName is the name of the sun of stage index (0-based).
func SunName(stage int) string { return fmt.Sprintf("Sun Gear %d", stage+1) }

// PlanetName is the name of planet index of stage (both 0-based).
func PlanetName(stage, index int) string { return fmt.Sprintf("Planet Gear %d.%d", stage+1, index+1) }

// CarrierName is the name of the carrier of stage index (0-based).
func CarrierName(stage int) string { return fmt.Sprintf("Carrier %d", stage+1) }

// Named lists every body of the gearbox with its name: the ring, then each stage's sun,
// planets and carrier.
func (g Gearbox) Named() []NamedBody {
	out := []NamedBody{{RingName, g.Ring}}
	for k, stage := range g.Stages {
		out = append(out, NamedBody{SunName(k), stage.Sun})
		for i, planet := range stage.Planets {
			out = append(out, NamedBody{PlanetName(k, i), planet})
		}
		out = append(out, NamedBody{CarrierName(k), stage.Carrier})
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

	sun, err := r.buildSun(ctx)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
	}
	planet, err := r.buildPlanet(ctx)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
	}
	var middle *decad.Body
	if p.Stages > 1 {
		stubTop := p.CarrierThickness + p.StageGap + p.GearThickness
		if middle, err = r.buildCarrier(ctx, stubTop); err != nil {
			return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
		}
	}
	last, err := r.buildCarrier(ctx, p.CarrierThickness+p.OutputShaftLength)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: %w", err)
	}

	// Collect every pose first: each master's last instance retires the master.
	var sunMoves, planetMoves, middleMoves, lastMoves []r3.Transform
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
		}
		m, err = pose(r.frame, 0, 0, 0, z+p.GearThickness+p.StageGap)
		if err != nil {
			return Gearbox{}, fmt.Errorf("planetarygearbox: carrier pose: %w", err)
		}
		if k == p.Stages-1 {
			lastMoves = append(lastMoves, m)
		} else {
			middleMoves = append(middleMoves, m)
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
	var middles []*decad.Body
	if middle != nil {
		if middles, err = instances(ctx, middle, middleMoves); err != nil {
			return Gearbox{}, fmt.Errorf("planetarygearbox: carriers: %w", err)
		}
	}
	lasts, err := instances(ctx, last, lastMoves)
	if err != nil {
		return Gearbox{}, fmt.Errorf("planetarygearbox: output carrier: %w", err)
	}
	carriers := append(middles, lasts...)
	for k := range p.Stages {
		box.Stages = append(box.Stages, Stage{
			Sun:     suns[k],
			Planets: planets[k*p.PlanetCount : (k+1)*p.PlanetCount],
			Carrier: carriers[k],
		})
	}
	return box, nil
}
