package planetarygearbox

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
)

// Container names, one per kind of body. Each kind has one container under the document root.
const (
	HousingContainerName        = "Housing"
	CoverBearingsContainerName  = "Cover Bearings"
	SunGearsContainerName       = "Sun Gears"
	PlanetGearsContainerName    = "Planet Gears"
	PlanetBearingsContainerName = "Planet Bearings"
	CarriersContainerName       = "Carriers"
	CarrierPinsContainerName    = "Carrier Pins"
	ShaftsContainerName         = "Shafts"
	BoltsContainerName          = "Bolts"
	NutsContainerName           = "Nuts"
	CollarsContainerName        = "Shaft Collars"
)

// Part is a gearbox body with its display name and the container Build files it in.
type Part struct {
	Name      string
	Container string
	Body      *decad.Body
}

// Parts lists every body of the gearbox with its name and container, in the order Build files them: the housing,
// the cover bearings, the bolts and the ear bolts, the nuts and the ear nuts, the collar rings, the input shaft, then
// each stage's sun, planets, planet bearings, pins, carrier and shaft.
func (g Gearbox) Parts() []Part {
	out := []Part{
		{RingName, HousingContainerName, g.Ring},
		{FrontCoverName, HousingContainerName, g.FrontCover},
		{RearCoverName, HousingContainerName, g.RearCover},
		{FrontBearingName, CoverBearingsContainerName, g.FrontBearing},
		{RearBearingName, CoverBearingsContainerName, g.RearBearing},
	}
	half := len(g.Bolts) / 2
	boltName := func(i int) string {
		if i >= half {
			return RearBoltName(i - half)
		}
		return FrontBoltName(i)
	}
	nutName := func(i int) string {
		if i >= half {
			return RearNutName(i - half)
		}
		return FrontNutName(i)
	}
	for i, b := range g.Bolts {
		out = append(out, Part{boltName(i), BoltsContainerName, b})
	}
	for i, b := range g.EarBolts {
		out = append(out, Part{EarBoltName(i), BoltsContainerName, b})
	}
	for i, n := range g.Nuts {
		out = append(out, Part{nutName(i), NutsContainerName, n})
	}
	for i, n := range g.EarNuts {
		out = append(out, Part{EarNutName(i), NutsContainerName, n})
	}
	out = append(out,
		Part{FrontCollarName, CollarsContainerName, g.FrontCollar},
		Part{RearCollarName, CollarsContainerName, g.RearCollar})
	out = append(out, Part{InputShaftName, ShaftsContainerName, g.InputShaft})
	for k, s := range g.Stages {
		out = append(out, Part{SunName(k), SunGearsContainerName, s.Sun})
		for i, b := range s.Planets {
			out = append(out, Part{PlanetName(k, i), PlanetGearsContainerName, b})
		}
		for i, b := range s.Bearings {
			out = append(out, Part{BearingName(k, i), PlanetBearingsContainerName, b})
		}
		for i, b := range s.Pins {
			out = append(out, Part{PinName(k, i), CarrierPinsContainerName, b})
		}
		out = append(out,
			Part{CarrierName(k), CarriersContainerName, s.Carrier},
			Part{ShaftName(k, len(g.Stages)), ShaftsContainerName, s.Shaft})
	}
	return out
}

// moveBodies puts every body of box in the container for its kind.
func moveBodies(doc *decad.Document, box Gearbox) error {
	root := doc.Root()
	for _, part := range box.Parts() {
		c, err := root.ChildNamed(part.Container)
		if err != nil {
			return fmt.Errorf("container %q: %w", part.Container, err)
		}
		if err := c.AddBody(part.Body); err != nil {
			return fmt.Errorf("place body in container %q: %w", part.Container, err)
		}
	}
	return nil
}
