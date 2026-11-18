package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// Poses of the clamp arm the "poses" view and the README show. The default pose is clamparm.DefaultParams.
var (
	clampRaisedPose = clamparm.Params{LinkLength: 120, Opening: 25, Shoulder: 90, Elbow: -30, Wrist: -60}
	clampFoldedPose = clamparm.Params{LinkLength: 120, Opening: 25, Shoulder: 100, Elbow: -115, Wrist: 15}
)

// clampBodyNames lists the 25 bodies of the clamp arm in the order newClampArm files them.
var clampBodyNames = []string{
	"Frame", "Pad", "Swivel", "Link A", "Link B", "Head", "Adapter",
	"Knob J1", "Knob J2", "Knob J3", "Knob J4", "Screw Knob", "Stud Knob",
	"Bolt J1", "Bolt J2", "Bolt J3", "Bolt J4", "Stud Bolt", "Screw Bolt",
	"Nut J1", "Nut J2", "Nut J3", "Nut J4", "Jaw Nut", "Adapter Nut",
}

// clampExplodedOrder lists every body in assembly order, which is the order the README's alt text follows.
var clampExplodedOrder = []string{
	"Frame", "Bolt J1", "Swivel", "Nut J1", "Knob J1",
	"Jaw Nut", "Screw Knob", "Screw Bolt", "Pad",
	"Link A", "Bolt J2", "Nut J2", "Knob J2",
	"Link B", "Bolt J3", "Nut J3", "Knob J3",
	"Head", "Bolt J4", "Nut J4", "Knob J4",
	"Stud Knob", "Stud Bolt", "Adapter Nut", "Adapter",
}

// clampPart is one image of the parts table: the shot's name and the bodies it stands for. The first body is drawn.
type clampPart struct {
	shot   string
	bodies []string
}

// clampParts lists the part images: one per distinct part, with the bodies that share it.
var clampParts = []clampPart{
	{"frame", []string{"Frame"}},
	{"pad", []string{"Pad"}},
	{"swivel", []string{"Swivel"}},
	{"link", []string{"Link A", "Link B"}},
	{"head", []string{"Head"}},
	{"adapter", []string{"Adapter"}},
	{"knob", []string{"Knob J1", "Knob J2", "Knob J3", "Knob J4"}},
	{"screw-knob", []string{"Screw Knob", "Stud Knob"}},
	{"bolt-25", []string{"Bolt J2", "Bolt J3", "Bolt J4"}},
	{"bolt-35", []string{"Bolt J1"}},
	{"bolt-40", []string{"Stud Bolt"}},
	{"bolt-70", []string{"Screw Bolt"}},
	{"nut", []string{"Nut J1", "Nut J2", "Nut J3", "Nut J4", "Jaw Nut", "Adapter Nut"}},
}

// clampViews are the full-size views, by preset name.
var clampViews = []string{
	"hero", "side", "back", "top", "poses", "exploded", "clamp-section", "joint-section", "interface-section",
	"interface",
}

// Exploded view magnitudes, in mm.
const (
	// clampExplodeBolt pulls a joint bolt out of its -Y face, clampExplodeNut and clampExplodeKnob the nut and the
	// knob out of the +Y face; the knob goes further, since the nut sits in its inner face.
	clampExplodeBolt = 45.0
	clampExplodeNut  = 20.0
	clampExplodeKnob = 40.0
	// clampExplodeSwivel lifts the swivel off the frame, and the J1 nut and knob above it.
	clampExplodeSwivel   = 20.0
	clampExplodeNutJ1    = 30.0
	clampExplodeKnobJ1   = 46.0
	clampExplodeLinkA    = 60.0
	clampExplodeLinkB    = 60.0
	clampExplodeHead     = 130.0
	clampExplodeAdapter  = 28.0
	clampExplodeAdNut    = 12.0
	clampExplodeStudBolt = 46.0
	clampExplodeStudKnob = 68.0
	clampExplodePad      = 60.0
	clampExplodeScrew    = 80.0
	clampExplodeScrewKn  = 100.0
	clampExplodeJawNut   = 35.0
)

// clampArm is one built clamp arm with every body tessellated.
type clampArm struct {
	dm     clamparm.Dimensions
	meshes map[string]*decad.Mesh
	// local maps each body's own frame (the one it was built in) to the world.
	local map[string]r3.Frame
}

