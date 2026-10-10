package spurgear_test

import (
	"math"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestDefaultGear(t *testing.T) {
	doc := decad.New()
	body, err := spurgear.Build(t.Context(), doc, spurgear.DefaultParams())
	require.NoError(t, err)
	require.True(t, body.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	volume, err := body.Volume()
	require.NoError(t, err)
	require.InDelta(t, 2218.77, volume.Value.Base(), volume.Bound.Base()+0.02)
	require.Less(t, volume.Bound.Base(), 0.5)
	box, err := body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 0, box.Min.Z, box.Bound.Base()+1e-8)
	require.InDelta(t, 10, box.Max.Z, box.Bound.Base()+1e-8)
	require.Greater(t, box.Max.X, 9.4)
	require.Less(t, box.Max.X, 9.6)
	roles := map[string]struct{}{}
	for _, face := range body.Faces() {
		for _, origin := range face.Origins() {
			if strings.HasPrefix(origin.Role, "fillet(0,") {
				roles[origin.Role] = struct{}{}
			}
		}
	}
	require.Len(t, roles, 2*spurgear.DefaultParams().Teeth)
}

func TestThroughBore(t *testing.T) {
	base, err := spurgear.Build(t.Context(), decad.New(), spurgear.DefaultParams())
	require.NoError(t, err)
	baseVolume, err := base.Volume()
	require.NoError(t, err)
	p := spurgear.DefaultParams()
	p.BoreDiameter = 3
	doc := decad.New()
	gear, err := spurgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, gear.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	withBore, err := gear.Volume()
	require.NoError(t, err)
	wantRemoved := math.Pi * math.Pow(p.BoreDiameter/2, 2) * p.Thickness
	require.InDelta(t, wantRemoved, baseVolume.Value.Base()-withBore.Value.Base(),
		baseVolume.Bound.Base()+withBore.Bound.Base()+0.02)
}

func TestBoreAndCapChamfer(t *testing.T) {
	p := spurgear.DefaultParams()
	p.BoreDiameter = 3
	withoutChamfer, err := spurgear.Build(t.Context(), decad.New(), p)
	require.NoError(t, err)
	baseVolume, err := withoutChamfer.Volume()
	require.NoError(t, err)

	p.CapChamfer = 0.1
	doc := decad.New()
	gear, err := spurgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, gear.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	chamferedVolume, err := gear.Volume()
	require.NoError(t, err)
	require.Greater(t, baseVolume.Value.Base()-chamferedVolume.Value.Base(),
		baseVolume.Bound.Base()+chamferedVolume.Bound.Base())
	for _, cap := range []decad.FeatureRef{decad.CapStart(gear), decad.CapEnd(gear)} {
		faces, err := decad.Faces(decad.FaceCreatedBy(cap)).Exactly(1).SelectFaces(gear)
		require.NoError(t, err)
		require.Len(t, faces[0].Loops(), 2)
		require.Len(t, faces[0].Loops()[1].Edges(), 1, "the bore rim stays sharp")
		circle, ok := faces[0].Loops()[1].Edges()[0].Curve().(decad.Circle3)
		require.True(t, ok)
		require.InDelta(t, p.BoreDiameter/2, circle.Radius.Base(), 1e-10)
	}
}

func TestCapChamferWithoutBore(t *testing.T) {
	p := spurgear.DefaultParams()
	plain, err := spurgear.Build(t.Context(), decad.New(), p)
	require.NoError(t, err)
	plainVolume, err := plain.Volume()
	require.NoError(t, err)

	p.CapChamfer = 0.1
	doc := decad.New()
	gear, err := spurgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, gear.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	chamferedVolume, err := gear.Volume()
	require.NoError(t, err)
	require.Greater(t, plainVolume.Value.Base()-chamferedVolume.Value.Base(),
		plainVolume.Bound.Base()+chamferedVolume.Bound.Base())

	for _, caps := range [][2]decad.FeatureRef{
		{decad.CapStart(plain), decad.CapStart(gear)},
		{decad.CapEnd(plain), decad.CapEnd(gear)},
	} {
		plainFaces, err := decad.Faces(decad.FaceCreatedBy(caps[0])).Exactly(1).SelectFaces(plain)
		require.NoError(t, err)
		chamferedFaces, err := decad.Faces(decad.FaceCreatedBy(caps[1])).Exactly(1).SelectFaces(gear)
		require.NoError(t, err)
		require.Len(t, chamferedFaces[0].Loops(), 1)
		plainArea, err := plainFaces[0].Area()
		require.NoError(t, err)
		chamferedArea, err := chamferedFaces[0].Area()
		require.NoError(t, err)
		require.Greater(t, plainArea.Value.Base()-chamferedArea.Value.Base(),
			plainArea.Bound.Base()+chamferedArea.Bound.Base())
	}
}

func TestEmbeddedThirtyToothGearWithoutRootFillets(t *testing.T) {
	p := spurgear.Params{
		Module: 0.5, Teeth: 30, PressureAngle: 30 * math.Pi / 180,
		Thickness: 2, Samples: 5, CapChamfer: 0.02,
	}
	doc := decad.New()
	gear, err := spurgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, gear.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	box, err := gear.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 0, box.Min.Z, box.Bound.Base()+1e-8)
	require.InDelta(t, p.Thickness, box.Max.Z, box.Bound.Base()+1e-8)
	chamferFaces := 0
	for _, face := range gear.Faces() {
		for _, origin := range face.Origins() {
			if strings.HasPrefix(origin.Role, "chamferCap(") {
				chamferFaces++
				break
			}
		}
	}
	require.Positive(t, chamferFaces)
	for _, cap := range []decad.FeatureRef{decad.CapStart(gear), decad.CapEnd(gear)} {
		faces, err := decad.Faces(decad.FaceCreatedBy(cap)).Exactly(1).SelectFaces(gear)
		require.NoError(t, err)
		require.Len(t, faces[0].Loops(), 1, "the no-bore cap has one loop")
	}
}

func TestEmbeddedSixtyToothGearWithRootFilletsAndBore(t *testing.T) {
	p := spurgear.Params{
		Module: 0.5, Teeth: 60, PressureAngle: 20 * math.Pi / 180,
		Thickness: 2, Samples: 15, BoreDiameter: 3, CapChamfer: 0.02,
	}
	doc := decad.New()
	gear, err := spurgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, gear.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	chamferFaces := 0
	for _, face := range gear.Faces() {
		for _, origin := range face.Origins() {
			if strings.HasPrefix(origin.Role, "chamferCap(") {
				chamferFaces++
				break
			}
		}
	}
	require.Positive(t, chamferFaces)
	for _, cap := range []decad.FeatureRef{decad.CapStart(gear), decad.CapEnd(gear)} {
		faces, err := decad.Faces(decad.FaceCreatedBy(cap)).Exactly(1).SelectFaces(gear)
		require.NoError(t, err)
		require.Len(t, faces[0].Loops(), 2)
		require.Len(t, faces[0].Loops()[1].Edges(), 1, "the bore rim stays sharp")
	}
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}
