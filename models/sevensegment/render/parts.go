package render

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/solidlens"
)

type part struct {
	name  string
	mesh  *decad.Mesh
	color solidlens.Color
}
type partSet []part
