package planetarygearbox_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/stretchr/testify/require"
)

// TestFastenersSitFlush checks that four M2 bolts and four nuts join each cover to the ring, that every bolt head
// ends level with its cover's outer face, and that every nut sits inside the ring, 1.8 mm from its end face. Each
// mounting ear holds a bolt, its head level with the ear's rear face and its tip inside the ear, and a nut wholly
// between the head and the tip, in a cavity inside the ear.
func TestFastenersSitFlush(t *testing.T) {
	_, box := defaultGearbox(t)
	requireFastenersFlush(t, box, planetarygearbox.DefaultParams())
}

// TestFastenersFitTheirPockets checks that no bolt or nut occupies material of the parts it joins: each bolt runs
// through its cover's hole and head pocket and the ring's hole, and each nut sits in a ring cavity.
func TestFastenersFitTheirPockets(t *testing.T) {
	// The overlap checks copy bodies into their document, so they use a gearbox of their own.
	box, err := planetarygearbox.Build(t.Context(), planetarygearbox.NewDocument(), planetarygearbox.DefaultParams())
	require.NoError(t, err)
	requireFastenersClear(t, box)
}

func TestFastenerFitRules(t *testing.T) {
	for name, c := range map[string]struct {
		edit func(*planetarygearbox.Params)
		want string
	}{
		"hole narrower than the shank": {func(p *planetarygearbox.Params) { p.BoltHoleDiameter = 1.5 }, "shank"},
		"hole under the M2 clearance":  {func(p *planetarygearbox.Params) { p.BoltHoleDiameter = 2.5 }, "clearance hole"},
		"cover too thin for the head":  {func(p *planetarygearbox.Params) { p.CoverThickness = 2.5 }, "bolt head"},
	} {
		t.Run(name, func(t *testing.T) {
			p := planetarygearbox.DefaultParams()
			c.edit(&p)
			_, err := planetarygearbox.Build(t.Context(), planetarygearbox.NewDocument(), p)
			require.ErrorContains(t, err, c.want)
		})
	}
}
