package turbinefan

// CI: isolated-tests

import (
	"fmt"
	"math"
	"math/big"
	"testing"
)

// Reflection preserves distances, curvatures and seam angles, so the right-hand
// certificate also covers the mirrored blade.
func TestBladeContinuousGeometry(t *testing.T) {
	for _, diameter := range []float64{128, 192, 224} {
		for _, pitch := range []float64{MinBladePitch, DefaultBladePitch, MaxBladePitch} {
			t.Run(fmt.Sprintf("%.0f/%.0f", diameter, pitch*180/math.Pi), func(t *testing.T) {
				requireBladeContinuousGeometry(t, diameter, pitch)
			})
		}
	}
}

func TestNearBedRejectsTightBend(t *testing.T) {
	r := bladeCertificateRow{Band: 4, Chart: 4, Face: 0, Z0: 0, Z1: 1,
		W: []*big.Rat{bprInt(1)}, AX: []*big.Rat{bprInt(2), bprInt(1)},
		BX: []*big.Rat{bprInt(2), bprInt(1)}, AY: []*big.Rat{bprInt(0)}, BY: []*big.Rat{bprInt(20)}}
	if err := bladeNearBedCurvature(r, 0, 3, .5); err == nil {
		t.Fatal("accepted a 1/120 mm inward radius")
	}
}
