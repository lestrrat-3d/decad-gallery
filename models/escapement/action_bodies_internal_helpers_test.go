package escapement

import (
	"sync"
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

var (
	wheelOnce    sync.Once
	wheelTeeth   [][]Pt
	wheelOff     float64
	wheelBuildEr error
)

// realWheelTeeth builds the real escape wheel once and returns its tooth polygons at z 23.5.
func realWheelTeeth(t *testing.T) ([][]Pt, float64) {
	t.Helper()
	wheelOnce.Do(func() {
		g, err := Layout(DefaultParams())
		if err != nil {
			wheelBuildEr = err
			return
		}
		b := newBuilder(t.Context(), NewDocument(), g)
		body, err := b.buildEscapeWheel()
		if err != nil {
			wheelBuildEr = err
			return
		}
		f, _ := r3.NewFrame(r3.NewVec(0, 0, 23.5), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
		o, err := outlineAt(t, body, f)
		if err != nil {
			wheelBuildEr = err
			return
		}
		wheelOff = o.OffSurface.Base()
		wheelTeeth, wheelBuildEr = toothPolys(outerLoop(o.Loops))
	})
	require.NoError(t, wheelBuildEr)
	return wheelTeeth, wheelOff
}
