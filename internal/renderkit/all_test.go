package renderkit_test

// CI: isolated-tests

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
	"github.com/stretchr/testify/require"
)

func TestJobPathJoinsRootDirAndOut(t *testing.T) {
	m := fakeModel("alpha", &shotLog{})
	got := renderkit.JobPath("root", m, renderkit.Job{Shot: "custom", Out: "images/a.png"})
	require.Equal(t, filepath.Join("root", "models", "alpha", "images", "a.png"), got)
}

func TestRenderAllRunsJobsInOrder(t *testing.T) {
	log := &shotLog{}
	a, b := fakeModel("alpha", log), fakeModel("beta", log)
	a.Jobs = []renderkit.Job{{Shot: "custom", Out: "images"}, {Shot: "custom", Kit: "", Out: "images/two"}}
	b.Jobs = []renderkit.Job{{Shot: "custom", Out: "images"}}
	var out bytes.Buffer
	require.NoError(t, renderkit.RenderAll(t.Context(), "root", []renderkit.Model{a, b}, &out))
	require.Equal(t, []string{"alpha/custom", "alpha/custom", "beta/custom"}, log.calls)
	require.Equal(t, filepath.Join("root", "models", "alpha", "images", "two"), log.opts[1].Output)
	require.Equal(t, "render alpha custom "+filepath.Join("root", "models", "alpha", "images")+"\n"+
		"render alpha custom "+filepath.Join("root", "models", "alpha", "images", "two")+"\n"+
		"render beta custom "+filepath.Join("root", "models", "beta", "images")+"\n", out.String())
}

func TestRenderAllKeepsGoingAndJoinsErrors(t *testing.T) {
	log := &shotLog{}
	a, b := fakeModel("alpha", log), fakeModel("beta", log)
	a.Jobs = []renderkit.Job{{Shot: "fail", Out: "images"}, {Shot: "custom", Out: "images/two"}}
	b.Jobs = []renderkit.Job{{Shot: "fail", Out: "images"}}
	var out bytes.Buffer
	err := renderkit.RenderAll(t.Context(), "root", []renderkit.Model{a, b}, &out)
	require.Error(t, err)
	require.Contains(t, err.Error(), "alpha/fail: boom")
	require.Contains(t, err.Error(), "beta/fail: boom")
	require.Equal(t, []string{"alpha/fail", "alpha/custom", "beta/fail"}, log.calls)
}

func TestRenderAllPassesJobKit(t *testing.T) {
	log := &shotLog{}
	a := fakeModel("alpha", log)
	a.AcceptsKit = true
	a.Jobs = []renderkit.Job{{Shot: "custom", Kit: "k1", Out: "images/kit"}}
	require.NoError(t, renderkit.RenderAll(t.Context(), "root", []renderkit.Model{a}, &bytes.Buffer{}))
	require.Equal(t, "k1", log.opts[0].Kit)
}

func TestRenderAllReturnsNilWithoutJobs(t *testing.T) {
	require.NoError(t, renderkit.RenderAll(t.Context(), "root", []renderkit.Model{fakeModel("alpha", &shotLog{})}, &bytes.Buffer{}))
}
