package turbinefan_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
)

func eachClutchCase(t *testing.T, fn func(t *testing.T, c fanCase, m *turbinefan.Model, doc *decad.Document)) {
	t.Helper()
	for _, c := range clutchCases {
		t.Run(c.name, func(t *testing.T) {
			m, doc := built(t, c)
			fn(t, c, m, doc)
		})
	}
}
