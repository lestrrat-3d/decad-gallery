package cycloidal_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/cycloidal"
	"github.com/stretchr/testify/require"
)

// Every pair of moving parts clears each other by the fit's minimum at the home pose, over the default and
// the smallest clearances Problems accepts, with one and two discs.
func TestMeasuredFits(t *testing.T) {
	t.Parallel()
	cases := []ciCase[func(*cycloidal.Params)]{
		{"one_disc", func(p *cycloidal.Params) {}},
		{"two_discs", func(p *cycloidal.Params) { p.DiscCount = 2 }},
		{"pins_12", func(p *cycloidal.Params) { p.PinCount = 12 }},
		{"two_discs_pins_20", func(p *cycloidal.Params) { p.DiscCount, p.PinCount = 2, 20 }},
		{"solid_cam_five_pins", func(p *cycloidal.Params) { p.InputShaftDiameter, p.OutputPinCount = 0, 5 }},
		{"minimum_clearances", func(p *cycloidal.Params) {
			p.DiscCount, p.DiskClearance, p.BearingClearance = 2, cycloidal.MinClearance, 2*cycloidal.MinClearance
			p.OutputPinClearance, p.DiscGap = cycloidal.MinClearance, cycloidal.MinCamNeck
		}},
		{"larger_clearances", func(p *cycloidal.Params) {
			p.DiskClearance, p.BearingClearance, p.OutputPinClearance = 0.6, 1.2, 0.6
		}},
	}
	ciRunCases(t, cases, func(t *testing.T, mod func(*cycloidal.Params)) {
		t.Parallel()
		p := cycloidal.DefaultParams()
		mod(&p)
		drive, err := cycloidal.Build(t.Context(), chainDocument(), p)
		require.NoError(t, err)
		const regular, press = 0.35, 0.05
		// The section measurement of the round pin and socket reads up to 0.03 mm under the nominal gap
		// (measured 0.025 for a 0.05 nominal), so the press fit must be proven to this bound.
		const pressMeasured = press - 0.03
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
			require.GreaterOrEqual(t, sectionGap(t, drive.OutputPlate, pin, plateMid), pressMeasured, "plate to pin %d", k+1)
		}
		if p.DiscCount == 2 {
			require.GreaterOrEqual(t, axialGap(t, drive.Discs[0], drive.Discs[1]), regular, "disc 1 to disc 2")
		}
		require.GreaterOrEqual(t, axialGap(t, drive.Discs[len(drive.Discs)-1], drive.OutputPlate), regular, "disc to plate")
		require.GreaterOrEqual(t, axialGap(t, drive.Cam, drive.OutputPlate), regular, "cam to plate")
	})
}
