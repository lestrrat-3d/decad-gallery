package bevelgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// children returns the names of the child containers of c, in creation order.
func children(c *decad.Container) []string {
	var names []string
	for _, child := range c.Children() {
		names = append(names, child.Name())
	}
	return names
}

// containerPath names the container holding body.
func containerPath(t *testing.T, doc *decad.Document, body *decad.Body) string {
	t.Helper()
	c, err := doc.ContainerOf(body)
	require.NoError(t, err)
	if c.Parent() == nil {
		return "/"
	}
	return c.Parent().Name() + "/" + c.Name()
}

func volumeOf(t *testing.T, body *decad.Body) (float64, float64) {
	t.Helper()
	v, err := body.Volume()
	require.NoError(t, err)
	return v.Value.Base(), v.Bound.Base()
}
