package planetarygearbox

// CI: isolated-tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCorners applies "Corners and roots" to every printed part of the default gearbox. The variants run in
// TestCornersVariants, which is too slow for pull request CI.
func TestCorners(t *testing.T) {
	cornersDefault.once.Do(func() {
		cornersDefault.box, cornersDefault.err = Build(t.Context(), NewDocument(), DefaultParams())
	})
	require.NoError(t, cornersDefault.err)
	requireCorners(t, cornersDefault.box, DefaultParams())
}

// TestCornersVariants applies "Corners and roots" to every parameter case that changes an outline: the other kits,
// every ear count the gallery tests, and two stages. It takes about 5 minutes, so it runs in the scheduled full suite
// and not in pull request CI. Each build takes under a minute; the 608ZZ kits' checks are the slow part of
// TestKitGearboxes, not their builds.
func TestCornersVariants(t *testing.T) {
	type variant struct {
		name  string
		build func(t *testing.T) Params
	}
	kit := func(name string) func(t *testing.T) Params {
		return func(t *testing.T) Params {
			p, err := DefaultParamsFor(name)
			require.NoError(t, err)
			return p
		}
	}
	ears := func(n int) func(t *testing.T) Params {
		return func(*testing.T) Params {
			p := DefaultParams()
			p.EarCount = n
			return p
		}
	}
	variants := []variant{
		{"planet605-shaft608", kit("planet605-shaft608")},
		{"planet608-shaft605", kit("planet608-shaft605")},
		{"planet608-shaft608", kit("planet608-shaft608")},
		{"EarCount 0", ears(0)},
		{"EarCount 1", ears(1)},
		{"EarCount 6", ears(6)},
		{"Stages 2", func(*testing.T) Params {
			p := DefaultParams()
			p.Stages = 2
			return p
		}},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			p := v.build(t)
			box, err := Build(t.Context(), NewDocument(), p)
			require.NoError(t, err)
			requireCorners(t, box, p)
		})
	}
}
