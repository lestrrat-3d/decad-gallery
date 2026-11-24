// Package sevensegment builds a cam-driven, 3D-printable seven-segment display: seven grooved face cams on one camshaft
// turn seven two-colour flags, and a one-way ratchet with two flexure pawls indexes the shaft, so every rest
// position of the knob shows one valid digit 0-9. See instructions.md for the construction contract.
//
// The model's local frame has its origin on the table under the digit's centre, +Z up, +Y toward the top of the digit
// (segment a) and -Y toward the knob.
package sevensegment

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
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
	if ctx == nil {
		return nil, errors.New("sevensegment: context must not be nil")
	}
	if doc == nil {
		return nil, errors.New("sevensegment: document must not be nil")
	}
	dm, err := p.Derive()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("sevensegment: before creating geometry: %w", err)
	}
	r := &run{ctx: ctx, doc: doc, p: p, dm: dm}
	out := &Parts{Dimensions: dm}
	spin := func(b *decad.Body) (*decad.Body, error) { return decad.Transform(ctx, b, r.camshaftTurn()) }
	type step struct {
		name  string
		build func() (*decad.Body, error)
		dst   **decad.Body
		pose  func(*decad.Body) (*decad.Body, error)
	}
	steps := []step{
		{"front wall", func() (*decad.Body, error) { return r.wall(true) }, &out.FrontWall, nil},
		{"rear wall", func() (*decad.Body, error) { return r.wall(false) }, &out.RearWall, nil},
		{"guide plate", r.columnGuidePlate, &out.GuidePlate, nil},
		{"cradle plate", r.cradlePlate, &out.CradlePlate, nil},
		{"top plate", r.flagTopPlate, &out.TopPlate, nil},
		{"shaft", r.shaft, &out.Shaft, spin},
		{"hub", r.hub, &out.Hub, spin},
		{"rotor", r.rotor, &out.Rotor, spin},
		{"housing", r.housing, &out.Housing, nil},
		{"button", r.button, &out.Button, nil},
		{"spring", func() (*decad.Body, error) { return r.spring(SpringRestLength) }, &out.Spring, spin},
	}
	for _, seg := range Segments() {
		steps = append(steps,
			step{"cam " + seg.String(), func() (*decad.Body, error) { return r.cam(seg) }, &out.Cams[seg], spin},
			step{"slider " + seg.String(), func() (*decad.Body, error) { return r.flagSlider(seg) }, &out.Sliders[seg],
				func(b *decad.Body) (*decad.Body, error) {
					lit, _ := Lit(p.Digit, seg)
					if !lit {
						return b, nil
					}
					lift, err := r3.Translation(r3.Vec{Z: Travel})
					if err != nil {
						return nil, err
					}
					return decad.Transform(ctx, b, lift)
				}})
	}
	var prototypes [2]Flag
	for _, seg := range Segments() {
		f, err := r.prototypeFlag(seg, &prototypes)
		if err != nil {
			return nil, fmt.Errorf("sevensegment: flag %s: %w", seg, err)
		}
		cx, cy, horizontal := segmentCentre(seg)
		axis := r3.Vec{Y: 1}
		if horizontal {
			axis = r3.Vec{X: 1}
		}
		turn, err := r3.RotationAround(r3.Vec{X: cx, Y: cy, Z: RotorAxisZ}, axis, units.Degrees(dm.FlagAngle(seg, dm.Theta)))
		if err != nil {
			return nil, fmt.Errorf("sevensegment: flag %s: turn: %w", seg, err)
		}
		for _, b := range []struct {
			name string
			in   *decad.Body
			dst  **decad.Body
		}{{"yellow face", f.Yellow, &out.Flags[seg].Yellow}, {"black body", f.Black, &out.Flags[seg].Black}} {
			posed, err := decad.Transform(ctx, b.in, turn)
			if err != nil {
				return nil, fmt.Errorf("sevensegment: flag %s %s: turn: %w", seg, b.name, err)
			}
			if *b.dst, err = placement.Place(ctx, posed, p.Placement); err != nil {
				return nil, fmt.Errorf("sevensegment: place flag %s %s: %w", seg, b.name, err)
			}
		}
	}
	if out.PivotBlocks, err = r.pivotBlocks(); err != nil {
		return nil, fmt.Errorf("sevensegment: pivot blocks: %w", err)
	}
	for _, st := range steps {
		body, err := st.build()
		if err != nil {
			return nil, fmt.Errorf("sevensegment: %s: %w", st.name, err)
		}
		if body == nil || !body.IsSolid() {
			return nil, fmt.Errorf("sevensegment: %s is not a solid", st.name)
		}
		if st.pose != nil {
			if body, err = st.pose(body); err != nil {
				return nil, fmt.Errorf("sevensegment: pose %s: %w", st.name, err)
			}
		}
		if *st.dst, err = placement.Place(ctx, body, p.Placement); err != nil {
			return nil, fmt.Errorf("sevensegment: place %s: %w", st.name, err)
		}
	}
	out.Sketches = r.sketches
	for i, site := range dm.Sites {
		if out.Sites[i], err = placeSite(site, p.Placement); err != nil {
			return nil, fmt.Errorf("sevensegment: place bolt %d: %w", i+1, err)
		}
	}
	return out, nil
}

// camshaftTurn is the turn of the camshaft group by theta about the shaft axis, right-handed about +Y.
func (r *run) camshaftTurn() r3.Transform {
	t, _ := r3.RotationAround(r3.Vec{Z: ShaftZ}, r3.Vec{Y: 1}, units.Degrees(r.dm.Theta))
	return t
}

// Build adds every body to doc, each in its container: 75 bodies. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error) {
	parts, err := BuildParts(ctx, doc, p)
	if err != nil {
		return nil, err
	}
	m := &Model{Parts: *parts}
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
	}
	for i, f := range []struct {
		origin, u, v r3.Vec
	}{
		{r3.Vec{Y: frontInnerY, Z: ShaftZ}, r3.Vec{X: 1}, r3.Vec{Z: 1}},
		{r3.Vec{Y: rearInnerY, Z: ShaftZ}, r3.Vec{Z: 1}, r3.Vec{X: 1}},
	} {
		frame, err := r3.NewFrame(f.origin, f.u, f.v)
		if err != nil {
			return nil, fmt.Errorf("sevensegment: bearing %d frame: %w", i+1, err)
		}
		bp := ballbearing.Params605ZZ()
		bp.Placement = &frame
		if p.Placement != nil {
			lift, err := r3.FromFrame(*p.Placement)
			if err != nil {
				return nil, fmt.Errorf("sevensegment: bearing %d placement: %w", i+1, err)
			}
			bp.Lift = &lift
		}
		if m.Bearings[i], err = ballbearing.Build(ctx, doc, bp); err != nil {
			return nil, fmt.Errorf("sevensegment: bearing %d: %w", i+1, err)
		}
		files = append(files, filed{"", BearingsContainerName, m.Bearings[i].Body})
	}
	for i, site := range m.Sites {
		if m.Bolts[i], err = m2fastener.PlaceBolt(ctx, doc, site); err != nil {
			return nil, fmt.Errorf("sevensegment: bolt %d: %w", i+1, err)
		}
		if m.Nuts[i], err = m2fastener.PlaceNut(ctx, doc, site); err != nil {
			return nil, fmt.Errorf("sevensegment: nut %d: %w", i+1, err)
		}
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
