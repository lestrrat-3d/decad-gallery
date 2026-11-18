package plaquestand_test

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/models/plaquestand"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func requireFullyConstrained(t *testing.T, s *sketch.Sketch) {
	t.Helper()
	report := s.Verify(t.Context())
	require.True(t, report.Status == sketch.FullyConstrained, "status %s", report.Status)
	require.Zero(t, report.DOF)
	require.Zero(t, len(report.Redundant))
	require.Zero(t, len(report.Conflicts))
	require.NoError(t, report.Check())
}

// requireBuilt builds p into a fresh document and checks what every case shares: two solid bodies, both sketches
// fully constrained, and the Stand and Plaque containers.
func requireBuilt(t *testing.T, p plaquestand.Params) (*decad.Document, *plaquestand.Model) {
	t.Helper()
	doc := decad.New()
	m, err := plaquestand.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Equal(t, 2, len(doc.Bodies()))
	require.True(t, m.Stand.IsSolid())
	require.True(t, m.Plaque.IsSolid())
	requireFullyConstrained(t, m.Profile)
	requireFullyConstrained(t, m.PlaqueSketch)

	names := map[string]*decad.Body{}
	for _, child := range doc.Root().Children() {
		bodies := child.Bodies()
		require.Equal(t, 1, len(bodies), child.Name())
		names[child.Name()] = bodies[0]
	}
	require.Equal(t, 2, len(names))
	require.Same(t, m.Stand, names[plaquestand.StandContainerName])
	require.Same(t, m.Plaque, names[plaquestand.PlaqueContainerName])
	require.NoError(t, doc.VerifyContainers().Err())
	return doc, m
}

func TestPlaqueStandBuildDefault(t *testing.T) {
	requireBuilt(t, plaquestand.DefaultParams())
}

// shoelace returns the absolute area of the closed loop in (Y, Z).
func shoelace(loop []r3.Vec) float64 {
	sum := 0.0
	for i, a := range loop {
		b := loop[(i+1)%len(loop)]
		sum += a.Y*b.Z - b.Y*a.Z
	}
	return math.Abs(sum) / 2
}

func TestPlaqueStandBuildCases(t *testing.T) {
	for _, tc := range cases() {
		t.Run(tc.name, func(t *testing.T) {
			_, m := requireBuilt(t, tc.params)
			dm := m.Dimensions
			p := tc.params
			sT, cT := dm.Axis.Y, dm.Axis.Z
			tPlaque := p.PlaqueThickness

			box, err := m.Stand.Bounds()
			require.NoError(t, err)
			tol := box.Bound.Base() + 1e-6
			require.InDelta(t, 0, box.Min.X, tol)
			require.InDelta(t, dm.StandWidth, box.Max.X, tol)
			require.InDelta(t, 0, box.Min.Y, tol)
			require.InDelta(t, dm.Depth, box.Max.Y, tol)
			require.InDelta(t, 0, box.Min.Z, tol)
			require.InDelta(t, dm.Height, box.Max.Z, tol)

			pbox, err := m.Plaque.Bounds()
			require.NoError(t, err)
			ptol := pbox.Bound.Base() + 1e-6
			x0 := (dm.StandWidth - p.PlaqueWidth) / 2
			require.InDelta(t, x0, pbox.Min.X, ptol)
			require.InDelta(t, x0+p.PlaqueWidth, pbox.Max.X, ptol)
			require.InDelta(t, dm.Seat.Y-tPlaque/2*cT, pbox.Min.Y, ptol)
			require.InDelta(t, dm.Seat.Y+tPlaque/2*cT+p.PlaqueHeight*sT, pbox.Max.Y, ptol)
			require.InDelta(t, dm.Seat.Z-tPlaque/2*sT, pbox.Min.Z, ptol)
			require.InDelta(t, dm.Seat.Z+tPlaque/2*sT+p.PlaqueHeight*cT, pbox.Max.Z, ptol)

			// The chamfer of both side faces' outlines removes e^2 (P - 2 pi e / 3) from the prism's volume.
			const e = plaquestand.ChamferSize
			A, P := dm.Outline.Area(), dm.Outline.Length()
			vol, err := m.Stand.Volume()
			require.NoError(t, err)
			require.LessOrEqual(t, vol.Bound.Base(), 10.0, "the volume bound must separate the chamfer")
			want := dm.StandWidth*A - e*e*(P-2*math.Pi*e/3)
			require.InDelta(t, want, vol.Value.Base(), vol.Bound.Base()+0.5)

			// One loop at each height. In the middle the section is the outline; 0.1 mm from a side face it is the
			// outline offset in by the chamfer's remaining 0.4 mm.
			for _, sec := range []struct{ h, area float64 }{
				{dm.StandWidth / 2, A},
				{0.1, A - 0.4*P + 0.16*math.Pi},
				{dm.StandWidth - 0.1, A - 0.4*P + 0.16*math.Pi},
			} {
				o, err := cornercheck.Outline(t.Context(), m.Stand, r3.Vec{X: 1}, sec.h)
				require.NoError(t, err)
				require.Len(t, o.Loops, 1, "loops at %g", sec.h)
				require.InDelta(t, sec.area, shoelace(o.Loops[0]), 3, "section area at %g", sec.h)
			}

			pvol, err := m.Plaque.Volume()
			require.NoError(t, err)
			require.InDelta(t, p.PlaqueWidth*p.PlaqueHeight*tPlaque, pvol.Value.Base(), pvol.Bound.Base()+1e-6)

			mesh, err := m.Stand.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, mesh.BoundaryVerified())
			require.True(t, mesh.VolumeVerified())

			limit := plaquestand.BedSize - 2*plaquestand.BedMargin
			require.LessOrEqual(t, box.Max.X-box.Min.X, limit)
			require.LessOrEqual(t, box.Max.Y-box.Min.Y, limit)
			require.LessOrEqual(t, box.Max.Z-box.Min.Z, limit)
			require.LessOrEqual(t, box.Max.X-box.Min.X, plaquestand.MaxStandWidth+tol)
		})
	}
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
