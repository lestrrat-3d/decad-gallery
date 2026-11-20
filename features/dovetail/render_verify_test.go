package dovetail_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/features/dovetail"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestDefaultDemoVerifyAllAtRenderTolerance tessellates both blocks of the default demo at 0.01 mm under VerifyAll. The
// renderer meshes them without verification at that tolerance.
func TestDefaultDemoVerifyAllAtRenderTolerance(t *testing.T) {
	demo, err := dovetail.BuildDemo(t.Context(), decad.New(), dovetail.DefaultParams())
	require.NoError(t, err)
	for name, body := range map[string]*decad.Body{"tail": demo.TailBlock, "socket": demo.SocketBlock} {
		_, err := body.Tessellate(t.Context(), units.Millimeters(.01), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, name)
	}
}
