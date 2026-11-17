package swisslever

import (
	"math"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// localMovement holds the default movement's bodies in their local frames (before any pose) and their meshes.
type localMovement struct {
	g      Geometry
	m      *Movement
	meshes map[string]*decad.Mesh
	err    error
}

var (
	localOnce sync.Once
	local     localMovement
)

// localParts builds every body in its local frame once, leaving out only what decad refuses today.
func localParts(t *testing.T) *localMovement {
	t.Helper()
	localOnce.Do(func() {
		g, err := Layout(DefaultParams())
		if err != nil {
			local.err = err
			return
		}
		b := newBuilder(t.Context(), NewDocument(), g)
		m := &Movement{}
		if err := b.buildAll(m); err != nil {
			local.err = err
			return
		}
		local = localMovement{g: g, m: m, meshes: map[string]*decad.Mesh{}}
		for _, s := range m.slots() {
			mesh, err := tessellate(t, *s.dst)
			if err != nil {
				local.err = fmtErr(s.name, err)
				return
			}
			local.meshes[s.name] = mesh
		}
	})
	require.NoError(t, local.err)
	return &local
}

type namedErr struct {
	name string
	err  error
}

func (e namedErr) Error() string { return e.name + ": " + e.err.Error() }
func (e namedErr) Unwrap() error { return e.err }

func fmtErr(name string, err error) error { return namedErr{name, err} }

// tessellate meshes body, verified, at the finest chord tolerance the body allows.
func tessellate(t *testing.T, body *decad.Body) (*decad.Mesh, error) {
	t.Helper()
	var mesh *decad.Mesh
	var err error
	for _, tol := range []float64{0.02, 0.03, 0.05} {
		mesh, err = body.Tessellate(t.Context(), units.Millimeters(tol), decad.WithVerification(decad.VerifyAll))
		if err == nil {
			return mesh, nil
		}
	}
	return nil, err
}

// turnPt applies a body pose to the XY point p.
func (pose bodyPose) turnPt(p Pt) Pt {
	if pose.local {
		return p.Rot(pose.angle).Add(pose.about)
	}
	return pose.about.Add(p.Sub(pose.about).Rot(pose.angle))
}

// posedMesh returns the meshgap mesh of body name at pose.
func (l *localMovement) posedMesh(t *testing.T, name string, pose bodyPose) meshgap.Mesh {
	t.Helper()
	mesh := l.meshes[name]
	vs := mesh.Vertices()
	out := make([]r3.Vec, len(vs))
	for i, v := range vs {
		p := pose.turnPt(Pt{v.X, v.Y})
		out[i] = r3.NewVec(p.X, p.Y, v.Z)
	}
	mg, err := meshgap.New(out, mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err)
	return mg
}

// posedLoops returns the section loops of body name at height z, posed, and the section's mesh bound.
func (l *localMovement) posedLoops(t *testing.T, name string, z float64, pose bodyPose) ([]outline, float64) {
	t.Helper()
	var body *decad.Body
	for _, s := range l.m.slots() {
		if s.name == name {
			body = *s.dst
		}
	}
	require.NotNil(t, body, name)
	loops, off := sectionLoops(t, body, z)
	out := make([]outline, len(loops))
	for i, lp := range loops {
		out[i] = make(outline, len(lp))
		for j, v := range lp {
			out[i][j] = pose.turnPt(v)
		}
	}
	return out, off
}

func pairIs(a, b, x, y string) bool { return (a == x && b == y) || (a == y && b == x) }

// byDesign lists the pairs that meet by design, with a gap no mesh proves apart: a shaft in a bearing bore (no
// clearance), the pillar tops on the front plate, a bolt head on its pocket floor, a bolt in its nut, a 605ZZ in its 0.05
// seat and each press fit with 0.05 round. TestFits measures those by section, with the bound.
func byDesign(a, b string) bool {
	for _, pr := range [][2]string{
		{EscapeArborName, BackEscapeBearingName}, {EscapeArborName, FrontEscapeBearingName},
		{PalletArborName, BackPalletBearingName}, {PalletArborName, FrontPalletBearingName},
		{BalanceStaffName, BackBalanceBearingName}, {BalanceStaffName, FrontBalanceBearingName},
		{BackPlateName, FrontPlateName},
		{EscapeWheelName, EscapeArborName}, {LeverName, PalletArborName},
		{EntryStoneName, LeverName}, {ExitStoneName, LeverName},
		{SafetyRollerName, BalanceStaffName}, {RollerTableName, BalanceStaffName},
		{BalanceWheelName, BalanceStaffName}, {HairspringRetainerName, BalanceStaffName},
		{SafetyRollerName, RollerTableName}, {RollerTableName, BalanceWheelName},
		{WindingKnobName, EscapeArborName}, {KnobCollarName, EscapeArborName}, {KnobCollarName, WindingKnobName},
		{BackEscapeBearingName, BackPlateName}, {BackPalletBearingName, BackPlateName},
		{BackBalanceBearingName, BackPlateName}, {FrontEscapeBearingName, FrontPlateName},
		{FrontPalletBearingName, FrontPlateName}, {FrontBalanceBearingName, FrontPlateName},
	} {
		if pairIs(a, b, pr[0], pr[1]) {
			return true
		}
	}
	for i := 0; i < pillarCount; i++ {
		if pairIs(a, b, BoltName(i), FrontPlateName) || pairIs(a, b, BoltName(i), NutName(i)) {
			return true
		}
	}
	return bySpring(a, b)
}

// bySpring lists the hairspring's close neighbours. Its mesh bound is wide (about 0.8 mm: the section reads Approximate),
// so a mesh cannot prove a gap of a few tenths; TestFits and TestPoseNoOverlap measure these pairs by the outline's
// vertices and by height.
func bySpring(a, b string) bool {
	for _, n := range []string{FrontPlateName, BalanceStaffName, BalanceWheelName, HairspringRetainerName} {
		if pairIs(a, b, HairspringName, n) {
			return true
		}
	}
	return false
}

// springGap returns the least distance from the vertices of loops that lie between rMin and rMax from centre to the
// polygons of other. The hairspring's outline is one long loop whose Clearance bound is wide, so its fits are measured
// from its vertices, which lie on the surface; the callers add the other outline's bound to their tolerance.
func springGap(loops []outline, centre Pt, rMin, rMax float64, other []outline) float64 {
	best := math.Inf(1)
	for _, l := range loops {
		for _, v := range l {
			if d := v.Sub(centre).Len(); d < rMin || d > rMax {
				continue
			}
			for _, o := range other {
				for j := range o {
					best = math.Min(best, segDist(v, o[j], o[(j+1)%len(o)]))
				}
			}
		}
	}
	return best
}

// pillarCount is the number of pillars, each with one bolt and one nut.
const pillarCount = 4

// byOutline lists the pairs that work against one another closely enough that the checks use their outlines.
func byOutline(a, b string) bool {
	return pairIs(a, b, EscapeWheelName, EntryStoneName) || pairIs(a, b, EscapeWheelName, ExitStoneName) ||
		pairIs(a, b, RollerTableName, LeverName)
}

// poseHairspringReach is the balance angle inside which the hairspring's hub bore clears the staff's flat corners.
const poseHairspringReach = 7 * deg

// nearBanking is how close to the banking angle the lever must be for its arm to read as resting on a post.
const nearBanking = 1 * deg

// TestPoseNoOverlap poses every body at several phases and requires that no two overlap: the wheel and the stones, and
// the pin and the lever, by section; every other pair by mesh.
func TestPoseNoOverlap(t *testing.T) {
	l := localParts(t)
	g := &l.g
	slots := l.m.slots()
	for _, phase := range []float64{0, 0.1, 0.226, 0.24, 0.25, 0.27, 0.6, 0.75} {
		s, err := g.StateAt(phase)
		require.NoError(t, err)
		table, err := poses(g, s)
		require.NoError(t, err)
		meshes := make([]meshgap.Mesh, len(slots))
		for i, sl := range slots {
			meshes[i] = l.posedMesh(t, sl.name, table[sl.name])
		}
		for i := range slots {
			for j := i + 1; j < len(slots); j++ {
				a, b := slots[i].name, slots[j].name
				if byDesign(a, b) || byOutline(a, b) {
					continue
				}
				// The lever's arm rests on a banking post at the banking, a stated contact 0.05° short of touching:
				// a gap under what a mesh proves, read by section below.
				if pairIs(a, b, LeverName, BackPlateName) && math.Abs(math.Abs(s.LeverAngle)-g.Banking) < nearBanking {
					continue
				}
				gap, separated, err := meshes[i].Exceeds(meshes[j], 0)
				require.NoError(t, err)
				require.True(t, separated, "phase %v: %s and %s are not proven apart (gap lower bound %v)", phase, a, b, gap)
			}
		}

		// The hairspring is drawn relaxed on the staff. Its hub bore clears the staff's flat corners only while the
		// balance stays within poseHairspringReach of rest; its faces clear the balance wheel and the retainer by height.
		spring, _ := l.posedLoops(t, HairspringName, 21, table[HairspringName])
		if math.Abs(s.BalanceAngle) <= poseHairspringReach {
			staff, off2 := l.posedLoops(t, BalanceStaffName, 21, table[BalanceStaffName])
			gap := springGap(spring, g.B, 0, 4, staff)
			require.GreaterOrEqual(t, gap, tightFit-0.02-off2, "phase %v: hairspring hub bore to the staff", phase)
		}
		require.GreaterOrEqual(t, springLow-balanceHigh, 0.25-3e-3, "phase %v: hairspring to the balance wheel", phase)
		require.GreaterOrEqual(t, retainerLow-(springLow+SpringParams().Height), 0.25-3e-3,
			"phase %v: hairspring to the retainer", phase)

		// The wheel's teeth and the stones, in the wheel's plane: the stones may touch a tooth, never enter it.
		wheel, off1 := l.posedLoops(t, EscapeWheelName, midHeight, table[EscapeWheelName])
		for _, name := range []string{EntryStoneName, ExitStoneName} {
			stone, off2 := l.posedLoops(t, name, midHeight, table[name])
			bound := 0.02 + off1 + off2
			require.LessOrEqual(t, loopsDepth(stone, wheel), bound, "phase %v: %s in the escape wheel", phase, name)
			require.LessOrEqual(t, loopsDepth(wheel, stone), bound, "phase %v: escape wheel in the %s", phase, name)
		}
		if math.Abs(math.Abs(s.LeverAngle)-g.Banking) < nearBanking {
			lever, off1 := l.posedLoops(t, LeverName, forkBandF, table[LeverName])
			posts, off2 := l.posedLoops(t, BackPlateName, forkBandF, table[BackPlateName])
			require.LessOrEqual(t, loopsDepth(lever, posts), 0.02+off1+off2, "phase %v: lever in a banking post", phase)
			require.LessOrEqual(t, loopsDepth(posts, lever), 0.02+off1+off2, "phase %v: banking post in the lever", phase)
		}
		// The impulse pin and the lever's fork, at the lever's band F: the pin may touch a corner round, never enter it.
		pin, off1 := l.posedLoops(t, RollerTableName, forkBandF, table[RollerTableName])
		lever, off2 := l.posedLoops(t, LeverName, forkBandF, table[LeverName])
		bound := 0.02 + off1 + off2
		require.LessOrEqual(t, loopsDepth(pin, lever), bound, "phase %v: pin in the lever", phase)
		require.LessOrEqual(t, loopsDepth(lever, pin), bound, "phase %v: lever in the pin", phase)
	}
}
