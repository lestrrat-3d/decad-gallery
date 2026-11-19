package onewayclutch_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
)

// TestOneWayClutchAdapter builds the 605ZZ and 608ZZ clutches and its standard adapter in one document and checks the adapter, its
// fasteners and every pair they meet: the adapter's shape and edge finishing, flush heads, nuts inside their pockets,
// bolt tips, the clearances round each shank, the seam, and the wall round each pocket.
func TestOneWayClutchAdapter(t *testing.T) {
	for _, shaft := range []string{"605ZZ", "608ZZ"} {
		t.Run(shaft, func(t *testing.T) {
			p := onewayclutch.DefaultParams()
			p.Shaft = shaft
			requireAdapter(t, p)
		})
	}
}
