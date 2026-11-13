package planetarygearbox

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupportedCombinationsAreValid(t *testing.T) {
	combos := SupportedCombinations()
	names := make([]string, 0, len(combos))
	for _, k := range combos {
		names = append(names, k.Name)
	}
	require.Equal(t, []string{
		"planet605-shaft605", "planet605-shaft608", "planet608-shaft605", "planet608-shaft608",
	}, names)

	first, err := DefaultParamsFor(combos[0].Name)
	require.NoError(t, err)
	require.Equal(t, DefaultParams(), first)

	for _, k := range combos {
		t.Run(k.Name, func(t *testing.T) {
			p, err := DefaultParamsFor(k.Name)
			require.NoError(t, err)
			require.NoError(t, p.Validate())
			require.Equal(t, k.PlanetBearing, p.PlanetBearing)
			require.Equal(t, k.ShaftBearing, p.ShaftBearing)
			require.InDelta(t, 1+float64(p.RingTeeth)/float64(p.SunTeeth), p.Ratio(), 1e-12)
			requireTableMatchesDerive(t, p)
		})
	}

	p, err := DefaultParamsFor("planet605-shaft608")
	require.NoError(t, err)
	d := derive(p)
	require.InDelta(t, 17.05, d.coverHub/2, 1e-9)
	require.InDelta(t, 29.35, d.rimInner-d.arm, 1e-9)
	require.InDelta(t, 28.1, d.coverBoss, 1e-9)
	require.Less(t, d.coverHub/2, d.rimInner-d.arm)

	p, err = DefaultParamsFor("planet608-shaft608")
	require.NoError(t, err)
	d = derive(p)
	require.InDelta(t, 26.35, d.rimInner-d.arm, 1e-9)
	require.InDelta(t, 0.43, d.planetRoot-p.Module-d.planetPocket/2, 1e-9)

	_, err = DefaultParamsFor("x")
	require.ErrorContains(t, err, `unknown kit "x"`)
}
