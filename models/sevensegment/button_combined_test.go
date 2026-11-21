package sevensegment

// CI: isolated-tests

import (
	"fmt"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func combinedMesh(t *testing.T, b *decad.Body) meshgap.Mesh {
	t.Helper()
	m, e := b.Tessellate(t.Context(), units.Millimeters(.05), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, e)
	out, e := meshgap.New(m.Vertices(), m.Triangles(), m.Bound().Base())
	require.NoError(t, e)
	return out
}

// TestCombinedMergedButton samples independent input phases and release lifts.
// It checks these components only; it does not model the complete button cycle.
func TestCombinedMergedButton(t *testing.T) {
	for _, pose := range [][2]float64{{0, 0}, {108, 1}, {180, 2}, {252, 1}, {360, 0}} {
		t.Run(fmt.Sprintf("phase_%g_lift_%g", pose[0], pose[1]), func(t *testing.T) {
			doc := decad.New()
			ctx := t.Context()
			idx, err := buildButtonIndex(ctx, doc, pose[0])
			require.NoError(t, err)
			det, err := buildButtonDetent(ctx, doc, pose[0])
			require.NoError(t, err)
			rel, err := buildButtonReleaseCrank(ctx, doc, pose[1])
			require.NoError(t, err)
			require.NoError(t, doc.VerifyContainers().Err())
			others := []struct {
				name string
				body *decad.Body
			}{{"wheel", idx.Wheel}, {"index crank", idx.Crank}, {"rivet", idx.Rivet}, {"shaft", idx.Shaft}, {"index key", idx.Key}, {"detent", det.Cam}, {"detent key", det.Key}}
			releases := []struct {
				name string
				body *decad.Body
			}{{"release crank", rel.Crank}, {"input pin", rel.Input}, {"output pin", rel.Output}, {"bias pin", rel.Bias}, {"pivot", rel.Pivot}}
			for _, a := range releases {
				am := combinedMesh(t, a.body)
				for _, b := range others {
					bm := combinedMesh(t, b.body)
					gap, ok, e := am.Exceeds(bm, .15)
					require.NoError(t, e)
					if !ok {
						t.Errorf("%s vs %s: gap %.6f", a.name, b.name, gap)
					}
				}
			}
			// The shaft and detent key have a line-to-line fit; their sampled
			// containment is checked separately in TestButtonDetentRealFit.
			for _, pair := range [][2]int{{3, 5}, {4, 5}} {
				am := combinedMesh(t, others[pair[0]].body)
				bm := combinedMesh(t, others[pair[1]].body)
				gap, ok, e := am.Exceeds(bm, 0)
				require.NoError(t, e)
				if !ok {
					t.Errorf("mating %s vs %s boundary %.6f", others[pair[0]].name, others[pair[1]].name, gap)
				}
			}
			for _, a := range others[:3] {
				am := combinedMesh(t, a.body)
				for _, b := range others[5:] {
					bm := combinedMesh(t, b.body)
					gap, ok, e := am.Exceeds(bm, .15)
					require.NoError(t, e)
					if !ok {
						t.Errorf("%s vs %s: gap %.6f", a.name, b.name, gap)
					}
				}
			}
		})
	}
}
