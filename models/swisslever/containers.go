package swisslever

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
)

// Container names. Each path below the document root names a kind of body.
const (
	FrameContainerName              = "Frame"
	BearingsContainerName           = "Bearings"
	ArborsContainerName             = "Arbors"
	EscapementContainerName         = "Escapement"
	EscapeWheelContainerName        = "Escape Wheel"
	LeverContainerName              = "Lever"
	PalletStonesContainerName       = "Pallet Stones"
	BalanceContainerName            = "Balance"
	RollersContainerName            = "Rollers"
	BalanceWheelContainerName       = "Balance Wheel"
	HairspringContainerName         = "Hairspring"
	HairspringRetainerContainerName = "Hairspring Retainer"
	DriveContainerName              = "Drive"
	WindingKnobContainerName        = "Winding Knob"
	CollarsContainerName            = "Collars"
	BoltsContainerName              = "Bolts"
	NutsContainerName               = "Nuts"
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
			m.BackBalanceBearing, m.FrontEscapeBearing, m.FrontPalletBearing, m.FrontBalanceBearing}},
		{[]string{ArborsContainerName}, []*decad.Body{m.EscapeArbor, m.PalletArbor, m.BalanceStaff}},
		{[]string{EscapementContainerName, EscapeWheelContainerName}, []*decad.Body{m.EscapeWheel}},
		{[]string{EscapementContainerName, LeverContainerName}, []*decad.Body{m.Lever}},
		{[]string{EscapementContainerName, PalletStonesContainerName}, []*decad.Body{m.EntryStone, m.ExitStone}},
		{[]string{BalanceContainerName, RollersContainerName}, []*decad.Body{m.SafetyRoller, m.RollerTable}},
		{[]string{BalanceContainerName, BalanceWheelContainerName}, []*decad.Body{m.BalanceWheel}},
		{[]string{BalanceContainerName, HairspringContainerName}, []*decad.Body{m.Hairspring}},
		{[]string{BalanceContainerName, HairspringRetainerContainerName}, []*decad.Body{m.HairspringRetainer}},
		{[]string{DriveContainerName, WindingKnobContainerName}, []*decad.Body{m.WindingKnob}},
		{[]string{CollarsContainerName}, []*decad.Body{m.KnobCollar}},
		{[]string{BoltsContainerName}, m.Bolts},
		{[]string{NutsContainerName}, m.Nuts},
	} {
		if err := move(e.path, e.bodies...); err != nil {
			return err
		}
	}
	return nil
}
