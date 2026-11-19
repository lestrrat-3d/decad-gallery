package turbinefan

// CI: isolated-tests

import (
	"math/big"
	"testing"
)

func TestBladeRootIntervalPlane(t *testing.T) {
	r := func(v int64) *big.Rat { return big.NewRat(v, 1) }
	p := bladeProofPatch{z0: 0, z1: 10, ax: []*big.Rat{r(1), r(1)}, ay: []*big.Rat{r(1)},
		bx: []*big.Rat{r(1), r(1)}, by: []*big.Rat{r(1)}, w: []*big.Rat{r(1)}}
	if err := bladeRootCurvature(p, 3); err != nil {
		t.Fatal(err)
	}
}

func TestBladeRootIntervalRejectsTightBend(t *testing.T) {
	r := func(v int64) *big.Rat { return big.NewRat(v, 1) }
	p := bladeProofPatch{z0: 0, z1: .5, ax: []*big.Rat{r(1), r(1)}, ay: []*big.Rat{r(0)},
		bx: []*big.Rat{r(1), r(1)}, by: []*big.Rat{r(1)}, w: []*big.Rat{r(1)}}
	if err := bladeRootCurvature(p, 3); err == nil {
		t.Fatal("tight inward bend was accepted")
	}
}
