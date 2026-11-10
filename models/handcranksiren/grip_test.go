package handcranksiren_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// region is an axis-aligned box in build coordinates.
type region struct{ min, max r3.Vec }

// The left hand's box round the grip's bar and the grip's finger window (instructions.md, Grip).
var (
	handBox      = region{r3.Vec{X: 57, Y: -100, Z: 15}, r3.Vec{X: 140, Y: -36.25, Z: 115}}
	fingerWindow = region{r3.Vec{X: 79, Y: -36.15, Z: 15}, r3.Vec{X: 105, Y: -6.35, Z: 115}}
)

// meets reports whether box, widened by margin on every side, intersects r.
func (r region) meets(box decad.Box, margin float64) bool {
	return box.Min.X-margin < r.max.X && box.Max.X+margin > r.min.X &&
		box.Min.Y-margin < r.max.Y && box.Max.Y+margin > r.min.Y &&
		box.Min.Z-margin < r.max.Z && box.Max.Z+margin > r.min.Z
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
		{"Min.X", grip.Min.X, 79}, {"Max.X", grip.Max.X, 105}, {"Min.Y", grip.Min.Y, -66.25},
		{"Max.Y", grip.Max.Y, -0.25}, {"Min.Z", grip.Min.Z, 5}, {"Max.Z", grip.Max.Z, 125},
	} {
		require.InDelta(t, row.want, row.got, tol, "grip %s", row.what)
	}
}
