package sevensegment_test

// CI: isolated-tests

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestSevenSegmentDefault(t *testing.T) {
	startDefault()
	t.Parallel()
	doc, m := builtDefault(t)
	require.Len(t, doc.Bodies(), 75)
	for _, b := range doc.Bodies() {
		require.True(t, b.IsSolid())
	}
	require.NotEmpty(t, m.Sketches)
	for i, s := range m.Sketches {
		report := s.Verify(t.Context())
		require.Equal(t, 0, report.DOF, "sketch %d", i)
		require.Empty(t, report.Redundant, "sketch %d", i)
		require.Empty(t, report.Conflicts, "sketch %d", i)
	}
	// No two printed bodies overlap: every pair is separated, the stated fits keeping at least their 0.05 mm.
	meshes := map[string]meshgap.Mesh{}
	boxes := map[string]decad.Box{}
	for name, b := range printed(m) {
		_, meshes[name] = meshOf(t, b, partTol(name))
		box, err := b.Bounds()
		require.NoError(t, err)
		boxes[name] = box
	}
	names := make([]string, 0, len(meshes))
	for n := range meshes {
		names = append(names, n)
	}
	pressed := map[string]struct{}{"hub": {}}
	for _, seg := range sevensegment.Segments() {
		pressed["cam "+seg.String()] = struct{}{}
	}
	shaftPairs := 0
	for i, a := range names {
		for _, b := range names[i+1:] {
			if !boxesTouch(boxes[a], boxes[b], 0.05) {
				continue
			}
			if isPivotPress(a, b) {
				continue // a pivot block is pressed 0.05 in its pocket: TestSevenSegmentPivotBlocks measures it on finer meshes
			}
			if isToothMate(a, b) {
				continue // the click indexer's mating teeth keep their land gaps: TestClickerLandGaps proves them
			}
			if strings.HasPrefix(a, "yellow face ") && b == "black body "+strings.TrimPrefix(a, "yellow face ") ||
				strings.HasPrefix(b, "yellow face ") && a == "black body "+strings.TrimPrefix(b, "yellow face ") {
				continue // the two bodies of one flag share their boundary faces
			}
			// Every pair is checked, the shaft's too. The parts pressed on the shaft keep its 0.05 mm press fit; their
			// meshes are read at a finer chord (0.005 mm) so that the proven bound is tighter than the fit. No other pair
			// overlaps by design, so there is no allow-list beyond the two bodies of one flag above.
			ma, mb, clearance := meshes[a], meshes[b], 0.0
			if other, ok := shaftMate(a, b); ok {
				if _, ok := pressed[other]; ok {
					_, ma = meshOf(t, printed(m)[a], pressTol(a))
					_, mb = meshOf(t, printed(m)[b], pressTol(b))
					clearance = 0.025 // the 0.05 mm press fit less the meshes' 0.025 mm allowance
					shaftPairs++
				}
			}
			sep, err := ma.ExceedsClearance(mb, clearance)
			require.NoError(t, err)
			require.True(t, sep, "%s and %s overlap or touch, or keep less than %.2f mm", a, b, clearance)
		}
	}
	require.Equal(t, len(pressed), shaftPairs, "every part pressed on the shaft is checked against it")
}

