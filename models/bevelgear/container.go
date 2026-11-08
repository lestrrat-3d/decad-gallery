package bevelgear

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch"
)

// Container names for a pair and its two members.
const (
	pairContainerName    = "Pair"
	drivingContainerName = "Driving Gear"
	pinionContainerName  = "Pinion Gear"
)

// pairContainers groups one pair: a container for the pair and one inside it for each member. A
// document that already holds a container of that name gets "name 2", "name 3" and so on
// (decad's NewUniqueChild), because sibling names are unique.
type pairContainers struct{ pair, driving, pinion *decad.Container }

func newPairContainers(doc *decad.Document) (pairContainers, error) {
	pair, err := doc.Root().NewUniqueChild(pairContainerName)
	if err != nil {
		return pairContainers{}, err
	}
	driving, err := pair.NewUniqueChild(drivingContainerName)
	if err != nil {
		return pairContainers{}, err
	}
	pinion, err := pair.NewUniqueChild(pinionContainerName)
	if err != nil {
		return pairContainers{}, err
	}
	return pairContainers{pair: pair, driving: driving, pinion: pinion}, nil
}

// addSketch places s in c. A nil c leaves s in the document's root container.
func addSketch(c *decad.Container, s *sketch.Sketch) error {
	if c == nil {
		return nil
	}
	if err := c.AddSketch(s); err != nil {
		return fmt.Errorf("place sketch in container %q: %w", c.Name(), err)
	}
	return nil
}

// addPlane places p in c. A nil c leaves p in the document's root container.
func addPlane(c *decad.Container, p *sketch.Plane) error {
	if c == nil {
		return nil
	}
	if err := c.AddPlane(p); err != nil {
		return fmt.Errorf("place plane in container %q: %w", c.Name(), err)
	}
	return nil
}

// moveBody puts body in c and returns it. A nil c leaves body where it is.
func moveBody(doc *decad.Document, body *decad.Body, c *decad.Container) (*decad.Body, error) {
	if c == nil {
		return body, nil
	}
	if err := doc.MoveBody(body, c); err != nil {
		return nil, fmt.Errorf("place body in container %q: %w", c.Name(), err)
	}
	return body, nil
}
