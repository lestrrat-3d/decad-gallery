package spinningwheel

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
)

// connectorContainerName holds the flexible connector that BuildConnector builds alone.
const connectorContainerName = "Flexible Connector"

// containerNames maps a Part.Group to the plural name of the container that holds its bodies.
var containerNames = map[string]string{
	"adjustment":  "Adjustments",
	"anchor":      "Anchors",
	"bearing":     "Bearings",
	"bobbin":      "Bobbin",
	"bush":        "Bushes",
	"brake-band":  "Brake Bands",
	"brake-guide": "Brake Guides",
	"collar":      "Collars",
	"compliant":   connectorContainerName,
	"crank":       "Cranks",
	"drive-band":  "Drive Bands",
	"fastener":    "Fasteners",
	"flyer":       "Flyer",
	"footman":     "Footman",
	"frame":       "Frame",
	"gimbal":      "Gimbal",
	"guide":       "Yarn Guides",
	"hinge":       "Hinges",
	"insert":      "Inserts",
	"key":         "Keys",
	"marker":      "Markers",
	"pin":         "Pins",
	"retainer":    "Retainers",
	"shaft":       "Shafts",
	"spacer":      "Spacers",
	"spring":      "Springs",
	"treadle":     "Treadle",
	"wheel":       "Wheel",
	"whorl":       "Whorls",
	"yarn":        "Yarn",
	"yarn-pack":   "Yarn Pack",
}

// placeParts moves every recorded part into the container for its group. It runs after the last operation, because
// decad refuses a Boolean whose operands sit in different containers.
func (b *builder) placeParts() {
	if b.err != nil {
		return
	}
	for _, part := range b.a.Parts {
		b.name = part.Name
		name, ok := containerNames[part.Group]
		if !ok {
			b.fail("place part", fmt.Errorf("no container for group %q", part.Group))
			return
		}
		container, err := b.doc.Root().ChildNamed(name)
		if err != nil {
			b.fail("create container", err)
			return
		}
		b.fail("place part", b.doc.MoveBody(part.Body, container))
	}
}

// placeConnector moves the connector body into its own container.
func placeConnector(doc *decad.Document, body *decad.Body) error {
	container, err := doc.Root().ChildNamed(connectorContainerName)
	if err != nil {
		return fmt.Errorf("create container %q: %w", connectorContainerName, err)
	}
	if err := doc.MoveBody(body, container); err != nil {
		return fmt.Errorf("place connector in container %q: %w", connectorContainerName, err)
	}
	return nil
}
