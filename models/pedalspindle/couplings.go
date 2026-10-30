package pedalspindle

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

func (b *builder) localOffset(name string, body *decad.Body, z float64) (*decad.Body, error) {
	return b.transform(name, body, r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
}

func (b *builder) capTool(name string, radius, flat, z0, z1 float64) (*decad.Body, error) {
	body, err := b.rectangle(name, flat, -radius-1, radius+1, radius+1, z1-z0, nil)
	if err != nil {
		return nil, err
	}
	return b.localOffset(name, body, z0)
}

func (b *builder) cut(name string, target, tool *decad.Body) (*decad.Body, error) {
	body, err := decad.Cut(b.ctx, target, tool)
	if err != nil {
		return nil, fmt.Errorf("%s: cut: %w", name, err)
	}
	return body, nil
}

func (b *builder) dBore(name string, target *decad.Body, radius, flat, thickness float64) (*decad.Body, error) {
	cylinder, err := b.disc(name+"-bore-circle", radius, 0, thickness+2)
	if err != nil {
		return nil, err
	}
	cylinder, err = b.localOffset(name+"-bore-circle", cylinder, -1)
	if err != nil {
		return nil, err
	}
	cap, err := b.capTool(name+"-bore-cap", radius, flat, -2, thickness+2)
	if err != nil {
		return nil, err
	}
	tool, err := b.cut(name+"-bore-tool", cylinder, cap)
	if err != nil {
		return nil, err
	}
	return b.cut(name+"-bore", target, tool)
}

func (b *builder) inputShaft(scale float64) (*decad.Body, error) {
	body, err := b.disc("input-shaft", 4, 0, 181*scale)
	if err != nil {
		return nil, err
	}
	front, err := b.capTool("input-shaft-front-flat", 4, 3, 85*scale, 181*scale+1)
	if err != nil {
		return nil, err
	}
	rear, err := b.capTool("input-shaft-rear-flat", 4, 3, -1, 35*scale)
	if err != nil {
		return nil, err
	}
	tool, err := decad.Union(b.ctx, front, rear)
	if err != nil {
		return nil, fmt.Errorf("input-shaft: union flat tools: %w", err)
	}
	return b.cut("input-shaft-flats", body, tool)
}

func (b *builder) intermediateShaft(scale float64) (*decad.Body, error) {
	body, err := b.disc("intermediate-shaft", 4, 0, 109*scale)
	if err != nil {
		return nil, err
	}
	tool, err := b.capTool("intermediate-shaft-flat", 4, 3, 46*scale, 109*scale+1)
	if err != nil {
		return nil, err
	}
	return b.cut("intermediate-shaft-flat", body, tool)
}

func (b *builder) flatSpindle(scale float64) (*decad.Body, error) {
	body, err := b.spindle(scale)
	if err != nil {
		return nil, err
	}
	tool, err := b.capTool("spindle-flat", 2.5, 1.75, 80*scale, 114*scale)
	if err != nil {
		return nil, err
	}
	return b.cut("spindle-flat", body, tool)
}
