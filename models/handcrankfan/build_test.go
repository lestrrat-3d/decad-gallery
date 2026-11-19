package handcrankfan_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
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
	for name, p := range map[string]handcrankfan.Params{
		"flat-three":  {CrankAngle: -0.7, BladePitch: 12 * math.Pi / 180},
		"steep-three": {CrankAngle: 2.5, BladePitch: 40 * math.Pi / 180},
	} {
		t.Run(name, func(t *testing.T) { checkFan(t, p) })
	}
}
