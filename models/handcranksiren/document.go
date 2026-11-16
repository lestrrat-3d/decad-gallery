package handcranksiren

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
)

// NewDocument returns a decad document whose Booleans admit the gearbox's ring and the siren's rotor. It is the
// planetary gearbox's document, with a chain tolerance of 0.1 mm; opts are applied after it. Build needs a document
// made with it.
func NewDocument(opts ...decad.DocumentOption) *decad.Document {
	return planetarygearbox.NewDocument(opts...)
}
