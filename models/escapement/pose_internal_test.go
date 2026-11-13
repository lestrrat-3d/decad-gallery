package escapement

import (
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
		m.Sketches = b.sketches
		local = localMovement{g: g, m: m, meshes: map[string]*decad.Mesh{}}
		for _, s := range m.slots() {
			mesh, err := tessellate(t, *s.dst)
			if err != nil {
				local.err = err
				return
			}
			local.meshes[s.name] = mesh
		}
	})
	require.NoError(t, local.err)
	return &local
}

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

// posedMesh returns the meshgap mesh of body name at pose.
func (l *localMovement) posedMesh(t *testing.T, name string, pose bodyPose) meshgap.Mesh {
	t.Helper()
	mesh := l.meshes[name]
	vs := mesh.Vertices()
	out := make([]r3.Vec, len(vs))
	for i, v := range vs {
		p := Pt{v.X, v.Y}
		if pose.local {
			p = p.Rot(pose.angle).Add(pose.about)
		} else {
			p = pose.about.Add(p.Sub(pose.about).Rot(pose.angle))
		}
		out[i] = r3.NewVec(p.X, p.Y, v.Z)
	}
	mg, err := meshgap.New(out, mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err)
	return mg
}

// posedLoops returns the outline loops of body name at height z, posed.
func (l *localMovement) posedLoops(t *testing.T, name string, z float64, pose bodyPose) [][]Pt {
	t.Helper()
	body := *l.slotByName(name)
	o := sectionAt(t, body, z)
	var out [][]Pt
	for _, lp := range loopsBy(o) {
		poly := make([]Pt, len(lp))
		for i, p := range lp {
			if pose.local {
				poly[i] = p.Rot(pose.angle).Add(pose.about)
			} else {
				poly[i] = pose.about.Add(p.Sub(pose.about).Rot(pose.angle))
			}
		}
		out = append(out, poly)
	}
	return out
}

func (l *localMovement) slotByName(name string) **decad.Body {
	for _, s := range l.m.slots() {
		if s.name == name {
			return s.dst
		}
	}
	panic("unknown body " + name)
}

// touching lists the pairs of bodies that touch by design: a shaft turning in a bearing bore (no clearance), the
// pillar tops on the front plate, a bolt head on its pocket floor, a bolt in its nut, and the suspension pin's shoulder
// on the front plate.
func touching(a, b string) bool {
	pairs := [][2]string{
		{EscapeArborName, BackEscapeBearingName}, {EscapeArborName, FrontEscapeBearingName},
		{PalletArborName, BackPalletBearingName}, {PalletArborName, FrontPalletBearingName},
		{GreatWheelArborName, BackGreatWheelBearingName}, {GreatWheelArborName, FrontGreatWheelBearingName},
		{SuspensionPinName, PendulumBearingName}, {SuspensionPinName, FrontPlateName},
		{BackPlateName, FrontPlateName},
	}
	// A 605ZZ in its 0.05 mm seat and a press-fit part on its arbor leave a gap below what a 0.02 mm mesh can prove, so
	// TestFits measures those by section. The mainspring's mesh bound is wide, so its 0.25 mm fits are measured in TestSpringFits.
	for _, bearing := range []string{BackEscapeBearingName, BackPalletBearingName, BackGreatWheelBearingName} {
		pairs = append(pairs, [2]string{bearing, BackPlateName})
	}
	for _, bearing := range []string{FrontEscapeBearingName, FrontPalletBearingName, FrontGreatWheelBearingName} {
		pairs = append(pairs, [2]string{bearing, FrontPlateName})
	}
	pairs = append(pairs, [2]string{PendulumBearingName, PendulumName}, [2]string{EscapeWheelName, EscapeArborName},
		[2]string{AnchorName, PalletArborName}, [2]string{CrutchName, PalletArborName},
		[2]string{PawlCarrierName, GreatWheelArborName}, [2]string{SpringSpacerName, GreatWheelArborName},
		[2]string{WindingKnobName, GreatWheelArborName},
		[2]string{SuspensionCapName, SuspensionPinName}, [2]string{SpringAnchorPinName, FrontPlateName},
		[2]string{MainspringName, GreatWheelArborName}, [2]string{MainspringName, PawlCarrierName},
		[2]string{MainspringName, SpringSpacerName}, [2]string{MainspringName, SpringAnchorPinName})
	for i := 0; i < 4; i++ {
		pairs = append(pairs, [2]string{BoltName(i), FrontPlateName}, [2]string{BoltName(i), NutName(i)})
	}
	for _, p := range pairs {
		if (a == p[0] && b == p[1]) || (a == p[1] && b == p[0]) {
			return true
		}
	}
	return false
}

