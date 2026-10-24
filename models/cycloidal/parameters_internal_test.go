package cycloidal

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// The parameter table holds the same derived dimensions the build reads from resolve.
func TestParameterTableMatchesResolve(t *testing.T) {
	for name, p := range map[string]Params{
		"default": DefaultParams(),
		"pin override": func() Params {
			p := DefaultParams()
			p.PinDiameter, p.OutputPinDiameter = 8, 6
			return p
		}(),
		"two discs": func() Params {
			p := DefaultParams()
			p.DiscCount = 2
			return p
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			doc := decad.New()
			require.NoError(t, recordParameters(doc, p))
			d := resolve(p)
			for want, got := range map[string]string{
				"pin_radius":                   "pin_radius",
				"output_hole_diameter":         "output_hole_diameter",
				"root_circle_diameter":         "root_circle_diameter",
				"housing_inner_diameter":       "housing_inner_diameter",
				"housing_outer_diameter":       "housing_outer_diameter",
				"resolved_output_pin_diameter": "resolved_output_pin_diameter",
				"output_plate_diameter":        "output_plate_diameter",
			} {
				v, err := doc.Param(got)
				require.NoError(t, err, want)
				expected := map[string]float64{
					"pin_radius":                   d.pinRadius,
					"output_hole_diameter":         d.outputHoleDiameter,
					"root_circle_diameter":         2 * d.rootRadius,
					"housing_inner_diameter":       d.housingInner,
					"housing_outer_diameter":       d.housingOuter,
					"resolved_output_pin_diameter": d.outputPinDiameter,
					"output_plate_diameter":        d.outputPlateDiameter,
				}[want]
				require.InDelta(t, expected, v.Base(), 1e-9, want)
			}
			lobes, err := doc.Param("lobes")
			require.NoError(t, err)
			require.Equal(t, float64(p.PinCount-1), lobes.Base())
			height, err := doc.Param("stack_height")
			require.NoError(t, err)
			require.InDelta(t, float64(p.DiscCount)*p.DiscThickness+float64(p.DiscCount-1)*p.DiscGap, height.Base(), 1e-9)
		})
	}
}
