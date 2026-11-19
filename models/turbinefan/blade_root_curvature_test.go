package turbinefan

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
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

func requireBladeContinuousGeometry(t *testing.T, diameter, pitch float64) {
	t.Helper()
	size, err := SizeFor(diameter)
	require.NoError(t, err)
	inspected := false
	_, err = buildBlade(t.Context(), NewDocument(), pitch, size,
		func(sections []decad.LoftSection, correspondence decad.CircularLoftCorrespondence) error {
			rows, err := bladeCertificateRows(sections, correspondence)
			if err != nil {
				return err
			}
			frame, err := sections[0].Sketch.Plane().Frame()
			if err != nil {
				return err
			}
			if err := bladeWorkingSeams(rows, frame.Origin().X, size.BladeStudRadius); err != nil {
				return err
			}
			nearBed := 0
			for _, row := range rows {
				if row.Band >= 3 && row.Face < 4 {
					if err := bladeNearBedCurvature(row, frame.Origin().X, size.BladeStudRadius,
						math.Nextafter(10.0/19, math.Inf(1))); err != nil {
						return err
					}
					nearBed++
				}
			}
			if nearBed != 180 {
				return fmt.Errorf("near-bed patch count %d, want 180", nearBed)
			}
			count := 0
			for _, row := range rows {
				if row.Band != 3 || row.Face >= 4 {
					continue
				}
				patch := bladeProofPatch{band: row.Band, face: row.Face, z0: row.Z0, z1: row.Z1, ax: row.AX, ay: row.AY, bx: row.BX, by: row.BY, w: row.W, linear: row.Linear}
				if err := bladeRootCurvature(patch, size.BladeStudRadius); err != nil {
					return err
				}
				count++
			}
			if count != 20 {
				return fmt.Errorf("root patch count %d, want 20", count)
			}
			inspected = true
			return nil
		})
	require.NoError(t, err)
	require.True(t, inspected)
}

func TestNearBedRejectsTightBend(t *testing.T) {
	r := bladeCertificateRow{Band: 4, Chart: 4, Face: 0, Z0: 0, Z1: 1,
		W: []*big.Rat{bprInt(1)}, AX: []*big.Rat{bprInt(2), bprInt(1)},
		BX: []*big.Rat{bprInt(2), bprInt(1)}, AY: []*big.Rat{bprInt(0)}, BY: []*big.Rat{bprInt(20)}}
	if err := bladeNearBedCurvature(r, 0, 3, .5); err == nil {
		t.Fatal("accepted a 1/120 mm inward radius")
	}
}
