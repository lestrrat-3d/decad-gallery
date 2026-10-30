package pedalspindle_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/pedalspindle"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestPedalSpindleDefault(t *testing.T) {
	doc := decad.New()
	a, err := pedalspindle.Build(t.Context(), doc, pedalspindle.DefaultParams())
	require.NoError(t, err)
	require.Equal(t, len(a.Parts), len(doc.Bodies()), "the document must contain only assembled parts")
	requireAssembly(t, a, 1)
	requireDimensions(t, a, 1)
	requireGearMesh(t, a)
}

func TestPedalSpindleVariants(t *testing.T) {
	for _, p := range []pedalspindle.Params{{Scale: 0.8, CrankAngle: math.Pi}, {Scale: 1.2, CrankAngle: -math.Pi / 2}} {
		t.Run(stringScale(p.Scale), func(t *testing.T) {
			a, err := pedalspindle.Build(t.Context(), decad.New(), p)
			require.NoError(t, err)
			requireAssembly(t, a, p.Scale)
			requireDimensions(t, a, p.Scale)
		})
	}
}

func stringScale(scale float64) string {
	if scale == 0.8 {
		return "small"
	}
	return "large"
}

// Failure reports contain names and scalar values rather than body or mesh graphs.
func requireAssembly(t *testing.T, a *pedalspindle.Assembly, scale float64) {
	t.Helper()
	require.Equal(t, 36, len(a.Parts))
	require.Equal(t, 32, len(a.Sketches))
	names := map[string]struct{}{}
	for _, part := range a.Parts {
		_, duplicate := names[part.Name]
		require.False(t, duplicate, "duplicate part %s", part.Name)
		names[part.Name] = struct{}{}
		require.True(t, part.Body.IsSolid(), "%s is not solid", part.Name)
		mesh, err := part.Body.Tessellate(t.Context(), units.Millimeters(0.1*scale),
			decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, "part %s", part.Name)
		require.True(t, mesh.BoundaryVerified(), "part %s", part.Name)
		require.True(t, mesh.VolumeVerified(), "part %s", part.Name)
	}
	for i, s := range a.Sketches {
		report := s.Verify(t.Context())
		require.True(t, report.Status == sketch.FullyConstrained, "sketch %d status %s", i, report.Status)
		require.Zero(t, report.DOF, "sketch %d", i)
		require.Len(t, report.Redundant, 0, "sketch %d", i)
		require.Len(t, report.Conflicts, 0, "sketch %d", i)
		require.NoError(t, report.Check(), "sketch %d", i)
	}
}

func partByName(t *testing.T, a *pedalspindle.Assembly, name string) *decad.Body {
	t.Helper()
	for _, p := range a.Parts {
		if p.Name == name {
			return p.Body
		}
	}
	t.Fatalf("missing part %s", name)
	return nil
}

