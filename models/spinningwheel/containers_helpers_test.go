package spinningwheel_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/stretchr/testify/require"
)

// The wheel builds the same body set for every parameter, so one default build per entry point covers
// each branch that changes the container layout.
var wantGroupContainers = map[string]string{
	"adjustment": "Adjustments", "anchor": "Anchors", "bearing": "Bearings", "bobbin": "Bobbin", "bush": "Bushes",
	"brake-band": "Brake Bands", "brake-guide": "Brake Guides", "collar": "Collars",
	"compliant": "Flexible Connector", "crank": "Cranks", "drive-band": "Drive Bands",
	"fastener": "Fasteners", "flyer": "Flyer", "footman": "Footman", "frame": "Frame", "gimbal": "Gimbal",
	"guide": "Yarn Guides", "hinge": "Hinges", "insert": "Inserts", "key": "Keys", "marker": "Markers",
	"pin": "Pins", "retainer": "Retainers", "shaft": "Shafts", "spacer": "Spacers", "spring": "Springs",
	"treadle": "Treadle", "wheel": "Wheel", "whorl": "Whorls", "yarn": "Yarn", "yarn-pack": "Yarn Pack",
}

func requirePartContainers(t *testing.T, doc *decad.Document, a *spinningwheel.Assembly) {
	t.Helper()
	require.NoError(t, doc.VerifyContainers().Err())
	used := map[string]struct{}{}
	for _, part := range a.Parts {
		want, ok := wantGroupContainers[part.Group]
		require.True(t, ok, "%s has unexpected group %q", part.Name, part.Group)
		container, err := doc.ContainerOf(part.Body)
		require.NoError(t, err, "%s", part.Name)
		require.Equal(t, doc.Root(), container.Parent(), "%s container must sit directly under the root", part.Name)
		require.Equal(t, want, container.Name(), "%s", part.Name)
		used[want] = struct{}{}
	}
	var got []string
	for _, child := range doc.Root().Children() {
		got = append(got, child.Name())
	}
	require.Len(t, got, len(used))
	for _, name := range got {
		require.Contains(t, used, name)
	}
}