func clampColor(name string) solidlens.Color {
	switch name {
	case "Frame":
		return solidlens.RGB(0.85, 0.45, 0.15)
	case "Pad":
		return solidlens.RGB(0.22, 0.24, 0.28)
	case "Swivel":
		return solidlens.RGB(0.17, 0.5, 0.55)
	case "Link A", "Link B":
		return solidlens.RGB(0.25, 0.45, 0.75)
	case "Head":
		return solidlens.RGB(0.55, 0.35, 0.7)
	case "Adapter":
		return solidlens.RGB(0.9, 0.75, 0.25)
	case "Screw Knob", "Stud Knob":
		return solidlens.RGB(0.8, 0.25, 0.25)
	}
	if isClampHardware(name) {
		return solidlens.RGB(0.62, 0.64, 0.68)
	}
	return solidlens.RGB(0.3, 0.65, 0.35)
}

// isClampHardware reports whether the body is a bought bolt or nut.
func isClampHardware(name string) bool {
	return strings.Contains(name, "Bolt") || strings.Contains(name, "Nut")
}

// newClampArm builds the clamp arm at p and tessellates all 25 bodies: hardware at 0.01 mm, printed parts at 0.05.
func newClampArm(ctx context.Context, p clamparm.Params) (*clampArm, error) {
	m, err := clamparm.Build(ctx, clamparm.NewDocument(), p)
	if err != nil {
		return nil, fmt.Errorf("build clamp arm: %w", err)
	}
	dm := m.Dimensions
	bodies := map[string]*decad.Body{
		"Frame": m.Clamp.Body, "Pad": m.Pad.Body, "Swivel": m.Swivel.Body, "Link A": m.Links[0].Body,
		"Link B": m.Links[1].Body, "Head": m.Head.Body, "Adapter": m.Adapter.Body,
		"Screw Knob": m.ScrewKnobs[0].Body, "Stud Knob": m.ScrewKnobs[1].Body,
		"Stud Bolt": m.Bolts[4].Body, "Screw Bolt": m.Bolts[5].Body,
		"Jaw Nut": m.Nuts[4].Body, "Adapter Nut": m.Nuts[5].Body,
	}
	frames := map[string]r3.Frame{
		"Frame": m.Clamp.Frame, "Pad": m.Pad.Frame, "Swivel": m.Swivel.Frame, "Link A": m.Links[0].Frame,
		"Link B": m.Links[1].Frame, "Head": m.Head.Frame, "Adapter": m.Adapter.Frame,
		"Screw Knob": m.ScrewKnobs[0].Frame, "Stud Knob": m.ScrewKnobs[1].Frame,
		"Jaw Nut": dm.JawNutFrame, "Adapter Nut": dm.AdapterNutFrame,
	}
	sites := map[string]m5fastener.Site{"Bolt J1": dm.PanSite, "Bolt J2": dm.JointSites[0], "Bolt J3": dm.JointSites[1],
		"Bolt J4": dm.JointSites[2], "Stud Bolt": dm.StudSite, "Screw Bolt": dm.ScrewSite}
	boltIndex := map[string]int{"Bolt J1": 0, "Bolt J2": 1, "Bolt J3": 2, "Bolt J4": 3, "Stud Bolt": 4, "Screw Bolt": 5}
	for i := range m.Knobs {
		name := fmt.Sprintf("Knob J%d", i+1)
		bodies[name], frames[name] = m.Knobs[i].Body, m.Knobs[i].Frame
		name = fmt.Sprintf("Nut J%d", i+1)
		bodies[name], frames[name] = m.Nuts[i].Body, dm.KnobNutFrames[i]
	}
	for name, site := range sites {
		bodies[name] = m.Bolts[boltIndex[name]].Body
		f, err := r3.NewFrame(site.Head, site.Corner, site.Dir.Cross(site.Corner))
		if err != nil {
			return nil, fmt.Errorf("frame of %s: %w", name, err)
		}
		frames[name] = f
	}
	arm := &clampArm{dm: dm, meshes: map[string]*decad.Mesh{}, local: frames}
	for _, name := range clampBodyNames {
		body, ok := bodies[name]
		if !ok {
			return nil, fmt.Errorf("clamp arm has no body %q", name)
		}
		tol := 0.05
		if isClampHardware(name) {
			tol = 0.01
		}
		mesh, err := body.Tessellate(ctx, units.Millimeters(tol), decad.WithVerification(decad.VerifyNone))
		if err != nil {
			return nil, fmt.Errorf("tessellate %s: %w", name, err)
		}
		arm.meshes[name] = mesh
	}
	return arm, nil
}

func clampModel(source solidlens.TriangleSource, color solidlens.Color) solidlens.Model {
	return solidlens.Model{Mesh: source, Material: solidlens.Matte(color),
		Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(0.04, 0.06, 0.08), Width: 1.2}}
}

