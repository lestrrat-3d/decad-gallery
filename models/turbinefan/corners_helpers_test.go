package turbinefan_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// counterboreRound is the set screw counterbore's 0.5 floor round, which a layer off the screw's line cuts at a slant and
// reads as a turn over 45 degrees.
func counterboreRound(dm turbinefan.Dimensions) cornercheck.Exempt {
	return cornercheck.Exempt{
		Reason: "counterbore floor round cut at a slant",
		Near: func(v r3.Vec) bool {
			r := math.Hypot(v.X, v.Z-turbinefan.SetScrewZ)
			return math.Abs(-v.Y-dm.CounterboreFloor) <= turbinefan.CounterboreRound+0.1 &&
				r >= m2fastener.HeadPocketRadius-turbinefan.CounterboreRound-0.05 && r <= m2fastener.HeadPocketRadius+0.05
		},
	}
}

// setScrewSlotEndRound is the set screw slot's rounded sloped end, which a layer cuts at a slant.
func setScrewSlotEndRound(dm turbinefan.Dimensions) cornercheck.Exempt {
	return cornercheck.Exempt{
		Reason: "set screw slot end round cut at a slant",
		Near: func(v r3.Vec) bool {
			return -v.Y >= dm.BoreFlat+turbinefan.PressWall-0.05 && -v.Y <= dm.BoreFlat+turbinefan.PressWall+2.05 &&
				math.Abs(v.X) <= flatsHalf+0.05 && v.Z >= turbinefan.SetScrewZ+m2fastener.NutCavityRadius/2-0.3
		},
	}
}

// optional returns the exemptions with each allowed to claim no finding.
func optional(exempt ...cornercheck.Exempt) []cornercheck.Exempt {
	for i := range exempt {
		exempt[i].Optional = true
	}
	return exempt
}

// setScrewSlotSideRounds is the 0.5 round where the set screw slot's straight sides meet its faces. A layer in the lower
// part of the slot (z up to 2.6) chords it into turns of 50.0 to 50.6 degrees on some hubs, just over the 50 degree limit,
// and under it on others, as the face levels fall differently.
func setScrewSlotSideRounds(dm turbinefan.Dimensions) cornercheck.Exempt {
	return cornercheck.Exempt{
		Reason:   "set screw slot side round chorded just over the limit",
		Optional: true,
		Near: func(v r3.Vec) bool {
			return -v.Y >= dm.BoreFlat+turbinefan.PressWall-0.05 && -v.Y <= dm.BoreFlat+turbinefan.PressWall+2.05 &&
				math.Abs(v.X) >= flatsHalf-0.6 && math.Abs(v.X) <= flatsHalf+0.05 && v.Z <= 2.6
		},
	}
}

// socketFloorRound is a blade socket's 0.5 floor round, which a layer off the socket's axis cuts at a slant and reads
// as a turn of 52 to 62 degrees near where the round meets the socket wall. It covers the round in each socket's own
// frame: BladeSocketStart to BladeSocketStart + BladeSocketFloorRound along the socket's axis, and BladeSocketRadius -
// BladeSocketFloorRound to BladeSocketRadius from it, each widened by 0.05 for the mesh. A layer far off the axis
// (z 12.375 on compact) meets that band up to BladeSocketRadius sideways, so the distance from the shaft axis alone
// does not bound it.
func socketFloorRound(sz turbinefan.Size) cornercheck.Exempt {
	return cornercheck.Exempt{
		Reason: "blade socket floor round cut at a slant",
		Near: func(v r3.Vec) bool {
			for k := 1; k <= turbinefan.BladeCount; k++ {
				d := bladeDir(k)
				along := v.X*d.X + v.Y*d.Y
				off := math.Hypot(v.Y*d.X-v.X*d.Y, v.Z-sz.HubMid)
				if along >= sz.BladeSocketStart-0.05 && along <= sz.BladeSocketStart+turbinefan.BladeSocketFloorRound+0.05 &&
					off >= sz.BladeSocketRadius-turbinefan.BladeSocketFloorRound-0.05 && off <= sz.BladeSocketRadius+0.05 {
					return true
				}
			}
			return false
		},
	}
}

