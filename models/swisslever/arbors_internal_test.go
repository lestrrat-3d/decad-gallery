package swisslever

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/stretchr/testify/require"
)

func TestArbors(t *testing.T) {
	b := arborTestBuilder(t)
	cases := []struct {
		name  string
		build func() (*decad.Body, error)
		at    Pt
		pts   meridianPoints
		spans []flatSpan
		seats [][2]float64 // z ranges of the round 5.0 bearing seats
	}{
		{"escape arbor", b.buildEscapeArbor, b.g.O, escapeArborPoints, escapeArborFlats, [][2]float64{{-5, 0}, {27, 32}}},
		{"pallet arbor", b.buildPalletArbor, b.g.P, palletArborPoints, palletArborFlats, [][2]float64{{-5, 0}, {27, 32}}},
		{"balance staff", b.buildBalanceStaff, b.g.B, palletArborPoints, palletArborFlats, [][2]float64{{-5, 0}, {27, 32}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body, err := c.build()
			require.NoError(t, err)
			require.True(t, body.IsSolid())
			// Volume: the revolve less, over each flat and its ramp, the circular segment of each section beyond it.
			want := meridianVolume(c.pts, c.spans)
			require.InDelta(t, want, arborVolume(t, body), 0.01*want)
			for _, sp := range c.spans {
				mid := (sp.Z0 + sp.Z1) / 2
				r := sectionRadius(c.pts, mid)
				require.Greater(t, r, sp.X, "the flat at z %v cuts a section", mid)
				require.False(t, arborInside(t, body, c.at.X+(sp.X+r)/2, c.at.Y, mid), "flat cut at z %v", mid)
				require.True(t, arborInside(t, body, c.at.X+sp.X-0.15, c.at.Y, mid), "shaft kept at z %v", mid)
				require.True(t, arborInside(t, body, c.at.X-r+0.1, c.at.Y, mid), "round side kept at z %v", mid)
				require.False(t, arborInside(t, body, c.at.X-r-0.05, c.at.Y, mid), "section radius at z %v", mid)
			}
			// A ramp is a plane at 30 degrees to the axis: 0.3 mm past the end of the flat the cut plane stands
			// 0.3 tan 30 degrees further out; the shaft is kept below it and cut above it.
			for _, sp := range c.spans {
				for _, ramp := range []struct {
					on bool
					z  float64
				}{{sp.RampEnd, sp.Z1 + 0.3}, {sp.RampStart, sp.Z0 - 0.3}} {
					if !ramp.on {
						continue
					}
					plane := sp.X + 0.3*math.Tan(coneHalfAngle)
					require.Greater(t, sectionRadius(c.pts, ramp.z), plane+0.05, "the ramp at z %v cuts a section", ramp.z)
					require.True(t, arborInside(t, body, c.at.X+plane-0.05, c.at.Y, ramp.z), "shaft kept under the ramp")
					require.False(t, arborInside(t, body, c.at.X+plane+0.05, c.at.Y, ramp.z), "ramp cut at z %v", ramp.z)
				}
			}
			// Each bearing seat is the full 5.0 mm round shaft: no flat and no clearance.
			for _, seat := range c.seats {
				mid := (seat[0] + seat[1]) / 2
				require.True(t, arborInside(t, body, c.at.X+2.45, c.at.Y, mid), "seat radius at z %v", mid)
				require.False(t, arborInside(t, body, c.at.X+2.55, c.at.Y, mid), "seat radius at z %v", mid)
				require.True(t, arborInside(t, body, c.at.X, c.at.Y+2.45, mid))
				require.False(t, arborInside(t, body, c.at.X, c.at.Y+2.55, mid))
			}
		})
	}
	require.NotEmpty(t, b.sketches)
}

// TestArborSketches verifies every sketch the arbor builders drew is fully constrained: DOF 0, no redundant constraint
// and no conflict.
func TestArborSketches(t *testing.T) {
	b := arborTestBuilder(t)
	for _, build := range []func() (*decad.Body, error){b.buildEscapeArbor, b.buildPalletArbor, b.buildBalanceStaff} {
		_, err := build()
		require.NoError(t, err)
	}
	// Meridian, one flat tool per flat and, for a ramp, its meridian: escape 2+... counted by the builders.
	require.GreaterOrEqual(t, len(b.sketches), 3+4)
	for _, s := range b.sketches {
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s))
	}
}

