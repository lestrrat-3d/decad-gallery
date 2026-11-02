package spinningwheel

import (
	"errors"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestPreviewContinuesWithoutPublishingFailedPart(t *testing.T) {
	doc := NewDocument()
	b, err := newAssemblyBuilder(t.Context(), doc, DefaultParams())
	require.NoError(t, err)
	preview := &Preview{Assembly: &b.a}
	b.preview = preview
	plane := frame(r3.Vec{}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	b.part("failed", "frame", "fixed", func() *decad.Body {
		body := b.cylinder(2, 3, plane)
		b.fail("construct", errors.New("deliberate source refusal"))
		return body
	})
	var completed *decad.Body
	b.part("completed", "frame", "fixed", func() *decad.Body {
		completed = b.cylinder(1, 2, plane)
		return completed
	})
	require.NoError(t, b.err)
	require.Len(t, preview.Issues, 1)
	require.Equal(t, "failed", preview.Issues[0].Part)
	require.Len(t, preview.Assembly.Parts, 1)
	require.Equal(t, "completed", preview.Assembly.Parts[0].Name)
	require.True(t, preview.Assembly.Parts[0].Body == completed)
	// A failed constructor can leave tools in its document. They must not enter
	// the registered part list consumed by the preview renderer.
	require.Equal(t, 2, len(doc.Bodies()))
}

func TestStrictPartFailureStillStopsConstruction(t *testing.T) {
	b, err := newAssemblyBuilder(t.Context(), NewDocument(), DefaultParams())
	require.NoError(t, err)
	cause := errors.New("deliberate source refusal")
	b.part("failed", "frame", "fixed", func() *decad.Body {
		b.fail("construct", cause)
		return nil
	})
	called := false
	b.part("later", "frame", "fixed", func() *decad.Body {
		called = true
		return nil
	})
	require.ErrorIs(t, b.err, cause)
	require.False(t, called)
	require.Empty(t, b.a.Parts)
}
