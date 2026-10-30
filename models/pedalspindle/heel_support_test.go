package pedalspindle_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/pedalspindle"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func requireHeelSupport(t *testing.T, a *pedalspindle.Assembly, scale float64) {
	t.Helper()
	for _, item := range []struct {
		name     string
		min, max r3.Vec
	}{
		{"heel-bracket-front", r3.NewVec(-250*scale, -222*scale, 18*scale), r3.NewVec(-210*scale, -214*scale, 84*scale)},
		{"heel-bracket-rear", r3.NewVec(-250*scale, -136*scale, 18*scale), r3.NewVec(-210*scale, -128*scale, 84*scale)},
		{"heel-pin", r3.NewVec(-230*scale-4, -230*scale, 60*scale-4), r3.NewVec(-230*scale+4, -120*scale, 60*scale+4)},
		{"heel-collar-front", r3.NewVec(-238*scale, -228*scale, 52*scale), r3.NewVec(-222*scale, -222*scale, 68*scale)},
		{"heel-collar-rear", r3.NewVec(-238*scale, -128*scale, 52*scale), r3.NewVec(-222*scale, -122*scale, 68*scale)},
		{"heel-spacer-front", r3.NewVec(-230*scale-6, -214*scale, 60*scale-6), r3.NewVec(-230*scale+6, -210.1*scale, 60*scale+6)},
		{"heel-spacer-rear", r3.NewVec(-230*scale-6, -139.9*scale, 60*scale-6), r3.NewVec(-230*scale+6, -136*scale, 60*scale+6)},
	} {
		body := partByName(t, a, item.name)
		box, err := body.Bounds()
		require.NoError(t, err)
		require.InDelta(t, 0, box.Min.Sub(item.min).Len(), box.Bound.Base()+1e-6, "%s min", item.name)
		require.InDelta(t, 0, box.Max.Sub(item.max).Len(), box.Bound.Base()+1e-6, "%s max", item.name)
		if item.name == "heel-pin" {
			continue
		}
		bores := 0
		for _, face := range body.Faces() {
			if face.Surface().Kind() != decad.KindPlane {
				continue
			}
			for _, loop := range face.Loops() {
				for _, edge := range loop.Edges() {
					circle, ok := edge.Curve().(decad.Circle3)
					if !ok || math.Abs(circle.Radius.Base()-4.1) > 1e-6 {
						continue
					}
					require.InDelta(t, a.State.Heel.X, circle.Center.X, 1e-6, "%s axis X", item.name)
					require.InDelta(t, a.State.Heel.Z, circle.Center.Z, 1e-6, "%s axis Z", item.name)
					bores++
				}
			}
		}
		require.Equal(t, 2, bores, "%s through bore", item.name)
	}
	// Read every stepped outline corner after placement, including the enlarged bearing seat.
	treadle := partByName(t, a, "treadle")
	for _, point := range [][2]float64{{-18, -18}, {18, -18}, {18, -12}, {212, -12}, {212, 12}, {18, 12}, {18, 18}, {-18, 18}} {
		x, z := point[0]*scale, point[1]*scale
		want := r3.NewVec(a.State.Heel.X+x*math.Cos(a.State.PedalAngle)-z*math.Sin(a.State.PedalAngle), 0,
			a.State.Heel.Z+x*math.Sin(a.State.PedalAngle)+z*math.Cos(a.State.PedalAngle))
		found := 0
		for _, vertex := range treadle.Vertices() {
			v := vertex.Position().Value
			if math.Hypot(v.X-want.X, v.Z-want.Z) < 1e-6 {
				found++
			}
		}
		require.Equal(t, 2, found, "stepped corner (%g,%g)", point[0], point[1])
	}
	volume, err := treadle.Volume()
	require.NoError(t, err)
	wantVolume := ((36*36+194*24)*scale*scale - math.Pi*(11*11+2.6*2.6)) * 70 * scale
	require.InDelta(t, wantVolume, volume.Value.Base(), volume.Bound.Base()+1e-5, "stepped treadle with stock bores")
}
