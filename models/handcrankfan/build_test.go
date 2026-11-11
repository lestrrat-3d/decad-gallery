package handcrankfan_test

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/lestrrat-3d/decad/parts"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func partNames(blades int) []string {
	names := []string{
		"handle", "handle-bolt-1", "handle-nut-1", "handle-bolt-2", "handle-nut-2",
		"gearbox-base", "gearbox-post", "gearbox-crank-wall", "gearbox-fan-wall", "gearbox-bolt-1", "gearbox-nut-1",
		"gearbox-bolt-2", "gearbox-nut-2", "gearbox-bolt-3", "gearbox-nut-3", "gearbox-bolt-4", "gearbox-nut-4",
		"gearbox-screw-1", "gearbox-screw-nut-1", "gearbox-screw-2", "gearbox-screw-nut-2",
		"gearbox-screw-3", "gearbox-screw-nut-3", "gearbox-screw-4", "gearbox-screw-nut-4", "driving-gear", "crank-shaft", "crank-arm",
		"crank-pin", "crank-grip", "crank-pin-bolt", "crank-pin-nut", "pinion", "fan-shaft", "fan-hub",
		"drive-shaft", "drive-gear", "crank-gear", "bearing-crank-post", "bearing-drive-post", "bearing-crank-wall",
		"bearing-drive-wall", "bearing-crank-block", "bearing-fan-wall", "bearing-fan-block",
	}
	for k := 1; k <= blades; k++ {
		names = append(names, fmt.Sprintf("fan-blade-%d", k), fmt.Sprintf("fan-bolt-%d", k), fmt.Sprintf("fan-nut-%d", k))
	}
	return names
}

func isRotor(name string) bool {
	return name == "fan-hub" || strings.HasPrefix(name, "fan-blade-") || strings.HasPrefix(name, "fan-bolt-") ||
		strings.HasPrefix(name, "fan-nut-")
}

// isFinished reports whether the part is checked for overlap: the rotor, the handle, the gearbox and the crank's own
// parts. The bevel pair and the spur pair, which mesh at their teeth, are left out.
func isFinished(name string) bool {
	switch name {
	case "handle", "gearbox-base", "gearbox-post", "gearbox-crank-wall", "gearbox-fan-wall", "crank-shaft", "drive-shaft", "crank-arm", "crank-pin", "crank-grip", "crank-pin-bolt", "crank-pin-nut":
		return true
	}
	if strings.HasPrefix(name, "bearing-") {
		return true
	}
	return isRotor(name) || strings.HasPrefix(name, "handle-bolt-") || strings.HasPrefix(name, "handle-nut-") ||
		strings.HasPrefix(name, "gearbox-bolt-") || strings.HasPrefix(name, "gearbox-nut-") || strings.HasPrefix(name, "gearbox-screw-")
}

func buildFan(t *testing.T, p handcrankfan.Params) (*decad.Document, map[string]handcrankfan.Part, *handcrankfan.Assembly) {
	t.Helper()
	doc := handcrankfan.NewDocument()
	a, err := handcrankfan.Build(t.Context(), doc, p)
	require.NoError(t, err)
	parts := map[string]handcrankfan.Part{}
	for _, part := range a.Parts {
		_, dup := parts[part.Name]
		require.False(t, dup, "duplicate part %s", part.Name)
		parts[part.Name] = part
		require.True(t, part.Body.IsSolid(), part.Name)
	}
	return doc, parts, a
}

func bounds(t *testing.T, part handcrankfan.Part) decad.Box {
	t.Helper()
	box, err := part.Body.Bounds()
	require.NoError(t, err)
	return box
}

