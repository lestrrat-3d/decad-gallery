package ballbearing_test

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/stretchr/testify/require"
)

func TestBearingCatalog(t *testing.T) {
	cat := ballbearing.Catalog()
	names := make([]string, 0, len(cat))
	for _, s := range cat {
		names = append(names, s.Name)
	}
	require.Equal(t, []string{"605ZZ", "608ZZ"}, names)
	require.Equal(t, ballbearing.Params605ZZ(), cat[0].Params())
	require.Equal(t, ballbearing.Params608ZZ(), cat[1].Params())

	for i, want := range [][3]float64{{5, 14.1, 5.1}, {8, 22.1, 8.1}} {
		s := cat[i]
		require.InDelta(t, want[0], s.ShaftDiameter(), 1e-9, s.Name)
		require.InDelta(t, want[1], s.SeatDiameter(), 1e-9, s.Name)
		require.InDelta(t, want[2], s.PressBoreDiameter(), 1e-9, s.Name)
	}

	got, ok := ballbearing.Lookup("608ZZ")
	require.True(t, ok)
	require.Equal(t, "608ZZ", got.Name)
	_, ok = ballbearing.Lookup("609ZZ")
	require.False(t, ok)
	_, ok = ballbearing.Lookup("605zz")
	require.False(t, ok)

	cat[0].Name = "changed"
	require.Equal(t, "605ZZ", ballbearing.Catalog()[0].Name)
}
