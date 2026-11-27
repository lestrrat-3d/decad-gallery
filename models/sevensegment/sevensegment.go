// Package sevensegment builds a cam-driven, 3D-printable seven-segment display: seven grooved face cams on one camshaft
// turn seven two-colour flags, and a one-way ratchet with two flexure pawls indexes the shaft, so every rest
// position of the knob shows one valid digit 0-9. See instructions.md for the construction contract.
//
// The model's local frame has its origin on the table under the digit's centre, +Z up, +Y toward the top of the digit
// (segment a) and -Y toward the knob.
package sevensegment

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/fastener"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Container names, one per kind of body.
const (
	FrameContainerName       = "Frame"
	WallsContainerName       = "Walls"
	PlatesContainerName      = "Plates"
	CamshaftContainerName    = "Camshaft"
	ShaftContainerName       = "Shaft"
	CamsContainerName        = "Cams"
	ClickerContainerName     = "Click Indexer"
	HubContainerName         = "Hub"
	RotorContainerName       = "Rotor"
	HousingContainerName     = "Housing"
	ButtonContainerName      = "Button"
	SpringContainerName      = "Spring"
	SlidersContainerName     = "Segment Sliders"
	GroovePinsContainerName  = "Groove Pins"
	BearingsContainerName    = "Bearings"
	BoltsContainerName       = "Bolts"
	NutsContainerName        = "Nuts"
	FlagsContainerName       = "Flags"
	FlagUnitName             = "Flag"
	YellowFaceContainerName  = "Yellow Face"
	BlackBodyContainerName   = "Black Body"
	PivotBlocksContainerName = "Pivot Blocks"
)

// Parts are the printed bodies, posed and placed, without the bought parts.
type Parts struct {
	FrontWall, RearWall, GuidePlate, CradlePlate, TopPlate *decad.Body
	Shaft                                                  *decad.Body
	// The click indexer: the keyed hub and the rotor turn with the shaft; the housing and the button do not.
	Hub, Rotor, Housing, Button, Spring *decad.Body
	Cams, Sliders                       [7]*decad.Body // indexed by Segment
	// GroovePins are the separate keyed pins of the a, g and d sliders, indexed by Segment; b to f have an integral pin and
	// their entries are nil. Each is one body that the export cuts into two identical halves on the plane x = 0.
	GroovePins [7]*decad.Body
	// Flags are the seven two-colour rotors, indexed by Segment.
	Flags [7]Flag
	// PivotBlocks are the 14 pivot blocks, indexed by Segment and then 0 for the rotor's -u end, 1 for its +u end.
	PivotBlocks [7][2]*decad.Body
	// Sketches are every sketch the parts were built from, each fully constrained.
	Sketches   []*sketch.Sketch
	Dimensions Dimensions
	// Sites are the ten bolts in the world frame after Placement.
	Sites [10]m2fastener.Site
}

// Model is a built display: its parts, bearings, bolts and nuts.
type Model struct {
	Parts
	Bearings [2]*ballbearing.Model // front, rear
	Bolts    [10]*fastener.Bolt
	Nuts     [10]*fastener.Nut
}

// BuildParts adds the printed bodies to doc, posed for p.Digit and placed, and puts them in no container. ctx and doc
// must not be nil.
func BuildParts(ctx context.Context, doc *decad.Document, p Params) (*Parts, error) {
	out, _, err := BuildPartsPosed(ctx, doc, p)
	return out, err
}

// Pose is a camshaft digit and the frame the parts are placed in. A nil Placement leaves the parts where the model's
// local frame puts them.
type Pose struct {
	Digit     int
	Placement *r3.Frame
}