func requireDimensions(t *testing.T, a *pedalspindle.Assembly, scale float64) {
	t.Helper()
	for _, item := range []struct {
		name     string
		min, max r3.Vec
	}{
		{"base", r3.NewVec(-270, -240, 0), r3.NewVec(150, 110, 18)},
		{"flywheel-rim", r3.NewVec(-245, 35, 225), r3.NewVec(65, 53, 535)},
		{"spindle", r3.NewVec(7.3, -270, 477.3), r3.NewVec(17.3, 30, 487.3)},
	} {
		b, err := partByName(t, a, item.name).Bounds()
		require.NoError(t, err)
		wantMin, wantMax := item.min.Scale(scale), item.max.Scale(scale)
		require.InDelta(t, 0, b.Min.Sub(wantMin).Len(), b.Bound.Base()+1e-6, "%s min", item.name)
		require.InDelta(t, 0, b.Max.Sub(wantMax).Len(), b.Bound.Base()+1e-6, "%s max", item.name)
	}
	for _, item := range []struct {
		name    string
		x, z, y float64
	}{
		{"input-gear", -90, 380, -30}, {"intermediate-pinion", 12.3, 380, -30},
		{"intermediate-gear", 12.3, 380, -60}, {"output-pinion", 12.3, 482.3, -60},
	} {
		body := partByName(t, a, item.name)
		box, err := body.Bounds()
		require.NoError(t, err)
		require.InDelta(t, (item.y-12)*scale, box.Min.Y, box.Bound.Base()+1e-6, "%s thickness", item.name)
		require.InDelta(t, item.y*scale, box.Max.Y, box.Bound.Base()+1e-6, "%s placement", item.name)
		caps := 0
		for _, face := range body.Faces() {
			if face.Surface().Kind() != decad.KindPlane || len(face.Loops()) != 2 {
				continue
			}
			bore := face.Loops()[1]
			require.Equal(t, 1, len(bore.Edges()), "%s bore rim", item.name)
			circle, ok := bore.Edges()[0].Curve().(decad.Circle3)
			require.True(t, ok, "%s has a circular bore", item.name)
			require.InDelta(t, 5.1*scale, circle.Radius.Base(), 1e-6, "%s bore radius", item.name)
			require.InDelta(t, item.x*scale, circle.Center.X, 1e-6, "%s bore center X", item.name)
			require.InDelta(t, item.z*scale, circle.Center.Z, 1e-6, "%s bore center Z", item.name)
			caps++
		}
		require.Equal(t, 2, caps, "%s has two bored caps", item.name)
	}
	// Read the actual pivot bores after placement, independently of Motion's closure equations.
	for _, item := range []struct {
		name    string
		centers []r3.Vec
	}{
		{"treadle", []r3.Vec{a.State.Heel, a.State.Toe}},
		{"footman", []r3.Vec{a.State.Toe, a.State.CrankPin}},
		{"crank", []r3.Vec{a.State.CrankPin}},
	} {
		for _, want := range item.centers {
			found := 0
			for _, face := range partByName(t, a, item.name).Faces() {
				if face.Surface().Kind() != decad.KindPlane {
					continue
				}
				for _, loop := range face.Loops() {
					for _, edge := range loop.Edges() {
						c, ok := edge.Curve().(decad.Circle3)
						if ok && math.Hypot(c.Center.X-want.X, c.Center.Z-want.Z) < 1e-6 {
							require.InDelta(t, 4.2*scale, c.Radius.Base(), 1e-6, "%s pivot radius", item.name)
							found++
						}
					}
				}
			}
			require.Equal(t, 2, found, "%s pivot must appear on both caps", item.name)
		}
	}
	spindle := partByName(t, a, "spindle")
	volume, err := spindle.Volume()
	require.NoError(t, err)
	wantVolume := math.Pi * (25*275 + 25.0/3*(25+5*.6+.6*.6)) * scale * scale * scale
	require.InDelta(t, wantVolume, volume.Value.Base(), volume.Bound.Base()+1e-5, "exact cylindrical shaft and conical tip")
}

// Sample one full input-tooth period. The second stage traverses three periods.
func requireGearMesh(t *testing.T, a *pedalspindle.Assembly) {
	t.Helper()
	for i := range 13 {
		delta := 2 * math.Pi * float64(i) / (51 * 12)
		for _, pair := range []struct {
			first, second     string
			centerA, centerB  r3.Vec
			angleA, angleB, y float64
		}{
			{"input-gear", "intermediate-pinion", a.State.Input, a.State.Intermediate, delta, -3 * delta, -36},
			{"intermediate-gear", "output-pinion", a.State.Intermediate, a.State.Output, -3 * delta, 9 * delta, -66},
		} {
			plane, err := r3.NewFrame(r3.NewVec(0, pair.y, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 0, 1))
			require.NoError(t, err)
			outlines := make([]decad.SectionOutline, 0, 2)
			for _, member := range []struct {
				name   string
				center r3.Vec
				angle  float64
			}{
				{pair.first, pair.centerA, pair.angleA}, {pair.second, pair.centerB, pair.angleB},
			} {
				body, err := decad.Copy(t.Context(), partByName(t, a, member.name),
					decad.WithCopyRotation(member.center, r3.NewVec(0, -1, 0), units.Radians(member.angle)))
				require.NoError(t, err, "%s phase %d", member.name, i)
				outline, err := body.SectionOutline(t.Context(), plane, units.Millimeters(0.03))
				require.NoError(t, err, "%s phase %d", member.name, i)
				require.Equal(t, 2, len(outline.Loops), "%s has outer outline and bore", member.name)
				outlines = append(outlines, outline)
			}
			gap, err := outlines[0].Clearance(outlines[1])
			require.NoError(t, err, "%s phase %d", pair.first, i)
			require.False(t, gap.Overlaps, "%s phase %d", pair.first, i)
			require.Greater(t, gap.Lower().Base(), 0.0, "%s phase %d certified gap", pair.first, i)
			require.Less(t, gap.Distance.Value.Base(), 0.4, "%s phase %d stays in mesh", pair.first, i)
		}
	}
}
