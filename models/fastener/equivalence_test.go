package fastener_test

// This file is temporary. It proves the preset-driven builds match the old m2bolt, m5bolt, m2nut and m5nut packages.
// It is deleted together with those packages.

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/fastener"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/decad-gallery/models/m5bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m5nut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestEquivalentConstants(t *testing.T) {
	m2 := mustLookup(t, "M2")
	m5 := mustLookup(t, "M5")
	sc := m2.Bolt.SocketCap
	require.Equal(t, m2bolt.ShankDiameter, m2.Bolt.ShankDiameter)
	require.Equal(t, m2bolt.HeadDiameter, sc.HeadDiameter)
	require.Equal(t, m2bolt.HeadHeight, m2.Bolt.HeadHeight)
	require.Equal(t, m2bolt.SocketAcrossFlats, sc.SocketAcrossFlats)
	require.Equal(t, m2bolt.SocketDepth, sc.SocketDepth)
	require.Equal(t, m2bolt.HeadChamferWidth, sc.HeadChamferWidth)
	require.Equal(t, m2bolt.HeadChamferAngle, sc.HeadChamferAngle)
	require.Equal(t, m2bolt.UnderheadRadius, sc.UnderheadRadius)
	require.Equal(t, m2bolt.TipChamfer, m2.Bolt.TipChamfer)
	require.Equal(t, m2bolt.DrillPointAngle, sc.DrillPointAngle)
	require.Equal(t, m2bolt.DrillPointBaseRadius, sc.DrillPointBaseRadius)
	require.Equal(t, m2bolt.PilotDepth, sc.PilotDepth)
	require.Equal(t, m2bolt.StandardLengths(), m2.Bolt.StandardLengths)

	require.Equal(t, m5bolt.ShankDiameter, m5.Bolt.ShankDiameter)
	require.Equal(t, m5bolt.AcrossFlats, m5.Bolt.Hex.AcrossFlats)
	require.Equal(t, m5bolt.HeadHeight, m5.Bolt.HeadHeight)
	require.Equal(t, m5bolt.BearingDiameter, m5.Bolt.Hex.BearingDiameter)
	require.Equal(t, m5bolt.ChamferAngle, fastener.HexChamferAngle)
	require.Equal(t, m5bolt.TipChamfer, m5.Bolt.TipChamfer)
	require.Equal(t, m5bolt.StandardLengths(), m5.Bolt.StandardLengths)
	require.Equal(t, m5bolt.AcrossCorners(), m5.Bolt.Hex.AcrossCorners())

	require.Equal(t, m2nut.BoreDiameter, m2.Nut.BoreDiameter)
	require.Equal(t, m2nut.AcrossFlats, m2.Nut.AcrossFlats)
	require.Equal(t, m2nut.Thickness, m2.Nut.Thickness)
	require.Equal(t, m2nut.BearingDiameter, m2.Nut.BearingDiameter)
	require.Equal(t, m2nut.AcrossCorners(), m2.Nut.AcrossCorners())
	require.Equal(t, m5nut.BoreDiameter, m5.Nut.BoreDiameter)
	require.Equal(t, m5nut.AcrossFlats, m5.Nut.AcrossFlats)
	require.Equal(t, m5nut.Thickness, m5.Nut.Thickness)
	require.Equal(t, m5nut.BearingDiameter, m5.Nut.BearingDiameter)
	require.Equal(t, m5nut.AcrossCorners(), m5.Nut.AcrossCorners())
}

