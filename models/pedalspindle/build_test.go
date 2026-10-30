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
	requireHeelSupport(t, a, 1)
	require.Equal(t, len(a.Parts), len(doc.Bodies()), "the document must contain only assembled parts")
	requireAssembly(t, a, 1)
	requireDimensions(t, doc, a, 1, math.Pi/3)
	requireStockBearings(t, doc, a, 1)
	requireReturnEnergy(t, a)
	requireGearMesh(t, a)
}

func TestPedalSpindleVariants(t *testing.T) {
	for _, p := range []pedalspindle.Params{{Scale: 0.8, CrankAngle: math.Pi}, {Scale: 1.2, CrankAngle: -math.Pi / 2}} {
		t.Run(stringScale(p.Scale), func(t *testing.T) {
			doc := decad.New()
			a, err := pedalspindle.Build(t.Context(), doc, p)
			require.NoError(t, err)
			requireAssembly(t, a, p.Scale)
			requireDimensions(t, doc, a, p.Scale, p.CrankAngle)
			requireStockBearings(t, doc, a, p.Scale)
			requireHeelSupport(t, a, p.Scale)
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
	require.Equal(t, 48, len(a.Parts))
	require.Equal(t, 58, len(a.Sketches))
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

func requireDimensions(t *testing.T, doc *decad.Document, a *pedalspindle.Assembly, scale, crankAngle float64) {
	t.Helper()
	for _, item := range []struct {
		name     string
		min, max r3.Vec
	}{
		{"base", r3.NewVec(-270, -240, 0), r3.NewVec(150, 110, 18)},
		{"flywheel-rim", r3.NewVec(-245, 35, 225), r3.NewVec(65, 53, 535)},
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
		boreRadius := 4.1
		flat, angle := 3.1, math.Mod(crankAngle, 2*math.Pi)
		if item.name == "intermediate-pinion" || item.name == "intermediate-gear" {
			angle = a.State.IntermediateAngle
		}
		if item.name == "output-pinion" {
			boreRadius = 2.6
			flat, angle = 1.85, a.State.SpindleAngle
		}
		box, err := body.Bounds()
		require.NoError(t, err)
		require.InDelta(t, (item.y-12)*scale, box.Min.Y, box.Bound.Base()+1e-6, "%s thickness", item.name)
		require.InDelta(t, item.y*scale, box.Max.Y, box.Bound.Base()+1e-6, "%s placement", item.name)
		requireDBore(t, doc.OperationHistory(), body, r3.NewVec(item.x*scale, 0, item.z*scale), (item.y-6)*scale, angle, boreRadius, flat)
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
			radius := 7.0
			if item.name == "treadle" {
				radius = 11
			}
			if item.name == "crank" || (item.name == "treadle" && want.Sub(a.State.Toe).Len() < 1e-6) {
				radius = 2.6
			}
			found := 0
			for _, face := range partByName(t, a, item.name).Faces() {
				if face.Surface().Kind() != decad.KindPlane {
					continue
				}
				for _, loop := range face.Loops() {
					for _, edge := range loop.Edges() {
						c, ok := edge.Curve().(decad.Circle3)
						if ok && math.Hypot(c.Center.X-want.X, c.Center.Z-want.Z) < 1e-6 {
							require.InDelta(t, radius, c.Radius.Base(), 1e-6, "%s stock pivot radius", item.name)
							found++
						}
					}
				}
			}
			require.Equal(t, 2, found, "%s pivot must appear on both caps", item.name)
		}
	}
	spindle := partByName(t, a, "spindle")
	box, err := spindle.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 0, box.Min.Sub(r3.NewVec(12.3*scale-2.5, -270*scale, 482.3*scale-2.5)).Len(), box.Bound.Base()+1e-6)
	require.InDelta(t, 0, box.Max.Sub(r3.NewVec(12.3*scale+2.5, 30*scale, 482.3*scale+2.5)).Len(), box.Bound.Base()+1e-6)
	volume, err := spindle.Volume()
	require.NoError(t, err)
	wantVolume := math.Pi * (6.25*275 + 25.0/3*(6.25+2.5*.3+.3*.3)) * scale
	wantVolume -= circularCapArea(2.5, 1.75) * 34 * scale
	require.InDelta(t, wantVolume, volume.Value.Base(), volume.Bound.Base()+1e-5, "exact cylindrical shaft and conical tip")
}

// Check the imported bearing bodies after the actual spindle consumer placed them.
func requireStockBearings(t *testing.T, doc *decad.Document, a *pedalspindle.Assembly, scale float64) {
	t.Helper()
	for _, item := range []struct {
		name                   string
		center                 r3.Vec
		bore, outer, low, high float64
	}{
		{"input-bearing-front", a.State.Input, 8, 22, 0, 7},
		{"input-bearing-rear", a.State.Input, 8, 22, 18*scale - 7, 18 * scale},
		{"intermediate-bearing-front", a.State.Intermediate, 8, 22, 0, 7},
		{"intermediate-bearing-rear", a.State.Intermediate, 8, 22, 18*scale - 7, 18 * scale},
		{"output-bearing-front", a.State.Output, 5, 14, 0, 5},
		{"output-bearing-rear", a.State.Output, 5, 14, 18*scale - 5, 18 * scale},
		{"heel-bearing-front", a.State.Heel, 8, 22, -210 * scale, -210*scale + 7},
		{"heel-bearing-rear", a.State.Heel, 8, 22, -140*scale - 7, -140 * scale},
		{"footman-bearing-crank", a.State.CrankPin, 5, 14, -125*scale - 2.5, -125*scale + 2.5},
		{"footman-bearing-toe", a.State.Toe, 5, 14, -125*scale - 2.5, -125*scale + 2.5},
	} {
		body := partByName(t, a, item.name)
		box, err := body.Bounds()
		require.NoError(t, err)
		wantMin := r3.NewVec(item.center.X-item.outer/2, item.low, item.center.Z-item.outer/2)
		wantMax := r3.NewVec(item.center.X+item.outer/2, item.high, item.center.Z+item.outer/2)
		require.InDelta(t, 0, box.Min.Sub(wantMin).Len(), box.Bound.Base()+1e-6, "%s stock bounds", item.name)
		require.InDelta(t, 0, box.Max.Sub(wantMax).Len(), box.Bound.Base()+1e-6, "%s stock bounds", item.name)
		volume, err := body.Volume()
		require.NoError(t, err)
		wantVolume := math.Pi * (item.outer*item.outer - item.bore*item.bore) / 4 * (item.high - item.low)
		require.InDelta(t, wantVolume, volume.Value.Base(), volume.Bound.Base()+1e-6, "%s envelope volume", item.name)
		caps := 0
		for _, face := range body.Faces() {
			if face.Surface().Kind() != decad.KindPlane {
				continue
			}
			require.Equal(t, 2, len(face.Loops()), "%s annular cap", item.name)
			circle, ok := face.Loops()[1].Edges()[0].Curve().(decad.Circle3)
			require.True(t, ok)
			require.InDelta(t, item.bore/2, circle.Radius.Base(), 1e-6, "%s stock bore", item.name)
			require.InDelta(t, 1, math.Abs(circle.Axis.Y), 1e-6, "%s bearing axis", item.name)
			caps++
		}
		require.Equal(t, 2, caps, "%s cap count", item.name)
	}
	board := partByName(t, a, "bearing-board")
	for _, shaft := range []struct {
		name   string
		center r3.Vec
		radius float64
	}{
		{"input-shaft", a.State.Input, 11},
		{"intermediate-shaft", a.State.Intermediate, 11},
		{"spindle", a.State.Output, 7},
	} {
		found := 0
		for _, face := range board.Faces() {
			if face.Surface().Kind() != decad.KindPlane {
				continue
			}
			for _, loop := range face.Loops() {
				for _, edge := range loop.Edges() {
					c, ok := edge.Curve().(decad.Circle3)
					if ok && math.Hypot(c.Center.X-shaft.center.X, c.Center.Z-shaft.center.Z) < 1e-6 {
						require.InDelta(t, shaft.radius, c.Radius.Base(), 1e-6, "%s stock housing bore", shaft.name)
						found++
					}
				}
			}
		}
		require.Equal(t, 2, found, "%s housing bore must pass through board", shaft.name)
		radius := 4.0
		backY := 65 * scale
		if shaft.name == "intermediate-shaft" {
			backY = 26 * scale
		}
		if shaft.name == "spindle" {
			radius = 2.5
			backY = 30 * scale
		}
		requireRoundJournal(t, doc.OperationHistory(), partByName(t, a, shaft.name), shaft.center, 9*scale, radius, backY-9*scale)
	}
}

func circularCapArea(radius, flat float64) float64 {
	return radius*radius*math.Acos(flat/radius) - flat*math.Sqrt(radius*radius-flat*flat)
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