// TestArborCollars checks the shaft half of the arbor collars on the built arbors: the shoulders and cones, the body
// steps and their 0.3 land, and the press gaps between each carried part's bore corner and the cone it faces.
func TestArborCollars(t *testing.T) {
	b := arborTestBuilder(t)
	escape, err := b.buildEscapeArbor()
	require.NoError(t, err)
	pallet, err := b.buildPalletArbor()
	require.NoError(t, err)
	staff, err := b.buildBalanceStaff()
	require.NoError(t, err)

	escapeCone, palletCone := arborCollarCone(escapeArborPoints, 4), arborCollarCone(palletArborPoints, 4)
	// Escape front shoulder [13.3634, 26.75]: length 13.39. Pallet and staff shoulders [0.25, 3.2866]: length 3.04.
	requireConeCollar(t, escape, b.g.O, 26.75, escapeConeLow, 20, escapeCone, "escape front shoulder")
	require.InDelta(t, 13.3634, escapeConeLow, 1e-4)
	require.InDelta(t, 14.0562, escapeConeHigh, 1e-4)
	requireConeCollar(t, pallet, b.g.P, 0.25, palletConeHigh, 1.2, palletCone, "pallet shoulder")
	requireConeCollar(t, staff, b.g.B, 0.25, palletConeHigh, 1.2, palletCone, "staff shoulder")
	require.InDelta(t, 2.5938, palletConeLow, 1e-4)
	require.InDelta(t, 3.2866, palletConeHigh, 1e-4)
	require.InDelta(t, 3.04, palletConeHigh-collarBackZ, 0.005)
	requireCone(t, escape, arborCollarCone(escapeArborPoints, 8), "escape end step")

	// The carried parts' bore corners stand 0.05 from the cone they face, measured normal to it: the escape wheel's
	// front bore corner (r 3.35, z 13.35); the lever's and the safety roller's rear bore corner (r 3.35, z 3.30).
	require.InDelta(t, pressGap, escapeCone.gapTo(BodyBoreRadius, wheelHigh), 1e-3, "escape wheel corner to the cone")
	require.InDelta(t, pressGap, palletCone.gapTo(BodyBoreRadius, palletPartLow), 1e-3, "lever corner to the cone")

	// Body steps: escape step at 0.25 with the wheel's rear face 10.10 from it; pallet step at 26.75 with the lever's
	// front face (z 9.80) 16.95 from it; staff step at 26.75 with the retainer's front face (z 25.90) 1.10 from the
	// front bearing's rear face (z 27). Every step stands 0.25 from the bearing it faces.
	requireBodyStep(t, escape, b.g.O, collarBackZ, 5, wheelLow, collarBackZ, "escape body step")
	require.InDelta(t, 10.10, wheelLow-collarBackZ, 1e-9)
	requireBodyStep(t, pallet, b.g.P, collarFrontZ, 20, 9.80, collarFrontZ, "pallet body step")
	require.InDelta(t, 16.95, collarFrontZ-9.80, 1e-9)
	requireBodyStep(t, staff, b.g.B, collarFrontZ, 24, 25.90, 27.0, "staff body step")
	require.InDelta(t, 1.10, 27.0-25.90, 1e-9)
	require.InDelta(t, 0.25, 27.0-collarFrontZ, 1e-9, "front body steps to the front bearing")
	require.InDelta(t, 0.25, collarBackZ-0.0, 1e-9, "shoulders and the escape step to the back bearing")

	// The 0.3 land: a step r 3.3 against the 605ZZ inner ring face of 8.0 across leaves the ring from the chamfer's
	// reach (r 3.0) to the step's radius; the shoulder's r 3.7 stays inside the ring face (r 4.0).
	require.GreaterOrEqual(t, bodyRadius-boreChamferReach, 0.3-1e-9)
	require.Less(t, CollarRadius, 4.0)
}
