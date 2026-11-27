package sevensegment_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentDigitsUpright checks the truth table in the knob-up stance. The box stands on its back wall with the button
// end up, so segment a (at +Y) is the bottom of the digit and each pair a-d, b-e, c-f swaps its role against the usual order.
func TestSevenSegmentDigitsUpright(t *testing.T) {
	t.Parallel()
	// The lit segments of each digit as the viewer reads them: top, upper right, lower right, bottom, lower left, upper
	// left and middle, written as letters of the usual order.
	usual := [10]string{"abcdef", "bc", "abdeg", "abcdg", "bcfg", "acdfg", "acdefg", "abc", "abcdefg", "abcdfg"}
	swap := map[rune]sevensegment.Segment{'a': sevensegment.SegD, 'b': sevensegment.SegE, 'c': sevensegment.SegF,
		'd': sevensegment.SegA, 'e': sevensegment.SegB, 'f': sevensegment.SegC, 'g': sevensegment.SegG}
	for digit, letters := range usual {
		want := map[sevensegment.Segment]bool{}
		for _, l := range letters {
			want[swap[l]] = true
		}
		for _, s := range sevensegment.Segments() {
			lit, err := sevensegment.Lit(digit, s)
			require.NoError(t, err)
			require.Equal(t, want[s], lit, "digit %d segment %s", digit, s)
		}
	}
}