// checkFan runs every check on one real build.
func checkFan(t *testing.T, p handcrankfan.Params) {
	doc, parts, a := buildFan(t, p)
	names := partNames(p.Blades)
	require.Len(t, a.Parts, len(names))
	for _, name := range names {
		require.Contains(t, parts, name)
	}
	require.Len(t, doc.Bodies(), len(names), "no tool or prototype body remains")
	require.NotEmpty(t, a.Sketches)
	for i, s := range a.Sketches {
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
	}
	// The crank shaft's grooved end is checked at 0.2 mm.
	for name, tolerance := range map[string]float64{"handle": 0.1, "gearbox-base": 0.1, "gearbox-post": 0.1, "gearbox-crank-wall": 0.1, "gearbox-fan-wall": 0.1, "crank-arm": 0.1, "crank-shaft": 0.2,
		"crank-pin": 0.1, "crank-grip": 0.1, "fan-hub": 0.1, "fan-blade-1": 0.1, "fan-bolt-1": 0.1, "crank-gear": 0.1, "gearbox-nut-1": 0.1, "handle-nut-1": 0.1} {
		mesh, err := parts[name].Body.Tessellate(t.Context(), units.Millimeters(tolerance), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, name)
		require.True(t, mesh.BoundaryVerified(), name)
		require.True(t, mesh.VolumeVerified(), name)
	}
	t.Run("orientation", func(t *testing.T) { requireOrientation(t, p, parts) })
	t.Run("pitch", func(t *testing.T) { requirePitch(t, p, parts) })
	t.Run("joint", func(t *testing.T) { requireJoint(t, p, parts) })
	t.Run("grip", func(t *testing.T) { requireGrip(t, parts) })
	t.Run("sweep", func(t *testing.T) { requireSweepClearance(t, parts, a.State, p.Blades) })
	// Fits, fasteners and gear mesh are measured at the default parameters only: the other parameters turn parts
	// rigidly about their own axes or repeat identical sockets.
	if p == handcrankfan.DefaultParams() {
		t.Run("fasteners", func(t *testing.T) { requireFasteners(t, doc, p, parts) })
		t.Run("clearances", func(t *testing.T) { requireClearances(t, p, parts) })
		t.Run("mesh", func(t *testing.T) { requireSpurMesh(t, p, parts) })
	}
	// Decad plans the explosion from axis-aligned boxes, and the arm's box depends on the crank angle, so the stage order
	// is searched and checked at the default pose only.
	if p.CrankAngle == handcrankfan.DefaultParams().CrankAngle {
		t.Run("explode", func(t *testing.T) { requireExplode(t, doc, a, parts) })
	}
	t.Run("gears", func(t *testing.T) { requireGearPlacement(t, p, parts) })
	t.Run("overlap", func(t *testing.T) { requireNoOverlap(t, doc, p, parts) })
	t.Run("copies", func(t *testing.T) { requireIdenticalBlades(t, p, parts) })
}

func requireOrientation(t *testing.T, p handcrankfan.Params, parts map[string]handcrankfan.Part) {
	box := map[string]decad.Box{}
	for name, part := range parts {
		box[name] = bounds(t, part)
	}
	const tol = 0.2
	for name, b := range box {
		switch name {
		case "crank-arm", "crank-pin", "crank-grip":
			continue
		}
		require.Less(t, b.Max.X, box["crank-pin"].Max.X, "%s is left of the crank", name)
	}
	require.Greater(t, box["crank-grip"].Min.X, box["gearbox-crank-wall"].Max.X, "the crank handle is outside the crank bearing block")
	require.Greater(t, box["crank-shaft"].Max.X, 54.0, "the crank shaft leaves the cover toward +X")
	require.InDelta(t, 0, (box["fan-shaft"].Min.X+box["fan-shaft"].Max.X)/2, tol, "the fan shaft lies on X = 0")
	rotorFront := math.Inf(1)
	for name, b := range box {
		if isRotor(name) {
			rotorFront = math.Min(rotorFront, b.Min.Y)
		}
	}
	for name, b := range box {
		// The raked grip's base reaches in front of the rotor's plane, but it hangs below the lowest blade tip, and the
		// overlap check proves it clears the blades.
		if strings.HasPrefix(name, "fan-") || name == "handle" {
			continue
		}
		require.Greater(t, b.Min.Y, rotorFront, "%s is in front of the rotor", name)
	}
	require.InDelta(t, -96, box["fan-hub"].Min.Y, tol, "the hub front face")
	require.InDelta(t, -76, box["fan-hub"].Max.Y, tol, "the hub back face")
	// Handheld: the grip hangs under the gearbox floor, is long enough for a hand, and has no stand around it.
	require.InDelta(t, 101.15, box["handle"].Max.Z, tol, "the grip starts just under the gearbox floor")
	require.InDelta(t, 101.4, box["gearbox-base"].Min.Z, tol, "the floor is where the grip ends")
	require.InDelta(t, 101.4-8, box["gearbox-fan-wall"].Min.Z, tol, "the fan-side wall's ears hang 8 below the floor, beside the grip")
	require.Greater(t, box["handle"].Max.Z-box["handle"].Min.Z, 100.0, "the grip is long enough for a hand")
	for name, b := range box {
		require.GreaterOrEqual(t, b.Min.Z, box["handle"].Min.Z-tol, "%s hangs below the grip end", name)
	}
	require.InDelta(t, -100, box["fan-shaft"].Min.Y, tol)
	for k := 1; k <= p.Blades; k++ {
		bolt := box[fmt.Sprintf("fan-bolt-%d", k)]
		require.GreaterOrEqual(t, bolt.Min.Y, -96.0-0.01, "bolt %d's head is flush with the hub's front face", k)
		require.Less(t, bolt.Max.Y, -76.0, "bolt %d is inside the hub's back face", k)
	}
}

