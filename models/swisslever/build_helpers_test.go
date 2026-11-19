package swisslever_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/swisslever"
	"github.com/stretchr/testify/require"
)

// expectedNames lists the bodies in the order of the Bodies table of instructions.md, each with its container path.
var expectedNames = []struct{ name, path string }{
	{"Back Plate", "Frame"}, {"Front Plate", "Frame"},
	{"Back Escape Bearing", "Bearings"}, {"Back Pallet Bearing", "Bearings"}, {"Back Balance Bearing", "Bearings"},
	{"Front Escape Bearing", "Bearings"}, {"Front Pallet Bearing", "Bearings"}, {"Front Balance Bearing", "Bearings"},
	{"Escape Arbor", "Arbors"}, {"Pallet Arbor", "Arbors"}, {"Balance Staff", "Arbors"},
	{"Escape Wheel", "Escapement/Escape Wheel"}, {"Lever", "Escapement/Lever"},
	{"Entry Stone", "Escapement/Pallet Stones"}, {"Exit Stone", "Escapement/Pallet Stones"},
	{"Safety Roller", "Balance/Rollers"}, {"Roller Table", "Balance/Rollers"},
	{"Balance Wheel", "Balance/Balance Wheel"}, {"Hairspring", "Balance/Hairspring"},
	{"Hairspring Retainer", "Balance/Hairspring Retainer"},
	{"Winding Knob", "Drive/Winding Knob"}, {"Knob Collar", "Collars"},
	{"Bolt 1", "Bolts"}, {"Bolt 2", "Bolts"}, {"Bolt 3", "Bolts"}, {"Bolt 4", "Bolts"},
	{"Nut 1", "Nuts"}, {"Nut 2", "Nuts"}, {"Nut 3", "Nuts"}, {"Nut 4", "Nuts"},
}

func requireInventory(t *testing.T, doc *decad.Document, m *swisslever.Movement) {
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
		require.Equal(t, "/"+expectedNames[i].path, c.Path(), "container of %s", nb.Name)
	}
	require.Len(t, doc.Bodies(), len(expectedNames), "only the movement's bodies are live")
	require.NoError(t, doc.VerifyContainers().Err())
}