// moved returns the mesh of one body, moved by offset.
func (a *clampArm) moved(name string, offset r3.Vec) (solidlens.TriangleSource, error) {
	t, err := r3.Translation(offset)
	if err != nil {
		return nil, fmt.Errorf("offset of %s: %w", name, err)
	}
	return rotatedMesh{TriangleSource: a.meshes[name], move: t}, nil
}

// models returns the models of the named bodies (every body when names is empty), each moved by its offset.
func (a *clampArm) models(names []string, at r3.Vec, offsets map[string]r3.Vec) ([]solidlens.Model, error) {
	if len(names) == 0 {
		names = clampBodyNames
	}
	var out []solidlens.Model
	for _, name := range names {
		src, err := a.moved(name, at.Add(offsets[name]))
		if err != nil {
			return nil, err
		}
		out = append(out, clampModel(src, clampColor(name)))
	}
	return out, nil
}

// clampExplodedOffsets returns the exploded view's world offset of every body, for the default pose. Each joint's
// bolt leaves its -Y face and its nut and knob leave the +Y face, along the joint axis; the swivel lifts off the frame
// along J1's axis, link B, the head and the adapter move apart along the chain, and the pad, the screw and its knob
// go down. A bolt, nut and knob go with the part they were built on.
func (a *clampArm) clampExplodedOffsets() map[string]r3.Vec {
	dm := a.dm
	y, z := dm.Axes[1], dm.Axes[0]
	swivel := z.Scale(clampExplodeSwivel)
	linkA := dm.LinkA.U().Scale(clampExplodeLinkA)
	linkB := dm.LinkB.U().Scale(clampExplodeLinkB)
	head := dm.Head.U().Scale(clampExplodeHead)
	off := map[string]r3.Vec{"Frame": {}}
	off["Swivel"] = swivel
	off["Bolt J1"] = y.Scale(-clampExplodeBolt)
	off["Nut J1"] = z.Scale(clampExplodeNutJ1)
	off["Knob J1"] = z.Scale(clampExplodeKnobJ1)
	for i, base := range []r3.Vec{swivel, linkA, linkB} {
		n := i + 2
		off[fmt.Sprintf("Bolt J%d", n)] = base.Add(y.Scale(-clampExplodeBolt))
		off[fmt.Sprintf("Nut J%d", n)] = base.Add(y.Scale(clampExplodeNut))
		off[fmt.Sprintf("Knob J%d", n)] = base.Add(y.Scale(clampExplodeKnob))
	}
	off["Link A"], off["Link B"], off["Head"] = linkA, linkB, head
	n := dm.Interface.N()
	off["Adapter"] = head.Add(n.Scale(clampExplodeAdapter))
	off["Adapter Nut"] = head.Add(n.Scale(clampExplodeAdNut))
	off["Stud Bolt"] = head.Add(n.Scale(-clampExplodeStudBolt))
	off["Stud Knob"] = head.Add(n.Scale(-clampExplodeStudKnob))
	off["Pad"] = z.Scale(-clampExplodePad)
	off["Screw Bolt"] = z.Scale(-clampExplodeScrew)
	off["Screw Knob"] = z.Scale(-clampExplodeScrewKn)
	off["Jaw Nut"] = y.Scale(clampExplodeJawNut)
	return off
}

// clampCamera looks at target from direction dir with the given up, and frames the models.
func clampCamera(models []solidlens.Model, target r3.Vec, dir r3.Vec, up r3.Vec, aspect, margin float64) solidlens.Camera {
	u, _ := dir.Normalize()
	camera := solidlens.Camera{Position: solidlens.Vec(target.Add(u.Scale(1500))), Target: solidlens.Vec(target),
		Up: solidlens.Vec(up), FOV: 30, Near: 1, Far: 8000}
	return fitCamera(camera, models, aspect, margin)
}

// centreOf returns the middle of the models' bounds.
func centreOf(models []solidlens.Model) r3.Vec {
	low, high := modelBounds(models)
	return low.Add(high).Scale(0.5)
}

var (
	clampHeroDir = r3.NewVec(0.55, -1, 0.5)
	clampSideDir = r3.NewVec(0, 1, 0)
	clampBackDir = r3.NewVec(-0.6, 0.9, 0.5)
	clampTopDir  = r3.NewVec(0, 0, 1)
	// clampExplodedDir shows the exploded stack from the front, a little to the right and above, so the bolts out of
	// the -Y faces and the knobs out of the +Y faces separate on the screen.
	clampExplodedDir = r3.NewVec(0.9, -1, 0.35)
	clampZ           = r3.NewVec(0, 0, 1)
	clampY           = r3.NewVec(0, 1, 0)
)