func TestSevenSegmentContainers(t *testing.T) {
	startDefault()
	t.Parallel()
	doc, m := builtDefault(t)
	require.NoError(t, doc.VerifyContainers().Err())
	want := map[string]struct{ parent, kind string }{
		"front wall":   {sevensegment.FrameContainerName, sevensegment.WallsContainerName},
		"rear wall":    {sevensegment.FrameContainerName, sevensegment.WallsContainerName},
		"guide plate":  {sevensegment.FrameContainerName, sevensegment.PlatesContainerName},
		"cradle plate": {sevensegment.FrameContainerName, sevensegment.PlatesContainerName},
		"top plate":    {sevensegment.FrameContainerName, sevensegment.PlatesContainerName},
		"shaft":        {sevensegment.CamshaftContainerName, sevensegment.ShaftContainerName},
		"hub":          {sevensegment.ClickerContainerName, sevensegment.HubContainerName},
		"rotor":        {sevensegment.ClickerContainerName, sevensegment.RotorContainerName},
		"housing":      {sevensegment.ClickerContainerName, sevensegment.HousingContainerName},
		"button":       {sevensegment.ClickerContainerName, sevensegment.ButtonContainerName},
	}
	for _, s := range sevensegment.Segments() {
		want["cam "+s.String()] = struct{ parent, kind string }{sevensegment.CamshaftContainerName,
			sevensegment.CamsContainerName}
		want["slider "+s.String()] = struct{ parent, kind string }{"", sevensegment.SlidersContainerName}
	}
	for name, b := range printed(m) {
		if strings.HasPrefix(name, "yellow face") || strings.HasPrefix(name, "black body") ||
			strings.HasPrefix(name, "pivot block") {
			continue // checked below
		}
		c, err := doc.ContainerOf(b)
		require.NoError(t, err, name)
		require.Equal(t, want[name].kind, c.Name(), name)
		if want[name].parent == "" {
			require.Equal(t, doc.Root(), c.Parent(), name)
		} else {
			require.Equal(t, want[name].parent, c.Parent().Name(), name)
		}
	}
	// Each flag is a unit of the Flags container holding its Yellow Face and its Black Body.
	for _, s := range sevensegment.Segments() {
		for kind, b := range map[string]*decad.Body{sevensegment.YellowFaceContainerName: m.Flags[s].Yellow,
			sevensegment.BlackBodyContainerName: m.Flags[s].Black} {
			c, err := doc.ContainerOf(b)
			require.NoError(t, err)
			require.Equal(t, kind, c.Name())
			require.True(t, strings.HasPrefix(c.Parent().Name(), sevensegment.FlagUnitName), "segment %s", s)
			require.Equal(t, sevensegment.FlagsContainerName, c.Parent().Parent().Name())
		}
	}
	// The 14 pivot blocks all sit in one Pivot Blocks container under the root.
	for _, s := range sevensegment.Segments() {
		for end, b := range m.PivotBlocks[s] {
			c, err := doc.ContainerOf(b)
			require.NoError(t, err, "pivot block %s %d", s, end)
			require.Equal(t, sevensegment.PivotBlocksContainerName, c.Name())
			require.Equal(t, doc.Root(), c.Parent())
		}
	}
	for i, b := range m.Bearings {
		c, err := doc.ContainerOf(b.Body)
		require.NoError(t, err, "bearing %d", i)
		require.Equal(t, sevensegment.BearingsContainerName, c.Name())
	}
	for i := range m.Bolts {
		c, err := doc.ContainerOf(m.Bolts[i].Body)
		require.NoError(t, err)
		require.Equal(t, sevensegment.BoltsContainerName, c.Name())
		c, err = doc.ContainerOf(m.Nuts[i].Body)
		require.NoError(t, err)
		require.Equal(t, sevensegment.NutsContainerName, c.Name())
	}
}

