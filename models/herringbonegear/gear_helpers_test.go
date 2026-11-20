package herringbonegear_test

import (
	"strings"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/herringbonegear"
)

func smallPlacedParams() herringbonegear.Params {
	p := herringbonegear.DefaultParams()
	p.Teeth, p.Samples, p.Thickness = 8, 5, 2
	p.BoreDiameter, p.CapChamfer = 1, 0.1
	return p
}

func meshEuler(mesh *decad.Mesh) int {
	edges := make(map[[2]int]struct{})
	triangles := mesh.Triangles()
	for _, triangle := range triangles {
		for i := range 3 {
			a, b := triangle[i], triangle[(i+1)%3]
			if a > b {
				a, b = b, a
			}
			edges[[2]int{a, b}] = struct{}{}
		}
	}
	return len(mesh.Vertices()) - len(edges) + len(triangles)
}

// filletRoles counts the distinct root fillets of gear by their face origin roles.
func filletRoles(gear *decad.Body) int {
	roles := map[string]struct{}{}
	for _, face := range gear.Faces() {
		for _, origin := range face.Origins() {
			if strings.HasPrefix(origin.Role, "fillet(") {
				roles[origin.Role] = struct{}{}
			}
		}
	}
	return len(roles)
}
