package pedalspindle_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/pedalspindle"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestPedalSpindleFrameStaysWithGuardRemoved(t *testing.T) {
	doc := decad.New()
	a, err := pedalspindle.Build(t.Context(), doc, pedalspindle.DefaultParams())
	require.NoError(t, err)
	requirePermanentFrame(t, a, 1)
	before := make(map[string]decad.Box, len(a.Parts))
	for _, part := range a.Parts {
		box, err := part.Body.Bounds()
		require.NoError(t, err)
		before[part.Name] = box
	}
	removed := 0
	for _, part := range a.Parts {
		if part.Group == "guard" {
			require.NoError(t, decad.Remove(part.Body))
			removed++
		}
	}
	require.Equal(t, 5, removed)
	require.Equal(t, len(a.Parts)-removed, len(doc.Bodies()))
	remaining := make(map[*decad.Body]struct{}, len(doc.Bodies()))
	for _, body := range doc.Bodies() {
		remaining[body] = struct{}{}
	}
	for _, part := range a.Parts {
		if part.Group == "guard" {
			continue
		}
		_, exists := remaining[part.Body]
		require.True(t, exists, "%s must remain when the cover is removed", part.Name)
		box, err := part.Body.Bounds()
		require.NoError(t, err)
		require.Zero(t, box.Min.Sub(before[part.Name].Min).Len(), "%s lower bounds moved", part.Name)
		require.Zero(t, box.Max.Sub(before[part.Name].Max).Len(), "%s upper bounds moved", part.Name)
	}
}

func requirePermanentFrame(t *testing.T, a *pedalspindle.Assembly, scale float64) {
	t.Helper()
	for _, axis := range []struct {
		name  string
		width float64
	}{
		{"input", 7}, {"intermediate", 7}, {"output", 5},
	} {
		front, err := partByName(t, a, axis.name+"-bearing-front").Bounds()
		require.NoError(t, err)
		rear, err := partByName(t, a, axis.name+"-bearing-rear").Bounds()
		require.NoError(t, err)
		span := (rear.Min.Y + rear.Max.Y - front.Min.Y - front.Max.Y) / 2
		require.Greater(t, span, 80.0, "%s bearings must straddle the gears", axis.name)
		require.InDelta(t, 113*scale-axis.width, span, 1e-6, "%s bearing-center span", axis.name)
		require.InDelta(t, -95*scale, front.Min.Y, 1e-6)
		require.InDelta(t, 18*scale, rear.Max.Y, 1e-6)
	}
	frame := partByName(t, a, "front-bearing-frame")
	requirePartBox(t, a, "front-bearing-frame", r3.NewVec(-175*scale, -95*scale, 18*scale),
		r3.NewVec(100*scale, -83*scale, 510*scale))
	base, err := partByName(t, a, "base").Bounds()
	require.NoError(t, err)
	box, err := frame.Bounds()
	require.NoError(t, err)
	require.InDelta(t, base.Max.Z, box.Min.Z, 1e-6, "integral frame feet must meet the base")
	volume, err := frame.Volume()
	require.NoError(t, err)
	area := (275*492-235*280-235*(44+70.3))*scale*scale - math.Pi*(2*121+49+4*3.3*3.3)
	require.InDelta(t, area*12*scale, volume.Value.Base(), volume.Bound.Base()+1e-5, "open portal volume")
	requirePortalCaps(t, frame, scale)
	guard, err := partByName(t, a, "guard-front").Bounds()
	require.NoError(t, err)
	flywheel, err := partByName(t, a, "flywheel-rim").Bounds()
	require.NoError(t, err)
	for i, corner := range [][2]float64{{-165, 308}, {90, 308}, {-165, 500}, {90, 500}} {
		x, z := corner[0]*scale, corner[1]*scale
		standoff := fmt.Sprintf("frame-standoff-%d", i)
		requirePartBox(t, a, standoff, r3.NewVec(x-10*scale, -83*scale, z-10*scale),
			r3.NewVec(x+10*scale, 0, z+10*scale))
		bolt := fmt.Sprintf("frame-bolt-%d", i)
		requirePartBox(t, a, bolt, r3.NewVec(x-6, -95*scale-4.2, z-6), r3.NewVec(x+6, 30*scale, z+6))
		nut := fmt.Sprintf("frame-nut-%d", i)
		requirePartBox(t, a, nut, r3.NewVec(x-6, 18*scale, z-6), r3.NewVec(x+6, 18*scale+5.5, z+6))
		for _, housing := range []string{"bearing-board", "front-bearing-frame"} {
			requireHousingBore(t, partByName(t, a, housing), r3.NewVec(x, 0, z), 3.3, housing+" / "+bolt)
		}
		for _, item := range []struct {
			name   string
			volume float64
		}{
			{standoff, math.Pi * (100*scale*scale - 3.3*3.3) * 83 * scale},
			{bolt, math.Pi * (9*125*scale + 36*4.2)},
			{nut, math.Pi * (36 - 9) * 5.5},
		} {
			got, err := partByName(t, a, item.name).Volume()
			require.NoError(t, err)
			require.InDelta(t, item.volume, got.Value.Base(), got.Bound.Base()+1e-5, "%s hardware envelope", item.name)
		}
		bb, err := partByName(t, a, bolt).Bounds()
		require.NoError(t, err)
		require.Greater(t, bb.Min.Y-guard.Max.Y-bb.Bound.Base()-guard.Bound.Base(), .59, "%s cover clearance", bolt)
		require.Greater(t, flywheel.Min.Y-bb.Max.Y-flywheel.Bound.Base()-bb.Bound.Base(), 3.99, "%s flywheel clearance", bolt)
	}
}

