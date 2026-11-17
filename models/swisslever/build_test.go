package swisslever_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/swisslever"
	"github.com/lestrrat-3d/r3"
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

func TestInventory(t *testing.T) {
	doc, m := swisslever.SharedAt(t, swisslever.DefaultParams().Phase)
	requireInventory(t, doc, m)

	doc, m = swisslever.SharedAt(t, 0.6)
	requireInventory(t, doc, m)
	require.Equal(t, 0.6, m.State.Phase)
}

func TestBuildValidates(t *testing.T) {
	doc := swisslever.NewDocument()
	p := swisslever.DefaultParams()
	p.Amplitude = 0.1
	_, err := swisslever.Build(t.Context(), doc, p)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Amplitude")
	require.Empty(t, doc.Bodies(), "Build rejects the parameters before it touches the document")
	_, err = swisslever.Build(nil, doc, swisslever.DefaultParams()) //nolint:staticcheck // the nil context is the case under test
	require.Error(t, err)
	_, err = swisslever.Build(t.Context(), nil, swisslever.DefaultParams())
	require.Error(t, err)
	require.Empty(t, doc.Bodies())
}

func TestSketches(t *testing.T) {
	_, m := swisslever.SharedAt(t, swisslever.DefaultParams().Phase)
	require.NotEmpty(t, m.Sketches)
	for i, s := range m.Sketches {
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
	}
}

func TestPlacement(t *testing.T) {
	_, base := swisslever.SharedAt(t, swisslever.DefaultParams().Phase)
	frame, err := r3.NewFrame(r3.NewVec(10, -5, 7), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	p := swisslever.DefaultParams()
	p.Placement = &frame
	doc, m, err := swisslever.BuildForTest(t, p)
	require.NoError(t, err)
	requireInventory(t, doc, m)

	want, got := base.Named(), m.Named()
	for i := range want {
		wv, err := want[i].Body.Volume()
		require.NoError(t, err)
		gv, err := got[i].Body.Volume()
		require.NoError(t, err)
		require.InDelta(t, wv.Value.Base(), gv.Value.Base(), 1e-3*wv.Value.Base(), "volume of %s", want[i].Name)
		wb, err := want[i].Body.Bounds()
		require.NoError(t, err)
		gb, err := got[i].Body.Bounds()
		require.NoError(t, err)
		// The frame maps (x, y, z) to (10 + z, -5 + x, 7 + y).
		require.InDelta(t, 10+wb.Min.Z, gb.Min.X, 2e-3, "bounds of %s", want[i].Name)
		require.InDelta(t, 10+wb.Max.Z, gb.Max.X, 2e-3, want[i].Name)
		require.InDelta(t, -5+wb.Min.X, gb.Min.Y, 2e-3, want[i].Name)
		require.InDelta(t, -5+wb.Max.X, gb.Max.Y, 2e-3, want[i].Name)
		require.InDelta(t, 7+wb.Min.Y, gb.Min.Z, 2e-3, want[i].Name)
		require.InDelta(t, 7+wb.Max.Y, gb.Max.Z, 2e-3, want[i].Name)
	}
}
