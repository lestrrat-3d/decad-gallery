package handcrankfan_test

// CI: isolated-tests

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/stretchr/testify/require"
)

func TestHandCrankFanDefault(t *testing.T) {
	skipUnlessFull(t)
	checkFan(t, handcrankfan.DefaultParams())
}

// TestHandCrankFanContainers checks the containers of the shared default build, which takes about two minutes.
func TestHandCrankFanContainers(t *testing.T) {
	doc, _, a := sharedDefault(t)
	requireContainers(t, doc, a)
}

func TestHandCrankFanVariant(t *testing.T) {
	skipUnlessFull(t)
	variants := map[string]handcrankfan.Params{
		"flat-three":  {CrankAngle: -0.7, BladePitch: 12 * math.Pi / 180},
		"steep-three": {CrankAngle: 2.5, BladePitch: 40 * math.Pi / 180},
	}
	if expected := os.Getenv("HANDCRANKFAN_VARIANTS"); expected != "" {
		names := make([]string, 0, len(variants))
		for name := range variants {
			names = append(names, name)
		}
		require.ElementsMatch(t, strings.Split(expected, ","), names)
	}
	for name, p := range variants {
		t.Run(name, func(t *testing.T) { checkFan(t, p) })
	}
}