// requireSameBody compares two bodies by counts and scalars only, never by passing a mesh to an assertion.
func requireSameBody(t *testing.T, label string, a, b *decad.Body) {
	t.Helper()
	require.Equal(t, len(a.Faces()), len(b.Faces()), label+" face count")
	ka, kb := faceKinds(a), faceKinds(b)
	require.Equal(t, len(ka), len(kb), label+" kind count")
	for k, n := range ka {
		require.Equal(t, n, kb[k], "%s kind %v", label, k)
	}
	boxA, err := a.Bounds()
	require.NoError(t, err)
	boxB, err := b.Bounds()
	require.NoError(t, err)
	for i, pair := range [][2]float64{
		{boxA.Min.X, boxB.Min.X}, {boxA.Min.Y, boxB.Min.Y}, {boxA.Min.Z, boxB.Min.Z},
		{boxA.Max.X, boxB.Max.X}, {boxA.Max.Y, boxB.Max.Y}, {boxA.Max.Z, boxB.Max.Z},
	} {
		require.LessOrEqual(t, math.Abs(pair[0]-pair[1]), 1e-9, "%s bounds[%d]", label, i)
	}
	va, err := a.Volume()
	require.NoError(t, err)
	vb, err := b.Volume()
	require.NoError(t, err)
	require.LessOrEqual(t, math.Abs(va.Value.Base()-vb.Value.Base()), 1e-9, label+" volume")

	ma, err := a.Tessellate(t.Context(), units.Millimeters(.03))
	require.NoError(t, err)
	mb, err := b.Tessellate(t.Context(), units.Millimeters(.03))
	require.NoError(t, err)
	va2, vb2 := ma.Vertices(), mb.Vertices()
	require.Equal(t, len(va2), len(vb2), label+" vertex count")
	ta, tb := ma.Triangles(), mb.Triangles()
	require.Equal(t, len(ta), len(tb), label+" triangle count")
	mismatch := 0
	for i := range ta {
		if ta[i] != tb[i] {
			mismatch++
		}
	}
	require.Zero(t, mismatch, label+" mismatched triangles")
	maxDist := 0.0
	for i := range va2 {
		d := va2[i].Sub(vb2[i])
		maxDist = math.Max(maxDist, math.Sqrt(d.X*d.X+d.Y*d.Y+d.Z*d.Z))
	}
	require.LessOrEqual(t, maxDist, 1e-9, label+" max vertex distance")
}

func frameFor(t *testing.T, origin r3.Vec) *r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(origin, r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	return &f
}

func TestEquivalentBolts(t *testing.T) {
	type build struct {
		old func(doc *decad.Document, l float64, f *r3.Frame) (*decad.Body, error)
	}
	olds := map[string]build{
		"M2": {func(doc *decad.Document, l float64, f *r3.Frame) (*decad.Body, error) {
			m, err := m2bolt.Build(t.Context(), doc, m2bolt.Params{Length: l, Placement: f})
			if err != nil {
				return nil, err
			}
			return m.Body, nil
		}},
		"M5": {func(doc *decad.Document, l float64, f *r3.Frame) (*decad.Body, error) {
			m, err := m5bolt.Build(t.Context(), doc, m5bolt.Params{Length: l, Placement: f})
			if err != nil {
				return nil, err
			}
			return m.Body, nil
		}},
	}
	placed := map[string]*r3.Frame{
		"M2": frameFor(t, r3.NewVec(10, 5, 0)),
		"M5": frameFor(t, r3.NewVec(10, 20, 30)),
	}
	for _, name := range fastener.Names() {
		for _, l := range mustLookup(t, name).Bolt.StandardLengths {
			frames := []*r3.Frame{nil}
			if l == 10 || l == 25 {
				frames = append(frames, placed[name])
			}
			for _, f := range frames {
				label := name + " bolt"
				oldBody, err := olds[name].old(decad.New(), l, f)
				require.NoError(t, err, label)
				nb, err := fastener.BuildBolt(t.Context(), decad.New(), fastener.BoltParams{Preset: name, Length: l, Placement: f})
				require.NoError(t, err, label)
				requireSameBody(t, label, oldBody, nb.Body)
			}
		}
	}
}

func TestEquivalentNuts(t *testing.T) {
	for _, name := range fastener.Names() {
		for _, f := range []*r3.Frame{nil, frameFor(t, r3.NewVec(10, 5, 0))} {
			var oldBody *decad.Body
			if name == "M2" {
				m, err := m2nut.Build(t.Context(), decad.New(), m2nut.Params{Placement: f})
				require.NoError(t, err)
				oldBody = m.Body
			} else {
				m, err := m5nut.Build(t.Context(), decad.New(), m5nut.Params{Placement: f})
				require.NoError(t, err)
				oldBody = m.Body
			}
			nn, err := fastener.BuildNut(t.Context(), decad.New(), fastener.NutParams{Preset: name, Placement: f})
			require.NoError(t, err)
			requireSameBody(t, name+" nut", oldBody, nn.Body)
		}
	}
}
