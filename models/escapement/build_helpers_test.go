package escapement_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/escapement"
	"github.com/stretchr/testify/require"
)

type built struct {
	doc *decad.Document
	m   *escapement.Movement
}

// defaultMovement returns the default movement, built once for the whole test binary.
func defaultMovement(t *testing.T) built {
	t.Helper()
	doc, m, err := escapement.SharedDefault(t)
	require.NoError(t, err)
	return built{doc, m}
}

// expectedNames lists the bodies in the order of the Bodies table of instructions.md, each with its container path.
var expectedNames = []struct{ name, path string }{
	{"Back Plate", "Frame"}, {"Front Plate", "Frame"},
	{"Back Escape Bearing", "Bearings"}, {"Back Pallet Bearing", "Bearings"}, {"Back Great Wheel Bearing", "Bearings"},
	{"Front Escape Bearing", "Bearings"}, {"Front Pallet Bearing", "Bearings"},
	{"Front Great Wheel Bearing", "Bearings"}, {"Pendulum Bearing", "Bearings"},
	{"Escape Arbor", "Arbors"}, {"Pallet Arbor", "Arbors"}, {"Great Wheel Arbor", "Arbors"},
	{"Escape Wheel", "Escapement/Escape Wheel"}, {"Anchor", "Escapement/Anchor"},
	{"Crutch", "Pendulum Assembly/Crutch"}, {"Pendulum", "Pendulum Assembly/Pendulum"},
	{"Suspension Pin", "Pendulum Assembly/Suspension"}, {"Suspension Cap", "Pendulum Assembly/Suspension"},
	{"Great Wheel", "Drive/Great Wheel"}, {"Pawl Carrier", "Drive/Pawl Carrier"}, {"Mainspring", "Drive/Mainspring"},
	{"Spring Retainer", "Drive/Spring Retainer"}, {"Spring Anchor Pin", "Drive/Spring Anchor"}, {"Winding Knob", "Drive/Winding Knob"},
	{"Crutch Collar", "Collars"}, {"Knob Collar", "Collars"},
	{"Bolt 1", "Bolts"}, {"Bolt 2", "Bolts"}, {"Bolt 3", "Bolts"}, {"Bolt 4", "Bolts"},
	{"Nut 1", "Nuts"}, {"Nut 2", "Nuts"}, {"Nut 3", "Nuts"}, {"Nut 4", "Nuts"},
}

func requireInventory(t *testing.T, doc *decad.Document, m *escapement.Movement) {
	t.Helper()
	named := m.Named()
	require.Len(t, named, len(expectedNames))
	seen := map[string]struct{}{}
	for i, nb := range named {
		require.Equal(t, expectedNames[i].name, nb.Name)
		_, dup := seen[nb.Name]
		require.False(t, dup, "duplicate name %s", nb.Name)
		seen[nb.Name] = struct{}{}
		require.True(t, nb.Body.IsSolid(), "%s is a solid", nb.Name)
		c, err := doc.ContainerOf(nb.Body)
		require.NoError(t, err)
		// Path is Root-relative: strip the root's slash and compare with the table.
		require.Equal(t, "/"+expectedNames[i].path, c.Path(), "container of %s", nb.Name)
	}
	require.Len(t, doc.Bodies(), len(expectedNames), "only the movement's bodies are live")
	require.NoError(t, doc.VerifyContainers().Err())
}
