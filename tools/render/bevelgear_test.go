package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

func TestBevelPairsBuildConcurrently(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	startupDeadline := time.NewTimer(5 * time.Second)
	defer startupDeadline.Stop()
	spiral, straight := &decad.Mesh{}, &decad.Mesh{}
	build := func(mesh *decad.Mesh) func(context.Context) (bevelPairMesh, error) {
		return func(context.Context) (bevelPairMesh, error) {
			started <- struct{}{}
			<-release
			return bevelPairMesh{pinion: mesh}, nil
		}
	}
	type result struct {
		pairs bevelPairs
		err   error
	}
	done := make(chan result, 1)
	go func() {
		pairs, err := buildBevelPairsWith(t.Context(), build(spiral), build(straight))
		done <- result{pairs, err}
	}()
	for range 2 {
		select {
		case <-started:
		case <-startupDeadline.C:
			t.Fatal("both bevel pair builds did not start within 5 seconds")
		}
	}
	close(release)
	released = true
	finishDeadline := time.NewTimer(5 * time.Second)
	defer finishDeadline.Stop()
	var got result
	select {
	case got = <-done:
	case <-finishDeadline.C:
		t.Fatal("bevel pair builds did not finish within 5 seconds")
	}
	require.NoError(t, got.err)
	require.True(t, got.pairs.spiral.pinion == spiral)
	require.True(t, got.pairs.straight.pinion == straight)
}

func TestBevelPairsFailureCancelsAndJoins(t *testing.T) {
	for _, failing := range []string{"spiral", "straight"} {
		t.Run(failing, func(t *testing.T) {
			failure := errors.New("build failed")
			started, stopped := make(chan struct{}), make(chan struct{})
			wait := func(ctx context.Context) (bevelPairMesh, error) {
				close(started)
				<-ctx.Done()
				close(stopped)
				return bevelPairMesh{}, ctx.Err()
			}
			fail := func(context.Context) (bevelPairMesh, error) {
				<-started
				return bevelPairMesh{}, failure
			}
			spiral, straight := wait, fail
			if failing == "spiral" {
				spiral, straight = fail, wait
			}
			_, err := buildBevelPairsWith(t.Context(), spiral, straight)
			require.ErrorIs(t, err, failure)
			require.ErrorIs(t, err, context.Canceled)
			select {
			case <-stopped:
			default:
				t.Fatal("other build was not joined")
			}
		})
	}
}

func TestBevelPairsParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	build := func(ctx context.Context) (bevelPairMesh, error) { return bevelPairMesh{}, ctx.Err() }
	_, err := buildBevelPairsWith(ctx, build, build)
	require.ErrorIs(t, err, context.Canceled)
}

func TestBevelPairsRealBuild(t *testing.T) {
	pairs, err := buildBevelPairs(t.Context())
	require.NoError(t, err)
	require.NotNil(t, pairs.spiral.pinion)
	require.NotNil(t, pairs.spiral.driving)
	require.NotNil(t, pairs.straight.pinion)
	require.NotNil(t, pairs.straight.driving)
}