// clampExplodedScene is the exploded view.
func (a *clampArm) clampExplodedScene() (solidlens.Scene, error) {
	models, err := a.models(clampExplodedOrder, r3.Vec{}, a.clampExplodedOffsets())
	if err != nil {
		return solidlens.Scene{}, err
	}
	camera := clampCamera(models, centreOf(models), clampExplodedDir, clampZ, 16.0/9, 0.04)
	return bevelScene(camera, models), nil
}

// capped returns m clipped by plane, with a lit cap in a lighter shade of the part's colour.
func capped(m solidlens.Model, color solidlens.Color, plane solidlens.ClipPlane) solidlens.Model {
	plane.Cap = &solidlens.Material{Color: solidlens.RGB(color.R+(1-color.R)*0.35, color.G+(1-color.G)*0.35,
		color.B+(1-color.B)*0.35), Unlit: true}
	m.Clip = &plane
	return m
}

// clampSection cuts the named bodies by plane and frames the box lo to hi from the removed side.
func (a *clampArm) clampSection(names []string, plane solidlens.ClipPlane, dir, up r3.Vec, lo, hi r3.Vec) (solidlens.Scene, error) {
	var models []solidlens.Model
	for _, name := range names {
		src, err := a.moved(name, r3.Vec{})
		if err != nil {
			return solidlens.Scene{}, err
		}
		models = append(models, capped(clampModel(src, clampColor(name)), clampColor(name), plane))
	}
	frame := []solidlens.Model{{Mesh: boxCorners{lo, hi}}}
	camera := clampCamera(frame, lo.Add(hi).Scale(0.5), dir, up, 16.0/9, 0.04)
	return bevelScene(camera, models), nil
}

// view draws one full-size view of the default pose.
func (a *clampArm) view(name string) (solidlens.Scene, error) {
	dm := a.dm
	i := dm.Interface.Origin()
	switch name {
	case "exploded":
		return a.clampExplodedScene()
	case "clamp-section":
		return a.clampSection([]string{"Frame", "Pad", "Screw Knob", "Screw Bolt", "Jaw Nut", "Bolt J1", "Nut J1",
			"Knob J1", "Swivel"},
			solidlens.ClipPlane{Normal: solidlens.Vec{Y: -1}}, r3.NewVec(0.1, -1, 0.08), clampZ,
			r3.NewVec(-18, -16, -106), r3.NewVec(58, 16, 44))
	case "joint-section":
		j := dm.Joints[2]
		u := dm.LinkA.U()
		plane := solidlens.ClipPlane{Origin: solidlens.Vec(j), Normal: solidlens.Vec(u)}
		return a.clampSection([]string{"Link A", "Link B", "Knob J3", "Bolt J3", "Nut J3"}, plane,
			u, clampY, j.Sub(r3.NewVec(40, 34, 40)), j.Add(r3.NewVec(75, 34, 40)))
	case "interface-section":
		plane := solidlens.ClipPlane{Normal: solidlens.Vec{Y: -1}}
		return a.clampSection([]string{"Head", "Adapter", "Stud Bolt", "Stud Knob", "Adapter Nut"}, plane,
			r3.NewVec(0.1, -1, 0.08), clampZ, i.Add(r3.NewVec(-55, -22, -42)), i.Add(r3.NewVec(35, 22, 38)))
	case "interface":
		models, err := a.models([]string{"Head", "Stud Bolt", "Stud Knob"}, r3.Vec{}, nil)
		if err != nil {
			return solidlens.Scene{}, err
		}
		frame := []solidlens.Model{{Mesh: boxCorners{i.Add(r3.NewVec(-55, -22, -26)), i.Add(r3.NewVec(35, 22, 22))}}}
		camera := clampCamera(frame, i, r3.NewVec(0.8, -1, 0.6), clampZ, 16.0/9, 0.04)
		return bevelScene(camera, models), nil
	}
	dirs := map[string]r3.Vec{"hero": clampHeroDir, "side": clampSideDir, "back": clampBackDir, "top": clampTopDir}
	dir, ok := dirs[name]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown clamp arm view %q", name)
	}
	models, err := a.models(nil, r3.Vec{}, nil)
	if err != nil {
		return solidlens.Scene{}, err
	}
	up := clampZ
	if name == "top" {
		up = clampY
	}
	return bevelScene(clampCamera(models, centreOf(models), dir, up, 16.0/9, 0.05), models), nil
}

