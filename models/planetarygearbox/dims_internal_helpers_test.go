package planetarygearbox

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

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
		{"shaft_diameter", d.shaft}, {"sun_shaft_diameter", d.sunShaft}, {"shaft_bore_diameter", d.shaftBore},
		{"shaft_collar_diameter", d.collar}, {"cover_pocket_diameter", d.coverPocket},
		{"cover_lip_bore_diameter", d.coverLip}, {"cover_boss_diameter", d.coverBoss},
		{"cover_hub_diameter", d.coverHub}, {"boss_diameter", d.boss}, {"arm_width", d.arm},
		{"orbit_radius", d.orbit}, {"ring_outer_radius", d.ringOuter}, {"bolt_circle_radius", d.boltCircle},
		{"ring_length", d.ringLength},
	} {
		v, err := doc.Param(c.name)
		require.NoError(t, err, c.name)
		got := v.Base()
		if tableValueDiffers(math.Abs(got - c.want)) {
			t.Errorf("table %s = %v, derive = %v", c.name, got, c.want)
		}
	}
}

// CI: isolated-predicate
func tableValueDiffers(distance float64) bool { return distance > 1e-9 }
