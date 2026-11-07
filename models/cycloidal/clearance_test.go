package cycloidal_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/cycloidal"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// sectionGap returns the proven lower bound on the gap between a and b in the plane z = height above the drive
// axis frame, including both sections' mesh bounds.
func sectionGap(t *testing.T, a, b *decad.Body, height float64) float64 {
	t.Helper()
	plane, err := r3.NewFrame(r3.NewVec(0, 0, height), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	tol := units.Millimeters(0.01)
	first, err := a.SectionOutline(t.Context(), plane, tol)
	require.NoError(t, err)
	second, err := b.SectionOutline(t.Context(), plane, tol)
	require.NoError(t, err)
	gap, err := first.Clearance(second)
	require.NoError(t, err)
	require.False(t, gap.Overlaps)
	return gap.Lower().Base()
}

// axialGap returns the proven lower bound on the distance from the top of low to the bottom of high.
func axialGap(t *testing.T, low, high *decad.Body) float64 {
	t.Helper()
	lo, err := low.Bounds()
	require.NoError(t, err)
	hi, err := high.Bounds()
	require.NoError(t, err)
	return hi.Min.Z - lo.Max.Z - lo.Bound.Base() - hi.Bound.Base()
}

// Every pair of moving parts clears each other by the fit's minimum at the home pose, over the default and
// the smallest clearances Problems accepts, with one and two discs.
func TestMeasuredFits(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		mod  func(p *cycloidal.Params)
	}{
		{"one_disc", func(p *cycloidal.Params) {}},
		{"two_discs", func(p *cycloidal.Params) { p.DiscCount = 2 }},
		{"pins_12", func(p *cycloidal.Params) { p.PinCount = 12 }},
		{"two_discs_pins_20", func(p *cycloidal.Params) { p.DiscCount, p.PinCount = 2, 20 }},
		{"solid_cam_five_pins", func(p *cycloidal.Params) { p.InputShaftDiameter, p.OutputPinCount = 0, 5 }},
		{"minimum_clearances", func(p *cycloidal.Params) {
			p.DiscCount, p.DiskClearance, p.BearingClearance = 2, cycloidal.MinClearance, 2*cycloidal.MinClearance
			p.OutputPinClearance, p.DiscGap = cycloidal.MinClearance, cycloidal.MinClearance
		}},
		{"larger_clearances", func(p *cycloidal.Params) {
			p.DiskClearance, p.BearingClearance, p.OutputPinClearance = 0.5, 1, 0.5
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := cycloidal.DefaultParams()
			p.ChamferSize = 0
			tc.mod(&p)
			drive, err := cycloidal.Build(t.Context(), chainDocument(), p)
			require.NoError(t, err)
			const regular, tight = 0.35, 0.2
			for i, disc := range drive.Discs {
				mid := float64(i)*(p.DiscThickness+p.DiscGap) + p.DiscThickness/2
				require.GreaterOrEqual(t, sectionGap(t, disc, drive.Housing, mid), regular, "disc %d to casing", i+1)
				require.GreaterOrEqual(t, sectionGap(t, disc, drive.Cam, mid), regular, "disc %d to cam", i+1)
				for k, pin := range drive.OutputPins {
					require.GreaterOrEqual(t, sectionGap(t, disc, pin, mid), regular, "disc %d to pin %d", i+1, k+1)
				}
			}
			plateMid := float64(p.DiscCount)*p.DiscThickness + float64(p.DiscCount-1)*p.DiscGap + 1 + p.OutputPlateThickness/2
			for k, pin := range drive.OutputPins {
				require.GreaterOrEqual(t, sectionGap(t, drive.OutputPlate, pin, plateMid), tight, "plate to pin %d", k+1)
			}
			if p.DiscCount == 2 {
				require.GreaterOrEqual(t, axialGap(t, drive.Discs[0], drive.Discs[1]), regular, "disc 1 to disc 2")
			}
			require.GreaterOrEqual(t, axialGap(t, drive.Discs[len(drive.Discs)-1], drive.OutputPlate), regular, "disc to plate")
			require.GreaterOrEqual(t, axialGap(t, drive.Cam, drive.OutputPlate), regular, "cam to plate")
		})
	}
}