// meshRing returns the vertices of the blade mesh, with the pose undone, whose X is the nearest ring to r.
func meshRing(verts []r3.Vec, r float64) (float64, []r3.Vec) {
	ring := math.Inf(1)
	for _, v := range verts {
		if math.Abs(v.X-r) < math.Abs(ring-r) {
			ring = v.X
		}
	}
	var out []r3.Vec
	for _, v := range verts {
		if math.Abs(v.X-ring) <= 1e-6 {
			out = append(out, v)
		}
	}
	return ring, out
}

func farthest(ring []r3.Vec, from r3.Vec, skip func(r3.Vec) bool) r3.Vec {
	var best r3.Vec
	distance := -1.0
	for _, v := range ring {
		if skip != nil && skip(v) {
			continue
		}
		if d := v.Sub(from).Len(); d > distance {
			best, distance = v, d
		}
	}
	return best
}

// requirePitch checks blade 1, which lies along +X before the fan turns. With the pose undone, the mesh vertices of
// one radial ring give the chord: from the midpoint of the trailing edge to the leading edge. The leading edge must
// lie upstream (larger Y) so air goes toward -Y, at the angle the twist formula gives, and the section must be as
// thick as the table says.
func requirePitch(t *testing.T, p handcrankfan.Params, parts map[string]handcrankfan.Part) {
	state, err := handcrankfan.Motion(p)
	require.NoError(t, err)
	blade := parts["fan-blade-1"].Body
	undo, err := r3.RotationAround(state.Apex, r3.Vec{Y: -1}, units.Radians(-state.FanAngle))
	require.NoError(t, err)

	mesh, err := blade.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	var verts []r3.Vec
	for _, v := range mesh.Vertices() {
		verts = append(verts, undo.Apply(v))
	}
	for _, r := range []float64{30, 40, 50, 60} {
		x, ring := meshRing(verts, r)
		require.NotEmpty(t, ring, "ring near radius %g", r)
		stack := r3.Vec{X: x, Y: -86, Z: 170}
		te1 := farthest(ring, stack, nil)
		le := farthest(ring, te1, nil)
		te2 := farthest(ring, le, func(v r3.Vec) bool { return v.Sub(te1).Len() < 0.2 })
		mid := te1.Add(te2).Scale(0.5)
		require.Greater(t, le.Y-mid.Y, 0.0, "the leading edge is upstream at radius %g", r)
		angle := math.Atan2(le.Y-mid.Y, le.Z-mid.Z)
		// The blades are mirrored, so the leading edge leads toward -Z and the chord angle is pi minus beta.
		require.InDelta(t, math.Pi-handcrankfan.Beta(x, p.BladePitch), angle, 0.5*math.Pi/180, "chord angle at radius %g", x)
		requireThickness(t, handcrankfan.ThicknessRatio(x), angle, ring, 0.03, x)
	}
	// The root is nearly round.
	x, ring := meshRing(verts, 15)
	requireThickness(t, 1, 0, ring, 0.1, x)
}