// BuildPartsPosed is BuildParts for p. It also poses the same bodies for each of extra, and returns one *Parts per
// extra pose, in order. Each body is built once, unposed, and the extra poses are copies of it, so an extra pose costs
// a small fraction of a build. An extra pose's parts are bit for bit those of BuildParts with that digit and
// placement. A body that the digit does not move and that has the same placement in two poses is the same
// *decad.Body in both. Every other body of an extra pose is a new body of doc, filed in no container.
//
// The extra poses carry the other fields of p, so extra[i] differs from p in Digit and Placement only. An invalid
// extra pose fails as BuildParts fails for it. Each extra Parts has the main Parts's Sketches. ctx and doc must not be
// nil.
func BuildPartsPosed(ctx context.Context, doc *decad.Document, p Params, extra ...Pose) (*Parts, []*Parts, error) {
	out := &Parts{}
	extras := make([]*Parts, len(extra))
	for i := range extras {
		extras[i] = &Parts{}
	}
	if err := out.build(ctx, doc, p, extra, extras, partSteps(out, extras)); err != nil {
		return nil, nil, err
	}
	return out, extras, nil
}

// build runs steps, which fill out and extras, and then sets the dimensions, the sketches and the bolt sites of each.
func (out *Parts) build(ctx context.Context, doc *decad.Document, p Params, poses []Pose, extras []*Parts,
	steps []buildStep) error {
	dm, poseDims, sketches, err := runPosedSteps(ctx, doc, p, poses, steps, stepWorkers())
	if err != nil {
		return err
	}
	out.Dimensions, out.Sketches = dm, sketches
	if out.Sites, err = placeSites(dm, p.Placement); err != nil {
		return err
	}
	for i, e := range extras {
		e.Dimensions, e.Sketches = poseDims[i], sketches
		if e.Sites, err = placeSites(poseDims[i], poses[i].Placement); err != nil {
			return err
		}
	}
	return nil
}

// placeSites carries every bolt site of dm through the placement f.
func placeSites(dm Dimensions, f *r3.Frame) ([10]m2fastener.Site, error) {
	var out [10]m2fastener.Site
	for i, site := range dm.Sites {
		var err error
		if out[i], err = placeSite(site, f); err != nil {
			return out, fmt.Errorf("sevensegment: place bolt %d: %w", i+1, err)
		}
	}
	return out, nil
}

// camshaftTurn is the turn of the camshaft group by theta about the shaft axis, right-handed about +Y.
func (r *run) camshaftTurn() r3.Transform {
	t, _ := r3.RotationAround(r3.Vec{Z: ShaftZ}, r3.Vec{Y: 1}, units.Degrees(r.dm.Theta))
	return t
}

// Build adds every body to doc, each in its container: 78 bodies. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error) {
	m, _, err := BuildPosed(ctx, doc, p)
	return m, err
}

// BuildPosed is Build for p, and also poses the printed parts for each of extra as BuildPartsPosed does. The extra
// poses hold no bought parts and their bodies are filed in no container, so the 78 bodies and the containers of the
// returned model are those of Build. ctx and doc must not be nil.
func BuildPosed(ctx context.Context, doc *decad.Document, p Params, extra ...Pose) (*Model, []*Parts, error) {
	m := &Model{}
	extras := make([]*Parts, len(extra))
	for i := range extras {
		extras[i] = &Parts{}
	}
	if err := m.build(ctx, doc, p, extra, extras, append(partSteps(&m.Parts, extras), hardwareSteps(m)...)); err != nil {
		return nil, nil, err
	}
	model, err := m.file(doc)
	if err != nil {
		return nil, nil, err
	}
	return model, extras, nil
}

