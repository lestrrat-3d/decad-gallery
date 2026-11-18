package clamparm_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

const (
	// rangeClearance is the loose clearance every joint keeps at its limits.
	rangeClearance = 0.5
	// rangeReach is how far from a joint's axis point the mating faces reach: the fork's slot is 26 deep. A
	// triangle farther than this from the point is some other feature and keeps its place in the mesh.
	rangeReach = 30.0
)

// asmBody is a named body of a joint's side.
type asmBody struct {
	name string
	body *decad.Body
}

// asmSides splits the model's 25 bodies at joint k (0 for J1 to 3 for J4) into the bodies before it on the chain from
// the desk (proximal) and the bodies after it (distal). Each joint's bolt, nut and knob go with its fork part, which
// is on the proximal side: the head and the nut bear on it. The clamp screw, its knob, its nut and the pad are proximal.
func asmSides(m *clamparm.Model, k int) ([]asmBody, []asmBody) {
	printedNames, printed := asmPrinted(m)
	boltNames, bolts := asmBolts(m)
	nutNames, nuts := asmNuts(m)
	var all []asmBody
	for i, p := range printed {
		all = append(all, asmBody{printedNames[i], p.Body})
	}
	for i, b := range bolts {
		all = append(all, asmBody{boltNames[i], b})
	}
	for i, n := range nuts {
		all = append(all, asmBody{nutNames[i], n})
	}
	// Distal names per joint.
	distal := [4]map[string]struct{}{
		set("swivel", "link A", "link B", "head", "adapter", "knob J2", "knob J3", "knob J4", "stud knob", "bolt J2",
			"bolt J3", "bolt J4", "stud", "nut J2", "nut J3", "nut J4", "adapter nut"),
		set("link A", "link B", "head", "adapter", "knob J3", "knob J4", "stud knob", "bolt J3", "bolt J4", "stud",
			"nut J3", "nut J4", "adapter nut"),
		set("link B", "head", "adapter", "knob J4", "stud knob", "bolt J4", "stud", "nut J4", "adapter nut"),
		set("head", "adapter", "stud knob", "stud", "adapter nut"),
	}
	var prox, dist []asmBody
	for _, b := range all {
		if _, ok := distal[k][b.name]; ok {
			dist = append(dist, b)
		} else {
			prox = append(prox, b)
		}
	}
	return prox, dist
}

func set(names ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(names))
	for _, n := range names {
		out[n] = struct{}{}
	}
	return out
}

// maskRole says which mating features of a part asmMasked leaves out of its mesh.
type maskRole int

const (
	maskNone maskRole = iota
	// maskTongue is the tongue part of J2 to J4: the tongue's faces at |y| 4, its chamfers (|y| 3.5 to 4) and its hole.
	maskTongue
	// maskFork is the fork part of J2 to J4: the fins' inner faces (|y| 4.26), their chamfers (|y| 4.26 to 4.76) and
	// their holes.
	maskFork
	// maskJ1 is J1's frame: the planes z 14, 14.26, 22.26 and 22.52 and the J1 hole walls.
	maskJ1
	// maskJ1Tongue is J1's swivel: maskJ1 and the chamfers along its two faces (z 14.26 to 14.76 and 21.76 to 22.26).
	maskJ1Tongue
)

const (
	maskTol = 1e-3
	// maskHoleRadius is a bolt hole's radius, 2.8, with room for tessellation.
	maskHoleRadius = 2.85
)

// maskBand is the |y| range, from lo to hi, whose faces of one side a role leaves out, besides the flat face at lo or
// hi. The flat face is the mating face: the tongue's at 4 (the band's hi), the fork's at 4.26 (the band's lo).
func maskBand(role maskRole) (flat, lo, hi float64) {
	if role == maskTongue {
		return clamparm.TongueThickness / 2, clamparm.TongueThickness/2 - 0.5, clamparm.TongueThickness / 2
	}
	return clamparm.TongueThickness/2 + 0.26, clamparm.TongueThickness/2 + 0.26, clamparm.TongueThickness/2 + 0.76
}

