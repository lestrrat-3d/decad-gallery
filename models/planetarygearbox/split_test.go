package planetarygearbox

// CI: isolated-tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSplitShafts applies "Split shafts" to every shaft and pin of the default gearbox.
func TestSplitShafts(t *testing.T) {
	cornersDefault.once.Do(func() {
		cornersDefault.box, cornersDefault.err = Build(t.Context(), NewDocument(), DefaultParams())
	})
	require.NoError(t, cornersDefault.err)
	requireSplit(t, cornersDefault.box, DefaultParams())
}

// TestSplitShaftsVariants applies "Split shafts" to every parameter case that changes a shaft or a pin: the other
// kits, two stages and no flat. It takes several minutes, so it runs in the scheduled full suite and not in pull
// request CI.
func TestSplitShaftsVariants(t *testing.T) {
	kit := func(name string) func(t *testing.T) Params {
		return func(t *testing.T) Params {
			p, err := DefaultParamsFor(name)
			require.NoError(t, err)
			return p
		}
	}
	variants := []struct {
		name  string
		build func(t *testing.T) Params
	}{
		{"planet605-shaft608", kit("planet605-shaft608")},
		{"planet608-shaft605", kit("planet608-shaft605")},
		{"planet608-shaft608", kit("planet608-shaft608")},
		{"Stages 2", func(*testing.T) Params {
			p := DefaultParams()
			p.Stages = 2
			return p
		}},
		{"Stages 2, 608ZZ shaft", func(t *testing.T) Params {
			p := kit("planet605-shaft608")(t)
			p.Stages = 2
			return p
		}},
		{"FlatDepth 0", func(*testing.T) Params {
			p := DefaultParams()
			p.FlatDepth = 0
			return p
		}},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			p := v.build(t)
			box, err := Build(t.Context(), NewDocument(), p)
			require.NoError(t, err)
			requireSplit(t, box, p)
		})
	}
}
