package escapement_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/escapement"
	"github.com/lestrrat-3d/r3"
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
	{"Spring Spacer", "Drive/Spring Spacer"}, {"Spring Anchor Pin", "Drive/Spring Anchor"}, {"Winding Knob", "Drive/Winding Knob"},
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

func TestInventory(t *testing.T) {
	d := defaultMovement(t)
	requireInventory(t, d.doc, d.m)

	p := escapement.DefaultParams()
	p.Phase = 0.3
	doc := escapement.NewDocument()
	m, err := escapement.Build(t.Context(), doc, p)
	require.NoError(t, err)
	requireInventory(t, doc, m)
	require.Equal(t, 0.3, m.State.Phase)
}

func TestBuildValidates(t *testing.T) {
	doc := escapement.NewDocument()
	p := escapement.DefaultParams()
	p.PendulumLength = 10
	_, err := escapement.Build(t.Context(), doc, p)
	require.Error(t, err)
	require.Contains(t, err.Error(), "PendulumLength")
	require.Empty(t, doc.Bodies(), "Build rejects the parameters before it touches the document")
	_, err = escapement.Build(nil, doc, escapement.DefaultParams()) //nolint:staticcheck // the nil context is the case under test
	require.Error(t, err)
	_, err = escapement.Build(t.Context(), nil, escapement.DefaultParams())
	require.Error(t, err)
}

func TestSketches(t *testing.T) {
	d := defaultMovement(t)
	require.NotEmpty(t, d.m.Sketches)
	for i, s := range d.m.Sketches {
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
	}
}

func TestPlacement(t *testing.T) {
	d := defaultMovement(t)
	frame, err := r3.NewFrame(r3.NewVec(10, -5, 7), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	p := escapement.DefaultParams()
	p.Placement = &frame
	doc := escapement.NewDocument()
	m, err := escapement.Build(t.Context(), doc, p)
	require.NoError(t, err)
	requireInventory(t, doc, m)

	want, got := d.m.Named(), m.Named()
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
