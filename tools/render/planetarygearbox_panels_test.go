package main

import (
	"context"
	"errors"
	"image"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlanetaryPanelsOrderAndLimit(t *testing.T) {
	started := make(chan int, 3)
	release := make(chan struct{})
	var active, peak atomic.Int32
	render := func(_ context.Context, i int) (image.Image, error) {
		n := active.Add(1)
		for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
		}
		started <- i
		if i < 2 {
			<-release
		}
		active.Add(-1)
		return image.NewRGBA(image.Rect(0, 0, i+1, 1)), nil
	}
	done := make(chan []image.Image, 1)
	go func() {
		images, err := renderPlanetaryPanels(t.Context(), 3, render)
		if err != nil {
			t.Error(err)
		}
		done <- images
	}()
	first, second := <-started, <-started
	close(release)
	images := <-done
	require.NotEqual(t, first, second)
	require.Less(t, first, 2)
	require.Less(t, second, 2)
	require.Len(t, images, 3)
	for i, img := range images {
		require.Equal(t, i+1, img.Bounds().Dx())
	}
	require.EqualValues(t, 2, peak.Load())
}

func TestPlanetaryPanelsFailureCancelsAndJoins(t *testing.T) {
	failure := errors.New("panel failed")
	started, stopped := make(chan struct{}), make(chan struct{})
	render := func(ctx context.Context, i int) (image.Image, error) {
		if i == 0 {
			close(started)
			<-ctx.Done()
			close(stopped)
			return nil, ctx.Err()
		}
		<-started
		return nil, failure
	}
	_, err := renderPlanetaryPanels(t.Context(), 3, render)
	require.ErrorIs(t, err, failure)
	require.ErrorIs(t, err, context.Canceled)
	select {
	case <-stopped:
	default:
		t.Fatal("panel worker was not joined")
	}
}

func TestPlanetaryPanelsParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := renderPlanetaryPanels(ctx, 3, func(ctx context.Context, _ int) (image.Image, error) {
		return nil, ctx.Err()
	})
	require.ErrorIs(t, err, context.Canceled)
}
