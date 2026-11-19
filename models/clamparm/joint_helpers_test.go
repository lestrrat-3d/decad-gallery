package clamparm_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

const boundTol = 0.01

func requireFullyConstrained(t *testing.T, sketches []*sketch.Sketch) {
	t.Helper()
	require.NotEmpty(t, sketches)
	for i, s := range sketches {
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
	}
}

func requireBounds(t *testing.T, body *decad.Body, min, max r3.Vec) {
	t.Helper()
	box, err := body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, min.X, box.Min.X, boundTol)
	require.InDelta(t, min.Y, box.Min.Y, boundTol)
	require.InDelta(t, min.Z, box.Min.Z, boundTol)
	require.InDelta(t, max.X, box.Max.X, boundTol)
	require.InDelta(t, max.Y, box.Max.Y, boundTol)
	require.InDelta(t, max.Z, box.Max.Z, boundTol)
}

func place(ctx context.Context, t *testing.T, body *decad.Body, f r3.Frame) *decad.Body {
	t.Helper()
	moved, err := placement.Place(ctx, body, &f)
	require.NoError(t, err)
	return moved
}

// requireDisjoint requires that a and b share no volume. Intersect retires its operands, so the caller passes bodies it
// builds for this check only.
func requireDisjoint(t *testing.T, a, b *decad.Body) {
	t.Helper()
	_, err := decad.Intersect(t.Context(), a, b)
	require.Error(t, err)
	var be *decad.BooleanError
	require.True(t, errors.As(err, &be), "not a boolean failure: %v", err)
	require.Equal(t, decad.BooleanEmpty, be.Code, "operands share volume: %v", err)
}

// joint3 is the default J3 joint in the world frame: link B, its knob, bolt and nut, each built in a fresh document
// so that a check may consume it and placed by the derived frames.
type joint3 struct {
	link, knob, bolt, nut *decad.Body
	linkSketches          []*sketch.Sketch
	knobSketches          []*sketch.Sketch
	site                  m5fastener.Site
}

func buildJoint3(t *testing.T) joint3 {
	t.Helper()
	ctx := t.Context()
	p := clamparm.DefaultParams()
	d, err := p.Derive()
	require.NoError(t, err)
	doc := clamparm.NewDocument()
	var j joint3
	link, ls, err := clamparm.BuildLink(ctx, doc, p)
	require.NoError(t, err)
	j.linkSketches = ls
	j.link = place(ctx, t, link, d.LinkB)
	knob, ks, err := clamparm.BuildKnob(ctx, doc, p)
	require.NoError(t, err)
	j.knobSketches = ks
	j.knob = place(ctx, t, knob, d.KnobFrames[2])

	j.site = d.JointSites[1]
	bolt, err := m5fastener.PlaceBolt(ctx, doc, j.site)
	require.NoError(t, err)
	j.bolt = bolt.Body
	nut, err := m5fastener.PlaceNut(ctx, doc, d.KnobNutFrames[2])
	require.NoError(t, err)
	j.nut = nut.Body
	return j
}
