package main

// CI: isolated-tests

import (
	"slices"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
)

func TestPlanetaryKitParts(t *testing.T) {
	defaults := shotsOf(planetaryParts)
	same, err := planetarygearbox.DefaultParamsFor("planet608-shaft608")
	if err != nil {
		t.Fatal(err)
	}
	if got := shotsOf(planetaryKitParts(same)); !slices.Equal(got, defaults) {
		t.Fatalf("matching bearings: got %v, want %v", got, defaults)
	}
	mixed, err := planetarygearbox.DefaultParamsFor("planet605-shaft608")
	if err != nil {
		t.Fatal(err)
	}
	got := shotsOf(planetaryKitParts(mixed))
	var want []string
	for _, shot := range defaults {
		if shot == "bearing" {
			want = append(want, "planet-bearing", "cover-bearing")
			continue
		}
		want = append(want, shot)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("mixed bearings: got %v, want %v", got, want)
	}
}