// byOutline lists the pairs that move against one another closely enough that the checks use their outlines.
func byOutline(a, b string) bool {
	set := map[[2]string]struct{}{
		{EscapeWheelName, AnchorName}: {}, {EscapeWheelName, GreatWheelName}: {}, {GreatWheelName, PawlCarrierName}: {},
	}
	_, ok := set[[2]string{a, b}]
	_, ok2 := set[[2]string{b, a}]
	return ok || ok2
}

func TestPoseNoOverlap(t *testing.T) {
	l := localParts(t)
	g := &l.g
	for _, phase := range []float64{0, 0.1, 0.35, 0.6, 0.85} {
		s, err := g.StateAt(phase)
		require.NoError(t, err)
		table := poses(g, s)
		slots := l.m.slots()
		meshes := make([]meshgap.Mesh, len(slots))
		for i, sl := range slots {
			meshes[i] = l.posedMesh(t, sl.name, table[sl.name])
		}
		for i := range slots {
			for j := i + 1; j < len(slots); j++ {
				a, b := slots[i].name, slots[j].name
				if touching(a, b) || byOutline(a, b) {
					continue
				}
				gap, separated, err := meshes[i].Exceeds(meshes[j], 0)
				require.NoError(t, err)
				require.True(t, separated, "phase %v: %s and %s are not proven apart (gap lower bound %v)", phase, a, b, gap)
			}
		}
		// Pairs checked by outline.
		wheel := l.posedLoops(t, EscapeWheelName, 23.5, table[EscapeWheelName])
		anchor := l.posedLoops(t, AnchorName, 23.5, table[AnchorName])
		require.False(t, overlap(wheel[0], anchor[0]), "phase %v: escape wheel and anchor", phase)
		pinion := l.posedLoops(t, EscapeWheelName, 3, table[EscapeWheelName])
		great := l.posedLoops(t, GreatWheelName, 3, table[GreatWheelName])
		require.False(t, overlap(pinion[0], great[0]), "phase %v: pinion and great wheel", phase)
		pocket := l.posedLoops(t, GreatWheelName, 7, table[GreatWheelName])[1]
		drum := l.posedLoops(t, PawlCarrierName, 7, table[PawlCarrierName])[0]
		require.True(t, fitsIn(drum, pocket), "phase %v: the carrier's pawls stay inside the ratchet pocket", phase)
		// The arbor turns inside the fixed spring hub's bore without touching it.
		arbor := l.posedLoops(t, GreatWheelArborName, 14.25, table[GreatWheelArborName])[0]
		var bore []Pt
		for _, lp := range l.posedLoops(t, MainspringName, 14.25, table[MainspringName]) {
			near := true
			for _, v := range lp {
				if v.Sub(g.G).Len() > 4 {
					near = false
					break
				}
			}
			if near {
				bore = lp
			}
		}
		require.NotNil(t, bore, "the spring hub's bore")
		require.True(t, fitsIn(arbor, bore), "phase %v: the arbor turns inside the spring's bore", phase)
		// The arbor turns 3.25° across a period against the fixed spring, so the flat corners close the 0.30 mm of the build
		// pose by about 0.084 mm at the worst phase, and the 0.30 mm bore clearance keeps the Tight 0.2 mm minimum.
		require.GreaterOrEqual(t, distPolys(arbor, bore), 0.2, "phase %v: arbor to spring bore", phase)
	}
}