func TestSevenSegmentFasteners(t *testing.T) {
	startDefault()
	t.Parallel()
	_, m := builtDefault(t)
	const tol = 0.02
	parts := printed(m)
	pmeshes := map[string]*decad.Mesh{}
	grids := map[string]*crossingGrid{}
	for name, b := range parts {
		pmeshes[name], _ = meshOf(t, b, partTol(name))
		grids[name] = newCrossingGrid(pmeshes[name])
	}
	for i, site := range m.Sites {
		base := site.Head.Dot(site.Dir)
		bolt, _ := meshOf(t, m.Bolts[i].Body, 0.01)
		lo, hi := along(bolt, site.Dir)
		require.InDelta(t, base, lo, tol, "bolt %d head top is flush", i+1)
		require.InDelta(t, base+site.Tip(), hi, tol, "bolt %d tip", i+1)
		// The tip stays inside a face: it does not pass the farthest face of the parts around the bolt's axis. The faces are
		// read from the part meshes, not from the site the bolt was built from.
		far := math.Inf(-1)
		for _, mesh := range pmeshes {
			for _, v := range mesh.Vertices() {
				d := v.Sub(site.Head)
				along := d.Dot(site.Dir)
				if along < 0 || d.Sub(site.Dir.Scale(along)).Len() > 2.5 {
					continue
				}
				far = math.Max(far, v.Dot(site.Dir))
			}
		}
		require.LessOrEqual(t, hi, far+tol, "bolt %d tip stands outside the parts' far face (%.2f)", i+1, far)
		nut, _ := meshOf(t, m.Nuts[i].Body, 0.01)
		nlo, nhi := along(nut, site.Dir)
		require.GreaterOrEqual(t, nlo, base+site.Cavity-tol, "nut %d lies inside its cavity", i+1)
		require.LessOrEqual(t, nhi, base+site.CavityFar()+tol, "nut %d lies inside its cavity", i+1)
		centre := r3.Vec{}
		for _, v := range nut.Vertices() {
			centre = centre.Add(v)
		}
		centre = centre.Scale(1 / float64(len(nut.Vertices())))
		// No bolt point lies inside a part it passes. A head vertex is probed 0.1 mm toward the head's middle, so that the
		// vertices on the flush top and on the underside (which rests on the pocket floor) are read from inside the head: a
		// head that reaches into the part beside its pocket puts those probes in the material. The shank is read at its
		// vertices.
		for name, grid := range grids {
			for _, v := range bolt.Vertices() {
				p, depth := v, v.Sub(site.Head).Dot(site.Dir)
				switch {
				case depth <= m2fastener.HeadPocketDepth+0.05 && depth < m2fastener.HeadPocketDepth/2:
					p = v.Add(site.Dir.Scale(0.1))
				case depth <= m2fastener.HeadPocketDepth+0.05:
					p = v.Sub(site.Dir.Scale(0.1))
				}
				if grid.pointIn(p) {
					require.Fail(t, "bolt overlaps a part", "bolt %d vertex %v (probe %v) inside %s", i+1, v, p, name)
				}
			}
			// The nut lies in its cavity: no vertex, probed 0.05 mm toward the nut's middle, is in a part, and the nut keeps
			// at least 0.05 mm (the coarse part mesh's own bound) from every part near it.
			for _, v := range nut.Vertices() {
				toward := centre.Sub(v)
				if grid.pointIn(v.Add(toward.Scale(0.05 / toward.Len()))) {
					require.Fail(t, "nut overlaps a part", "nut %d vertex %v inside %s", i+1, v, name)
				}
			}
		}
		_, nutGap := meshOf(t, m.Nuts[i].Body, 0.01)
		nutBox, err := m.Nuts[i].Body.Bounds()
		require.NoError(t, err)
		for name, b := range parts {
			box, err := b.Bounds()
			require.NoError(t, err)
			if !boxesTouch(box, nutBox, 0.5) {
				continue
			}
			_, partGap := meshOf(t, b, 0.05)
			ok, err := nutGap.ExceedsClearance(partGap, 0.05)
			require.NoError(t, err)
			require.True(t, ok, "nut %d keeps 0.05 mm from %s", i+1, name)
		}
	}
}

// TestSevenSegmentBearings checks the two 605ZZ bearings against the parts around them: the wall seat grips the outer
// race with 0.05 mm of radial clearance (P2), the shaft's journal equals the bore (N1, a nominal exception: no clearance
// and no overlap), and no other printed part touches a bearing.
func TestSevenSegmentBearings(t *testing.T) {
	startDefault()
	t.Parallel()
	_, m := builtDefault(t)
	parts := printed(m)
	for i, walls := range []struct {
		name string
		wall *decad.Body
	}{{"front wall", m.FrontWall}, {"rear wall", m.RearWall}} {
		bearing := m.Bearings[i].Body
		box, err := bearing.Bounds()
		require.NoError(t, err)
		mid := (box.Min.Y + box.Max.Y) / 2
		race := sevensegment.SectionAt(t, bearing, mid)
		require.Len(t, race, 2, "%s bearing section: outer ring and bore", walls.name)
		outer, bore := race[:1], race[1:]
		if radius(race[0]) < radius(race[1]) {
			outer, bore = race[1:], race[:1]
		}
		seat := sevensegment.LoopsGap(outer, sevensegment.SectionAt(t, walls.wall, mid))
		require.InDelta(t, sevensegment.PressFit, seat, 0.02, "%s seat to the outer race (P2)", walls.name)
		journal := sevensegment.SectionAt(t, m.Shaft, mid)
		require.InDelta(t, radius(bore[0]), radius(journal[0]), 0.01, "%s: journal and bore (N1)", walls.name)
		// No overlap: the journal's radius never exceeds the bore's.
		for _, p := range journal[0] {
			require.LessOrEqual(t, p.Len(), radius(bore[0])+0.005, "%s: the journal stays inside the bore", walls.name)
		}
		_, bearingGap := meshOf(t, bearing, 0.05)
		for name, b := range parts {
			if b == walls.wall || b == m.Shaft {
				continue
			}
			pbox, err := b.Bounds()
			require.NoError(t, err)
			if !boxesTouch(pbox, box, 0.05) {
				continue
			}
			_, partGap := meshOf(t, b, 0.05)
			ok, err := bearingGap.ExceedsClearance(partGap, 0)
			require.NoError(t, err)
			require.True(t, ok, "%s bearing and %s touch", walls.name, name)
		}
	}
}

