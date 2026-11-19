package cycloidal_test

// CI: isolated-tests

import (
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/cycloidal"
	"github.com/stretchr/testify/require"
)

func TestDefaultParamsAreValid(t *testing.T) {
	require.Empty(t, cycloidal.Problems(cycloidal.DefaultParams()))
	require.NoError(t, cycloidal.Validate(cycloidal.DefaultParams()))
}

// A one-disc drive ignores DiscGap, and a two-disc drive accepts the smallest gap.
func TestDiscGapLimit(t *testing.T) {
	p := cycloidal.DefaultParams()
	p.DiscGap = 0
	require.Empty(t, cycloidal.Problems(p))
	p.DiscCount, p.DiscGap = 2, cycloidal.MinCamNeck
	require.Empty(t, cycloidal.Problems(p))
}

func TestProblems(t *testing.T) {
	cases := []struct {
		name string
		mod  func(p *cycloidal.Params)
		want string
	}{
		{"two discs need even pins", func(p *cycloidal.Params) { p.DiscCount, p.PinCount = 2, 15 },
			"Two discs require an even Pin Count and an even Output Pin Count (currently N=15, M=6)."},
		{"two discs need even output pins", func(p *cycloidal.Params) { p.DiscCount, p.OutputPinCount = 2, 5 },
			"currently N=16, M=5"},
		{"pin diameter override out of range", func(p *cycloidal.Params) { p.PinDiameter = 1 },
			"Pin Diameter must be between 3.00 mm and 17.56 mm (currently 1.00)."},
		{"pin circle too small for the auto pin", func(p *cycloidal.Params) { p.PinCircleDiameter = 10 },
			"Pin geometry out of range"},
		{"output pin override of zero is not an override", func(p *cycloidal.Params) { p.OutputPinCircleDiameter = 5 },
			"Output pins vanish"},
		{"output holes overlap", func(p *cycloidal.Params) { p.OutputPinDiameter = 25 },
			"Output holes overlap"},
		{"eccentricity beyond the cusp limit", func(p *cycloidal.Params) { p.Eccentricity = 3 },
			"Eccentricity too large — reduce it below 2.81 mm"},
		{"output pin circle beyond the root circle", func(p *cycloidal.Params) { p.OutputPinCircleDiameter = 90 },
			"Output Pin Circle too large"},
		{"shaft bore as large as the cam", func(p *cycloidal.Params) { p.InputShaftDiameter = 30 },
			"Input Shaft Diameter must be less than Center Bearing Diameter (30.00 mm)."},
		{"shaft bore leaves the cam", func(p *cycloidal.Params) { p.InputShaftDiameter = 28 },
			"Input bore doesn't fit inside the cam — set Input Shaft Diameter below 27.00 mm"},
		{"center bore meets the output holes", func(p *cycloidal.Params) { p.CenterBearingDiameter = 45 },
			"Disk center bore overlaps the output holes"},
		{"pin count floor", func(p *cycloidal.Params) { p.PinCount = 3 }, "Pin Count must be at least 4"},
		{"disc count", func(p *cycloidal.Params) { p.DiscCount = 3 }, "Disc Count must be 1 or 2"},
		{"negative clearance", func(p *cycloidal.Params) { p.DiskClearance = -1 }, "Disk Clearance must not be negative"},
		{"disk clearance below the fit", func(p *cycloidal.Params) { p.DiskClearance = 0.35 },
			"Disk Clearance must be at least 0.55 mm"},
		{"bearing clearance below the fit", func(p *cycloidal.Params) { p.BearingClearance = 0.7 },
			"Bearing Clearance must be at least 1.10 mm"},
		{"output pin clearance below the fit", func(p *cycloidal.Params) { p.OutputPinClearance = 0 },
			"Output Pin Clearance must be at least 0.55 mm"},
		{"zero chamfer", func(p *cycloidal.Params) { p.ChamferSize = 0 },
			"Chamfer Size must be at least 0.50 mm"},
		{"chamfer below the edge finish minimum", func(p *cycloidal.Params) { p.ChamferSize = 0.49 },
			"Chamfer Size must be at least 0.50 mm"},
		{"touching discs", func(p *cycloidal.Params) { p.DiscCount, p.DiscGap = 2, 0 },
			"Disc Gap must be at least 1.40 mm"},
		{"gap below the cam neck", func(p *cycloidal.Params) { p.DiscCount, p.DiscGap = 2, 0.55 },
			"Disc Gap must be at least 1.40 mm"},
		{"housing wall below the press-fit wall", func(p *cycloidal.Params) { p.HousingWall = 1.5 },
			"Housing Wall must be at least 1.60 mm (currently 1.50 mm)."},
		{"plate sockets too close", func(p *cycloidal.Params) {
			p.Eccentricity, p.OutputPinDiameter, p.CenterBearingDiameter = 0.2, 23.4, 20
		},
			"Output plate sockets need 1.60 mm of wall between them"},
		{"cam bore wall too thin", func(p *cycloidal.Params) { p.InputShaftDiameter = 24 },
			"The input bore of the cam needs 1.60 mm of wall"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := cycloidal.DefaultParams()
			tc.mod(&p)
			problems := strings.Join(cycloidal.Problems(p), "\n")
			require.Contains(t, problems, tc.want)
			require.ErrorIs(t, cycloidal.Validate(p), cycloidal.ErrInvalid)
		})
	}
}

// The undercut bound is the binding eccentricity limit, about 2.4 mm for the defaults.
func TestUndercutBound(t *testing.T) {
	p := cycloidal.DefaultParams()
	p.Eccentricity = 2.7
	problems := cycloidal.Problems(p)
	require.Len(t, problems, 1)
	require.Contains(t, problems[0], "undercuts/self-intersects. Reduce Eccentricity below 2.4")
}

// The wall clauses each fail alone: the plate's socket-to-socket wall at 1.5 mm, and the cam's input bore wall.
func TestPressFitWallLimits(t *testing.T) {
	p := cycloidal.DefaultParams()
	p.Eccentricity, p.OutputPinDiameter, p.CenterBearingDiameter = 0.2, 23.4, 20
	problems := cycloidal.Problems(p)
	require.Len(t, problems, 1)
	require.Contains(t, problems[0], "sockets need 1.60 mm of wall between them")
	require.Contains(t, problems[0], "(currently 1.50 mm)")
	p.OutputPinDiameter = 23.2
	require.Empty(t, cycloidal.Problems(p))

	// A 24 mm shaft in a 30 mm cam leaves 15 - 1.5 - 12.05 = 1.45 mm. With two discs the bridge is the thinner wall:
	// a 22 mm shaft leaves 1.20 mm there.
	q := cycloidal.DefaultParams()
	q.InputShaftDiameter = 24
	problems = cycloidal.Problems(q)
	require.Len(t, problems, 1)
	require.Contains(t, problems[0], "(currently 1.45 mm)")
	q.InputShaftDiameter = 8
	q.DiscCount, q.InputShaftDiameter = 2, 22
	problems = cycloidal.Problems(q)
	require.Len(t, problems, 1)
	require.Contains(t, problems[0], "The input bore of the cam needs 1.60 mm of wall")
}