// j1Planes are J1's mating planes as distances along its axis from the swivel's bottom face (z 14.26): the top jaw's
// face z 14, the swivel's faces z 14.26 and 22.26 and the upper fin's underside z 22.52.
var j1Planes = []float64{-0.26, 0, 8, 8.26}

// asmMasked is the body's mesh without its mating features at the joint, per role. Those faces stand 0.26 mm apart
// by design (the Tight tongue-in-fork fit) at every angle and the bolt stands 0.3 from its hole, so a gap measured
// with them could never reach 0.5. Everything else stays, so the fold band, the slot end and the J1 gap end are
// measured: a triangle is dropped only when it is one of the role's mating features, within reach of the axis point
// (J1's swivel rides over the top jaw at every pan angle, so J1 has no reach limit).
func asmMasked(t testing.TB, body *decad.Body, centre, axis r3.Vec, role maskRole, reach float64) meshgap.Mesh {
	t.Helper()
	_, mesh := asmFineMesh(t, body)
	vertices, faces := asmMaskedFaces(mesh, centre, axis, role, reach)
	mg, err := meshgap.New(vertices, faces, mesh.Bound().Base())
	require.NoError(t, err)
	return mg
}

// asmMaskedFaces is the vertices and the kept triangles of asmMasked.
func asmMaskedFaces(mesh *decad.Mesh, centre, axis r3.Vec, role maskRole, reach float64) ([]r3.Vec, [][3]int) {
	vertices := mesh.Vertices()
	if role == maskNone {
		return vertices, mesh.Triangles()
	}
	flat, lo, hi := maskBand(role)
	mating := func(tri [3]int) bool {
		var along, radial [3]float64
		near := true
		for i, v := range tri {
			d := vertices[v].Sub(centre)
			along[i] = d.Dot(axis)
			radial[i] = d.Sub(axis.Scale(along[i])).Len()
			near = near && d.Len() <= reach
		}
		if !near {
			return false
		}
		// Hole wall: every vertex on the hole's cylinder.
		if radial[0] <= maskHoleRadius && radial[1] <= maskHoleRadius && radial[2] <= maskHoleRadius {
			return true
		}
		if role == maskJ1Tongue {
			for _, band := range [][2]float64{{0, 0.5}, {7.5, 8}} {
				if along[0] >= band[0]-maskTol && along[0] <= band[1]+maskTol &&
					along[1] >= band[0]-maskTol && along[1] <= band[1]+maskTol &&
					along[2] >= band[0]-maskTol && along[2] <= band[1]+maskTol {
					return true
				}
			}
		}
		if role == maskJ1 || role == maskJ1Tongue {
			for _, p := range j1Planes {
				if math.Abs(along[0]-p) <= maskTol && math.Abs(along[1]-p) <= maskTol &&
					math.Abs(along[2]-p) <= maskTol {
					return true
				}
			}
			return false
		}
		for _, sign := range []float64{1, -1} {
			inBand, onFlat := true, true
			for _, a := range along {
				a *= sign
				inBand = inBand && a >= lo-maskTol && a <= hi+maskTol
				onFlat = onFlat && math.Abs(a-flat) <= maskTol
			}
			if inBand || onFlat {
				return true
			}
		}
		return false
	}
	var faces [][3]int
	for _, tri := range mesh.Triangles() {
		if !mating(tri) {
			faces = append(faces, tri)
		}
	}
	return vertices, faces
}

// jointRoles names the printed part that is each joint's fork (proximal) and tongue (distal).
var jointRoles = [4][2]string{{"frame", "swivel"}, {"swivel", "link A"}, {"link A", "link B"}, {"link B", "head"}}

