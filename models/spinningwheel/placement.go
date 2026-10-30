package spinningwheel

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

func (b *builder) place(body *decad.Body, f r3.Frame) *decad.Body {
	if b.err != nil {
		return nil
	}

	transform, err := r3.FromFrame(f)
	if err != nil {
		b.fail("create placement transform", err)
		return nil
	}

	placed, err := decad.Transform(b.ctx, body, transform)
	if err != nil {
		b.fail("transform body", err)
		return nil
	}
	return placed
}
