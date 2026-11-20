package plaquestand_test

// CI: isolated-tests

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/models/plaquestand"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestPlaqueStandBuildDefault(t *testing.T) {
	requireBuilt(t, plaquestand.DefaultParams())
}

func TestPlaqueStandBuildCases(t *testing.T) {
	for _, tc := range cases() {
		t.Run(tc.name, func(t *testing.T) {
			_, m := requireBuilt(t, tc.params)
			dm := m.Dimensions
			p := tc.params
			sT, cT := dm.Axis.Y, dm.Axis.Z
			tPlaque := p.PlaqueThickness
			const e, wb = plaquestand.ChamferSize, plaquestand.BracketWidth
			limit := plaquestand.BedSize - 2*plaquestand.BedMargin

			A, P := dm.Outline.Area(), dm.Outline.Length()
			for k, b := range m.Brackets {
				x0 := float64(k) * (dm.Span - wb)
				box, tol := bounds(t, b)
				require.InDelta(t, x0, box.Min.X, tol)
				require.InDelta(t, x0+wb, box.Max.X, tol)
				require.InDelta(t, 0, box.Min.Y, tol)
				require.InDelta(t, dm.Depth, box.Max.Y, tol)
				require.InDelta(t, 0, box.Min.Z, tol)
				require.InDelta(t, dm.Height, box.Max.Z, tol)
				require.LessOrEqual(t, box.Max.Y-box.Min.Y, limit)
				require.LessOrEqual(t, box.Max.Z-box.Min.Z, limit)

				// A prism less the side chamfers, the hex pocket (sharp) and the hole above it: the 0.6 mm round
				// leaves about 1.2 mm³ more.
				vol, err := b.Volume()
				require.NoError(t, err)
				hex := 2 * (3 * math.Sqrt(3) / 2) * 2.51 * 2.51
				hole := math.Pi * 1.3 * 1.3 * (dm.ChannelFloor - 2)
				want := wb*A - e*e*(P-2*math.Pi*e/3) - hex - hole
				require.InDelta(t, want, vol.Value.Base(), vol.Bound.Base()+3)

				// One loop at each height: in the middle (clear of the pocket and hole) the section is the outline;
				// 0.1 mm from a side face it is the outline offset in by the chamfer's remaining 0.4 mm.
				for _, sec := range []struct{ h, area float64 }{
					{x0 + wb/2 + 4, A},
					{x0 + 0.1, A - 0.4*P + 0.16*math.Pi},
					{x0 + wb - 0.1, A - 0.4*P + 0.16*math.Pi},
				} {
					o, err := cornercheck.Outline(t.Context(), b, r3.Vec{X: 1}, sec.h)
					require.NoError(t, err)
					require.Len(t, o.Loops, 1, "loops at %g", sec.h)
					require.InDelta(t, sec.area, shoelace(o.Loops[0]), 3, "bracket %d section area at %g", k+1, sec.h)
				}

				mesh, err := b.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
				require.NoError(t, err)
				require.True(t, mesh.BoundaryVerified())
				require.True(t, mesh.VolumeVerified())
			}

			rbox, rtol := bounds(t, m.Rail)
			require.InDelta(t, 0, rbox.Min.X, rtol)
			require.InDelta(t, dm.Span, rbox.Max.X, rtol)
			require.InDelta(t, dm.RailCenter-plaquestand.RailWidth/2, rbox.Min.Y, rtol)
			require.InDelta(t, dm.RailCenter+plaquestand.RailWidth/2, rbox.Max.Y, rtol)
			require.InDelta(t, plaquestand.BaseThickness-plaquestand.RailHeight+plaquestand.JointFit, rbox.Min.Z, rtol)
			require.InDelta(t, plaquestand.BaseThickness+plaquestand.JointFit, rbox.Max.Z, rtol)
			require.LessOrEqual(t, rbox.Max.X-rbox.Min.X, limit)
			rail := plaquestand.RailWidth*plaquestand.RailHeight - 4*(1-math.Pi/4)*plaquestand.RailRound*plaquestand.RailRound
			rvol, err := m.Rail.Volume()
			require.NoError(t, err)
			head := math.Pi * 2.2 * 2.2 * 2
			rhole := math.Pi * 1.3 * 1.3 * (plaquestand.RailHeight - 2)
			require.InDelta(t, rail*dm.Span-2*(head+rhole), rvol.Value.Base(), rvol.Bound.Base()+0.2)
			rmesh, err := m.Rail.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, rmesh.BoundaryVerified())
			require.True(t, rmesh.VolumeVerified())

			pbox, ptol := bounds(t, m.Plaque)
			x0 := (dm.Span - p.PlaqueWidth) / 2
			require.InDelta(t, x0, pbox.Min.X, ptol)
			require.InDelta(t, x0+p.PlaqueWidth, pbox.Max.X, ptol)
			require.InDelta(t, dm.Seat.Y-tPlaque/2*cT, pbox.Min.Y, ptol)
			require.InDelta(t, dm.Seat.Y+tPlaque/2*cT+p.PlaqueHeight*sT, pbox.Max.Y, ptol)
			require.InDelta(t, dm.Seat.Z-tPlaque/2*sT, pbox.Min.Z, ptol)
			require.InDelta(t, dm.Seat.Z+tPlaque/2*sT+p.PlaqueHeight*cT, pbox.Max.Z, ptol)
			pvol, err := m.Plaque.Volume()
			require.NoError(t, err)
			require.InDelta(t, p.PlaqueWidth*p.PlaqueHeight*tPlaque, pvol.Value.Base(), pvol.Bound.Base()+1e-6)
			require.LessOrEqual(t, dm.Span, plaquestand.MaxSpan)
		})
	}
}