// TestSevenSegmentPlacement moves the whole display three ways: by 100 mm along X, by (10, 20, 30), and by a quarter turn
// about Z. Each part's bounds and each bolt site must follow the placement. Decad builds all three (an earlier note here
// said it refused slider b for most translations; these and several others build).
func TestSevenSegmentPlacement(t *testing.T) {
	type placement struct {
		name         string
		origin, u, v r3.Vec
	}
	cases := []placement{
		{"100 along X", r3.Vec{X: 100}, r3.Vec{X: 1}, r3.Vec{Y: 1}},
		{"offset 10, 20, 30", r3.Vec{X: 10, Y: 20, Z: 30}, r3.Vec{X: 1}, r3.Vec{Y: 1}},
		{"quarter turn about Z", r3.Vec{}, r3.Vec{Y: 1}, r3.Vec{X: -1}},
	}
	builds := make([]build[*sevensegment.Parts], len(cases))
	frames := make([]r3.Frame, len(cases))
	for i, c := range cases {
		var err error
		frames[i], err = r3.NewFrame(c.origin, c.u, c.v)
		require.NoError(t, err)
		p := sevensegment.DefaultParams()
		p.Placement = &frames[i]
		builds[i].start(func(ctx context.Context) (*sevensegment.Parts, error) {
			return sevensegment.BuildParts(ctx, sevensegment.NewDocument(), p)
		})
	}
	startDefault()
	t.Parallel()
	_, base := builtDefault(t)
	plain := printed(base)
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			frame := frames[i]
			p := sevensegment.DefaultParams()
			p.Placement = &frame
			parts := builds[i].built(t, func(ctx context.Context) (*sevensegment.Parts, error) {
				return sevensegment.BuildParts(ctx, sevensegment.NewDocument(), p)
			})
			place, err := r3.FromFrame(frame)
			require.NoError(t, err)
			for name, b := range printed(&sevensegment.Model{Parts: *parts}) {
				got, err := b.Bounds()
				require.NoError(t, err)
				want, err := plain[name].Bounds()
				require.NoError(t, err)
				// The placed box is the box of the plain one's corners, which a quarter turn keeps axis-aligned.
				lo, hi := r3.Vec{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}, r3.Vec{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}
				for _, x := range []float64{want.Min.X, want.Max.X} {
					for _, y := range []float64{want.Min.Y, want.Max.Y} {
						for _, z := range []float64{want.Min.Z, want.Max.Z} {
							q := place.Apply(r3.Vec{X: x, Y: y, Z: z})
							lo = r3.Vec{X: math.Min(lo.X, q.X), Y: math.Min(lo.Y, q.Y), Z: math.Min(lo.Z, q.Z)}
							hi = r3.Vec{X: math.Max(hi.X, q.X), Y: math.Max(hi.Y, q.Y), Z: math.Max(hi.Z, q.Z)}
						}
					}
				}
				for axis, pair := range map[string][2]float64{"x": {lo.X, got.Min.X}, "y": {lo.Y, got.Min.Y}, "z": {lo.Z, got.Min.Z},
					"max x": {hi.X, got.Max.X}, "max y": {hi.Y, got.Max.Y}, "max z": {hi.Z, got.Max.Z}} {
					require.InDelta(t, pair[0], pair[1], 0.02, "%s: %s", name, axis)
				}
			}
			for k, s := range parts.Sites {
				head, dir := place.Apply(base.Sites[k].Head), place.ApplyDir(base.Sites[k].Dir)
				require.InDelta(t, head.X, s.Head.X, 1e-9, "site %d head x", k)
				require.InDelta(t, head.Y, s.Head.Y, 1e-9, "site %d head y", k)
				require.InDelta(t, head.Z, s.Head.Z, 1e-9, "site %d head z", k)
				require.InDelta(t, dir.X, s.Dir.X, 1e-9, "site %d dir x", k)
				require.InDelta(t, dir.Y, s.Dir.Y, 1e-9, "site %d dir y", k)
				require.InDelta(t, dir.Z, s.Dir.Z, 1e-9, "site %d dir z", k)
			}
		})
	}
	p := sevensegment.DefaultParams()
	p.Placement = &r3.Frame{}
	_, err := sevensegment.BuildParts(t.Context(), sevensegment.NewDocument(), p)
	require.EqualError(t, err, "sevensegment: invalid placement frame")
}

