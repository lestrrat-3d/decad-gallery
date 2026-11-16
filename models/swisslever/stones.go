package swisslever

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch"
)

// Pallet stone heights in mm: each stone's section is extruded over z 3.30 to 13.90.
const (
	stoneLow    = 3.30
	stoneLength = 10.6
)

// stoneBody builds one pallet stone in the lever frame about the pallet arbor: the section's locking corner C and
// letting-off corner F are polar points; the top corner Hs lies StoneLength along the locking face and the back corner
// G on the back face's line, with the top parallel to the impulse face.
func (b *builder) stoneBody(what string, st StoneSection, uAngle float64) (*decad.Body, error) {
	s, err := b.flat(what, b.g.P.X, b.g.P.Y, stoneLow)
	if err != nil {
		return nil, err
	}
	ref := xRef(s)
	c := polar(s, ref, st.C.Len(), st.C.Angle())
	f := polar(s, ref, st.F.Len(), st.F.Angle())
	hs := s.CreatePoint(st.Hs.X, st.Hs.Y)
	g := s.CreatePoint(st.G.X, st.G.Y)
	locking := s.CreateLine(c, hs)
	impulse := s.CreateLine(c, f)
	back := s.CreateLine(f, g)
	top := s.CreateLine(g, hs)
	s.AddConstraint(
		sketch.NewAngle(ref, locking, uAngle/deg), sketch.NewDistance(c, hs, StoneLength),
		sketch.NewAngle(ref, back, (uAngle+StoneKey)/deg), sketch.NewParallel(top, impulse),
	)
	body, err := b.extrude(s, 0, stoneLength, what)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// buildStones builds the entry and the exit pallet stone, each pressed into its slot in the lever.
func (b *builder) buildStones() (*decad.Body, *decad.Body, error) {
	entry, err := b.stoneBody("entry stone", b.g.Entry, b.g.UEAngle)
	if err != nil {
		return nil, nil, fmt.Errorf("swisslever: %w", err)
	}
	exit, err := b.stoneBody("exit stone", b.g.Exit, b.g.UXAngle)
	if err != nil {
		return nil, nil, fmt.Errorf("swisslever: %w", err)
	}
	return entry, exit, nil
}
