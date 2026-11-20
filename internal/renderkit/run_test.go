package renderkit_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
	"github.com/stretchr/testify/require"
)

func TestRunRejectsKitWhenModelDoesNotAcceptIt(t *testing.T) {
	log := &shotLog{}
	err := renderkit.Run(t.Context(), fakeModel("alpha", log), "custom", renderkit.Options{Output: "x", Kit: "k"})
	require.EqualError(t, err, "-kit is not supported by -model alpha")
	require.Empty(t, log.calls)
}

func TestRunAcceptsKitWhenModelAllowsIt(t *testing.T) {
	log := &shotLog{}
	m := fakeModel("alpha", log)
	m.AcceptsKit = true
	require.NoError(t, renderkit.Run(t.Context(), m, "custom", renderkit.Options{Output: "x", Kit: "k"}))
	require.Equal(t, "k", log.opts[0].Kit)
}

func TestRunRejectsReadmeSizes(t *testing.T) {
	m := fakeModel("alpha", &shotLog{})
	m.Shots["readme"] = m.Shots["custom"]
	err := renderkit.Run(t.Context(), m, "readme", renderkit.Options{Output: "x", Width: 10})
	require.EqualError(t, err, "readme output uses fixed image sizes")
	err = renderkit.Run(t.Context(), m, "readme", renderkit.Options{Output: "x", Height: 10})
	require.EqualError(t, err, "readme output uses fixed image sizes")
}

func TestRunRequiresOutput(t *testing.T) {
	err := renderkit.Run(t.Context(), fakeModel("alpha", &shotLog{}), "custom", renderkit.Options{})
	require.EqualError(t, err, "-out is required")
}

func TestRunPassesOptionsInContext(t *testing.T) {
	log := &shotLog{}
	opts := renderkit.Options{Output: "x", Width: 3, Height: 4, Frames: 5}
	require.NoError(t, renderkit.Run(t.Context(), fakeModel("alpha", log), "custom", opts))
	require.Equal(t, []renderkit.Options{opts}, log.opts)
}

func TestRunPrefersCustomShotOverPreset(t *testing.T) {
	log := &shotLog{}
	m := fakeModel("alpha", log)
	m.Presets["custom"] = m.Presets["preset"]
	require.NoError(t, renderkit.Run(t.Context(), m, "custom", renderkit.Options{Output: "x"}))
	require.Equal(t, []string{"alpha/custom"}, log.calls)
}

func TestRunRejectsNegativeSize(t *testing.T) {
	err := renderkit.Run(t.Context(), fakeModel("alpha", &shotLog{}), "preset", renderkit.Options{Output: "x", Width: -1})
	require.EqualError(t, err, "image dimensions must be positive")
}

func TestRunWrapsSceneError(t *testing.T) {
	log := &shotLog{}
	err := renderkit.Run(t.Context(), fakeModel("alpha", log), "preset", renderkit.Options{Output: "x"})
	require.EqualError(t, err, "build alpha/preset scene: scene stub")
	require.Equal(t, []string{"alpha/preset"}, log.calls)
}

func TestRunRejectsUnknownShot(t *testing.T) {
	err := renderkit.Run(t.Context(), fakeModel("alpha", &shotLog{}), "nope", renderkit.Options{Output: "x"})
	require.EqualError(t, err, `unknown image preset "nope" for model "alpha"`)
}

func TestOptionsFromWithoutOptionsIsZero(t *testing.T) {
	require.Equal(t, renderkit.Options{}, renderkit.OptionsFrom(t.Context()))
}
