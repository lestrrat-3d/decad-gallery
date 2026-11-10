package main

import (
	"context"
	"errors"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
)

func TestPlanetaryReadmeConcurrentBuilds(t *testing.T) {
	box, params, fastener, err := buildPlanetaryReadmeGearboxes(t.Context(), buildPlanetaryGearbox)
	if err != nil {
		t.Fatal(err)
	}
	if params.Stages != 2 || len(box.Named()) != 40 {
		t.Fatalf("README gearbox has %d stages and %d named bodies", params.Stages, len(box.Named()))
	}
	if len(fastener.Stages) != 1 || len(fastener.Named()) == 0 {
		t.Fatalf("fastener gearbox has %d stages and %d named bodies", len(fastener.Stages), len(fastener.Named()))
	}
}

func TestPlanetaryReadmeBuildFailureJoinsWork(t *testing.T) {
	firstErr := errors.New("two-stage build failed")
	secondErr := errors.New("one-stage build stopped")
	started := make(chan struct{})
	stopped := make(chan struct{})
	build := func(ctx context.Context, stages int) (planetarygearbox.Gearbox, planetarygearbox.Params, error) {
		if stages == 1 {
			close(started)
			<-ctx.Done()
			close(stopped)
			return planetarygearbox.Gearbox{}, planetarygearbox.Params{}, secondErr
		}
		<-started
		return planetarygearbox.Gearbox{}, planetarygearbox.Params{}, firstErr
	}
	_, _, _, err := buildPlanetaryReadmeGearboxes(t.Context(), build)
	if !errors.Is(err, firstErr) || !errors.Is(err, secondErr) {
		t.Fatalf("build error does not contain both failures: %v", err)
	}
	select {
	case <-stopped:
	default:
		t.Fatal("one-stage build was not joined")
	}
}
