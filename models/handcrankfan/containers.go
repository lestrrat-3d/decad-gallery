package handcrankfan

import (
	"fmt"
	"strings"

	"github.com/lestrrat-3d/decad"
)

// Container names, one per kind of body.
const (
	HousingContainerName  = "Housing" // gearbox-base, gearbox-post, gearbox-crank-wall, gearbox-fan-wall
	GripContainerName     = "Grip"    // the handle
	CrankContainerName    = "Crank"   // crank-arm, crank-pin, crank-grip
	HubContainerName      = "Hub"     // fan-hub
	BladesContainerName   = "Blades"
	ShaftsContainerName   = "Shafts"
	BearingsContainerName = "Bearings"
	BoltsContainerName    = "Bolts"
	NutsContainerName     = "Nuts"
	SpurPairContainerName = "Spur Pair" // crank-gear and drive-gear
)

// containerName returns the name of the container for a part that is not in the bevel pair.
func containerName(part Part) (string, error) {
	switch part.Name {
	case "crank-gear", "drive-gear":
		return SpurPairContainerName, nil
	}
	switch part.Group {
	case "housing":
		return HousingContainerName, nil
	case "handle":
		return GripContainerName, nil
	case "arm", "pin", "grip":
		return CrankContainerName, nil
	case "rotor":
		return HubContainerName, nil
	case "blade":
		return BladesContainerName, nil
	case "shaft":
		return ShaftsContainerName, nil
	case "bearing":
		return BearingsContainerName, nil
	case "hardware":
		if strings.Contains(part.Name, "nut") {
			return NutsContainerName, nil
		}
		return BoltsContainerName, nil
	}
	return "", fmt.Errorf("no container for group %q", part.Group)
}

// moveBodies puts every part's body in the container for its kind. It creates a container when it adds that
// container's first body, so no container is ever empty. Decad cuts only within one container, so it runs last.
// bevel holds the containers bevelgear.BuildPair made for driving-gear and pinion: the bodies go back into them
// (the pair is the "Pair" container with "Driving Gear" and "Pinion Gear" inside), because Decad cannot remove the
// emptied containers and VerifyContainers rejects an empty one.
func moveBodies(doc *decad.Document, parts []Part, bevel map[string]*decad.Container) error {
	root := doc.Root()
	for _, part := range parts {
		var target *decad.Container
		if c, ok := bevel[part.Name]; ok {
			target = c
		} else {
			name, err := containerName(part)
			if err != nil {
				return fmt.Errorf("place %s: %w", part.Name, err)
			}
			c, err := root.ChildNamed(name)
			if err != nil {
				return fmt.Errorf("container %q: %w", name, err)
			}
			target = c
		}
		if err := target.AddBody(part.Body); err != nil {
			return fmt.Errorf("place %s in container %q: %w", part.Name, target.Name(), err)
		}
	}
	return nil
}
