package handcranksiren_test

import (
	"math"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// region is an axis-aligned box in build coordinates.
type region struct{ min, max r3.Vec }

// The left hand's box round the grip's bar and the grip's finger window (instructions.md, Grip).
var (
	handBox      = region{r3.Vec{X: 57, Y: -110, Z: 12}, r3.Vec{X: 140, Y: -46.25, Z: 118}}
	fingerWindow = region{r3.Vec{X: 79, Y: -46.15, Z: 12}, r3.Vec{X: 105, Y: -16.35, Z: 118}}
)

const (
	// knuckleRoom is the room the fingers bending round the bar's front edge need, in mm.
	knuckleRoom = 25.0
	// minWindowHeight is the least height an adult hand's breadth needs, in mm.
	minWindowHeight = 100.0
)

// meets reports whether box, widened by margin on every side, intersects r.
func (r region) meets(box decad.Box, margin float64) bool {
	return box.Min.X-margin < r.max.X && box.Max.X+margin > r.min.X &&
		box.Min.Y-margin < r.max.Y && box.Max.Y+margin > r.min.Y &&
		box.Min.Z-margin < r.max.Z && box.Max.Z+margin > r.min.Z
}

// distanceTo returns the distance from the point to the box, which is 0 for a point inside it.
func distanceTo(box decad.Box, p r3.Vec) float64 {
	d := func(lo, hi, v float64) float64 { return math.Max(0, math.Max(lo-v, v-hi)) }
	return math.Sqrt(math.Pow(d(box.Min.X, box.Max.X, p.X), 2) + math.Pow(d(box.Min.Y, box.Max.Y, p.Y), 2) +
		math.Pow(d(box.Min.Z, box.Max.Z, p.Z), 2))
}

// requireKnuckleRoom checks that no other part comes within knuckleRoom of the bar's inner front edge (X 79, Y -46.25)
// at any Z across the window, on the X < 79 side where the bent fingers go. It measures to each part's bounding box, which
// never overstates the room.
func requireKnuckleRoom(t *testing.T, parts map[string]handcranksiren.Part) {
	for name, part := range parts {
		box := bounds(t, part)
		if name == "grip" || box.Min.X >= 79 {
			continue
		}
		box.Max.X = math.Min(box.Max.X, 79)
		for z := 12.0; z <= 118.0; z += 1 {
			got := distanceTo(box, r3.Vec{X: 79, Y: -46.25, Z: z})
			require.GreaterOrEqual(t, got, knuckleRoom, "%s is too close to the bar's front edge at Z %.0f", name, z)
		}
	}
}

// requireWindowHeight checks that the finger window is at least minWindowHeight tall 0.25 mm from the bar's face. It sections
// the grip across X midway between the bolt columns and measures the open run through the window's middle.
func requireWindowHeight(t *testing.T, parts map[string]handcranksiren.Part) {
	origin := r3.Vec{X: 92}
	outline, err := parts["grip"].Body.SectionOutline(t.Context(), frameAt(t, origin, r3.Vec{Y: 1}, r3.Vec{Z: 1}),
		units.Millimeters(fitTolerance))
	require.NoError(t, err)
	s := sectionAt{outline: outline, origin: origin, u: r3.Vec{Y: 1}, v: r3.Vec{Z: 1}}
	const step = 0.05
	open := func(z float64) bool { return !s.contains(r3.Vec{Y: -46.0, Z: z}) }
	require.True(t, open(65), "the window is open at its middle")
	lo, hi := 65.0, 65.0
	for open(lo - step) {
		lo -= step
	}
	for open(hi + step) {
		hi += step
	}
	require.GreaterOrEqual(t, hi-lo, minWindowHeight-0.2, "the window is %.1f tall at the bar face", hi-lo)
}

// requireGrip checks that the grip fits a left hand and clears every other part: nothing else enters the hand box or the
// finger window, the grip sits left of the gearbox and behind the horn, the crank sits right of the gearbox, and the
// grip has the designed bounds.
func requireGrip(t *testing.T, parts map[string]handcranksiren.Part) {
	for name, part := range parts {
		if name == "grip" {
			continue
		}
		box := bounds(t, part)
		require.False(t, handBox.meets(box, box.Bound.Base()), "%s enters the hand box", name)
		require.False(t, fingerWindow.meets(box, 0), "%s enters the finger window", name)
		// The arm, pin, handle and pin hardware are the crank; crank-gear and crank-shaft lie inside the gearbox.
		if strings.HasPrefix(name, "crank-arm") || strings.HasPrefix(name, "crank-pin") || name == "crank-handle" {
			require.GreaterOrEqual(t, box.Min.Y, 38.6-1e-3-box.Bound.Base(), "%s is right of the gearbox", name)
		}
	}
	grip := bounds(t, parts["grip"])
	tol := 1e-3 + grip.Bound.Base()
	require.LessOrEqual(t, grip.Max.Y, -0.25+tol, "the grip is left of the gearbox")
	require.Greater(t, grip.Min.X, bounds(t, parts["horn"]).Max.X, "the grip is behind the horn")
	for _, row := range []struct {
		what      string
		got, want float64
	}{
		{"Min.X", grip.Min.X, 79}, {"Max.X", grip.Max.X, 105}, {"Min.Y", grip.Min.Y, -76.25},
		{"Max.Y", grip.Max.Y, -0.25}, {"Min.Z", grip.Min.Z, 5}, {"Max.Z", grip.Max.Z, 125},
	} {
		require.InDelta(t, row.want, row.got, tol, "grip %s", row.what)
	}
}
