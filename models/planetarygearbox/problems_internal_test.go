package planetarygearbox

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewFitChecks(t *testing.T) {
	for name, tc := range map[string]struct {
		pass, fail string
		want       string
	}{
		"shaft collar vs inner race": {
			collarRaceProblem(7.4, 8, "605ZZ"), collarRaceProblem(7.4, 7.0, "605ZZ"),
			"the 7.4 mm shaft collar is not narrower than the 7 mm inner race of the 605ZZ shaft bearing",
		},
		"pin collar vs inner race": {
			pinRaceProblem(7.4, 8, "605ZZ"), pinRaceProblem(10.4, 10, "608ZZ"),
			"the 10.4 mm pin thick part is not narrower than the 10 mm inner race of the 608ZZ planet bearing",
		},
		"shaft collar through cover lip": {
			collarLipProblem(7.4, 10), collarLipProblem(7.4, 7.8),
			"the 7.4 mm shaft collar does not pass the 7.8 mm cover lip bore with 0.25 on each side",
		},
		"centre hole vs pin sockets": {
			centreHoleProblem(21, 7.4, 7.4), centreHoleProblem(8, 7.4, 7.4),
			"the carrier's centre hole and pin sockets leave 0.5 of plate between them, less than 1",
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.Empty(t, tc.pass)
			require.Equal(t, tc.want, tc.fail)
		})
	}

	for _, k := range SupportedCombinations() {
		p, err := DefaultParamsFor(k.Name)
		require.NoError(t, err)
		d := derive(p)
		require.Empty(t, collarRaceProblem(d.collar, shaftBearing(p).InnerRaceDiameter, p.ShaftBearing), k.Name)
		require.Empty(t, pinRaceProblem(d.pinCollar, planetBearing(p).InnerRaceDiameter, p.PlanetBearing), k.Name)
		require.Empty(t, collarLipProblem(d.collar, d.coverLip), k.Name)
		require.Empty(t, centreHoleProblem(d.orbit, d.collar, d.pinCollar), k.Name)
	}
}

// TestEarSpacingCountsRootRounds checks that the ear-spacing problem accounts for the rounds where each ear meets
// the ring. With the default ring, 17 ears are 18.0 mm apart: more than the 11 mm the ears and their gap need, less
// than the 18.1 mm they need with their 3.55 mm root rounds.
func TestEarSpacingCountsRootRounds(t *testing.T) {
	p := DefaultParams()
	d := derive(p)
	half := earHalfWidth(p.EarBoreDiameter)
	p.EarCount = 17
	chord := 2 * earCenterRadius(d.ringOuter, p.EarBoreDiameter) * math.Sin(math.Pi/17)
	require.Greater(t, chord, 2*half+earGap)
	require.Less(t, chord, 2*(half+earRootRound(p))+earGap)
	require.Contains(t, strings.Join(earProblems(p, d), "\n"), "neighbouring ear bores")

	p.EarCount = 16
	require.NotContains(t, strings.Join(earProblems(p, d), "\n"), "neighbouring ear bores")
}
