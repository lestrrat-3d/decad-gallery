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

func TestEscapementAction(t *testing.T) {
	teeth, wheelOffSurface := realWheelTeeth(t)
	cases := map[string]func(*Params){
		"default": func(*Params) {},
		"wide": func(p *Params) {
			p.LockAngle, p.LiftAngle, p.Amplitude = 1.25*deg, 3*deg, 6*deg
		},
		"long lock": func(p *Params) {
			p.LockAngle, p.LiftAngle, p.Amplitude = 1.5*deg, 2*deg, 6*deg
		},
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			p := DefaultParams()
			edit(&p)
			g, err := Layout(p)
			require.NoError(t, err)
			b := newBuilder(t.Context(), NewDocument(), g)
			body, err := b.buildAnchor()
			require.NoError(t, err)
			f, _ := r3.NewFrame(r3.NewVec(0, 0, 23.5), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
			o, err := outlineAt(t, body, f)
			require.NoError(t, err)
			anchor := simplify(outerLoop2(o.Loops, g.P), 0.004)
			c := &checker{g: &g, in: actionInput{teeth: teeth, anchor: anchor}}
			res, err := c.run()
			require.NoError(t, err)
			checkAction(t, &g, res, wheelOffSurface+o.OffSurface.Base())
			t.Logf("advance %v lock %v lift %v others %.3f root %.3f top %.3f recoil %g", res.periodAdvance,
				res.lock, res.lift, res.others, res.rootGap, res.topGap, res.maxRecoil)
		})
	}
}
