package cycloidal_test

import (
	"context"
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

// cornerCheck is one check of "Corners and roots" on a drive's bodies: a body cut by horizontal sections, or the
// housing, cam or plate cut by an axial plane.
type cornerCheck struct {
	name string
	run  func(t require.TestingT, ctx context.Context)
}

// cornerChecks lists the checks requireCorners makes on the drive built from p, in print orientation, axis +Z (the
// README's Printing section). The rotor discs are checked as built, chamfered. The housing and the cam are also cut
// by planes through the drive axis, normal to X and to Y, for the "Corners between layers" bullet: the casing
// floor, and the neck between the two sections of a two-disc cam.
func cornerChecks(p cycloidal.Params, drive *cycloidal.Drive) []cornerCheck {
	z := r3.Vec{Z: 1}
	var checks []cornerCheck
	for _, nb := range drive.Named() {
		checks = append(checks, cornerCheck{nb.Name, func(t require.TestingT, ctx context.Context) {
			fs, err := cornercheck.Check(ctx, nb.Body, z, nil)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		}})
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
			checks = append(checks, cornerCheck{a.name + " axial", func(t require.TestingT, ctx context.Context) {
				for _, nb := range drive.Named() {
					if nb.Name != a.name {
						continue
					}
					fs, err := cornercheck.CheckAt(ctx, nb.Body, normal, []float64{0.001}, a.exempt)
					require.NoError(t, err)
					require.Empty(t, fs, cornercheck.Report(fs, 5))
				}
			}})
		}
	}
	return checks
}

// requireCorners applies "Corners and roots" to every printed body of the drive built from p, one subtest per check.
func requireCorners(t *testing.T, p cycloidal.Params, drive *cycloidal.Drive) {
	t.Helper()
	for _, check := range cornerChecks(p, drive) {
		t.Run(check.name, func(t *testing.T) { check.run(t, t.Context()) })
	}
}

// namedT prefixes every failure message with the check's name.
type namedT struct {
	*testing.T
	name string
}

func (n namedT) Errorf(format string, args ...any) {
	n.Helper()
	n.T.Errorf("%s: "+format, append([]any{n.name}, args...)...)
}

// requireCornersInline makes the same checks as requireCorners in t itself, for a case of a table that the CI case
// planner runs by name, where a case may not start subtests. A failure names its check, and the first one ends t.
func requireCornersInline(t *testing.T, p cycloidal.Params, drive *cycloidal.Drive) {
	t.Helper()
	for _, check := range cornerChecks(p, drive) {
		check.run(namedT{t, check.name}, t.Context())
	}
}