// TestPlaqueStandProfileConditioning guards the 26-segment profile sketch against sketch's near-singular gate (4e-5):
// every combination of each parameter's minimum, midpoint and maximum keeps the conditioning at 6e-5 or more.
func TestPlaqueStandProfileConditioning(t *testing.T) {
	axis := func(lo, hi float64) []float64 { return []float64{lo, (lo + hi) / 2, hi} }
	worst := math.Inf(1)
	for _, w := range axis(plaquestand.MinPlaqueWidth, plaquestand.MaxPlaqueWidth) {
		for _, h := range axis(plaquestand.MinPlaqueHeight, plaquestand.MaxPlaqueHeight) {
			for _, th := range axis(plaquestand.MinPlaqueThickness, plaquestand.MaxPlaqueThickness) {
				for _, r := range axis(plaquestand.MinRecline, plaquestand.MaxRecline) {
					for _, lip := range axis(plaquestand.MinLipHeight, plaquestand.MaxLipHeight) {
						for _, c := range axis(plaquestand.MinSlotClearance, plaquestand.MaxSlotClearance) {
							p := plaquestand.Params{PlaqueWidth: w, PlaqueHeight: h, PlaqueThickness: th, Recline: r,
								LipHeight: lip, SlotClearance: c}
							cond, err := plaquestand.ProfileConditioning(t.Context(), p)
							require.NoError(t, err, "%+v", p)
							worst = math.Min(worst, cond)
							require.GreaterOrEqual(t, cond, 6e-5, "%+v", p)
						}
					}
				}
			}
		}
	}
	t.Logf("least conditioning %g", worst)
}

func TestPlaqueStandBuildRejectsInvalidInputs(t *testing.T) {
	doc := decad.New()
	var bad []plaquestand.Params
	for _, f := range []struct {
		set    func(*plaquestand.Params, float64)
		lo, hi float64
	}{
		{func(p *plaquestand.Params, v float64) { p.PlaqueWidth = v }, plaquestand.MinPlaqueWidth, plaquestand.MaxPlaqueWidth},
		{func(p *plaquestand.Params, v float64) { p.PlaqueHeight = v }, plaquestand.MinPlaqueHeight, plaquestand.MaxPlaqueHeight},
		{func(p *plaquestand.Params, v float64) { p.PlaqueThickness = v }, plaquestand.MinPlaqueThickness,
			plaquestand.MaxPlaqueThickness},
		{func(p *plaquestand.Params, v float64) { p.Recline = v }, plaquestand.MinRecline, plaquestand.MaxRecline},
		{func(p *plaquestand.Params, v float64) { p.LipHeight = v }, plaquestand.MinLipHeight, plaquestand.MaxLipHeight},
		{func(p *plaquestand.Params, v float64) { p.SlotClearance = v }, plaquestand.MinSlotClearance,
			plaquestand.MaxSlotClearance},
	} {
		for _, v := range []float64{f.lo - 0.01, f.hi + 0.01} {
			p := plaquestand.DefaultParams()
			f.set(&p, v)
			bad = append(bad, p)
		}
	}
	require.Len(t, bad, 12)
	for i, p := range bad {
		_, err := plaquestand.Build(t.Context(), doc, p)
		require.Error(t, err, "case %d", i)
	}
	require.Zero(t, len(doc.Bodies()))

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := plaquestand.Build(ctx, doc, plaquestand.DefaultParams())
	require.ErrorIs(t, err, context.Canceled)
	_, err = plaquestand.Build(t.Context(), nil, plaquestand.DefaultParams())
	require.ErrorContains(t, err, "document")
	_, err = plaquestand.Build(nil, doc, plaquestand.DefaultParams()) //nolint:staticcheck
	require.ErrorContains(t, err, "context")
	require.Zero(t, len(doc.Bodies()))
}
