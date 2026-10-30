package pedalspindle

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
)

func (b *builder) cut(target, tool *decad.Body) *decad.Body {
	if b.err != nil {
		return nil
	}
	body, err := decad.Cut(b.ctx, target, tool)
	if err != nil {
		b.err = fmt.Errorf("cut body: %w", err)
	}
	return body
}

func (b *builder) copy(body *decad.Body) *decad.Body {
	if b.err != nil {
		return nil
	}
	c, err := decad.Copy(b.ctx, body)
	if err != nil {
		b.err = fmt.Errorf("copy body: %w", err)
	}
	return c
}

func (b *builder) dBore(target *decad.Body, radius, flat, thickness float64) *decad.Body {
	tool := b.disc(radius, 0, thickness+2, -1)
	cap := b.rect(flat, -radius-1, radius+1, radius+1, thickness+4, -2)
	tool = b.cut(tool, cap)
	return b.cut(target, tool)
}

func (b *builder) flats(target *decad.Body, radius, flat float64, spans [][2]float64) *decad.Body {
	if b.err != nil {
		return nil
	}
	tools := make([]*decad.Body, 0, len(spans))
	for _, span := range spans {
		tools = append(tools, b.rect(flat, -radius-1, radius+1, radius+1, span[1]-span[0], span[0]))
		if b.err != nil {
			return nil
		}
	}
	tool := tools[0]
	if len(tools) > 1 {
		var err error
		tool, err = decad.Union(b.ctx, tool, tools[1:]...)
		if err != nil {
			b.err = fmt.Errorf("union shaft flat tools: %w", err)
			return nil
		}
	}
	return b.cut(target, tool)
}

func (b *builder) gears(a, phaseB, phaseC float64) {
	if b.err != nil {
		return
	}
	s := b.scale
	p := spurgear.DefaultParams()
	p.Module = 3 * s
	p.Thickness = 12 * s
	p.PressureAngle = 20 * math.Pi / 180
	p.Samples = 15
	p.CapChamfer = 0
	p.BoreDiameter = 0
	p.Teeth = 51
	large, err := spurgear.Build(b.ctx, b.doc, p)
	if err != nil {
		b.err = fmt.Errorf("construct 51-tooth gear: %w", err)
		b.add("intermediate-gear", "gear", nil)
		return
	}
	large = b.dBore(large, 4.1, 3.1, 12*s)
	input := b.copy(large)
	b.add("input-gear", "gear", b.place(input, -90*s, -30*s, 380*s, a))
	b.add("intermediate-gear", "gear", b.place(large, 12.3*s, -60*s, 380*s, phaseB))
	if b.err != nil {
		return
	}
	p.Teeth = 17
	small, err := spurgear.Build(b.ctx, b.doc, p)
	if err != nil {
		b.err = fmt.Errorf("construct 17-tooth gear: %w", err)
		b.add("output-pinion", "gear", nil)
		return
	}
	intermediate := b.copy(small)
	intermediate = b.dBore(intermediate, 4.1, 3.1, 12*s)
	small = b.dBore(small, 2.6, 1.85, 12*s)
	b.add("intermediate-pinion", "gear", b.place(intermediate, 12.3*s, -30*s, 380*s, phaseB))
	b.add("output-pinion", "gear", b.place(small, 12.3*s, -60*s, 482.3*s, phaseC))
}
