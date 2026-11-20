package clamparm_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/stretchr/testify/require"
)

func TestClampArmEdgeFinishing(t *testing.T) {
	_, m := asmBuild(t, asmCase{"default", clamparm.DefaultParams(), -1})
	parts := map[string]clamparm.Part{
		"frame": m.Clamp, "pad": m.Pad, "swivel": m.Swivel, "link A": m.Links[0], "link B": m.Links[1],
		"head": m.Head, "adapter": m.Adapter, "knob J1": m.Knobs[0], "knob J2": m.Knobs[1], "knob J3": m.Knobs[2],
		"knob J4": m.Knobs[3], "screw knob": m.ScrewKnobs[0], "stud knob": m.ScrewKnobs[1],
	}
	all := [4]bool{true, true, true, true}
	// A y face outline's tongue end is narrower than the face, so only the fork end's side reads the chamfer.
	fin := [4]bool{true, true, false, true}
	faces := []edgeFace{
		{"frame", 1, false, all, 0}, {"frame", 1, true, all, 0},
		{"pad", 2, false, all, 0}, {"pad", 2, true, all, 0},
		{"swivel", 2, false, [4]bool{true, false, true, true}, 0}, {"swivel", 2, true, [4]bool{false, false, true, true}, 0},
		{"swivel", 1, false, [4]bool{false, true, false, true}, 0}, {"swivel", 1, true, [4]bool{false, true, false, true}, 0},
		{"link A", 1, false, fin, 0}, {"link A", 1, true, fin, 0}, {"link B", 1, false, fin, 0}, {"link B", 1, true, fin, 0},
		{"head", 1, false, [4]bool{true, false, false, false}, 0}, {"head", 1, true, [4]bool{true, false, false, false}, 0},
		{"adapter", 2, false, all, 0}, {"adapter", 2, true, all, 0},
	}
	// The tongue faces at y = +-4 of the links and the head: their outlines carry the chamfer too, tangent to the round.
	for _, n := range []string{"link A", "link B", "head"} {
		faces = append(faces, edgeFace{n, 1, false, [4]bool{false, false, true, false}, -tongueFace + inset},
			edgeFace{n, 1, true, [4]bool{false, false, true, false}, tongueFace - inset})
	}
	// The swivel's tongue top face at z = 8 (round end about J1) and the y = +-4.26 fin inner faces of the swivel (round
	// end about J2) and of the links (round end about the far joint): the section sits 0.25 inside each face.
	const finFace = 4.51
	faces = append(faces,
		edgeFace{"swivel", 2, true, [4]bool{true, false, false, false}, 7.75},
		edgeFace{"swivel", 1, false, [4]bool{false, false, false, true}, -finFace},
		edgeFace{"swivel", 1, true, [4]bool{false, false, false, true}, finFace},
		edgeFace{"link A", 1, false, [4]bool{false, false, false, true}, -finFace},
		edgeFace{"link A", 1, true, [4]bool{false, false, false, true}, finFace},
		edgeFace{"link B", 1, false, [4]bool{false, false, false, true}, -finFace},
		edgeFace{"link B", 1, true, [4]bool{false, false, false, true}, finFace})
	for _, n := range []string{"knob J1", "knob J2", "knob J3", "knob J4", "screw knob", "stud knob"} {
		faces = append(faces, edgeFace{n, 2, false, all, 0}, edgeFace{n, 2, true, all, 0})
	}
	for _, f := range faces {
		name := f.part + "/axis" + string(rune('x'+f.axis))
		if f.at != 0 {
			name += "-at"
		}
		if f.upper {
			name += "+"
		} else {
			name += "-"
		}
		t.Run(name, func(t *testing.T) {
			p := parts[f.part]
			lo, hi := localBounds(t, p)
			level := lo[f.axis] + inset
			if f.upper {
				level = hi[f.axis] - inset
			}
			if f.at != 0 {
				level = f.at
			}
			o := asmSectionOutline(t, p.Body, sectionFrame(t, p, f.axis, level))
			require.NotEmpty(t, o.Loops)
			var oa, ob [2]int
			switch f.axis {
			case 0:
				oa, ob = [2]int{1, 1}, [2]int{2, 2}
			case 1:
				oa, ob = [2]int{2, 2}, [2]int{0, 0}
			default:
				oa, ob = [2]int{0, 0}, [2]int{1, 1}
			}
			ax := [2]int{oa[0], ob[0]}
			gotLo, gotHi := [2]float64{math.Inf(1), math.Inf(1)}, [2]float64{math.Inf(-1), math.Inf(-1)}
			for _, l := range o.Loops {
				for _, v := range l {
					lv := p.Frame.ToLocal(v)
					c := [3]float64{lv.X, lv.Y, lv.Z}
					for i, a := range ax {
						gotLo[i], gotHi[i] = math.Min(gotLo[i], c[a]), math.Max(gotHi[i], c[a])
					}
				}
			}
			bound := o.OffSurface.Base() + 0.06
			for i, a := range ax {
				if f.sides[2*i] {
					require.InDelta(t, lo[a]+inset, gotLo[i], bound, "axis %d low side", a)
				}
				if f.sides[2*i+1] {
					require.InDelta(t, hi[a]-inset, gotHi[i], bound, "axis %d high side", a)
				}
			}
		})
	}
}