func requireHousingBore(t *testing.T, body *decad.Body, center r3.Vec, radius float64, name string) {
	t.Helper()
	found := 0
	for _, face := range body.Faces() {
		if face.Surface().Kind() != decad.KindPlane {
			continue
		}
		for _, loop := range face.Loops() {
			for _, edge := range loop.Edges() {
				circle, ok := edge.Curve().(decad.Circle3)
				if !ok || math.Hypot(circle.Center.X-center.X, circle.Center.Z-center.Z) > 1e-6 {
					continue
				}
				require.InDelta(t, radius, circle.Radius.Base(), 1e-6, "%s housing radius", name)
				require.InDelta(t, 1, math.Abs(circle.Axis.Y), 1e-6, "%s coaxial seat", name)
				found++
			}
		}
	}
	require.Equal(t, 2, found, "%s must have a through bore", name)
}

func requirePartBox(t *testing.T, a *pedalspindle.Assembly, name string, low, high r3.Vec) {
	t.Helper()
	box, err := partByName(t, a, name).Bounds()
	require.NoError(t, err)
	require.InDelta(t, 0, box.Min.Sub(low).Len(), box.Bound.Base()+1e-6, "%s lower bounds", name)
	require.InDelta(t, 0, box.Max.Sub(high).Len(), box.Bound.Base()+1e-6, "%s upper bounds", name)
}

func requirePortalCaps(t *testing.T, frame *decad.Body, scale float64) {
	t.Helper()
	caps := 0
	for _, face := range frame.Faces() {
		if face.Surface().Kind() != decad.KindPlane || len(face.Loops()) != 10 {
			continue
		}
		caps++
		outer := face.Loops()[0].Edges()
		require.Equal(t, 8, len(outer))
		for _, vertex := range [][2]float64{{-175, 18}, {-155, 18}, {-155, 298}, {80, 298}, {80, 18}, {100, 18}, {100, 510}, {-175, 510}} {
			found := false
			for _, edge := range outer {
				p := edge.Start().Position().Value
				found = found || math.Hypot(p.X-vertex[0]*scale, p.Z-vertex[1]*scale) < 1e-6
			}
			require.True(t, found, "portal vertex (%g,%g)", vertex[0], vertex[1])
		}
		circles, windows := 0, 0
		for _, loop := range face.Loops()[1:] {
			if len(loop.Edges()) == 1 {
				_, ok := loop.Edges()[0].Curve().(decad.Circle3)
				require.True(t, ok, "frame port must be circular")
				circles++
				continue
			}
			require.Equal(t, 4, len(loop.Edges()), "frame window must have four edges")
			minX, maxX, minZ, maxZ := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
			for _, edge := range loop.Edges() {
				p := edge.Start().Position().Value
				minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
				minZ, maxZ = math.Min(minZ, p.Z), math.Max(maxZ, p.Z)
			}
			require.InDelta(t, -155*scale, minX, 1e-6, "window left side")
			require.InDelta(t, 80*scale, maxX, 1e-6, "window right side")
			if math.Abs(minZ-318*scale) < 1e-6 {
				require.InDelta(t, 362*scale, maxZ, 1e-6, "lower service window")
			} else {
				require.InDelta(t, 398*scale, minZ, 1e-6, "upper service window")
				require.InDelta(t, 468.3*scale, maxZ, 1e-6, "upper service window")
			}
			windows++
		}
		require.Equal(t, 7, circles)
		require.Equal(t, 2, windows)
	}
	require.Equal(t, 2, caps, "portal must have two holed cap faces")
}
