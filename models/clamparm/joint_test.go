package clamparm_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestClampArmJointLinkEdgeFinishing(t *testing.T) {
	for _, l := range []float64{80, 120, 150} {
		p := clamparm.DefaultParams()
		p.LinkLength = l
		doc := clamparm.NewDocument()
		body, sketches, err := clamparm.BuildLink(t.Context(), doc, p)
		require.NoError(t, err, "L=%g", l)
		require.True(t, body.IsSolid())
		requireFullyConstrained(t, sketches)
		requireBounds(t, body, r3.NewVec(-10, -10, -10), r3.NewVec(l+10, 10, 10))
	}
}

func TestClampArmJointSmoke(t *testing.T) {
	j := buildJoint3(t)

	t.Run("parts are solid and sketches fully constrained", func(t *testing.T) {
		require.True(t, j.link.IsSolid())
		require.True(t, j.knob.IsSolid())
		requireFullyConstrained(t, j.linkSketches)
		requireFullyConstrained(t, j.knobSketches)
		// Knob: the outline, the nut pocket hexagon and the part mark.
		require.Len(t, j.knobSketches, 3)
	})

	t.Run("bounds", func(t *testing.T) {
		// Link B is posed about Y, so its Y extent is unchanged.
		lb, err := j.link.Bounds()
		require.NoError(t, err)
		require.InDelta(t, -10, lb.Min.Y, boundTol)
		require.InDelta(t, 10, lb.Max.Y, boundTol)
		// The knob stands on the fin's far side: its inner face is 20 mm from the bolt head, its top 32 mm.
		box, err := j.knob.Bounds()
		require.NoError(t, err)
		require.InDelta(t, 10, box.Min.Y, boundTol)
		require.InDelta(t, 22, box.Max.Y, boundTol)
	})

	t.Run("bolt head is flush in its pocket", func(t *testing.T) {
		lb, err := j.link.Bounds()
		require.NoError(t, err)
		bb, err := j.bolt.Bounds()
		require.NoError(t, err)
		require.InDelta(t, lb.Min.Y, bb.Min.Y, boundTol)
		// The tip ends 28.5 mm past the head's top face: past the nut's top, inside the knob's outer face.
		kb, err := j.knob.Bounds()
		require.NoError(t, err)
		nb, err := j.nut.Bounds()
		require.NoError(t, err)
		require.InDelta(t, 18.5, bb.Max.Y, boundTol)
		require.Greater(t, bb.Max.Y, nb.Max.Y)
		require.Less(t, bb.Max.Y, kb.Max.Y)
	})

	t.Run("nut is inside the knob pocket", func(t *testing.T) {
		kb, err := j.knob.Bounds()
		require.NoError(t, err)
		nb, err := j.nut.Bounds()
		require.NoError(t, err)
		// The pocket floor is 3 mm above the knob's inner face; the nut stands on it.
		require.InDelta(t, kb.Min.Y+3, nb.Min.Y, boundTol)
		require.Less(t, nb.Max.Y, kb.Max.Y)
		require.Greater(t, nb.Min.X, kb.Min.X)
		require.Less(t, nb.Max.X, kb.Max.X)
		require.Greater(t, nb.Min.Z, kb.Min.Z)
		require.Less(t, nb.Max.Z, kb.Max.Z)
	})
}

func TestClampArmJointNoOverlap(t *testing.T) {
	pairs := []struct {
		name string
		a, b func(joint3) *decad.Body
	}{
		{"bolt and link", func(j joint3) *decad.Body { return j.bolt }, func(j joint3) *decad.Body { return j.link }},
		{"bolt and knob", func(j joint3) *decad.Body { return j.bolt }, func(j joint3) *decad.Body { return j.knob }},
		{"nut and link", func(j joint3) *decad.Body { return j.nut }, func(j joint3) *decad.Body { return j.link }},
		{"nut and knob", func(j joint3) *decad.Body { return j.nut }, func(j joint3) *decad.Body { return j.knob }},
	}
	for _, tc := range pairs {
		t.Run(tc.name, func(t *testing.T) {
			j := buildJoint3(t)
			requireDisjoint(t, tc.a(j), tc.b(j))
		})
	}
}

func TestClampArmJointSwivel(t *testing.T) {
	body, sketches, err := clamparm.SwivelBlank(t.Context(), clamparm.NewDocument(), clamparm.DefaultParams())
	require.NoError(t, err)
	require.True(t, body.IsSolid())
	require.Len(t, sketches, 2)
	requireFullyConstrained(t, sketches)
	requireBounds(t, body, r3.NewVec(-10, -10, 0), r3.NewVec(80, 10, 20))
}

// TestClampArmJointOverlapCheckDetects shows that the disjoint check fails for a bolt whose head is buried in the fin.
func TestClampArmJointOverlapCheckDetects(t *testing.T) {
	j := buildJoint3(t)
	site := j.site
	site.Head = site.Head.Add(r3.NewVec(0, 4, 0))
	bolt, err := m5fastener.PlaceBolt(t.Context(), j.link.Document(), site)
	require.NoError(t, err)
	shared, err := decad.Intersect(t.Context(), bolt.Body, j.link)
	require.NoError(t, err)
	require.True(t, shared.IsSolid())
}
