package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

func TestReleaseCrankRealAssembly(t *testing.T) {
	doc := decad.New()
	_, err := Build(t.Context(), doc, Params{Digit: 9})
	if err != nil {
		t.Fatal(err)
	}
	_, err = buildButtonIndex(t.Context(), doc, 0)
	if err != nil {
		t.Fatal(err)
	}
	retained := doc.Bodies()
	for _, lift := range []float64{0, .5, 1, 1.5, 2} {
		release, err := buildButtonReleaseCrank(t.Context(), doc, lift)
		if err != nil {
			t.Fatal(err)
		}
		if err := doc.VerifyContainers().Err(); err != nil {
			t.Fatal(err)
		}
		bodies := []*decad.Body{release.Crank, release.Input, release.Output, release.Bias, release.Pivot}
		unique := make(map[*decad.Body]struct{}, len(bodies))
		for _, b := range bodies {
			if b == nil {
				t.Fatalf("lift %g: nil body", lift)
			}
			unique[b] = struct{}{}
		}
		for i, b := range bodies {
			container, e := doc.ContainerOf(b)
			if e != nil {
				t.Fatal(e)
			}
			want := "Release Pins"
			if i == 0 {
				want = "Release Crank"
			}
			if container.Name() != want {
				t.Fatalf("lift %g: body %d in %s, want %s", lift, i, container.Name(), want)
			}
		}
		if len(unique) != len(bodies) {
			t.Fatalf("lift %g: got %d distinct bodies", lift, len(unique))
		}
		meshes := make([]meshgap.Mesh, len(bodies))
		boxes := make([]decad.Box, len(bodies))
		for i, b := range bodies {
			mesh, e := b.Tessellate(t.Context(), units.Millimeters(.002), decad.WithVerification(decad.VerifyAll))
			if e != nil {
				t.Fatalf("release body %d: %v", i, e)
			}
			meshes[i], e = meshgap.New(mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base())
			if e != nil {
				t.Fatal(e)
			}
			boxes[i], e = b.Bounds()
			if e != nil {
				t.Fatal(e)
			}
		}
		for i := 1; i < len(bodies); i++ {
			gap, ok, e := meshes[0].Exceeds(meshes[i], 0)
			if e != nil || !ok {
				t.Fatalf("pin %d clearance %g: %v", i, gap, e)
			}
		}
		for _, b := range retained {
			c, e := doc.ContainerOf(b)
			if e != nil {
				t.Fatal(e)
			}
			skip := false
			for p := c; p != nil; p = p.Parent() {
				if p.Name() == "Knob" || p.Name() == "Ratchet Wheel" || p.Name() == "Detent Plate" {
					skip = true
				}
			}
			if skip {
				continue
			}
			box, e := b.Bounds()
			if e != nil {
				t.Fatal(e)
			}
			for i, other := range boxes {
				if !(other.Max.X+other.Bound.Base() < box.Min.X-box.Bound.Base() || box.Max.X+box.Bound.Base() < other.Min.X-other.Bound.Base()) {
					t.Fatalf("retained %s and release body %d lack x separation", c.Path(), i)
				}
			}
		}
		rot, e := r3.RotationAround(r3.NewVec(0, -26, 10), r3.NewVec(1, 0, 0), units.Radians(math.Asin((1-lift)/10)))
		if e != nil {
			t.Fatal(e)
		}
		frame, e := r3.NewFrame(r3.NewVec(10, -140, 52), r3.NewVec(1, 0, 0), r3.NewVec(0, 0, 1))
		if e != nil {
			t.Fatal(e)
		}
		for i, center := range [][2]float64{{-26, 20}, {-36, 10}, {-26, 15}} {
			for _, v := range []struct{ x, shift float64 }{{-45.45, .15}, {-43.55, -.15}} {
				p := rot.Apply(r3.NewVec(v.x, center[0]+1.15, center[1]))
				world := frame.ToWorld(p)
				anchor := frame.ToWorld(rot.Apply(r3.NewVec(-44.5, center[0]+2, center[1])))
				loc, e := release.Crank.LocatePoint(t.Context(), anchor, units.Millimeters(.002))
				if e != nil || loc != decad.PointInside {
					t.Fatalf("anchor %d %v %v", i, loc, e)
				}
				delta := world.Sub(anchor)
				// A certified interior anchor and a path bounded away from the verified
				// mesh prove the witness lies inside the crank despite an undecided
				// direct point classification at the witness.
				step := math.Sqrt(delta.Dot(delta)) / 32
				min := math.Inf(1)
				for j := 0; j <= 32; j++ {
					pt := anchor.Add(delta.Scale(float64(j) / 32))
					dd, er := meshes[0].PointDistance(pt)
					if er != nil {
						t.Fatal(er)
					}
					min = math.Min(min, dd-meshes[0].Bound()-step/2)
				}
				if min < 1e-7 {
					t.Fatalf("path %d %g", i, min)
				}
				t.Logf("pin %d shift %g minimum certified margin %g", i, v.shift, min)
				shift, e := r3.Translation(r3.NewVec(v.shift, 0, 0))
				if e != nil {
					t.Fatal(e)
				}
				copy, e := decad.Copy(t.Context(), bodies[i+1], decad.WithCopyPlacement(shift))
				if e != nil {
					t.Fatal(e)
				}
				loc, e = copy.LocatePoint(t.Context(), world, units.Millimeters(.001))
				if e != nil || loc != decad.PointInside {
					t.Fatalf("pin witness %d: %s %v", i, loc, e)
				}
				if e := decad.Remove(copy); e != nil {
					t.Fatal(e)
				}
			}
		}
	}
}

func TestReleaseCrankRejectsInvalidInputs(t *testing.T) {
	for _, lift := range []float64{-0.01, 2.01, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := buildButtonReleaseCrank(t.Context(), decad.New(), lift); err == nil {
			t.Fatalf("lift %g accepted", lift)
		}
	}
	if _, err := buildButtonReleaseCrank(nil, decad.New(), 0); err == nil {
		t.Fatal("nil context accepted")
	}
	if _, err := buildButtonReleaseCrank(t.Context(), nil, 0); err == nil {
		t.Fatal("nil document accepted")
	}
}
