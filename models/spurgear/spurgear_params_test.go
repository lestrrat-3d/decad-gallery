package spurgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/stretchr/testify/require"
)

// The build reads the gear's dimensions and feature values from the document's
// parameter table, so a generator can read the names and expressions.
func TestBuildReadsParameterTable(t *testing.T) {
	p := spurgear.DefaultParams()
	p.BoreDiameter = 3
	p.CapChamfer = 0.1
	doc := decad.New()
	_, err := spurgear.Build(t.Context(), doc, p)
	require.NoError(t, err)

	read := map[string]string{}
	for _, b := range doc.Bindings() {
		read[b.Name] = b.Expr
	}
	for name, expr := range map[string]string{
		"root_radius":      "(module * teeth - 2.5 * module) / 2",
		"fillet_radius":    "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2",
		"bore_tool_length": "thickness + 2 * tool_margin",
	} {
		require.Equal(t, expr, read[name], name)
	}
	for _, name := range []string{"teeth", "thickness", "cap_chamfer"} {
		require.Contains(t, read, name)
	}

	root, err := doc.Param("root_radius")
	require.NoError(t, err)
	require.InDelta(t, 7.25, root.Base(), 1e-12)
	teeth, err := doc.ParamInt("teeth")
	require.NoError(t, err)
	require.Equal(t, p.Teeth, teeth)
}

// Editing an input rebuilds the model with the derived values that follow.
func TestParameterTableFollowsInputs(t *testing.T) {
	p := spurgear.DefaultParams()
	p.Teeth = 20
	doc := decad.New()
	_, err := spurgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	root, err := doc.Param("root_radius")
	require.NoError(t, err)
	require.InDelta(t, 8.75, root.Base(), 1e-12)
}
