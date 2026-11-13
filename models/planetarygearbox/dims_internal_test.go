package planetarygearbox

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

func TestDefaultDimensionsUnchanged(t *testing.T) {
	d := derive(DefaultParams())
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"orbit", d.orbit, 21}, {"planetRoot", d.planetRoot, 12.98}, {"planetTip", d.planetTip, 15.48},
		{"ringTip", d.ringTip, 34.5}, {"ringRoot", d.ringRoot, 37}, {"ringOuter", d.ringOuter, 44.1},
		{"pin", d.pin, 5}, {"pinCollar", d.pinCollar, 7.4}, {"planetPocket", d.planetPocket, 14.1},
		{"planetLip", d.planetLip, 10}, {"boss", d.boss, 11}, {"arm", d.arm, 7}, {"collar", d.collar, 7.4},
		{"shaft", d.shaft, 5}, {"shaftBore", d.shaftBore, 5.1}, {"coverPocket", d.coverPocket, 14.1},
		{"coverLip", d.coverLip, 10}, {"coverBoss", d.coverBoss, 20.1}, {"coverHub", d.coverHub, 26.1},
		{"boltCircle", d.boltCircle, 40.55}, {"rimInner", d.rimInner, 36.85}, {"stagePitch", d.stagePitch, 13},
		{"ringLength", d.ringLength, 14},
	} {
		require.InDelta(t, c.want, c.got, 1e-9, c.name)
	}
}

func TestParameterTableMatchesDerive(t *testing.T) {
	requireTableMatchesDerive(t, DefaultParams())
}

// requireTableMatchesDerive requires every derived name of the parameter table to equal its derive field. A mismatch
// reports the name and both values only.
func requireTableMatchesDerive(t *testing.T, p Params) {
	t.Helper()
	doc := decad.New()
	require.NoError(t, defineParameters(doc.Params(), p))
	d := derive(p)
	for _, c := range []struct {
		name string
		want float64
	}{
		{"pin_diameter", d.pin}, {"pin_collar_diameter", d.pinCollar},
		{"planet_pocket_diameter", d.planetPocket}, {"planet_lip_bore_diameter", d.planetLip},
		{"shaft_diameter", d.shaft}, {"shaft_bore_diameter", d.shaftBore},
		{"shaft_collar_diameter", d.collar}, {"cover_pocket_diameter", d.coverPocket},
		{"cover_lip_bore_diameter", d.coverLip}, {"cover_boss_diameter", d.coverBoss},
		{"cover_hub_diameter", d.coverHub}, {"boss_diameter", d.boss}, {"arm_width", d.arm},
		{"orbit_radius", d.orbit}, {"ring_outer_radius", d.ringOuter}, {"bolt_circle_radius", d.boltCircle},
		{"ring_length", d.ringLength},
	} {
		v, err := doc.Param(c.name)
		require.NoError(t, err, c.name)
		if math.Abs(v.Base()-c.want) > 1e-9 {
			t.Errorf("table %s = %v, derive = %v", c.name, v.Base(), c.want)
		}
	}
}
