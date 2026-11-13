package main

import (
	"slices"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
)

func shotsOf(parts []planetaryPart) []string {
	var shots []string
	for _, p := range parts {
		shots = append(shots, p.shot)
	}
	return shots
}

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

func TestCheckKitFlag(t *testing.T) {
	if err := checkKitFlag("planetarygearbox", "planet605-shaft608"); err != nil {
		t.Fatal(err)
	}
	if err := checkKitFlag("spurgear", "planet605-shaft608"); err == nil {
		t.Fatal("-kit with another model succeeded")
	}
	if err := checkKitFlag("spurgear", ""); err != nil {
		t.Fatal(err)
	}
}