// clampPosesScene draws the arms of the default, raised and folded poses side by side along +X, 40 mm apart.
func clampPosesScene(arms []*clampArm) (solidlens.Scene, error) {
	var models []solidlens.Model
	var x float64
	for _, a := range arms {
		part, err := a.models(nil, r3.Vec{}, nil)
		if err != nil {
			return solidlens.Scene{}, err
		}
		low, high := modelBounds(part)
		shift := r3.Vec{X: x - low.X}
		part, err = a.models(nil, shift, nil)
		if err != nil {
			return solidlens.Scene{}, err
		}
		models = append(models, part...)
		x += high.X - low.X + 40
	}
	return bevelScene(clampCamera(models, centreOf(models), r3.NewVec(0.2, -1, 0.3), clampZ, 16.0/9, 0.04), models), nil
}

// partScene draws one body alone in its own frame, the way it prints.
func (a *clampArm) partScene(name string) (solidlens.Scene, error) {
	move, err := r3.FromFrame(a.local[name])
	if err != nil {
		return solidlens.Scene{}, err
	}
	back, err := move.Inverse()
	if err != nil {
		return solidlens.Scene{}, err
	}
	mesh := rotatedMesh{TriangleSource: a.meshes[name], move: back}
	models := []solidlens.Model{clampModel(mesh, clampColor(name))}
	low, high := modelBounds(models)
	span := high.Sub(low).Len()
	centre := low.Add(high).Scale(0.5)
	camera := solidlens.Camera{Position: solidlens.Vec(centre.Add(r3.Vec{X: 0.9 * span, Y: -0.9 * span, Z: 0.8 * span})),
		Target: solidlens.Vec(centre), Up: solidlens.Vec{Z: 1}, FOV: 35}
	return bevelScene(fitCamera(camera, models, 16.0/9, 0.1), models), nil
}

// clampArmView returns the preset scene of one full-size view. "poses" builds all three poses.
func clampArmView(name string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		if name == "poses" {
			arms, err := clampPoseArms(ctx)
			if err != nil {
				return solidlens.Scene{}, err
			}
			return clampPosesScene(arms)
		}
		a, err := newClampArm(ctx, clamparm.DefaultParams())
		if err != nil {
			return solidlens.Scene{}, err
		}
		return a.view(name)
	}
}

// clampArmPart returns the preset scene of one part image.
func clampArmPart(part clampPart) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		a, err := newClampArm(ctx, clamparm.DefaultParams())
		if err != nil {
			return solidlens.Scene{}, err
		}
		return a.partScene(part.bodies[0])
	}
}

// clampPoseArms builds the default, raised and folded arms.
func clampPoseArms(ctx context.Context) ([]*clampArm, error) {
	var arms []*clampArm
	for _, p := range []clamparm.Params{clamparm.DefaultParams(), clampRaisedPose, clampFoldedPose} {
		a, err := newClampArm(ctx, p)
		if err != nil {
			return nil, err
		}
		arms = append(arms, a)
	}
	return arms, nil
}

// renderClampArmReadme builds the three poses once and writes every README image.
func renderClampArmReadme(ctx context.Context, outputDir string) error {
	arms, err := clampPoseArms(ctx)
	if err != nil {
		return err
	}
	a := arms[0]
	write := func(file string, scene solidlens.Scene, width, height int) error {
		return renderPNG(ctx, filepath.Join(outputDir, file), scene, width, height)
	}
	for _, view := range clampViews {
		var scene solidlens.Scene
		if view == "poses" {
			scene, err = clampPosesScene(arms)
		} else {
			scene, err = a.view(view)
		}
		if err != nil {
			return fmt.Errorf("clamp arm %s: %w", view, err)
		}
		if err := write(view+".png", scene, 1600, 900); err != nil {
			return err
		}
		if view == "hero" {
			if err := write("index.png", scene, 480, 270); err != nil {
				return err
			}
		}
	}
	for _, part := range clampParts {
		scene, err := a.partScene(part.bodies[0])
		if err != nil {
			return fmt.Errorf("clamp arm part %s: %w", part.shot, err)
		}
		if err := write(filepath.Join("parts", part.shot+".png"), scene, 480, 270); err != nil {
			return err
		}
	}
	return nil
}

// clampArmPartByShot returns the preset scene of the part image named shot.
func clampArmPartByShot(shot string) func(context.Context) (solidlens.Scene, error) {
	for _, part := range clampParts {
		if part.shot == shot {
			return clampArmPart(part)
		}
	}
	return func(context.Context) (solidlens.Scene, error) {
		return solidlens.Scene{}, fmt.Errorf("unknown clamp arm part %q", shot)
	}
}