// requireBladeCorners checks blade k along its radial line d, in the hub's mid plane. The stud is cut at the radii where
// its bolt hole crosses it; the hole's four corners (r 9.5 and 10.2) are the one exemption, where the material angle is
// 64 to 90 degrees and the checker adds the stud's curvature. The loft is cut at radii read from the station table
// (the root's and the airfoil's, each scaled with its station), because it has no planar faces to give heights. The step
// where the stud meets the root face (RootRadius) is a corner between layers, which no section reads; instructions.md
// lists it.
func requireBladeCorners(t *testing.T, m *turbinefan.Model, k int) {
	t.Helper()
	sz := m.Dimensions.Size
	j, s := sz.HubScale, sz.Scale
	d := bladeDir(k)
	centre := r3.Vec{Z: sz.HubMid}
	body := m.Blades[k-1]
	// The bolt hole's radius is fixed, so the layers through it lie at its centre and 0.7 off it, not scaled.
	stud := []float64{7.0 * j, sz.BladeBoltAt, sz.BladeBoltAt + 0.7, 12.5 * j}
	fs, err := cornercheck.CheckAt(t.Context(), body, d, stud, []cornercheck.Exempt{
		// At 128 the four corners read as tips of 125 to 130 degrees. A stud of radius 4.05 (192, 224) meets the hole at
		// a wider angle and reads none, so the exemption need not claim a finding.
		{Reason: "bolt hole across the stud", Near: cornercheck.Disc(centre, d, sz.BladeStudRadius+0.1), Optional: true},
	})
	require.NoError(t, err)
	require.Empty(t, fs, cornercheck.Report(fs, 5))
	loft := []float64{sz.RootRadius + 0.2*j, sz.RootRadius + 2.1*j}
	for _, r := range []float64{18, 22, 26, 30, 36, 39.2, 44, 52, 56, 60, 62, 63, 63.8} {
		loft = append(loft, r*s)
	}
	fs, err = cornercheck.CheckAt(t.Context(), body, d, loft, nil)
	require.NoError(t, err)
	require.Empty(t, fs, cornercheck.Report(fs, 5))
}

// requireBladePrintCorners checks a fresh blade lying in its print pose at its rest tilt, read across the bed (+Z) at
// every 1 mm from z 0.5 to its top: no planar face lies parallel to the bed, so MidHeights gives no height. Three
// listed corner groups are exempt:
//   - the step where the stud meets the root face: inside corners whose blade-local radius lies within 1.0 of
//     RootRadius. Lying, the step is inside the layers; it waits on a root fillet (instructions.md). required makes the
//     exemption claim a finding.
//   - the edges cut obliquely: a layer cuts the leading edge and the trailing-edge flat at the rest tilt, so each loop
//     ends in a stretched tip. Findings within 1.5 mm in x of their loop's least or greatest x are exempt.
//   - the bolt hole across the stud, the corner the radial-line check lists too: lying, the layers cut the hole's walls
//     where it crosses the stud. Findings whose blade-local radius lies within HoleRadius + 0.5 of BladeBoltAt.
func requireBladePrintCorners(t *testing.T, d, pitch float64, hand turbinefan.Hand, required bool) {
	t.Helper()
	sz := mustSize(t, d)
	_, blade, tilt := posedBlade(t, d, pitch, hand)
	pose, err := turbinefan.PrintPoseFor(sz, turbinefan.DefaultBladePitch, hand, tilt)
	require.NoError(t, err)
	radial := pose.U().Cross(pose.V())
	box, err := blade.Bounds()
	require.NoError(t, err)
	var heights []float64
	for h := 0.5; h < box.Max.Z; h += 1 {
		heights = append(heights, h)
	}
	ends := map[float64][][2]float64{} // per height, each loop's least and greatest x
	for _, h := range heights {
		o, err := cornercheck.Outline(t.Context(), blade, r3.Vec{Z: 1}, h)
		require.NoError(t, err)
		for _, l := range o.Loops {
			lo, hi := math.Inf(1), math.Inf(-1)
			for _, p := range l {
				lo, hi = math.Min(lo, p.X), math.Max(hi, p.X)
			}
			ends[h] = append(ends[h], [2]float64{lo, hi})
		}
	}
	step := cornercheck.Exempt{
		Reason:   "stud-to-root step inside the layers",
		Optional: !required,
		Near: func(p r3.Vec) bool {
			return math.Abs(p.Sub(pose.Origin()).Dot(radial)-sz.RootRadius) <= 1.0
		},
	}
	oblique := cornercheck.Exempt{
		Reason:   "edge cut obliquely by a layer",
		Optional: true,
		Near: func(p r3.Vec) bool {
			for h, loops := range ends {
				if math.Abs(p.Z-h) > 1e-6 {
					continue
				}
				for _, e := range loops {
					if math.Abs(p.X-e[0]) <= 1.5 || math.Abs(p.X-e[1]) <= 1.5 {
						return true
					}
				}
			}
			return false
		},
	}
	hole := cornercheck.Exempt{
		Reason:   "bolt hole across the stud",
		Optional: true,
		Near: func(p r3.Vec) bool {
			return math.Abs(p.Sub(pose.Origin()).Dot(radial)-sz.BladeBoltAt) <= m2fastener.HoleRadius+0.5
		},
	}
	fs, err := cornercheck.CheckAt(t.Context(), blade, r3.Vec{Z: 1}, heights, []cornercheck.Exempt{step, oblique, hole})
	require.NoError(t, err)
	require.Empty(t, fs, cornercheck.Report(fs, 8))
}
