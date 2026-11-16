package cycloidal_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/models/cycloidal"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// housingFloorCorner lists the casing floor corner that a housing with more than MaxChamferedFloorPins pins keeps
// sharp: the 90° turn where the top of the base meets the contour wall, along the ring at z = -1 that follows the
// contour. It lies between the pin radius R - Rr and the contour's peak R - Rr + 2E, pushed out by the disk
// clearance. It is a corner between layers that opens upward, and the chamfer that would cut it waits for decad's
// mitred rim offsets (decad P7): the contour has a 13.8° crease where two branches meet at 24 pins, which decad's
// rimoffset does not offset (it offsets joints under 10° only). It matches "Retained sharp corners" in
// instructions.md. A housing with the chamfer has no such corner, so it lists nothing.
func housingFloorCorner(p cycloidal.Params) []cornercheck.Exempt {
	if p.PinCount <= cycloidal.MaxChamferedFloorPins {
		return nil
	}
	pinCircle := p.PinCircleDiameter / 2
	pin := p.PinDiameter / 2
	if p.PinDiameter == 0 {
		pin = (p.Eccentricity + pinCircle*math.Sin(math.Pi/float64(p.PinCount))) / 2
	}
	low := pinCircle - pin
	high := low + 2*p.Eccentricity + p.DiskClearance
	ring := cornercheck.Annulus(r3.Vec{}, r3.Vec{Z: 1}, low-0.3, high+0.3)
	return []cornercheck.Exempt{{
		Reason: "casing floor ring at the wall foot",
		Near:   func(v r3.Vec) bool { return math.Abs(v.Z+1) < 0.3 && ring(v) },
	}}
}

// requireCorners applies "Corners and roots" to every printed body of the drive built from p, in its print
// orientation, axis +Z (the README's Printing section). The rotor discs are checked as built, chamfered. The housing
// and the cam are also cut by planes through the drive axis, normal to X and to Y, for the "Corners between layers"
// bullet: the casing floor, and the neck between the two sections of a two-disc cam.
func requireCorners(t *testing.T, p cycloidal.Params, drive *cycloidal.Drive) {
	t.Helper()
	z := r3.Vec{Z: 1}
	for _, nb := range drive.Named() {
		t.Run(nb.Name, func(t *testing.T) {
			fs, err := cornercheck.Check(t.Context(), nb.Body, z, nil)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		})
	}
	axial := []struct {
		name   string
		exempt []cornercheck.Exempt
	}{
		{cycloidal.HousingName, housingFloorCorner(p)},
		{cycloidal.CamName, nil},
		{cycloidal.OutputPlateName, nil},
	}
	for _, a := range axial {
		for _, normal := range []r3.Vec{{X: 1}, {Y: 1}} {
			t.Run(a.name+" axial", func(t *testing.T) {
				for _, nb := range drive.Named() {
					if nb.Name != a.name {
						continue
					}
					fs, err := cornercheck.CheckAt(t.Context(), nb.Body, normal, []float64{0.001}, a.exempt)
					require.NoError(t, err)
					require.Empty(t, fs, cornercheck.Report(fs, 5))
				}
			})
		}
	}
}

func TestCorners(t *testing.T) {
	t.Run("one disc", func(t *testing.T) {
		p := cycloidal.DefaultParams()
		doc, drive, err := defaultFixture(t.Context(), p)
		require.NoError(t, err)
		count := len(doc.Bodies())
		requireCorners(t, p, drive)
		require.Len(t, doc.Bodies(), count)
	})
	t.Run("two discs", func(t *testing.T) {
		p := cycloidal.DefaultParams()
		p.DiscCount = 2
		drive, err := cycloidal.Build(t.Context(), chainDocument(), p)
		require.NoError(t, err)
		requireCorners(t, p, &drive)
	})
}

// TestCornersVariants runs the cases that change the rotor and casing outlines. The scheduled models job runs it; the
// pull request job does not.
func TestCornersVariants(t *testing.T) {
	for name, mod := range map[string]func(p *cycloidal.Params){
		"pins_12": func(p *cycloidal.Params) { p.PinCount = 12 },
		"pins_24": func(p *cycloidal.Params) { p.PinCount = 24 },
		"two_discs_pins_20": func(p *cycloidal.Params) {
			p.DiscCount, p.PinCount = 2, 20
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := cycloidal.DefaultParams()
			mod(&p)
			drive, err := cycloidal.Build(t.Context(), chainDocument(), p)
			require.NoError(t, err)
			requireCorners(t, p, &drive)
		})
	}
}
