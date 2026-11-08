package ballbearing_test

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestBearing608Envelope(t *testing.T) {
	doc := decad.New()
	p := ballbearing.DefaultParams()
	m, err := ballbearing.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Equal(t, 1, len(doc.Bodies()))
	require.True(t, m.Body.IsSolid())
	report := m.Sketch.Verify(t.Context())
	require.True(t, report.Status == sketch.FullyConstrained, "status %s", report.Status)
	require.Zero(t, report.DOF)
	require.Zero(t, len(report.Redundant))
	require.Zero(t, len(report.Conflicts))
	require.NoError(t, report.Check())
	box, err := m.Body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, -11, box.Min.X, box.Bound.Base()+1e-8)
	require.InDelta(t, 11, box.Max.X, box.Bound.Base()+1e-8)
	require.InDelta(t, -11, box.Min.Y, box.Bound.Base()+1e-8)
	require.InDelta(t, 11, box.Max.Y, box.Bound.Base()+1e-8)
	require.InDelta(t, 0, box.Min.Z, box.Bound.Base()+1e-8)
	require.InDelta(t, 7, box.Max.Z, box.Bound.Base()+1e-8)
	volume, err := m.Body.Volume()
	require.NoError(t, err)
	require.InDelta(t, math.Pi*(121-16)*7, volume.Value.Base(), volume.Bound.Base()+1e-6)
	require.Equal(t, 4, len(m.Body.Faces()))
	requireBore(t, m.Body, 4, r3.NewVec(0, 0, 1))
	mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(.05), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func requireBore(t *testing.T, body *decad.Body, radius float64, axis r3.Vec) {
	t.Helper()
	caps := 0
	for _, face := range body.Faces() {
		if face.Surface().Kind() != decad.KindPlane {
			continue
		}
		require.Equal(t, 2, len(face.Loops()))
		require.Equal(t, 1, len(face.Loops()[1].Edges()))
		circle, ok := face.Loops()[1].Edges()[0].Curve().(decad.Circle3)
		require.True(t, ok)
		require.InDelta(t, radius, circle.Radius.Base(), 1e-8)
		require.InDelta(t, 1, math.Abs(circle.Axis.Dot(axis)), 1e-8)
		caps++
	}
	require.Equal(t, 2, caps)
}

func TestBearingPlacedAndRepeated(t *testing.T) {
	doc := decad.New()
	first, err := ballbearing.Build(t.Context(), doc, ballbearing.DefaultParams())
	require.NoError(t, err)
	f, err := r3.NewFrame(r3.NewVec(20, -5, 7), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	p := ballbearing.DefaultParams()
	p.Placement = &f
	placed, err := ballbearing.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Equal(t, 2, len(doc.Bodies()))
	box, err := placed.Body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 0, box.Min.Sub(r3.NewVec(20, -16, -4)).Len(), box.Bound.Base()+1e-8)
	require.InDelta(t, 0, box.Max.Sub(r3.NewVec(27, 6, 18)).Len(), box.Bound.Base()+1e-8)
	requireBore(t, placed.Body, 4, r3.NewVec(1, 0, 0))
	for _, model := range []*ballbearing.Model{first, placed} {
		report := model.Sketch.Verify(t.Context())
		require.True(t, report.Status == sketch.FullyConstrained, "status %s", report.Status)
		require.Zero(t, report.DOF)
		require.Zero(t, len(report.Redundant))
		require.Zero(t, len(report.Conflicts))
		mesh, err := model.Body.Tessellate(t.Context(), units.Millimeters(.05), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		require.True(t, mesh.VolumeVerified())
	}
}

// A lifted envelope is the same solid as one placed by the composed frame, but its only transform is the lift.
func TestBearingLifted(t *testing.T) {
	doc := decad.New()
	lift, err := r3.RotationAround(r3.NewVec(3, 0, 0), r3.NewVec(0, 1, 0), units.Radians(.3))
	require.NoError(t, err)
	local, err := r3.NewFrame(r3.NewVec(2, -1, 5), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	p := ballbearing.DefaultParams()
	p.Placement, p.Lift = &local, &lift
	lifted, err := ballbearing.Build(t.Context(), doc, p)
	require.NoError(t, err)
	localTransform, err := r3.FromFrame(local)
	require.NoError(t, err)
	composed, err := localTransform.Then(lift)
	require.NoError(t, err)
	placement, err := r3.NewFrame(composed.Translation(), composed.Basis().EX, composed.Basis().EY)
	require.NoError(t, err)
	p.Placement, p.Lift = &placement, nil
	placed, err := ballbearing.Build(t.Context(), doc, p)
	require.NoError(t, err)
	a, err := lifted.Body.Bounds()
	require.NoError(t, err)
	b, err := placed.Body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 0, a.Min.Sub(b.Min).Len(), 1e-9)
	require.InDelta(t, 0, a.Max.Sub(b.Max).Len(), 1e-9)
	va, err := lifted.Body.Volume()
	require.NoError(t, err)
	vb, err := placed.Body.Volume()
	require.NoError(t, err)
	require.InDelta(t, vb.Value.Base(), va.Value.Base(), 1e-9)
	requireBore(t, lifted.Body, 4, lift.ApplyDir(r3.NewVec(0, 0, 1)))
	report := lifted.Sketch.Verify(t.Context())
	require.True(t, report.Status == sketch.FullyConstrained, "status %s", report.Status)
	require.Zero(t, report.DOF)
	frame, err := lifted.Sketch.Plane().Frame()
	require.NoError(t, err)
	require.Equal(t, local.Origin(), frame.Origin(), "the section lies on the local placement plane")
	invalid := ballbearing.DefaultParams()
	invalid.Lift = &r3.Transform{}
	_, err = ballbearing.Build(t.Context(), decad.New(), invalid)
	require.Error(t, err)
}

func TestBearingCustomDimensions(t *testing.T) {
	p := ballbearing.Params{BoreDiameter: 12, OuterDiameter: 26, Width: 6}
	m, err := ballbearing.Build(t.Context(), decad.New(), p)
	require.NoError(t, err)
	requireBore(t, m.Body, 6, r3.NewVec(0, 0, 1))
	volume, err := m.Body.Volume()
	require.NoError(t, err)
	require.InDelta(t, math.Pi*(169-36)*6, volume.Value.Base(), volume.Bound.Base()+1e-6)
	mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(.05), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.VolumeVerified())
	report := m.Sketch.Verify(t.Context())
	require.True(t, report.Status == sketch.FullyConstrained, "status %s", report.Status)
	require.Zero(t, report.DOF)
	require.Zero(t, len(report.Redundant))
	require.Zero(t, len(report.Conflicts))
}

