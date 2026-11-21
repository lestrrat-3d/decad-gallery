package sevensegment

import (
	"fmt"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// The part tests only read the bodies they build, so each part is built once per test binary, in a document and run
// of its own, and shared. A test that moves a part transforms it into a new body and leaves the shared one as it is.

// sharedEntry is one cached build: the run that built it (its sketches and dimensions) and what it returned.
type sharedEntry struct {
	once sync.Once
	r    *run
	body *decad.Body
	flag Flag
	// The rotor split: the yellow and black bodies before the flat, notches and stubs, and the rotor's volume.
	splitYellow, splitBlack *decad.Body
	rotorVolume             float64
	err                     error
}

var sharedParts sync.Map // key string -> *sharedEntry

// sharedRun returns the cache entry for key, building it with build in a fresh run for p the first time.
func sharedRun(t *testing.T, p Params, key string, build func(r *run, e *sharedEntry) error) *sharedEntry {
	t.Helper()
	v, _ := sharedParts.LoadOrStore(fmt.Sprintf("%s@%d", key, p.Digit), &sharedEntry{})
	e := v.(*sharedEntry)
	e.once.Do(func() {
		dm, err := p.Derive()
		if err != nil {
			e.err = err
			return
		}
		e.r = &run{ctx: t.Context(), doc: NewDocument(), p: p, dm: dm}
		// The run's context ends with the test that built it: a later build in a shared run fails instead of adding to
		// it, so tests take every part from these helpers.
		e.err = build(e.r, e)
	})
	require.NoError(t, e.err, key)
	return e
}

// sharedBody returns the body build makes for the default parameters, built once, and the run that built it.
func sharedBody(t *testing.T, key string, build func(*run) (*decad.Body, error)) (*run, *decad.Body) {
	t.Helper()
	return sharedBodyFor(t, DefaultParams(), key, build)
}

// sharedBodyFor is sharedBody for the parameters p.
func sharedBodyFor(t *testing.T, p Params, key string, build func(*run) (*decad.Body, error)) (*run, *decad.Body) {
	t.Helper()
	e := sharedRun(t, p, key, func(r *run, e *sharedEntry) error {
		var err error
		e.body, err = build(r)
		return err
	})
	return e.r, e.body
}

// sharedCam is segment s's cam, built once.
func sharedCam(t *testing.T, s Segment) (*run, *decad.Body) {
	t.Helper()
	return sharedBody(t, "cam "+s.String(), func(r *run) (*decad.Body, error) { return r.cam(s) })
}

// sharedSlider is segment s's flag slider, built once.
func sharedSlider(t *testing.T, s Segment) (*run, *decad.Body) {
	t.Helper()
	return sharedBody(t, "slider "+s.String(), func(r *run) (*decad.Body, error) { return r.flagSlider(s) })
}

// sharedWall is the front or rear wall, built once.
func sharedWall(t *testing.T, front bool) (*run, *decad.Body) {
	t.Helper()
	if front {
		return sharedBody(t, "front wall", func(r *run) (*decad.Body, error) { return r.wall(true) })
	}
	return sharedBody(t, "rear wall", func(r *run) (*decad.Body, error) { return r.wall(false) })
}

// sharedFlag is segment s's flag (its yellow and black bodies), built once.
func sharedFlag(t *testing.T, s Segment) Flag {
	t.Helper()
	e := sharedRun(t, DefaultParams(), "flag "+s.String(), func(r *run, e *sharedEntry) error {
		var err error
		e.flag, err = r.flag(s)
		return err
	})
	return e.flag
}

// sharedRotorSplit is r.rotorSplit, built once.
func sharedRotorSplit(t *testing.T) (yellow, black *decad.Body, rotorVolume float64) {
	t.Helper()
	e := sharedRun(t, DefaultParams(), "rotor split", func(r *run, e *sharedEntry) error {
		var err error
		e.splitYellow, e.splitBlack, e.rotorVolume, err = r.rotorSplit()
		return err
	})
	return e.splitYellow, e.splitBlack, e.rotorVolume
}
