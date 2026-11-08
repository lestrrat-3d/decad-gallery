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
	ShaftsContainerName         = "Shafts"
	BoltsContainerName          = "Bolts"
	NutsContainerName           = "Nuts"
)

// moveBodies puts every body of box in the container for its kind.
func moveBodies(doc *decad.Document, box Gearbox) error {
	root := doc.Root()
	move := func(name string, bodies ...*decad.Body) error {
		c, err := root.ChildNamed(name)
		if err != nil {
			return fmt.Errorf("container %q: %w", name, err)
		}
		for _, b := range bodies {
			if err := c.AddBody(b); err != nil {
				return fmt.Errorf("place body in container %q: %w", name, err)
			}
		}
		return nil
	}
	if err := move(HousingContainerName, box.Ring, box.FrontCover, box.RearCover); err != nil {
		return err
	}
	if err := move(CoverBearingsContainerName, box.FrontBearing, box.RearBearing); err != nil {
		return err
	}
	if err := move(BoltsContainerName, box.Bolts...); err != nil {
		return err
	}
	if err := move(NutsContainerName, box.Nuts...); err != nil {
		return err
	}
	if err := move(ShaftsContainerName, box.InputShaft); err != nil {
		return err
	}
	for _, s := range box.Stages {
		if err := move(SunGearsContainerName, s.Sun); err != nil {
			return err
		}
		if err := move(PlanetGearsContainerName, s.Planets...); err != nil {
			return err
		}
		if err := move(PlanetBearingsContainerName, s.Bearings...); err != nil {
			return err
		}
		if err := move(CarriersContainerName, s.Carrier); err != nil {
			return err
		}
		if err := move(ShaftsContainerName, s.Shaft); err != nil {
			return err
		}
	}
	return nil
}
