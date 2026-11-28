package cycloidal_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/cycloidal"
	"github.com/stretchr/testify/require"
)

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
		_, drive := sharedDrive(t, p)
		requireCorners(t, p, drive)
	})
}

// TestCornersVariants runs the cases that change the rotor and casing outlines. Each case builds its own drive, and the
// CI case planner runs them as separate workers, so a case starts no subtest.
func TestCornersVariants(t *testing.T) {
	cases := []ciCase[func(p *cycloidal.Params)]{
		{"pins_12", func(p *cycloidal.Params) { p.PinCount = 12 }},
		{"pins_24", func(p *cycloidal.Params) { p.PinCount = 24 }},
		{"two_discs_pins_20", func(p *cycloidal.Params) { p.DiscCount, p.PinCount = 2, 20 }},
	}
	ciRunCases(t, cases, func(t *testing.T, mod func(p *cycloidal.Params)) {
		p := cycloidal.DefaultParams()
		mod(&p)
		drive, err := cycloidal.Build(t.Context(), chainDocument(), p)
		require.NoError(t, err)
		requireCornersInline(t, p, &drive)
	})
}
