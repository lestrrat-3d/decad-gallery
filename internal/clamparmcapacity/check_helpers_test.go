package clamparmcapacity_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func documentation(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../models/clamparm/instructions.md")
	require.NoError(t, err)
	return string(data)
}