func TestBearingRejectsInvalidInputs(t *testing.T) {
	for _, item := range []struct {
		name string
		p    ballbearing.Params
	}{
		{"zero-bore", ballbearing.Params{OuterDiameter: 22, Width: 7}},
		{"negative-bore", ballbearing.Params{BoreDiameter: -1, OuterDiameter: 22, Width: 7}},
		{"zero-outside", ballbearing.Params{BoreDiameter: 8, Width: 7}},
		{"zero-width", ballbearing.Params{BoreDiameter: 8, OuterDiameter: 22}},
		{"equal-diameters", ballbearing.Params{BoreDiameter: 8, OuterDiameter: 8, Width: 7}},
		{"inverted-diameters", ballbearing.Params{BoreDiameter: 22, OuterDiameter: 8, Width: 7}},
		{"nan-bore", ballbearing.Params{BoreDiameter: math.NaN(), OuterDiameter: 22, Width: 7}},
		{"inf-bore", ballbearing.Params{BoreDiameter: math.Inf(1), OuterDiameter: 22, Width: 7}},
		{"nan-outside", ballbearing.Params{BoreDiameter: 8, OuterDiameter: math.NaN(), Width: 7}},
		{"inf-outside", ballbearing.Params{BoreDiameter: 8, OuterDiameter: math.Inf(1), Width: 7}},
		{"nan-width", ballbearing.Params{BoreDiameter: 8, OuterDiameter: 22, Width: math.NaN()}},
		{"inf-width", ballbearing.Params{BoreDiameter: 8, OuterDiameter: 22, Width: math.Inf(1)}},
		{"invalid-frame", ballbearing.Params{BoreDiameter: 8, OuterDiameter: 22, Width: 7, Placement: &r3.Frame{}}},
	} {
		t.Run(item.name, func(t *testing.T) {
			doc := decad.New()
			_, err := ballbearing.Build(t.Context(), doc, item.p)
			require.Error(t, err)
			require.Zero(t, len(doc.Bodies()))
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	doc := decad.New()
	_, err := ballbearing.Build(ctx, doc, ballbearing.DefaultParams())
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, len(doc.Bodies()))
	_, err = ballbearing.Build(t.Context(), nil, ballbearing.DefaultParams())
	require.ErrorContains(t, err, "document")
	// Build explicitly rejects a nil context at its public boundary.
	_, err = ballbearing.Build(nil, doc, ballbearing.DefaultParams()) //nolint:staticcheck
	require.ErrorContains(t, err, "context")
}
