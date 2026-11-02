package meshgap

import (
	"fmt"
	"math"
)

// Mount records candidate support or fastener connections. Ground must already
// have been checked against the floor by the consuming assembly.
type Mount struct {
	Part       string
	Ground     bool
	References []string
	Clearance  float64
}

// Nearest records one candidate's boundary-gap lower bound for bounded diagnostics.
type Nearest struct {
	Part     string
	LowerGap float64
}

// Reachable filters declared connections by their actual mesh boundary gaps,
// then finds paths to Ground nodes. A floating cycle cannot acquire support
// through a distant ground reference. Remaining paths are only a proximity
// screen; joint fit, retention and material interference need separate checks.
func Reachable(mounts []Mount, meshes map[string]Mesh) (map[string]struct{}, map[string]Nearest, error) {
	known := make(map[string]struct{}, len(mounts))
	for _, mount := range mounts {
		if mount.Part == "" {
			return nil, nil, fmt.Errorf("meshgap: mount has no part name")
		}
		if _, duplicate := known[mount.Part]; duplicate {
			return nil, nil, fmt.Errorf("meshgap: duplicate mount %s", mount.Part)
		}
		known[mount.Part] = struct{}{}
		mesh, present := meshes[mount.Part]
		if !present || len(mesh.triangles) == 0 {
			return nil, nil, fmt.Errorf("meshgap: missing mesh for %s", mount.Part)
		}
		if !finite(mount.Clearance) || mount.Clearance < 0 {
			return nil, nil, fmt.Errorf("meshgap: invalid clearance for %s", mount.Part)
		}
	}
	dependents := make(map[string][]string, len(mounts))
	nearest := make(map[string]Nearest, len(mounts))
	reached := make(map[string]struct{}, len(mounts))
	var queue []string
	for _, mount := range mounts {
		if mount.Ground {
			reached[mount.Part] = struct{}{}
			queue = append(queue, mount.Part)
		}
		closest := Nearest{LowerGap: math.Inf(1)}
		seen := make(map[string]struct{}, len(mount.References))
		for _, reference := range mount.References {
			if reference == mount.Part {
				return nil, nil, fmt.Errorf("meshgap: %s cannot support itself", mount.Part)
			}
			if _, present := known[reference]; !present {
				return nil, nil, fmt.Errorf("meshgap: %s references absent mount %s", mount.Part, reference)
			}
			if _, duplicate := seen[reference]; duplicate {
				continue
			}
			seen[reference] = struct{}{}
			gap, separated, err := meshes[mount.Part].Exceeds(meshes[reference], mount.Clearance)
			if err != nil {
				return nil, nil, fmt.Errorf("meshgap: %s / %s: %w", mount.Part, reference, err)
			}
			if gap < closest.LowerGap {
				closest = Nearest{Part: reference, LowerGap: gap}
			}
			if !separated {
				dependents[reference] = append(dependents[reference], mount.Part)
			}
		}
		nearest[mount.Part] = closest
	}
	for i := 0; i < len(queue); i++ {
		for _, part := range dependents[queue[i]] {
			if _, present := reached[part]; present {
				continue
			}
			reached[part] = struct{}{}
			queue = append(queue, part)
		}
	}
	return reached, nearest, nil
}
