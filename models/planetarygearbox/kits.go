package planetarygearbox

import (
	"fmt"
	"strings"
)

// Kit is one supported pair of bearings: Name is the kit's identifier, and PlanetBearing and ShaftBearing are
// catalog names for the two roles.
type Kit struct {
	Name, PlanetBearing, ShaftBearing string
}

// kits lists every supported kit with its tooth set at module 1 and three planets, the default first.
var kits = []struct {
	Kit
	sun, planet, ring int
}{
	{Kit{"planet605-shaft605", "605ZZ", "605ZZ"}, 13, 29, 71},
	{Kit{"planet605-shaft608", "605ZZ", "608ZZ"}, 14, 28, 70},
	{Kit{"planet608-shaft605", "608ZZ", "605ZZ"}, 13, 29, 71},
	{Kit{"planet608-shaft608", "608ZZ", "608ZZ"}, 14, 28, 70},
}

// SupportedCombinations returns every kit DefaultParamsFor accepts, the default kit first.
func SupportedCombinations() []Kit {
	out := make([]Kit, 0, len(kits))
	for _, k := range kits {
		out = append(out, k.Kit)
	}
	return out
}

// DefaultParamsFor returns DefaultParams with the kit's bearings and its tooth set. It returns an error naming the
// supported kits for an unknown name.
func DefaultParamsFor(kit string) (Params, error) {
	names := make([]string, 0, len(kits))
	for _, k := range kits {
		if k.Name == kit {
			p := DefaultParams()
			p.PlanetBearing, p.ShaftBearing = k.PlanetBearing, k.ShaftBearing
			p.SunTeeth, p.PlanetTeeth, p.RingTeeth = k.sun, k.planet, k.ring
			return p, nil
		}
		names = append(names, k.Name)
	}
	return Params{}, fmt.Errorf("planetarygearbox: unknown kit %q; choose one of %s", kit, strings.Join(names, ", "))
}
