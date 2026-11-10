package handcranksiren

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
)

// Container names, one per kind of body.
const (
	HousingContainerName   = "Housing" // gearbox-front, gearbox-frame, gearbox-rear, drum
	StatorContainerName    = "Stator"
	RotorContainerName     = "Rotor"
	HornContainerName      = "Horn"
	GearPairsContainerName = "Gear Pairs" // children "Pair" (stage 1) and "Pair 2" (stage 2)
	PairContainerName      = "Pair"
	ShaftsContainerName    = "Shafts"
	BearingsContainerName  = "Bearings"
	CrankContainerName     = "Crank" // crank-arm, crank-pin, crank-handle
	BoltsContainerName     = "Bolts"
	NutsContainerName      = "Nuts"
)

// pairOf returns 1 or 2 for the gears of stage 1 and stage 2, and 0 for any other part.
func pairOf(name string) int {
	switch name {
	case "crank-gear", "intermediate-pinion":
		return 1
	case "intermediate-gear", "rotor-pinion":
		return 2
	}
	return 0
}

// moveBodies puts every part's body in the container for its kind. It creates a container when it adds that
// container's first body, so no container is ever empty. Decad cuts only within one container, so it runs last.
func moveBodies(doc *decad.Document, parts []Part) error {
	root := doc.Root()
	pairs := map[int]*decad.Container{}
	for _, part := range parts {
		var target *decad.Container
		if n := pairOf(part.Name); n != 0 {
			if pairs[n] == nil {
				group, err := root.ChildNamed(GearPairsContainerName)
				if err != nil {
					return fmt.Errorf("container %q: %w", GearPairsContainerName, err)
				}
				pair, err := group.NewUniqueChild(PairContainerName)
				if err != nil {
					return fmt.Errorf("container %q: %w", PairContainerName, err)
				}
				pairs[n] = pair
			}
			target = pairs[n]
		} else {
			c, err := root.ChildNamed(part.Group)
			if err != nil {
				return fmt.Errorf("container %q: %w", part.Group, err)
			}
			target = c
		}
		if err := target.AddBody(part.Body); err != nil {
			return fmt.Errorf("place %s in container %q: %w", part.Name, target.Name(), err)
		}
	}
	return nil
}
