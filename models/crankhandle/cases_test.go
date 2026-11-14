package crankhandle_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
)

// crankCase is one parameter case. Every case changes the crank's outline or its bore.
type crankCase struct {
	name   string
	params func() crankhandle.Params
}

var crankCases = []crankCase{
	{"605ZZ", crankhandle.Params605ZZ},
	{"608-2RS", crankhandle.Params6082RS},
	{"regular-fit", func() crankhandle.Params {
		p := crankhandle.Params605ZZ()
		p.BoreClearance = 0.35
		return p
	}},
	{"long-thin", func() crankhandle.Params {
		p := crankhandle.Params605ZZ()
		p.Throw, p.Thickness, p.GripDiameter, p.GripLength, p.FlatDepth = 100, 8, 18, 45, 1
		return p
	}},
}

// builtCase is one case's build, shared by the tests: Build takes seconds and the tests only read its bodies.
type builtCase struct {
	m   *crankhandle.Model
	doc *decad.Document
}

var builtCases = map[string]builtCase{}

// built returns the case's build, building it on first use.
func built(t *testing.T, c crankCase) (*crankhandle.Model, *decad.Document) {
	t.Helper()
	if b, ok := builtCases[c.name]; ok {
		return b.m, b.doc
	}
	m, doc := build(t, c.params())
	builtCases[c.name] = builtCase{m: m, doc: doc}
	return m, doc
}
