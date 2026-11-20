package clamparm_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// localBounds is a part's mesh bounds in its local frame, indexed by axis (0 x, 1 y, 2 z).
func localBounds(t *testing.T, p clamparm.Part) (lo, hi [3]float64) {
	t.Helper()
	lo, hi = [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}, [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, v := range asmTessellate(t, p.Body, 0.05).Vertices() {
		l := p.Frame.ToLocal(v)
		for k, c := range [3]float64{l.X, l.Y, l.Z} {
			lo[k], hi[k] = math.Min(lo[k], c), math.Max(hi[k], c)
		}
	}
	return lo, hi
}

// sectionFrame is the plane through the part's local point with local coordinate level on axis, with a basis whose
// normal is that axis.
func sectionFrame(t *testing.T, p clamparm.Part, axis int, level float64) r3.Frame {
	t.Helper()
	var at r3.Vec
	var u, v r3.Vec
	local := func(a, b r3.Vec) (r3.Vec, r3.Vec) {
		return p.Frame.ToWorld(a).Sub(p.Frame.Origin()), p.Frame.ToWorld(b).Sub(p.Frame.Origin())
	}
	switch axis {
	case 0:
		at, u, v = r3.Vec{X: level}, r3.Vec{Y: 1}, r3.Vec{Z: 1}
	case 1:
		at, u, v = r3.Vec{Y: level}, r3.Vec{Z: 1}, r3.Vec{X: 1}
	default:
		at, u, v = r3.Vec{Z: level}, r3.Vec{X: 1}, r3.Vec{Y: 1}
	}
	wu, wv := local(u, v)
	f, err := r3.NewFrame(p.Frame.ToWorld(at), wu, wv)
	require.NoError(t, err)
	return f
}

// edgeFace is one large face of a part whose outline carries the 0.5 chamfer: the face at the part's local bound lo
// or hi on Axis. The section 0.25 inside it must lie 0.25 inside the part's bound on each listed side, where the
// outline's extreme is a chamfered edge. Sides indexes [lo, hi] of the first and then the second other axis: y, z for
// an x face, z, x for a y face and x, y for a z face.
type edgeFace struct {
	part  string
	axis  int
	upper bool
	sides [4]bool
	// at, when not zero, is the local level of the section, and the bounds expected are the part's, inset 0.25.
	at float64
}

// inset is how far a 0.5 chamfer pulls a section 0.25 inside a face; tongueFace is a tongue face's y.
const (
	inset      = edgefinish.MinChamfer / 2
	tongueFace = clamparm.TongueThickness / 2
)