// requireThickness compares the ring's extent across the chord direction with its extent along it.
func requireThickness(t *testing.T, want, angle float64, ring []r3.Vec, tol, x float64) {
	t.Helper()
	along := r3.Vec{Y: math.Sin(angle), Z: math.Cos(angle)}
	across := r3.Vec{Y: math.Cos(angle), Z: -math.Sin(angle)}
	lo, hi := [2]float64{math.Inf(1), math.Inf(1)}, [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, v := range ring {
		for i, axis := range []r3.Vec{along, across} {
			d := v.Y*axis.Y + v.Z*axis.Z
			lo[i], hi[i] = math.Min(lo[i], d), math.Max(hi[i], d)
		}
	}
	require.InDelta(t, want, (hi[1]-lo[1])/(hi[0]-lo[0]), tol, "thickness ratio at radius %g", x)
}

// requireJoint checks that the hub is empty where a stud, a bolt and a nut go. The overlap check proves they fit.
func requireJoint(t *testing.T, p handcrankfan.Params, parts map[string]handcrankfan.Part) {
	state, err := handcrankfan.Motion(p)
	require.NoError(t, err)
	pose, err := r3.RotationAround(state.Apex, r3.Vec{Y: -1}, units.Radians(state.FanAngle))
	require.NoError(t, err)
	locate := func(name string, at r3.Vec) decad.PointLocation {
		loc, err := parts[name].Body.LocatePoint(t.Context(), pose.Apply(at), units.Millimeters(0.05))
		require.NoError(t, err, name)
		return loc
	}
	require.Equal(t, decad.PointOutside, locate("fan-hub", r3.Vec{X: 8, Y: -86, Z: 170}), "the socket is empty")
	// The nut cavity, its roof and the head pocket sit within 0.5 mm of the hub's faces, where the faceted hub gives no
	// decided point location. The fasteners subtest proves them with section loops.
	require.Equal(t, decad.PointInside, locate("fan-hub", r3.Vec{X: 10, Y: -91, Z: 175}), "the hub between two blades is solid")
	require.Equal(t, decad.PointOutside, locate("fan-hub", r3.Vec{X: 9.6, Y: -86.2, Z: 170.3}), "the socket is empty past the bolt")
}

// requireGrip measures the grip's mesh rings at three heights: they are ovals of the table's width and length,
// raked toward the operator, and the crank handle's head sits inside its counterbore.
func requireGrip(t *testing.T, parts map[string]handcrankfan.Part) {
	mesh, err := parts["handle"].Body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	for _, want := range []struct{ z, width, length float64 }{{26 - 26.75, 31, 39}, {85 - 26.75, 34, 42.5}, {114 - 26.75, 31, 38}} {
		lo, hi := r3.Vec{X: math.Inf(1), Y: math.Inf(1)}, r3.Vec{X: math.Inf(-1), Y: math.Inf(-1)}
		for _, v := range mesh.Vertices() {
			if math.Abs(v.Z-want.z) > 1e-6 {
				continue
			}
			lo = r3.Vec{X: math.Min(lo.X, v.X), Y: math.Min(lo.Y, v.Y)}
			hi = r3.Vec{X: math.Max(hi.X, v.X), Y: math.Max(hi.Y, v.Y)}
		}
		require.InDelta(t, want.width, hi.X-lo.X, 0.3, "grip width at Z %g", want.z)
		require.InDelta(t, want.length, hi.Y-lo.Y, 0.3, "grip length at Z %g", want.z)
		// The grip hangs from the lowered floor: its heights are the table's minus 26.75.
		rake := (want.z + 26.75 - 128) * math.Tan(15*math.Pi/180)
		require.InDelta(t, -23.5+rake, (lo.Y+hi.Y)/2, 0.5, "the grip is raked toward the operator at Z %g", want.z)
	}
	pin, grip := bounds(t, parts["crank-pin"]), bounds(t, parts["crank-grip"])
	require.Less(t, pin.Max.X, grip.Max.X-0.3, "the pin head sits inside the handle's counterbore")
}

// requireNoOverlap checks every pair among the finished parts: no shared material and no undecided pair. It proves
// the rotor's stud fits its socket and its bolts their holes, the grip's screws their pilots, and the crank's pin,
// shaft and screws their pockets and holes. The gear pair, which meshes at its teeth, and the fan shaft, which
// touches its neighbours, are not parts of the check: the gear check has already retired the gears, and this check
// removes the shaft, so it must run after the checks that use them.
func requireNoOverlap(t *testing.T, doc *decad.Document, p handcrankfan.Params, parts map[string]handcrankfan.Part) {
	names := map[*decad.Body]string{}
	for name, part := range parts {
		names[part.Body] = name
	}
	// Two parts are checked when both are finished parts, or when one is a shaft or a bearing: a shaft has to slide
	// through every gear's spline bore and every bearing, so a shaft that shares material with any part, or is
	// undecided against it, means a groove and its tooth are out of line. The gears that mesh at their teeth are only
	// checked against the shafts and bearings.
	// The fasteners subtest proves a bolt against its head host and its own nut, so those pairs are not checked here.
	// A fastener counts once its nut is a part.
	proven := map[[2]string]struct{}{}
	fasteners, err := handcrankfan.Fasteners(p)
	require.NoError(t, err)
	for _, f := range fasteners {
		if _, ok := parts[f.Nut]; !ok {
			continue
		}
		proven[[2]string{f.Bolt, f.HeadHost}], proven[[2]string{f.HeadHost, f.Bolt}] = struct{}{}, struct{}{}
		proven[[2]string{f.Bolt, f.Nut}], proven[[2]string{f.Nut, f.Bolt}] = struct{}{}, struct{}{}
	}
	checked := func(a, b *decad.Body) bool {
		x, y := names[a], names[b]
		if _, ok := proven[[2]string{x, y}]; ok {
			return false
		}
		holder := func(name string) bool {
			return strings.HasSuffix(name, "-shaft") || strings.HasPrefix(name, "bearing-")
		}
		return isFinished(x) && isFinished(y) || holder(x) || holder(y)
	}
	report, err := doc.Verify(t.Context())
	require.NoError(t, err)
	for _, pair := range report.Interferences {
		require.False(t, checked(pair.A, pair.B), "%s and %s share material", names[pair.A], names[pair.B])
	}
	for _, d := range report.Diagnostics {
		if d.Pair == nil {
			continue
		}
		require.False(t, checked(d.Pair.A, d.Pair.B), "%s and %s are undecided: %s", names[d.Pair.A], names[d.Pair.B], d.Code)
	}
}

// requireExplode builds the parts set and explodes it with Decad. The handle slides over the pin's stub end, so
// along the crank axis the arm, the handle and the pin must follow in that order with gaps between them. The main
// parts all move: the gear case and its cover to the right, the gears, the shafts and the crank. The driving gear leaves
// to the left and the pinion up, and no pair of parts may still share space when the parts are apart.
func requireExplode(t *testing.T, doc *decad.Document, a *handcrankfan.Assembly, built map[string]handcrankfan.Part) {
	set, err := a.Set(doc)
	require.NoError(t, err)
	layout, err := a.Explode(t.Context(), set)
	require.NoError(t, err)
	require.Equal(t, "gearbox-base", layout.Ground)
	// Overlaps comes from axis-aligned boxes, and a captive nut always sits in its host's box. Those pairs are exempt;
	// the fasteners subtest proves the nut and its host share no material.
	fasteners, err := handcrankfan.Fasteners(a.Params)
	require.NoError(t, err)
	captive := map[string]string{}
	for _, f := range fasteners {
		captive[f.Nut] = f.NutHost
	}
	var overlaps [][2]string
	for _, pair := range layout.Overlaps {
		if captive[pair[0]] == pair[1] || captive[pair[1]] == pair[0] {
			continue
		}
		overlaps = append(overlaps, pair)
	}
	require.Empty(t, overlaps, "no pair of parts shares space when exploded")
	span := map[string][2]float64{}
	offset := map[string]r3.Vec{}
	for _, part := range layout.Parts {
		offset[part.Name] = part.Offset
		box := bounds(t, built[part.Name])
		span[part.Name] = [2]float64{box.Min.X + part.Offset.X, box.Max.X + part.Offset.X}
		switch {
		case part.Name == "gearbox-base":
			require.Zero(t, part.Offset.Len(), "the gearbox is the ground")
		case strings.HasPrefix(part.Name, "gearbox-screw-nut-"):
			// A captive nut in the ground piece has no joint and stays in its cavity.
			require.Zero(t, part.Offset.Len(), "%s stays in the base", part.Name)
		default:
			require.GreaterOrEqual(t, part.Depth, 1, "%s is joined to the gearbox", part.Name)
		}
		switch part.Name {
		case "drive-gear", "crank-gear", "drive-shaft", "crank-shaft", "crank-arm", "crank-pin", "crank-grip", "driving-gear",
			"pinion", "handle", "fan-hub":
			require.Greater(t, part.Offset.Len(), 0.0, "%s moves", part.Name)
		}
		switch part.Name {
		case "driving-gear", "crank-gear", "drive-gear":
			require.Greater(t, part.Offset.Z, 0.0, "%s lifts out of the open top", part.Name)
		case "pinion", "drive-shaft", "crank-shaft":
			require.Less(t, part.Offset.X, 0.0, "%s slides off to the left", part.Name)
		}
	}
	require.Less(t, span["crank-arm"][1], span["crank-grip"][0], "the handle is outside the arm")
	require.Less(t, span["crank-grip"][1], span["crank-pin"][0], "the pin is outside the handle")
	for _, f := range fasteners {
		if strings.HasPrefix(f.Nut, "gearbox-screw-nut-") {
			continue
		}
		require.InDelta(t, 0, offset[f.Nut].Sub(offset[f.NutHost]).Len(), 1e-9, "%s moves with %s", f.Nut, f.NutHost)
	}
	bom, err := set.BOM(t.Context())
	require.NoError(t, err)
	screws, nuts, bearings := 0, 0, 0
	for _, row := range bom.Rows {
		if row.Kind != parts.Purchased {
			continue
		}
		switch row.Spec {
		case "608-2RS ball bearing":
			bearings += row.Quantity
		case "M2 hex nut":
			nuts += row.Quantity
		default:
			screws += row.Quantity
		}
	}
	require.Equal(t, 11+a.Params.Blades, screws, "four ear bolts, four wall bolts, two grip bolts, the pin bolt and one for each blade")
	require.Equal(t, 11+a.Params.Blades, nuts, "one nut for each bolt")
	require.Equal(t, 7, bearings, "a bearing where each shaft crosses a wall")
}

// requireSweepClearance proves a hand on the crank handle cannot meet a blade. The crank's arm, pin and handle sweep
// a ring about the crank axis, which sits well below the fan axis, and the blades sweep a disc about the fan axis.
// Every vertex of the crank parts is turned about the crank axis through a full turn, and none may come within a hand's
// width of the blades' disc.
func requireSweepClearance(t *testing.T, built map[string]handcrankfan.Part, state handcrankfan.State, blades int) {
	vertices := func(name string) []r3.Vec {
		mesh, err := built[name].Body.Tessellate(t.Context(), units.Millimeters(0.2), decad.WithVerification(decad.VerifyNone))
		require.NoError(t, err, name)
		return mesh.Vertices()
	}
	const hand = 15.0
	front, back, disc := math.Inf(1), math.Inf(-1), 0.0
	for k := 1; k <= blades; k++ {
		for _, v := range vertices(fmt.Sprintf("fan-blade-%d", k)) {
			front, back = math.Min(front, v.Y), math.Max(back, v.Y)
			disc = math.Max(disc, math.Hypot(v.X, v.Z-state.Apex.Z))
		}
	}
	closest := math.Inf(1)
	for _, name := range []string{"crank-arm", "crank-pin", "crank-grip"} {
		for _, v := range vertices(name) {
			radius := math.Hypot(v.Y-state.Crank.Y, v.Z-state.Crank.Z)
			phase := math.Atan2(v.Z-state.Crank.Z, v.Y-state.Crank.Y)
			for turn := 0.0; turn < 2*math.Pi; turn += 3 * math.Pi / 180 {
				y, z := state.Crank.Y+radius*math.Cos(phase+turn), state.Crank.Z+radius*math.Sin(phase+turn)
				along := math.Max(math.Max(front-y, y-back), 0)
				across := math.Max(math.Hypot(v.X, z-state.Apex.Z)-disc, 0)
				closest = math.Min(closest, math.Hypot(along, across))
			}
		}
	}
	require.GreaterOrEqual(t, closest, hand, "a crank part comes within %.1f of the blades' disc", closest)
}

// requireIdenticalBlades turns every blade and bolt back to the first one's place and compares their bounds.
// It retires the bodies it turns, so it runs last.
func requireIdenticalBlades(t *testing.T, p handcrankfan.Params, parts map[string]handcrankfan.Part) {
	state, err := handcrankfan.Motion(p)
	require.NoError(t, err)
	for _, kind := range []string{"fan-blade-", "fan-bolt-", "fan-nut-"} {
		var first decad.Box
		for k := 1; k <= p.Blades; k++ {
			back := state.FanAngle + 2*math.Pi*float64(k-1)/float64(p.Blades)
			xf, err := r3.RotationAround(state.Apex, r3.Vec{Y: -1}, units.Radians(-back))
			require.NoError(t, err)
			moved, err := decad.Transform(t.Context(), parts[fmt.Sprintf("%s%d", kind, k)].Body, xf)
			require.NoError(t, err)
			box, err := moved.Bounds()
			require.NoError(t, err)
			require.NoError(t, decad.Remove(moved))
			if k == 1 {
				first = box
				continue
			}
			for _, pair := range [][2]float64{
				{first.Min.X, box.Min.X}, {first.Min.Y, box.Min.Y}, {first.Min.Z, box.Min.Z},
				{first.Max.X, box.Max.X}, {first.Max.Y, box.Max.Y}, {first.Max.Z, box.Max.Z},
			} {
				require.InDelta(t, pair[0], pair[1], 0.05, "%s%d matches %s1", kind, k, kind)
			}
		}
	}
}

func requireGearPlacement(t *testing.T, p handcrankfan.Params, parts map[string]handcrankfan.Part) {
	state, err := handcrankfan.Motion(p)
	require.NoError(t, err)
	// Undo the pose to compare with the build frame.
	back := func(body *decad.Body, axis r3.Vec, angle float64) decad.Box {
		xf, err := r3.RotationAround(state.Apex, axis, units.Radians(-angle))
		require.NoError(t, err)
		moved, err := decad.Transform(t.Context(), body, xf)
		require.NoError(t, err)
		box, err := moved.Bounds()
		require.NoError(t, err)
		require.NoError(t, decad.Remove(moved))
		return box
	}
	driving := back(parts["driving-gear"].Body, r3.Vec{X: 1}, state.CrankAngle)
	require.InDelta(t, 21, driving.Max.X, 0.05)
	pinion := back(parts["pinion"].Body, r3.Vec{Y: -1}, state.FanAngle)
	require.InDelta(t, -31.5, pinion.Min.Y, 0.05)
}

// skipUnlessFull skips a slow check. The default build takes about five minutes and the variants as long again, so
// they run only on demand: set HANDCRANKFAN_FULL=1. CI does not run them.
func skipUnlessFull(t *testing.T) {
	t.Helper()
	if os.Getenv("HANDCRANKFAN_FULL") == "" {
		t.Skip("slow: set HANDCRANKFAN_FULL=1 to run")
	}
}

func TestHandCrankFanDefault(t *testing.T) {
	skipUnlessFull(t)
	checkFan(t, handcrankfan.DefaultParams())
}

func TestHandCrankFanVariant(t *testing.T) {
	skipUnlessFull(t)
	for name, p := range map[string]handcrankfan.Params{
		"flat-three":  {CrankAngle: -0.7, BladePitch: 12 * math.Pi / 180, Blades: 3},
		"steep-three": {CrankAngle: 2.5, BladePitch: 40 * math.Pi / 180, Blades: 3},
	} {
		t.Run(name, func(t *testing.T) { checkFan(t, p) })
	}
}
