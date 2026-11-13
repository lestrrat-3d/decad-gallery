package planetarygearbox_test

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/stretchr/testify/require"
)

func TestProblemsNameForcedMinimums(t *testing.T) {
	require.Empty(t, planetarygearbox.Problems(planetarygearbox.DefaultParams()))

	shaft := planetarygearbox.DefaultParams()
	shaft.ShaftBearing = "608ZZ"
	pocket := planetarygearbox.DefaultParams()
	pocket.PlanetBearing = "608ZZ"
	pocket.SunTeeth, pocket.PlanetTeeth, pocket.RingTeeth = 13, 26, 65
	lip := planetarygearbox.DefaultParams()
	lip.PlanetBearing = "608ZZ"
	lip.GearThickness = 7.5

	for name, tc := range map[string]struct {
		p    planetarygearbox.Params
		want []string
	}{
		"sun wall": {shaft, []string{
			"shaft bore leaves less than one module",
			"SunTeeth must be at least 14 for the 8.1 mm shaft at module 1",
			"try 14/28/70",
		}},
		"planet pocket wall": {pocket, []string{"PlanetTeeth must be at least 28 for the 608ZZ planet bearing"}},
		"lip":                {lip, []string{"a 608ZZ planet bearing needs GearThickness >= 8"}},
	} {
		t.Run(name, func(t *testing.T) {
			probs := planetarygearbox.Problems(tc.p)
			require.NotEmpty(t, probs)
			for _, want := range tc.want {
				require.Contains(t, probs[0], want)
			}
			require.EqualError(t, tc.p.Validate(), probs[0])
		})
	}

	t.Run("input problems stop before the fit checks", func(t *testing.T) {
		p := planetarygearbox.DefaultParams()
		p.ShaftBearing = "608ZZ"
		p.SunTeeth, p.PlanetTeeth, p.RingTeeth = 14, 29, 72
		probs := planetarygearbox.Problems(p)
		require.Len(t, probs, 1)
		require.Contains(t, probs[0], "must divide by planet count")
	})
	t.Run("planet count zero does not panic", func(t *testing.T) {
		p := planetarygearbox.DefaultParams()
		p.PlanetCount = 0
		require.NotPanics(t, func() { require.NotEmpty(t, planetarygearbox.Problems(p)) })
	})
	t.Run("unknown bearing is reported once per role", func(t *testing.T) {
		p := planetarygearbox.DefaultParams()
		p.PlanetBearing, p.ShaftBearing = "609ZZ", "610ZZ"
		probs := planetarygearbox.Problems(p)
		require.Len(t, probs, 2)
		require.Contains(t, probs[0], `planet bearing "609ZZ" is not in the bearing catalog; choose one of 605ZZ, 608ZZ`)
		require.Contains(t, probs[1], `shaft bearing "610ZZ"`)
	})
}