// jointGapAbove reports whether every proximal body stands more than clearance from every distal body at joint k, with
// the joint's mating features left out of the fork and tongue parts (asmMasked). When it does not, the string names
// the nearest pair.
func jointGapAbove(t *testing.T, c asmCase, m *clamparm.Model, k int, clearance float64) (bool, string) {
	t.Helper()
	xf := asmWorld(t, c.params)
	centre := xf.Apply(m.Dimensions.Joints[k])
	axis := xf.ApplyDir(m.Dimensions.Axes[k])
	reach := rangeReach
	if k == 0 {
		reach = math.Inf(1)
	}
	prox, dist := asmSides(m, k)
	require.NotEmpty(t, prox)
	require.NotEmpty(t, dist)
	mask := func(bs []asmBody, partName string) []meshgap.Mesh {
		out := make([]meshgap.Mesh, len(bs))
		for i, b := range bs {
			role := maskNone
			if b.name == partName {
				switch {
				case k == 0 && partName == "swivel":
					role = maskJ1Tongue
				case k == 0:
					role = maskJ1
				case partName == jointRoles[k][0]:
					role = maskFork
				default:
					role = maskTongue
				}
			}
			out[i] = asmMasked(t, b.body, centre, axis, role, reach)
		}
		return out
	}
	pm, dm := mask(prox, jointRoles[k][0]), mask(dist, jointRoles[k][1])
	for i, p := range pm {
		for j, d := range dm {
			ok, err := p.ExceedsClearance(d, clearance)
			require.NoError(t, err)
			if !ok {
				gap, _, err := p.Exceeds(d, 0)
				require.NoError(t, err)
				return false, fmt.Sprintf("J%d in case %s: %s and %s are %g mm apart at the nearest (lower bound)", k+1,
					c.name, prox[i].name, dist[j].name, gap)
			}
		}
	}
	return true, ""
}

// requireJointClear requires every proximal body to stand more than 0.5 mm from every distal body at joint k.
func requireJointClear(t *testing.T, c asmCase, m *clamparm.Model, k int) {
	t.Helper()
	ok, why := jointGapAbove(t, c, m, k, rangeClearance)
	require.True(t, ok, "joint is too close at its limit, need more than %g: %s", rangeClearance, why)
}

// TestClampArmRangeOfMotion builds each limit case and requires the joint at its limit to keep 0.5 mm between its two
// sides, hardware included (instructions.md "Range of motion"); the other cases keep it at all four joints.
func TestClampArmRangeOfMotion(t *testing.T) {
	for _, c := range asmCases(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, m := asmBuild(t, c)
			if c.joint >= 0 {
				requireJointClear(t, c, m, c.joint)
				return
			}
			for k := range 4 {
				requireJointClear(t, c, m, k)
			}
		})
	}
}

// TestClampArmRangeLimitsMatchCases keeps the limit cases tied to the constants of params.go.
func TestClampArmRangeLimitsMatchCases(t *testing.T) {
	byName := map[string]clamparm.Params{}
	for _, c := range asmCases(t) {
		byName[c.name] = c.params
	}
	require.Equal(t, clamparm.J1Max, byName["pan-max"].Pan)
	require.Equal(t, -clamparm.J1Max, byName["pan-min"].Pan)
	require.Equal(t, clamparm.ShoulderMin, byName["shoulder-min"].Shoulder)
	require.Equal(t, clamparm.OpeningMax, byName["shoulder-min"].Opening)
	require.Equal(t, clamparm.JointMax, byName["shoulder-max"].Shoulder)
	require.Equal(t, -clamparm.JointMax, byName["elbow-min"].Elbow)
	require.Equal(t, clamparm.JointMax, byName["elbow-max"].Elbow)
	require.Equal(t, clamparm.WristMin, byName["wrist-min"].Wrist)
	require.Equal(t, clamparm.WristMax, byName["wrist-max"].Wrist)
	for name, p := range byName {
		require.NoError(t, p.Validate(), name)
	}
	// The default, raised and folded render poses stay valid down to JointMax 115.
	require.GreaterOrEqual(t, clamparm.JointMax, 115.0)
}
