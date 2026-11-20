package handcranksiren_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
)

func TestHandCrankSirenDefault(t *testing.T) { checkSiren(t, handcranksiren.DefaultParams()) }

// TestHandCrankSirenVariant builds the siren with the fewest and the most slots, at two crank angles.
func TestHandCrankSirenVariant(t *testing.T) {
	for name, p := range map[string]handcranksiren.Params{
		"slots 12": {CrankAngle: 0, Slots: 12},
		"slots 24": {CrankAngle: 5 * math.Pi / 4, Slots: 24},
	} {
		t.Run(name, func(t *testing.T) { checkSiren(t, p) })
	}
}