// TestSevenSegmentAllDigits poses the digit-0 build for each digit: the cams turned, the sliders lifted and the flags
// turned. It checks every cam against its slider (the pin keeps the Regular minimum from the groove), every slider
// against the plates, the other sliders and its own flag, and every flag against the plates and the other flags.
func TestSevenSegmentAllDigits(t *testing.T) {
	startDigit0()
	t.Parallel()
	parts := builtDigit0(t)
	dm := parts.Dimensions
	var cams, sliders, yellows, blacks [7]*decad.Mesh
	var fixed [3]meshgap.Mesh
	for _, s := range sevensegment.Segments() {
		cams[s] = rawMeshOf(t, parts.Cams[s], 0.05)
		sliders[s] = rawMeshOf(t, parts.Sliders[s], 0.05)
		yellows[s] = rawMeshOf(t, parts.Flags[s].Yellow, 0.05)
		blacks[s] = rawMeshOf(t, parts.Flags[s].Black, 0.05)
	}
	_, fixed[0] = meshOf(t, parts.TopPlate, 0.05)
	_, fixed[1] = meshOf(t, parts.GuidePlate, 0.05)
	_, fixed[2] = meshOf(t, parts.CradlePlate, 0.05)
	shaft := r3.Vec{Z: sevensegment.ShaftZ}
	for k := range 10 {
		var lifted, black, yellow [7]meshgap.Mesh
		for _, s := range sevensegment.Segments() {
			was, _ := sevensegment.Lit(0, s)
			now, _ := sevensegment.Lit(k, s)
			dz := sevensegment.Travel * (b2f(now) - b2f(was))
			lifted[s] = posed(t, sliders[s], shaft, r3.Vec{Y: 1}, 0, dz)
			point, axis := segmentAxis(s)
			dphi := dm.FlagAngle(s, 36*float64(k)) - dm.FlagAngle(s, 0)
			black[s] = posed(t, blacks[s], point, axis, dphi, 0)
			yellow[s] = posed(t, yellows[s], point, axis, dphi, 0)
			cam := posed(t, cams[s], shaft, r3.Vec{Y: 1}, 36*float64(k), 0)
			ok, err := cam.ExceedsClearance(lifted[s], 0.35)
			require.NoError(t, err)
			require.True(t, ok, "digit %d: pin of segment %s closer than 0.35 to its cam", k, s)
			for _, plate := range fixed {
				ok, err := plate.ExceedsClearance(lifted[s], 0)
				require.NoError(t, err)
				require.True(t, ok, "digit %d: slider %s touches a plate", k, s)
				for _, flag := range []meshgap.Mesh{black[s], yellow[s]} {
					ok, err := plate.ExceedsClearance(flag, 0)
					require.NoError(t, err)
					require.True(t, ok, "digit %d: flag %s touches a plate", k, s)
				}
			}
			// The tongue and pin keep their gaps to the notch and pocket walls (T7 is 0.20 on sections, T8 more; the mesh check is
			// coarser, 0.15).
			ok, err = black[s].ExceedsClearance(lifted[s], 0.15)
			require.NoError(t, err)
			require.True(t, ok, "digit %d: slider %s closer than 0.15 to its flag", k, s)
		}
		for i := range lifted {
			for j := i + 1; j < len(lifted); j++ {
				for _, pair := range [][2]meshgap.Mesh{{lifted[i], lifted[j]}, {black[i], black[j]}, {black[i], lifted[j]},
					{lifted[i], black[j]}} {
					ok, err := pair[0].ExceedsClearance(pair[1], 0)
					require.NoError(t, err)
					require.True(t, ok, "digit %d: segments %d and %d touch", k, i, j)
				}
			}
		}
	}
}
