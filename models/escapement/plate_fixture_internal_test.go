package escapement

import (
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

var backPlateFixture struct {
	once sync.Once
	body *decad.Body
	g    Geometry
	err  error
}

var frontPlateFixture struct {
	once sync.Once
	body *decad.Body
	g    Geometry
	err  error
}

func defaultBackPlate(t *testing.T) (*decad.Body, Geometry) {
	t.Helper()
	backPlateFixture.once.Do(func() {
		backPlateFixture.g, backPlateFixture.err = Layout(DefaultParams())
		if backPlateFixture.err != nil {
			return
		}
		b := newBuilder(t.Context(), NewDocument(), backPlateFixture.g)
		backPlateFixture.body, backPlateFixture.err = b.buildBackPlate()
	})
	require.NoError(t, backPlateFixture.err)
	return backPlateFixture.body, backPlateFixture.g
}

func defaultFrontPlate(t *testing.T) (*decad.Body, Geometry) {
	t.Helper()
	frontPlateFixture.once.Do(func() {
		frontPlateFixture.g, frontPlateFixture.err = Layout(DefaultParams())
		if frontPlateFixture.err != nil {
			return
		}
		b := newBuilder(t.Context(), NewDocument(), frontPlateFixture.g)
		frontPlateFixture.body, frontPlateFixture.err = b.buildFrontPlate()
	})
	require.NoError(t, frontPlateFixture.err)
	return frontPlateFixture.body, frontPlateFixture.g
}
