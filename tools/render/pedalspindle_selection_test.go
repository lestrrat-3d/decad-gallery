package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPedalSpindleDriveOmitsUnsupportedBearings(t *testing.T) {
	for _, name := range []string{
		"input-bearing-front", "input-bearing-rear",
		"intermediate-bearing-front", "intermediate-bearing-rear",
		"output-bearing-front", "output-bearing-rear",
		"heel-bearing-front", "heel-bearing-rear",
		"footman-bearing-crank", "footman-bearing-toe",
	} {
		mesh := pedalSpindleMesh{name: name, group: "bearing"}
		require.False(t, pedalSpindlePartVisible(mesh, "drive"), "%s must leave with its support", name)
		require.True(t, pedalSpindlePartVisible(mesh, "hero"), "%s must stay in the assembled view", name)
		require.True(t, pedalSpindlePartVisible(mesh, "guarded"), "%s must stay in the guarded assembly", name)
	}
	for _, mesh := range []pedalSpindleMesh{
		{name: "input-gear", group: "gear"},
		{name: "crank", group: "drive"},
		{name: "input-shaft", group: "steel"},
	} {
		require.True(t, pedalSpindlePartVisible(mesh, "drive"), "%s must stay in the drive", mesh.name)
	}
}
