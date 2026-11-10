package spurgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
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

func TestVariantBorePreservesParameterExpressions(t *testing.T) {
	p := spurgear.DefaultParams()
	p.Teeth, p.Samples, p.Thickness, p.BoreDiameter = 8, 5, 2, 2
	expressions := map[string]string{
		"root_radius":   "pitch_radius - 1.35 * module",
		"tip_radius":    "pitch_radius + 0.9 * module",
		"fillet_radius": "0 * root_radius",
	}
	variant := spurgear.Variant{Define: func(table *param.Table) error {
		for name, expr := range expressions {
			if err := table.SetExpr(name, expr, units.Millimeter); err != nil {
				return err
			}
		}
		return nil
	}}
	doc := decad.New()
	body, err := variant.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, body.IsSolid())
	bindings := map[string]string{}
	for _, binding := range doc.Bindings() {
		bindings[binding.Name] = binding.Expr
	}
	for name, expr := range expressions {
		require.Equal(t, expr, bindings[name], name)
	}
	for name, expected := range map[string]float64{"root_radius": 2.65, "tip_radius": 4.9, "fillet_radius": 0} {
		value, err := doc.Param(name)
		require.NoError(t, err, name)
		require.InDelta(t, expected, value.Base(), 1e-12, name)
	}
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