// file puts the bodies of m in their containers in doc.
func (m *Model) file(doc *decad.Document) (*Model, error) {
	root := doc.Root()
	units3 := map[string]*decad.Container{}
	unit := func(name string) (*decad.Container, error) {
		if c, ok := units3[name]; ok {
			return c, nil
		}
		c, err := root.NewChild(name)
		if err != nil {
			return nil, fmt.Errorf("sevensegment: container %s: %w", name, err)
		}
		units3[name] = c
		return c, nil
	}
	type filed struct {
		parent, kind string // parent "" for the root
		body         *decad.Body
	}
	files := []filed{
		{FrameContainerName, WallsContainerName, m.FrontWall}, {FrameContainerName, WallsContainerName, m.RearWall},
		{FrameContainerName, PlatesContainerName, m.GuidePlate}, {FrameContainerName, PlatesContainerName, m.CradlePlate},
		{FrameContainerName, PlatesContainerName, m.TopPlate},
		{CamshaftContainerName, ShaftContainerName, m.Shaft},
		{ClickerContainerName, HubContainerName, m.Hub},
		{ClickerContainerName, RotorContainerName, m.Rotor},
		{ClickerContainerName, HousingContainerName, m.Housing},
		{ClickerContainerName, ButtonContainerName, m.Button},
		{ClickerContainerName, SpringContainerName, m.Spring},
	}
	for _, seg := range Segments() {
		files = append(files, filed{CamshaftContainerName, CamsContainerName, m.Cams[seg]},
			filed{"", SlidersContainerName, m.Sliders[seg]})
		if m.GroovePins[seg] != nil {
			files = append(files, filed{"", GroovePinsContainerName, m.GroovePins[seg]})
		}
	}
	for i := range m.Bearings {
		files = append(files, filed{"", BearingsContainerName, m.Bearings[i].Body})
	}
	for i := range m.Bolts {
		files = append(files, filed{"", BoltsContainerName, m.Bolts[i].Body}, filed{"", NutsContainerName, m.Nuts[i].Body})
	}
	flags, err := root.NewChild(FlagsContainerName)
	if err != nil {
		return nil, fmt.Errorf("sevensegment: container %s: %w", FlagsContainerName, err)
	}
	for _, seg := range Segments() {
		unit, err := flags.NewUniqueChild(FlagUnitName)
		if err != nil {
			return nil, fmt.Errorf("sevensegment: container %s: %w", FlagUnitName, err)
		}
		for _, k := range []struct {
			kind string
			body *decad.Body
		}{{YellowFaceContainerName, m.Flags[seg].Yellow}, {BlackBodyContainerName, m.Flags[seg].Black}} {
			box, err := unit.ChildNamed(k.kind)
			if err != nil {
				return nil, fmt.Errorf("sevensegment: container %s: %w", k.kind, err)
			}
			if err := box.AddBody(k.body); err != nil {
				return nil, fmt.Errorf("sevensegment: add body to %s: %w", k.kind, err)
			}
		}
	}
	blocks, err := root.NewChild(PivotBlocksContainerName)
	if err != nil {
		return nil, fmt.Errorf("sevensegment: container %s: %w", PivotBlocksContainerName, err)
	}
	for _, seg := range Segments() {
		for _, b := range m.PivotBlocks[seg] {
			if err := blocks.AddBody(b); err != nil {
				return nil, fmt.Errorf("sevensegment: add body to %s: %w", PivotBlocksContainerName, err)
			}
		}
	}
	for _, f := range files {
		parent := root
		if f.parent != "" {
			if parent, err = unit(f.parent); err != nil {
				return nil, err
			}
		}
		box, err := parent.ChildNamed(f.kind)
		if err != nil {
			return nil, fmt.Errorf("sevensegment: container %s: %w", f.kind, err)
		}
		if err := box.AddBody(f.body); err != nil {
			return nil, fmt.Errorf("sevensegment: add body to %s: %w", f.kind, err)
		}
	}
	return m, nil
}

// placeSite carries a local bolt site through the placement.
func placeSite(s m2fastener.Site, f *r3.Frame) (m2fastener.Site, error) {
	if f == nil {
		return s, nil
	}
	xf, err := r3.FromFrame(*f)
	if err != nil {
		return s, err
	}
	s.Head, s.Dir, s.Corner = xf.Apply(s.Head), xf.ApplyDir(s.Dir), xf.ApplyDir(s.Corner)
	return s, nil
}
