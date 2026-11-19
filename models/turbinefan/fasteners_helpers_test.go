package turbinefan_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// flatsHalf is the distance from a nut slot's line to its straight sides.
var flatsHalf = m2fastener.NutCavityRadius * math.Sqrt(3) / 2

// requireAccess requires that no vertex of blade lies in blade k's nut path, the slot's mouth below the root: radial
// HubRadius - 1 to HubRadius + 4.7 along the radial line, within 2.81 of it sideways, and 0.3 above the slot's high
// face.
func requireAccess(t *testing.T, sz turbinefan.Size, blade []r3.Vec, k int) {
	t.Helper()
	u := bladeDir(k)
	v := r3.Vec{X: -u.Y, Y: u.X}
	for _, w := range blade {
		if r := w.Dot(u); r >= sz.HubRadius-1 && r <= sz.HubRadius+4.7 && math.Abs(w.Dot(v)) <= 2.81 &&
			w.Z < sz.HubLength-sz.BladeBoltCavity+0.3 {
			require.FailNow(t, fmt.Sprintf("a blade %d vertex blocks its nut's path at %v", k, w))
		}
	}
}
