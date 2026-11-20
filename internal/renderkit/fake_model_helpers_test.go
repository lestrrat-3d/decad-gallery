package renderkit_test

import (
	"context"
	"errors"
	"sync"

	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
	"github.com/lestrrat-3d/solidlens"
)

// shotLog records the options each fake shot received.
type shotLog struct {
	mu    sync.Mutex
	calls []string
	opts  []renderkit.Options
}

func (l *shotLog) record(name string, opts renderkit.Options) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, name)
	l.opts = append(l.opts, opts)
}

// fakeModel returns a model whose custom shot "custom" records its call and whose failing shot "fail" returns an error.
// Its preset "preset" builds an empty scene of 64 × 36 pixels.
func fakeModel(name string, log *shotLog) renderkit.Model {
	return renderkit.Model{
		Name: name,
		Dir:  "models/" + name,
		Presets: map[string]renderkit.Preset{
			"preset": {Width: 64, Height: 36, Scene: func(ctx context.Context) (solidlens.Scene, error) {
				log.record(name+"/preset", renderkit.OptionsFrom(ctx))
				return solidlens.Scene{}, errors.New("scene stub")
			}},
		},
		Shots: map[string]renderkit.ShotFunc{
			"custom": func(ctx context.Context, opts renderkit.Options) error {
				log.record(name+"/custom", opts)
				if renderkit.OptionsFrom(ctx) != opts {
					return errors.New("context options differ from the options argument")
				}
				return nil
			},
			"fail": func(ctx context.Context, opts renderkit.Options) error {
				log.record(name+"/fail", opts)
				return errors.New("boom")
			},
		},
	}
}
