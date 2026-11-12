package escapement

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
)

// Container names. Each path below the document root names a kind of body.
const (
	FrameContainerName       = "Frame"
	BearingsContainerName    = "Bearings"
	ArborsContainerName      = "Arbors"
	EscapementContainerName  = "Escapement"
	EscapeWheelContainerName = "Escape Wheel"
	AnchorContainerName      = "Anchor"
	PendulumAssemblyName     = "Pendulum Assembly"
	CrutchContainerName      = "Crutch"
	PendulumContainerName    = "Pendulum"
	SuspensionContainerName  = "Suspension"
	DriveContainerName       = "Drive"
	GreatWheelContainerName  = "Great Wheel"
	DrumContainerName        = "Drum"
	WindingKnobContainerName = "Winding Knob"
	BoltsContainerName       = "Bolts"
	NutsContainerName        = "Nuts"
)

// moveBodies puts every body of m in the container for its kind.
func moveBodies(doc *decad.Document, m *Movement) error {
	move := func(path []string, bodies ...*decad.Body) error {
		c := doc.Root()
		for _, name := range path {
			var err error
			if c, err = c.ChildNamed(name); err != nil {
				return fmt.Errorf("container %q: %w", name, err)
			}
		}
		for _, b := range bodies {
			if err := c.AddBody(b); err != nil {
				return fmt.Errorf("place body in container %q: %w", c.Path(), err)
			}
		}
		return nil
	}
	for _, e := range []struct {
		path   []string
		bodies []*decad.Body
	}{
		{[]string{FrameContainerName}, []*decad.Body{m.BackPlate, m.FrontPlate}},
		{[]string{BearingsContainerName}, []*decad.Body{m.BackEscapeBearing, m.BackPalletBearing,
			m.BackGreatWheelBearing, m.FrontEscapeBearing, m.FrontPalletBearing, m.FrontGreatWheelBearing,
			m.PendulumBearing}},
		{[]string{ArborsContainerName}, []*decad.Body{m.EscapeArbor, m.PalletArbor, m.GreatWheelArbor}},
		{[]string{EscapementContainerName, EscapeWheelContainerName}, []*decad.Body{m.EscapeWheel}},
		{[]string{EscapementContainerName, AnchorContainerName}, []*decad.Body{m.Anchor}},
		{[]string{PendulumAssemblyName, CrutchContainerName}, []*decad.Body{m.Crutch}},
		{[]string{PendulumAssemblyName, PendulumContainerName}, []*decad.Body{m.Pendulum}},
		{[]string{PendulumAssemblyName, SuspensionContainerName}, []*decad.Body{m.SuspensionPin, m.SuspensionCap}},
		{[]string{DriveContainerName, GreatWheelContainerName}, []*decad.Body{m.GreatWheel}},
		{[]string{DriveContainerName, DrumContainerName}, []*decad.Body{m.Drum}},
		{[]string{DriveContainerName, WindingKnobContainerName}, []*decad.Body{m.WindingKnob}},
		{[]string{BoltsContainerName}, m.Bolts},
		{[]string{NutsContainerName}, m.Nuts},
	} {
		if err := move(e.path, e.bodies...); err != nil {
			return err
		}
	}
	return nil
}
