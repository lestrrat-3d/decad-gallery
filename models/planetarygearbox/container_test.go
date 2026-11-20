package planetarygearbox_test

// CI: isolated-tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBodyContainers(t *testing.T) {
	doc, box := defaultGearbox(t)
	require.Len(t, box.Stages, 1)
	requireContainers(t, doc, box)
}
