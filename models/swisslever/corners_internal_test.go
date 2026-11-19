package swisslever

// CI: isolated-tests

import (
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestCorners applies "Corners and roots" to every printed body of the posed movement in its print orientation, axis
// +Z. Bearings, bolts and nuts are purchased parts. The arbors are read as split halves by TestSplitShafts, because a
// round shaft has no planar face parallel to the bed for the check to cut. The exemptions match "Retained sharp
// corners" in instructions.md, and a listed corner that matches no finding fails the check.
func TestCorners(t *testing.T) {
	_, m := sharedDefault(t)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	table, err := poses(&g, m.State)
	require.NoError(t, err)
	z := r3.Vec{Z: 1}
	at := func(name string, p Pt) r3.Vec {
		q := table[name].turnPt(p)
		return r3.Vec{X: q.X, Y: q.Y}
	}
	// The lever's slot corners are mating corners of the stones; the bottoms of the two gap rounds may read 0.1° over the
	// inside-corner limit on the section mesh.
	var lever []cornercheck.Exempt
	for _, slot := range [][4]Pt{g.EntrySlot, g.ExitSlot} {
		for _, v := range slot {
			lever = append(lever, cornercheck.Exempt{Reason: "slot corner",
				Near: cornercheck.Disc(at(LeverName, g.P.Add(v)), z, 0.5)})
		}
	}
	for _, c := range []Pt{g.GapRight, g.GapLeft} {
		lever = append(lever, cornercheck.Exempt{Reason: "gap round bottom", Optional: true,
			Near: cornercheck.Disc(at(LeverName, g.P.Add(Pt{c.X, c.Y - g.GapRadius})), z, 0.5)})
	}
	exempt := map[string][]cornercheck.Exempt{LeverName: lever}
	for _, nb := range m.Named() {
		name := nb.Name
		if strings.HasSuffix(name, "Bearing") || strings.HasPrefix(name, "Bolt ") || strings.HasPrefix(name, "Nut ") {
			continue
		}
		switch name {
		case EscapeArborName, PalletArborName, BalanceStaffName:
			continue
		}
		t.Run(name, func(t *testing.T) {
			fs, err := cornercheck.Check(t.Context(), nb.Body, z, exempt[name])
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		})
	}
}
